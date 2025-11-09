package service

import (
	"context"
	"errors"
	"fmt"
	businessunitserrors "skeji/internal/businessunits/errors"
	"skeji/internal/businessunits/repository"
	"skeji/internal/businessunits/validator"
	"skeji/pkg/config"
	apperrors "skeji/pkg/errors"
	"skeji/pkg/locale"
	"skeji/pkg/model"
	"skeji/pkg/sanitizer"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
)

type BusinessUnitService interface {
	Create(ctx context.Context, bu *model.BusinessUnit) error
	GetByID(ctx context.Context, id string) (*model.BusinessUnit, error)
	GetAll(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, int64, error)
	Update(ctx context.Context, id string, updates *model.BusinessUnitUpdate) error
	Delete(ctx context.Context, id string) error

	GetByAdminPhone(ctx context.Context, phone string) ([]*model.BusinessUnit, error)
	Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error)
}

type businessUnitService struct {
	repo      repository.BusinessUnitRepository
	validator *validator.BusinessUnitValidator
	cfg       *config.Config
}

func NewBusinessUnitService(
	repo repository.BusinessUnitRepository,
	validator *validator.BusinessUnitValidator,
	cfg *config.Config,
) BusinessUnitService {
	return &businessUnitService{
		repo:      repo,
		validator: validator,
		cfg:       cfg,
	}
}

func (s *businessUnitService) Create(ctx context.Context, bu *model.BusinessUnit) error {
	s.sanitize(bu)
	s.applyDefaultsForNewCreatedBusiness(bu)

	if err := s.validator.Validate(bu); err != nil {
		s.cfg.Log.Warn("Business unit validation failed",
			"name", bu.Name,
			"admin_phone", bu.AdminPhone,
			"error", err,
		)
		return apperrors.Validation("Business unit validation failed", map[string]any{
			"error": err.Error(),
		})
	}

	err := s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		existing, err := s.repo.FindByAdminPhone(sessCtx, bu.AdminPhone)
		if err != nil {
			return fmt.Errorf("failed to check for duplicates: %w", err)
		}

		for _, existingBU := range existing {
			if s.isDuplicate(bu, existingBU) {
				return apperrors.Conflict(fmt.Sprintf(
					"Business unit with similar details already exists (id: %s)",
					existingBU.ID,
				))
			}
		}

		if err := s.repo.Create(sessCtx, bu); err != nil {
			return fmt.Errorf("failed to create business unit: %w", err)
		}

		return nil
	})

	if err != nil {
		s.cfg.Log.Error("Failed to create business unit",
			"name", bu.Name,
			"admin_phone", bu.AdminPhone,
			"error", err,
		)
		return err
	}

	s.cfg.Log.Info("Business unit created successfully",
		"id", bu.ID,
		"name", bu.Name,
		"admin_phone", bu.AdminPhone,
		"timezone", bu.TimeZone,
	)

	return nil
}

func (s *businessUnitService) GetByID(ctx context.Context, id string) (*model.BusinessUnit, error) {
	if id == "" {
		return nil, apperrors.InvalidInput("Business unit ID cannot be empty")
	}

	bu, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, businessunitserrors.ErrNotFound) {
			return nil, apperrors.NotFoundWithID("Business unit", id)
		}
		if errors.Is(err, businessunitserrors.ErrInvalidID) {
			return nil, apperrors.InvalidInput("Invalid business unit ID format")
		}
		s.cfg.Log.Error("Failed to get business unit by ID",
			"id", id,
			"error", err,
		)
		return nil, apperrors.Internal("Failed to retrieve business unit", err)
	}

	return bu, nil
}

func (s *businessUnitService) GetAll(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var count int64
	var units []*model.BusinessUnit
	var errCount, errFind error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		var err error
		ctx, cancel := context.WithTimeout(ctx, s.cfg.ReadTimeout)
		defer cancel()
		count, err = s.repo.Count(ctx)
		if err != nil {
			s.cfg.Log.Error("Failed to count business units", "error", err)
			errCount = apperrors.Internal("Failed to count business units", err)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		ctx, cancel := context.WithTimeout(ctx, s.cfg.ReadTimeout)
		defer cancel()
		units, err = s.repo.FindAll(ctx, limit, offset)
		if err != nil {
			s.cfg.Log.Error("Failed to get all business units",
				"limit", limit,
				"offset", offset,
				"error", err,
			)
			errFind = apperrors.Internal("Failed to retrieve business units", err)
		}
	}()
	wg.Wait()

	if errCount != nil {
		return nil, 0, errCount
	}
	if errFind != nil {
		return nil, 0, errFind
	}

	return units, count, nil
}

