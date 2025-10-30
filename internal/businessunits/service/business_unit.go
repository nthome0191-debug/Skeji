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
	Update(ctx context.Context, id string, bu *model.BusinessUnit) error
	Delete(ctx context.Context, id string) error

	GetByAdminPhone(ctx context.Context, phone string) ([]*model.BusinessUnit, error)
	GetByCity(ctx context.Context, city string) ([]*model.BusinessUnit, error)
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
	s.applyDefaults(bu)

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

func (s *businessUnitService) Update(ctx context.Context, id string, bu *model.BusinessUnit) error {
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

	merged := s.mergeBusinessUnitUpdates(existing, bu)

	merged.ID = existing.ID
	merged.CreatedAt = existing.CreatedAt

	s.sanitize(merged)

	if err := s.validator.Validate(merged); err != nil {
		s.logger.Warn("Business unit update validation failed",
			"id", id,
			"error", err,
		)
		return apperrors.Validation("Business unit validation failed", map[string]any{
			"error": err.Error(),
		})
	}

	bu = merged // Use merged result for update

	if err := s.repo.Update(ctx, id, bu); err != nil {
		s.logger.Error("Failed to update business unit",
			"id", id,
			"error", err,
		)
		return apperrors.Internal("Failed to update business unit", err)
	}

	s.logger.Info("Business unit updated successfully",
		"id", id,
		"name", bu.Name,
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

func (s *businessUnitService) GetByCity(ctx context.Context, city string) ([]*model.BusinessUnit, error) {
	if city == "" {
		return nil, apperrors.InvalidInput("City cannot be empty")
	}

	units, err := s.repo.FindByCity(ctx, city)
	if err != nil {
		s.logger.Error("Failed to get business units by city",
			"city", city,
			"error", err,
		)
		return nil, apperrors.Internal("Failed to retrieve business units by city", err)
	}

	return units, nil
}

func (s *businessUnitService) Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error) {
	if len(cities) == 0 || len(labels) == 0 {
		return nil, apperrors.InvalidInput("Both search criteria (cities and labels) must be provided")
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
}

func (s *businessUnitService) applyDefaults(bu *model.BusinessUnit) {
	if bu.TimeZone == "" {
		bu.TimeZone = locale.InferTimezoneFromPhone(bu.AdminPhone)
	}

	if bu.Priority == 0 {
		bu.Priority = DefaultPriority
	}
}

func (s *businessUnitService) mergeBusinessUnitUpdates(existing, updates *model.BusinessUnit) *model.BusinessUnit {
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

	if len(updates.Maintainers) > 0 {
		merged.Maintainers = updates.Maintainers
	}

	if updates.Priority != 0 {
		merged.Priority = updates.Priority
	}

	if updates.TimeZone != "" {
		merged.TimeZone = updates.TimeZone
	}

	return &merged
}
