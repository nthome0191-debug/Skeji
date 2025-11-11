package service

import (
	"context"
	"errors"
	scheduleerrors "skeji/internal/schedules/errors"
	"skeji/internal/schedules/repository"
	"skeji/internal/schedules/validator"
	"skeji/pkg/config"
	apperrors "skeji/pkg/errors"
	"skeji/pkg/locale"
	"skeji/pkg/model"
	"skeji/pkg/sanitizer"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
)

type ScheduleService interface {
	Create(ctx context.Context, sc *model.Schedule) error
	GetByID(ctx context.Context, id string) (*model.Schedule, error)
	GetAll(ctx context.Context, limit int, offset int) ([]*model.Schedule, int64, error)
	Update(ctx context.Context, id string, updates *model.ScheduleUpdate) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, businessID string, city string) ([]*model.Schedule, error)
}

type scheduleService struct {
	repo      repository.ScheduleRepository
	validator *validator.ScheduleValidator
	cfg       *config.Config
}

func NewScheduleService(
	repo repository.ScheduleRepository,
	validator *validator.ScheduleValidator,
	cfg *config.Config,
) ScheduleService {
	return &scheduleService{
		repo:      repo,
		validator: validator,
		cfg:       cfg,
	}
}

func (s *scheduleService) Create(ctx context.Context, sc *model.Schedule) error {
	s.sanitize(sc)
	s.applyDefaults(sc)

	if err := s.validator.Validate(sc); err != nil {
		s.cfg.Log.Warn("Schedule validation failed",
			"name", sc.Name,
			"business_id", sc.BusinessID,
			"error", err,
		)
		return apperrors.Validation("Schedule validation failed", map[string]any{
			"error": err.Error(),
		})
	}

	err := s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		existing, err := s.repo.Search(sessCtx, sc.BusinessID, sc.City)
		if err != nil {
			return apperrors.Internal("Failed to check for existing schedules", err)
		}

		for _, e := range existing {
			if strings.EqualFold(e.Address, sc.Address) {
				return apperrors.Conflict("Schedule with the same address already exists for this business")
			}

			if strings.EqualFold(e.Name, sc.Name) {
				return apperrors.Conflict("Schedule with the same name and city already exists for this business")
			}
		}
		return s.repo.Create(sessCtx, sc)
	})
	if err != nil {
		s.cfg.Log.Error("Failed to create schedule",
			"name", sc.Name,
			"business_id", sc.BusinessID,
			"error", err,
		)
		return err
	}

	s.cfg.Log.Info("Schedule created successfully",
		"id", sc.ID,
		"name", sc.Name,
		"business_id", sc.BusinessID,
		"city", sc.City,
	)
	return nil
}

func (s *scheduleService) GetByID(ctx context.Context, id string) (*model.Schedule, error) {
	if id == "" {
		return nil, apperrors.InvalidInput("Schedule ID cannot be empty")
	}

	sc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, scheduleerrors.ErrNotFound) {
			return nil, apperrors.NotFoundWithID("Schedule", id)
		}
		if errors.Is(err, scheduleerrors.ErrInvalidID) {
			return nil, apperrors.InvalidInput("Invalid schedule ID format")
		}
		s.cfg.Log.Error("Failed to get schedule by ID",
			"id", id,
			"error", err,
		)
		return nil, apperrors.Internal("Failed to retrieve schedule", err)
	}

	return sc, nil
}

func (s *scheduleService) GetAll(ctx context.Context, limit int, offset int) ([]*model.Schedule, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = config.DefaultPaginationLimit
	}
	if offset < 0 {
		offset = 0
	}

	// Create shared context with timeout for both goroutines
	// This ensures coordinated cancellation if one operation times out
	sharedCtx, cancel := context.WithTimeout(ctx, s.cfg.ReadTimeout)
	defer cancel()

	var count int64
	var schedules []*model.Schedule
	var errCount, errFind error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error
		count, err = s.repo.Count(sharedCtx)
		if err != nil {
			s.cfg.Log.Error("Failed to count schedules", "error", err)
			errCount = apperrors.Internal("Failed to count schedules", err)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		schedules, err = s.repo.FindAll(sharedCtx, limit, offset)
		if err != nil {
			s.cfg.Log.Error("Failed to get all schedules",
				"limit", limit,
				"offset", offset,
				"error", err,
			)
			errFind = apperrors.Internal("Failed to retrieve schedules", err)
		}
	}()

	wg.Wait()
	if errCount != nil {
		return nil, 0, errCount
	}
	if errFind != nil {
		return nil, 0, errFind
	}
	return schedules, count, nil
}