func (s *businessUnitService) Update(ctx context.Context, id string, updates *model.BusinessUnitUpdate) error {
	if id == "" {
		return apperrors.InvalidInput("Business unit ID cannot be empty")
	}

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, businessunitserrors.ErrNotFound) {
			return apperrors.NotFoundWithID("Business unit", id)
		}
		if errors.Is(err, businessunitserrors.ErrInvalidID) {
			return apperrors.InvalidInput("Invalid business unit ID format")
		}
		return apperrors.Internal("Failed to check business unit existence", err)
	}

	s.sanitizeUpdate(updates)
	merged := s.mergeBusinessUnitUpdates(existing, updates)
	err = s.validator.Validate(merged)
	if err != nil {
		s.cfg.Log.Warn("Business unit validation failed",
			"name", merged.Name,
			"admin_phone", merged.AdminPhone,
			"id", id,
			"error", err,
		)
		return apperrors.Validation("Business unit validation failed", map[string]any{
			"error": err.Error(),
		})
	}

	if _, err := s.repo.Update(ctx, id, merged); err != nil {
		s.cfg.Log.Error("Failed to update business unit",
			"id", id,
			"error", err,
		)
		return apperrors.Internal("Failed to update business unit", err)
	}
	s.cfg.Log.Info("Business unit updated successfully",
		"id", id,
		"name", merged.Name,
	)

	return nil
}

func (s *businessUnitService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return apperrors.InvalidInput("Business unit ID cannot be empty")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, businessunitserrors.ErrNotFound) {
			return apperrors.NotFoundWithID("Business unit", id)
		}
		if errors.Is(err, businessunitserrors.ErrInvalidID) {
			return apperrors.InvalidInput("Invalid business unit ID format")
		}
		s.cfg.Log.Error("Failed to delete business unit",
			"id", id,
			"error", err,
		)
		return apperrors.Internal("Failed to delete business unit", err)
	}

	s.cfg.Log.Info("Business unit deleted successfully", "id", id)

	return nil
}

func (s *businessUnitService) GetByAdminPhone(ctx context.Context, phone string) ([]*model.BusinessUnit, error) {
	if phone == "" {
		return nil, apperrors.InvalidInput("Admin phone number cannot be empty")
	}

	phone = sanitizer.NormalizePhone(phone)

	units, err := s.repo.FindByAdminPhone(ctx, phone)
	if err != nil {
		s.cfg.Log.Error("Failed to get business units by admin phone",
			"phone", phone,
			"error", err,
		)
		return nil, apperrors.Internal("Failed to retrieve business units by phone", err)
	}

	return units, nil
}

func (s *businessUnitService) Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error) {
	if len(cities) == 0 || len(labels) == 0 {
		return nil, apperrors.InvalidInput("Both search criteria (cities and labels) must be provided")
	}

	originalCities := append([]string(nil), cities...)
	originalLabels := append([]string(nil), labels...)

	cities = sanitizer.NormalizeCities(cities)
	labels = sanitizer.NormalizeLabels(labels)

	if len(cities) == 0 || len(labels) == 0 {
		s.cfg.Log.Warn("Search criteria normalized to empty",
			"original_cities", originalCities,
			"original_labels", originalLabels,
			"normalized_cities", cities,
			"normalized_labels", labels,
		)
		return nil, apperrors.InvalidInput("Search criteria resulted in no valid items after normalization")
	}

	units, err := s.repo.Search(ctx, cities, labels)
	if err != nil {
		s.cfg.Log.Error("Failed to search business units",
			"cities", cities,
			"labels", labels,
			"error", err,
		)
		return nil, apperrors.Internal("Failed to search business units", err)
	}

	s.cfg.Log.Debug("Business units search completed",
		"cities", cities,
		"labels", labels,
		"results_count", len(units),
	)

	return units, nil
}

