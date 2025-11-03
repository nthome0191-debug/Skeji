package service

import (
	"context"
	"skeji/internal/businessunits/repository"
	"skeji/internal/businessunits/validator"
	apperrors "skeji/pkg/errors"
	"skeji/pkg/locale"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"skeji/pkg/sanitizer"
	"strings"
)

const (
	DefaultPriority = 10
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
	logger    *logger.Logger
}

func NewBusinessUnitService(
	repo repository.BusinessUnitRepository,
	validator *validator.BusinessUnitValidator,
	logger *logger.Logger,
) BusinessUnitService {
	return &businessUnitService{
		repo:      repo,
		validator: validator,
		logger:    logger,
	}
}

func (s *businessUnitService) Create(ctx context.Context, bu *model.BusinessUnit) error {
	s.sanitize(bu)
	s.applyDefaultsForNewCreatedBusiness(bu)

	if err := s.validator.Validate(bu); err != nil {
		s.logger.Warn("Business unit validation failed",
			"name", bu.Name,
			"admin_phone", bu.AdminPhone,
			"error", err,
		)
		return apperrors.Validation("Business unit validation failed", map[string]any{
			"error": err.Error(),
		})
	}

	if err := s.repo.Create(ctx, bu); err != nil {
		s.logger.Error("Failed to create business unit",
			"name", bu.Name,
			"admin_phone", bu.AdminPhone,
			"error", err,
		)
		return apperrors.Internal("Failed to create business unit", err)
	}

	s.logger.Info("Business unit created successfully",
		"id", bu.ID,
		"name", bu.Name,
		"admin_phone", bu.AdminPhone,
		"priority", bu.Priority,
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
		if strings.Contains(err.Error(), "not found") {
			return nil, apperrors.NotFoundWithID("Business unit", id)
		}
		s.logger.Error("Failed to get business unit by ID",
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

	// Get total count and items in parallel would be ideal, but for now sequential
	count, err := s.repo.Count(ctx)
	if err != nil {
		s.logger.Error("Failed to count business units", "error", err)
		return nil, 0, apperrors.Internal("Failed to count business units", err)
	}

	units, err := s.repo.FindAll(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get all business units",
			"limit", limit,
			"offset", offset,
			"error", err,
		)
		return nil, 0, apperrors.Internal("Failed to retrieve business units", err)
	}

	return units, count, nil
}

func (s *businessUnitService) Update(ctx context.Context, id string, updates *model.BusinessUnitUpdate) error {
	if id == "" {
		return apperrors.InvalidInput("Business unit ID cannot be empty")
	}

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return apperrors.NotFoundWithID("Business unit", id)
		}
		return apperrors.Internal("Failed to check business unit existence", err)
	}

	s.sanitizeUpdate(updates)

	merged := s.mergeBusinessUnitUpdates(existing, updates)

	merged.ID = existing.ID
	merged.CreatedAt = existing.CreatedAt

	if err := s.validator.Validate(merged); err != nil {
		s.logger.Warn("Business unit update validation failed",
			"id", id,
			"error", err,
		)
		return apperrors.Validation("Business unit validation failed", map[string]any{
			"error": err.Error(),
		})
	}

	if err := s.repo.Update(ctx, id, merged); err != nil {
		s.logger.Error("Failed to update business unit",
			"id", id,
			"error", err,
		)
		return apperrors.Internal("Failed to update business unit", err)
	}

	s.logger.Info("Business unit updated successfully",
		"id", id,
		"name", merged.Name,
	)

	return nil
}

func (s *businessUnitService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return apperrors.InvalidInput("Business unit ID cannot be empty")
	}

	// Note: In production, you might want to check for dependent entities
	// (e.g., active bookings) before allowing deletion

	if err := s.repo.Delete(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return apperrors.NotFoundWithID("Business unit", id)
		}
		s.logger.Error("Failed to delete business unit",
			"id", id,
			"error", err,
		)
		return apperrors.Internal("Failed to delete business unit", err)
	}

	s.logger.Info("Business unit deleted successfully", "id", id)

	return nil
}

func (s *businessUnitService) GetByAdminPhone(ctx context.Context, phone string) ([]*model.BusinessUnit, error) {
	if phone == "" {
		return nil, apperrors.InvalidInput("Admin phone number cannot be empty")
	}

	phone = sanitizer.NormalizePhone(phone)

	units, err := s.repo.FindByAdminPhone(ctx, phone)
	if err != nil {
		s.logger.Error("Failed to get business units by admin phone",
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
		s.logger.Warn("Search criteria normalized to empty",
			"original_cities", originalCities,
			"original_labels", originalLabels,
			"normalized_cities", cities,
			"normalized_labels", labels,
		)
		return nil, apperrors.InvalidInput("Search criteria resulted in no valid items after normalization")
	}

	units, err := s.repo.Search(ctx, cities, labels)
	if err != nil {
		s.logger.Error("Failed to search business units",
			"cities", cities,
			"labels", labels,
			"error", err,
		)
		return nil, apperrors.Internal("Failed to search business units", err)
	}

	s.logger.Debug("Business units search completed",
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
	bu.Priority = sanitizer.NormalizePriority(bu.Priority)
}

func (s *businessUnitService) sanitizeUpdate(updates *model.BusinessUnitUpdate) {
	if updates.Name != "" {
		updates.Name = sanitizer.NormalizeName(updates.Name)
	}
	if len(updates.Cities) > 0 {
		updates.Cities = sanitizer.NormalizeCities(updates.Cities)
	}
	if len(updates.Labels) > 0 {
		updates.Labels = sanitizer.NormalizeLabels(updates.Labels)
	}
	if updates.AdminPhone != "" {
		updates.AdminPhone = sanitizer.NormalizePhone(updates.AdminPhone)
	}
	if updates.Maintainers != nil {
		normalized := sanitizer.NormalizeMaintainers(*updates.Maintainers)
		updates.Maintainers = &normalized
	}
	if updates.Priority != nil {
		normalized := sanitizer.NormalizePriority(*updates.Priority)
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
		bu.Priority = DefaultPriority
	}
}

func (s *businessUnitService) mergeBusinessUnitUpdates(existing *model.BusinessUnit, updates *model.BusinessUnitUpdate) *model.BusinessUnit {
	merged := *existing

	if updates.Name != "" {
		merged.Name = updates.Name
	}

	if len(updates.Cities) > 0 {
		merged.Cities = updates.Cities
	}

	if len(updates.Labels) > 0 {
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

	return &merged
}