func (s *scheduleService) Update(ctx context.Context, id string, updates *model.ScheduleUpdate) error {
	if id == "" {
		return apperrors.InvalidInput("Schedule ID cannot be empty")
	}

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, scheduleerrors.ErrNotFound) {
			return apperrors.NotFoundWithID("Schedule", id)
		}
		if errors.Is(err, scheduleerrors.ErrInvalidID) {
			return apperrors.InvalidInput("Invalid schedule ID format")
		}
		return apperrors.Internal("Failed to check schedule existence", err)
	}

	s.sanitizeUpdate(updates)
	merged := s.mergeScheduleUpdates(existing, updates)
	if err := s.validator.Validate(merged); err != nil {
		s.cfg.Log.Warn("Schedule validation failed",
			"name", merged.Name,
			"business_id", merged.BusinessID,
			"id", id,
			"error", err,
		)
		return apperrors.Validation("Schedule validation failed", map[string]any{
			"error": err.Error(),
		})
	}

	err = s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		existingSchedules, err := s.repo.Search(sessCtx, merged.BusinessID, merged.City)
		if err != nil {
			return apperrors.Internal("Failed to check for duplicate schedules", err)
		}
		for _, e := range existingSchedules {
			if e.ID == merged.ID {
				continue
			}
			if strings.EqualFold(e.Address, merged.Address) {
				return apperrors.Conflict("Another schedule with the same address already exists for this business")
			}
			if strings.EqualFold(e.Name, merged.Name) && strings.EqualFold(e.City, merged.City) {
				return apperrors.Conflict("Another schedule with the same name and city already exists for this business")
			}
		}
		if _, err := s.repo.Update(sessCtx, id, merged); err != nil {
			s.cfg.Log.Error("Failed to update schedule",
				"id", id,
				"error", err,
			)
			return apperrors.Internal("Failed to update schedule", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	s.cfg.Log.Info("Schedule updated successfully", "id", id, "name", merged.Name)
	return nil
}

func (s *scheduleService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return apperrors.InvalidInput("Schedule ID cannot be empty")
	}

	err := s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		if err := s.repo.Delete(sessCtx, id); err != nil {
			if errors.Is(err, scheduleerrors.ErrNotFound) {
				return apperrors.NotFoundWithID("Schedule", id)
			}
			if errors.Is(err, scheduleerrors.ErrInvalidID) {
				return apperrors.InvalidInput("Invalid schedule ID format")
			}
			s.cfg.Log.Error("Failed to delete schedule",
				"id", id,
				"error", err,
			)
			return apperrors.Internal("Failed to delete schedule", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.cfg.Log.Info("Schedule deleted successfully", "id", id)
	return nil
}

func (s *scheduleService) Search(ctx context.Context, businessID string, city string) ([]*model.Schedule, error) {
	if businessID == "" {
		return nil, apperrors.InvalidInput("Business_id must be provided, city is optional")
	}

	city = sanitizer.TrimAndNormalize(city)
	businessID = sanitizer.TrimAndNormalize(businessID)

	schedules, err := s.repo.Search(ctx, businessID, city)
	if err != nil {
		s.cfg.Log.Error("Failed to search schedules",
			"business_id", businessID,
			"city", city,
			"error", err,
		)
		return nil, apperrors.Internal("Failed to search schedules", err)
	}

	s.cfg.Log.Debug("Schedules search completed",
		"business_id", businessID,
		"city", city,
		"results_count", len(schedules),
	)

	return schedules, nil
}

func (s *scheduleService) sanitize(sc *model.Schedule) {
	sc.Name = sanitizer.NormalizeName(sc.Name)
	sc.City = sanitizer.TrimAndNormalize(sc.City)
	sc.Address = sanitizer.TrimAndNormalize(sc.Address)
}

func (s *scheduleService) sanitizeUpdate(updates *model.ScheduleUpdate) {
	if updates.Name != "" {
		updates.Name = sanitizer.NormalizeName(updates.Name)
	}
	if updates.City != "" {
		updates.City = sanitizer.TrimAndNormalize(updates.City)
	}
	if updates.Address != "" {
		updates.Address = sanitizer.TrimAndNormalize(updates.Address)
	}
}

func (s *scheduleService) applyDefaults(sc *model.Schedule) {
	if sc.DefaultMeetingDurationMin == 0 {
		sc.DefaultMeetingDurationMin = s.cfg.DefaultMeetingDurationMin
	}
	if sc.DefaultBreakDurationMin == 0 {
		sc.DefaultBreakDurationMin = s.cfg.DefaultBreakDurationMin
	}
	if sc.MaxParticipantsPerSlot == 0 {
		sc.MaxParticipantsPerSlot = s.cfg.DefaultMaxParticipantsPerSlot
	}
	if sc.StartOfDay == "" {
		sc.StartOfDay = s.cfg.DefaultStartOfDay
	}
	if sc.EndOfDay == "" {
		sc.EndOfDay = s.cfg.DefaultEndOfDay
	}
	if len(sc.WorkingDays) == 0 {
		switch locale.DetectRegion(sc.TimeZone) {
		case "IL":
			sc.WorkingDays = s.cfg.DefaultWorkingDaysIsrael
		case "US":
			sc.WorkingDays = s.cfg.DefaultWorkingDaysUs
		default:
			sc.WorkingDays = s.cfg.DefaultWorkingDaysIsrael
		}
	}
}

func (s *scheduleService) mergeScheduleUpdates(existing *model.Schedule, updates *model.ScheduleUpdate) *model.Schedule {
	merged := *existing

	if updates.Name != "" {
		merged.Name = updates.Name
	}
	if updates.City != "" {
		merged.City = updates.City
	}
	if updates.Address != "" {
		merged.Address = updates.Address
	}
	if updates.StartOfDay != "" {
		merged.StartOfDay = updates.StartOfDay
	}
	if updates.EndOfDay != "" {
		merged.EndOfDay = updates.EndOfDay
	}
	if updates.WorkingDays != nil {
		merged.WorkingDays = updates.WorkingDays
	}
	if updates.DefaultMeetingDurationMin != nil {
		merged.DefaultMeetingDurationMin = *updates.DefaultMeetingDurationMin
	}
	if updates.DefaultBreakDurationMin != nil {
		merged.DefaultBreakDurationMin = *updates.DefaultBreakDurationMin
	}
	if updates.MaxParticipantsPerSlot != nil {
		merged.MaxParticipantsPerSlot = *updates.MaxParticipantsPerSlot
	}
	if updates.Exceptions != nil {
		// Validate that all exception dates are in YYYY-MM-DD format
		for _, date := range *updates.Exceptions {
			if len(date) != 10 || date[4] != '-' || date[7] != '-' {
				s.cfg.Log.Warn("Invalid exception date format",
					"date", date,
					"expected_format", "YYYY-MM-DD",
				)
				// Skip invalid dates
				continue
			}
			// Basic validation: check that it looks like a date
			if date < "1900-01-01" || date > "2100-12-31" {
				s.cfg.Log.Warn("Exception date out of reasonable range",
					"date", date,
				)
				continue
			}
		}
		merged.Exceptions = *updates.Exceptions
	}
	if updates.TimeZone != "" {
		merged.TimeZone = updates.TimeZone
	}

	merged.ID = existing.ID
	merged.CreatedAt = existing.CreatedAt
	return &merged
}