func (s *businessUnitService) sanitize(bu *model.BusinessUnit) {
	bu.Name = sanitizer.NormalizeName(bu.Name)
	bu.Cities = sanitizer.NormalizeCities(bu.Cities)
	bu.Labels = sanitizer.NormalizeLabels(bu.Labels)
	bu.AdminPhone = sanitizer.NormalizePhone(bu.AdminPhone)
	bu.Maintainers = sanitizer.NormalizeMaintainers(bu.Maintainers)
	bu.WebsiteURL = sanitizer.NormalizeURL(bu.WebsiteURL)
	bu.Priority = sanitizer.NormalizePriority(s.cfg, bu.Priority)
}

func (s *businessUnitService) sanitizeUpdate(updates *model.BusinessUnitUpdate) {
	if updates.Name != "" {
		updates.Name = sanitizer.NormalizeName(updates.Name)
	}
	if updates.Cities != nil {
		if len(updates.Cities) == 0 {
			s.cfg.Log.Warn("Attempted to update cities with empty array")
		} else {
			updates.Cities = sanitizer.NormalizeCities(updates.Cities)
		}
	}
	if updates.Labels != nil {
		if len(updates.Labels) == 0 {
			s.cfg.Log.Warn("Attempted to update labels with empty array")
		} else {
			updates.Labels = sanitizer.NormalizeLabels(updates.Labels)
		}
	}
	if updates.AdminPhone != "" {
		updates.AdminPhone = sanitizer.NormalizePhone(updates.AdminPhone)
		if updates.AdminPhone == "" {
			updates.AdminPhone = "invalid_result"
		}
	}
	if updates.Maintainers != nil {
		normalized := sanitizer.NormalizeMaintainers(*updates.Maintainers)
		updates.Maintainers = &normalized
	}
	if updates.Priority != nil {
		normalized := sanitizer.NormalizePriority(s.cfg, *updates.Priority)
		updates.Priority = &normalized
	}
	if updates.WebsiteURL != nil {
		normalized := sanitizer.NormalizeURL(*updates.WebsiteURL)
		updates.WebsiteURL = &normalized
	}
	if updates.TimeZone != "" {
		updates.TimeZone = sanitizer.TrimAndNormalize(updates.TimeZone)
	}
}

func (s *businessUnitService) applyDefaultsForNewCreatedBusiness(bu *model.BusinessUnit) {
	if bu.TimeZone == "" {
		bu.TimeZone = locale.InferTimezoneFromPhone(bu.AdminPhone)
	}
	if bu.Priority == 0 {
		bu.Priority = int64(s.cfg.DefaultBusinessPriority)
	}
}

func (s *businessUnitService) mergeBusinessUnitUpdates(existing *model.BusinessUnit, updates *model.BusinessUnitUpdate) *model.BusinessUnit {
	merged := *existing

	if updates.Name != "" {
		merged.Name = updates.Name
	}

	if updates.Cities != nil {
		merged.Cities = updates.Cities
	}

	if updates.Labels != nil {
		merged.Labels = updates.Labels
	}

	if updates.AdminPhone != "" {
		merged.AdminPhone = updates.AdminPhone
	}

	if updates.Maintainers != nil {
		merged.Maintainers = *updates.Maintainers
	}

	if updates.Priority != nil {
		merged.Priority = *updates.Priority
	}

	if updates.TimeZone != "" {
		merged.TimeZone = updates.TimeZone
	}

	if updates.WebsiteURL != nil {
		merged.WebsiteURL = *updates.WebsiteURL
	}

	merged.ID = existing.ID
	merged.CreatedAt = existing.CreatedAt

	return &merged
}

func (s *businessUnitService) isDuplicate(newBU, existingBU *model.BusinessUnit) bool {
	if sanitizer.NormalizeNameForComparison(newBU.Name) != sanitizer.NormalizeNameForComparison(existingBU.Name) {
		return false
	}

	if !s.setsOverlap(newBU.Cities, existingBU.Cities) {
		return false
	}

	if !s.setsOverlap(newBU.Labels, existingBU.Labels) {
		return false
	}

	return true
}

func (s *businessUnitService) setsOverlap(set1, set2 []string) bool {
	return s.isSubset(set1, set2) || s.isSubset(set2, set1)
}

func (s *businessUnitService) isSubset(subset, superset []string) bool {
	supersetMap := make(map[string]bool)
	for _, item := range superset {
		supersetMap[item] = true
	}

	for _, item := range subset {
		if !supersetMap[item] {
			return false
		}
	}

	return true
}
