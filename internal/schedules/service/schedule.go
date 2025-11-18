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
	GetAll(ctx context.Context, limit int, offset int64) ([]*model.Schedule, int64, error)
	Update(ctx context.Context, id string, updates *model.ScheduleUpdate) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, businessID string, city string, limit int, offset int64) ([]*model.Schedule, int64, error)
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
	s.applyDefaults(sc)
	s.sanitize(sc)
	err := s.verifyLimitPerBusinessUnit(ctx, sc)
	if err != nil {
		return err
	}
	err = s.validate(sc)
	if err != nil {
		return err
	}
	err = s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		err = s.verifyDuplication(sessCtx, sc)
		if err != nil {
			return err
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

func (s *scheduleService) GetAll(ctx context.Context, limit int, offset int64) ([]*model.Schedule, int64, error) {

	var count int64
	var schedules []*model.Schedule
	var errCount, errFind error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error
		count, err = s.repo.Count(ctx)
		if err != nil {
			s.cfg.Log.Error("Failed to count schedules", "error", err)
			errCount = apperrors.Internal("Failed to count schedules", err)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		schedules, err = s.repo.FindAll(ctx, limit, offset)
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
	merged := s.mergeScheduleUpdates(existing, updates)
	s.sanitize(merged)
	err = s.validate(merged)
	if err != nil {
		return err
	}
	err = s.verifyLimitPerBusinessUnit(ctx, merged)
	if err != nil {
		return err
	}

	err = s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		err = s.verifyDuplication(sessCtx, merged)
		if err != nil {
			return err
		}
		_, err := s.repo.Update(sessCtx, id, merged)
		return err
	})
	if err != nil {
		s.cfg.Log.Error("Failed to update schedule",
			"id", id,
			"error", err,
		)
		return apperrors.Internal("Failed to update schedule", err)
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

func (s *scheduleService) Search(ctx context.Context, businessID string, city string, limit int, offset int64) ([]*model.Schedule, int64, error) {
	if businessID == "" {
		return nil, 0, apperrors.InvalidInput("Business_id must be provided, city is optional")
	}

	city = sanitizer.SanitizeCityOrLabel(city)

	var count int64
	var schedules []*model.Schedule
	var errCount, errFind error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error
		count, err = s.repo.CountBySearch(ctx, businessID, city)
		if err != nil {
			s.cfg.Log.Error("Failed to count schedules by search",
				"business_id", businessID,
				"city", city,
				"error", err,
			)
			errCount = apperrors.Internal("Failed to count schedules", err)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		schedules, err = s.repo.Search(ctx, businessID, city, limit, offset)
		if err != nil {
			s.cfg.Log.Error("Failed to search schedules",
				"business_id", businessID,
				"city", city,
				"limit", limit,
				"offset", offset,
				"error", err,
			)
			errFind = apperrors.Internal("Failed to search schedules", err)
		}
	}()

	wg.Wait()

	if errCount != nil {
		return nil, 0, errCount
	}
	if errFind != nil {
		return nil, 0, errFind
	}

	s.cfg.Log.Debug("Schedules search completed",
		"business_id", businessID,
		"city", city,
		"results_count", len(schedules),
		"total_count", count,
	)

	return schedules, count, nil
}

func (s *scheduleService) sanitize(sc *model.Schedule) {
	sc.Name = sanitizer.SanitizeNameOrAddress(sc.Name)
	sc.City = sanitizer.SanitizeCityOrLabel(sc.City)
	sc.Address = sanitizer.SanitizeNameOrAddress(sc.Address)
	sc.WorkingDays = sanitizer.SanitizeSlice(sc.WorkingDays, sanitizer.SanitizeCityOrLabel)
	sc.Exceptions = sanitizer.SanitizeSlice(sc.Exceptions, sanitizer.SanitizeNameOrAddress)
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
	if sc.Exceptions == nil {
		sc.Exceptions = []string{}
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
		merged.Exceptions = append([]string{}, *updates.Exceptions...)
	}
	if updates.TimeZone != "" {
		merged.TimeZone = updates.TimeZone
	}

	merged.ID = existing.ID
	merged.CreatedAt = existing.CreatedAt
	return &merged
}

func (s *scheduleService) validate(sc *model.Schedule) error {
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
	return nil
}

func (s *scheduleService) verifyDuplication(ctx context.Context, sc *model.Schedule) error {
	// For duplicate checking, we fetch with a reasonable limit
	// In practice, a business shouldn't have more than 1000 schedules in a single city
	const maxSchedulesPerCity = config.DefaultPaginationLimit
	existingSchedules, err := s.repo.Search(ctx, sc.BusinessID, sc.City, maxSchedulesPerCity, 0)
	if err != nil {
		return apperrors.Internal("Failed to check for duplicate schedules", err)
	}
	for _, e := range existingSchedules {
		if e.ID == sc.ID {
			continue
		}
		if strings.EqualFold(e.Address, sc.Address) {
			return apperrors.Conflict("Another schedule with the same address already exists for this business")
		}
		if strings.EqualFold(e.Name, sc.Name) && strings.EqualFold(e.City, sc.City) {
			return apperrors.Conflict("Another schedule with the same name and city already exists for this business")
		}
	}
	return nil
}

func (s *scheduleService) verifyLimitPerBusinessUnit(ctx context.Context, sc *model.Schedule) error {
	_, total, err := s.Search(ctx, sc.BusinessID, "", 10, 0)
	if err != nil {
		return err
	}
	if total >= int64(config.DefaultMaxSchedulesPerBusinessUnits) {
		return apperrors.Conflict("Business unit exceeded num of allowed schedules")
	}

	return nil
}
