package service

import (
	"context"
	"errors"
	"fmt"
	"skeji/internal/businessunits/repository"
	"skeji/internal/businessunits/validator"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"strings"
)

// Service errors
var (
	ErrBusinessUnitNotFound = errors.New("business unit not found")
	ErrInvalidInput         = errors.New("invalid input")
	ErrUnauthorized         = errors.New("unauthorized operation")
)

const (
	DefaultPriority        = 10
	DefaultTimezone        = "UTC"
	IsraelTimezone         = "Asia/Jerusalem"
	USTimezoneDefault      = "America/New_York" // Default to Eastern Time (most populous timezone)
	IsraelPhonePrefix      = "+972"
	IsraelPhonePrefixAlt   = "972" // Without plus sign
	USPhonePrefix          = "+1"
)

// BusinessUnitService defines business logic operations for business units
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

// NewBusinessUnitService creates a new business unit service
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

// Create creates a new business unit with defaults and validation
func (s *businessUnitService) Create(ctx context.Context, bu *model.BusinessUnit) error {
	// Apply defaults before validation
	s.applyDefaults(bu)

	// Validate
	if err := s.validator.Validate(bu); err != nil {
		s.logger.Warn("Business unit validation failed",
			"name", bu.Name,
			"admin_phone", bu.AdminPhone,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// Create in repository
	if err := s.repo.Create(ctx, bu); err != nil {
		s.logger.Error("Failed to create business unit",
			"name", bu.Name,
			"admin_phone", bu.AdminPhone,
			"error", err,
		)
		return fmt.Errorf("failed to create business unit: %w", err)
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

// GetByID retrieves a business unit by ID
func (s *businessUnitService) GetByID(ctx context.Context, id string) (*model.BusinessUnit, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id cannot be empty", ErrInvalidInput)
	}

	bu, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrBusinessUnitNotFound
		}
		s.logger.Error("Failed to get business unit by ID",
			"id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get business unit: %w", err)
	}

	return bu, nil
}

// GetAll retrieves all business units with pagination and total count
func (s *businessUnitService) GetAll(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, int64, error) {
	// Set reasonable defaults for pagination
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // Max limit for safety
	}
	if offset < 0 {
		offset = 0
	}

	// Get total count and items in parallel would be ideal, but for now sequential
	count, err := s.repo.Count(ctx)
	if err != nil {
		s.logger.Error("Failed to count business units", "error", err)
		return nil, 0, fmt.Errorf("failed to count business units: %w", err)
	}

	units, err := s.repo.FindAll(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get all business units",
			"limit", limit,
			"offset", offset,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get business units: %w", err)
	}

	return units, count, nil
}

// Update updates an existing business unit
func (s *businessUnitService) Update(ctx context.Context, id string, bu *model.BusinessUnit) error {
	if id == "" {
		return fmt.Errorf("%w: id cannot be empty", ErrInvalidInput)
	}

	// Check if business unit exists first
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrBusinessUnitNotFound
		}
		return fmt.Errorf("failed to check business unit existence: %w", err)
	}

	// Preserve fields that shouldn't be updated through this method
	bu.ID = existing.ID
	bu.CreatedAt = existing.CreatedAt

	// Apply defaults for any missing optional fields
	s.applyDefaults(bu)

	// Validate updated data
	if err := s.validator.Validate(bu); err != nil {
		s.logger.Warn("Business unit update validation failed",
			"id", id,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, id, bu); err != nil {
		s.logger.Error("Failed to update business unit",
			"id", id,
			"error", err,
		)
		return fmt.Errorf("failed to update business unit: %w", err)
	}

	s.logger.Info("Business unit updated successfully",
		"id", id,
		"name", bu.Name,
	)

	return nil
}

// Delete deletes a business unit by ID
func (s *businessUnitService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: id cannot be empty", ErrInvalidInput)
	}

	// Note: In production, you might want to check for dependent entities
	// (e.g., active bookings) before allowing deletion

	if err := s.repo.Delete(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrBusinessUnitNotFound
		}
		s.logger.Error("Failed to delete business unit",
			"id", id,
			"error", err,
		)
		return fmt.Errorf("failed to delete business unit: %w", err)
	}

	s.logger.Info("Business unit deleted successfully", "id", id)

	return nil
}

// GetByAdminPhone retrieves business units by admin phone number
func (s *businessUnitService) GetByAdminPhone(ctx context.Context, phone string) ([]*model.BusinessUnit, error) {
	if phone == "" {
		return nil, fmt.Errorf("%w: phone cannot be empty", ErrInvalidInput)
	}

	units, err := s.repo.FindByAdminPhone(ctx, phone)
	if err != nil {
		s.logger.Error("Failed to get business units by admin phone",
			"phone", phone,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get business units by phone: %w", err)
	}

	return units, nil
}

// GetByCity retrieves business units in a specific city
func (s *businessUnitService) GetByCity(ctx context.Context, city string) ([]*model.BusinessUnit, error) {
	if city == "" {
		return nil, fmt.Errorf("%w: city cannot be empty", ErrInvalidInput)
	}

	units, err := s.repo.FindByCity(ctx, city)
	if err != nil {
		s.logger.Error("Failed to get business units by city",
			"city", city,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get business units by city: %w", err)
	}

	return units, nil
}

// Search searches for business units by cities and/or labels
func (s *businessUnitService) Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error) {
	if len(cities) == 0 && len(labels) == 0 {
		return nil, fmt.Errorf("%w: at least one search criteria (cities or labels) must be provided", ErrInvalidInput)
	}

	units, err := s.repo.Search(ctx, cities, labels)
	if err != nil {
		s.logger.Error("Failed to search business units",
			"cities", cities,
			"labels", labels,
			"error", err,
		)
		return nil, fmt.Errorf("failed to search business units: %w", err)
	}

	s.logger.Debug("Business units search completed",
		"cities", cities,
		"labels", labels,
		"results_count", len(units),
	)

	return units, nil
}

// applyDefaults sets default values for optional fields
func (s *businessUnitService) applyDefaults(bu *model.BusinessUnit) {
	// Set default timezone based on phone number if not provided
	if bu.TimeZone == "" {
		bu.TimeZone = s.inferTimezoneFromPhone(bu.AdminPhone)
	}

	// Set default priority if not provided (0 is the zero value, treat as not set)
	if bu.Priority == 0 {
		bu.Priority = DefaultPriority
	}
}

// inferTimezoneFromPhone infers timezone from phone number country code
func (s *businessUnitService) inferTimezoneFromPhone(phone string) string {
	// Normalize phone number for comparison
	normalizedPhone := strings.TrimSpace(phone)

	// Check for Israel country code
	if strings.HasPrefix(normalizedPhone, IsraelPhonePrefix) ||
		strings.HasPrefix(normalizedPhone, IsraelPhonePrefixAlt) {
		return IsraelTimezone
	}

	// Check for US/Canada country code
	// NOTE: US has multiple timezones (Eastern, Central, Mountain, Pacific, Alaska, Hawaii)
	// We default to Eastern Time as it covers the most populous region.
	// Users should explicitly provide timezone if they're in other US timezones.
	if strings.HasPrefix(normalizedPhone, USPhonePrefix) {
		s.logger.Debug("US/Canada number detected, using default Eastern Time",
			"phone", phone,
			"timezone", USTimezoneDefault,
		)
		return USTimezoneDefault
	}

	// Fallback to UTC for unsupported countries (should not reach here due to validation)
	s.logger.Warn("Unexpected country code passed validation",
		"phone", phone,
		"timezone", DefaultTimezone,
	)

	return DefaultTimezone
}
