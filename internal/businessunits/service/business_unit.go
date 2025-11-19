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
	GetAll(ctx context.Context, limit int, offset int64) ([]*model.BusinessUnit, int64, error)
	Update(ctx context.Context, id string, updates *model.BusinessUnitUpdate) error
	Delete(ctx context.Context, id string) error

	GetByPhone(ctx context.Context, phone string, cities []string, labels []string, limit int, offset int64) ([]*model.BusinessUnit, int64, error)
	Search(ctx context.Context, cities []string, labels []string, limit int, offset int64) ([]*model.BusinessUnit, int64, error)
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
	s.applyDefaults(bu)
	s.sanitize(bu)
	err := s.verifyLimitPerPhoneAdmin(ctx, bu)
	if err != nil {
		return err
	}
	err = s.validate(bu)
	if err != nil {
		return err
	}
	s.populateCityLabelPairs(bu)
	err = s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		err = s.verifyDuplication(sessCtx, bu)
		if err != nil {
			return err
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

func (s *businessUnitService) GetAll(ctx context.Context, limit int, offset int64) ([]*model.BusinessUnit, int64, error) {
	var count int64
	var units []*model.BusinessUnit
	var errCount, errFind error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		var err error
		count, err = s.repo.Count(ctx)
		if err != nil {
			s.cfg.Log.Error("Failed to count business units", "error", err)
			errCount = apperrors.Internal("Failed to count business units", err)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
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
	merged := s.mergeBusinessUnitUpdates(existing, updates)
	s.sanitize(merged)
	err = s.validate(merged)
	if err != nil {
		return err
	}
	s.populateCityLabelPairs(merged)
	err = s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		err = s.verifyDuplication(sessCtx, merged)

		if _, err = s.repo.Update(sessCtx, id, merged); err != nil {
			s.cfg.Log.Error("Failed to update business unit",
				"id", id,
				"error", err,
			)
			return apperrors.Internal("Failed to update business unit", err)
		}
		return nil
	})

	if err != nil {
		return err
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
	err := s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		if err := s.repo.Delete(sessCtx, id); err != nil {
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
		return nil
	})
	if err != nil {
		return err
	}
	s.cfg.Log.Info("Business unit deleted successfully", "id", id)
	return nil
}

func (s *businessUnitService) GetByPhone(ctx context.Context, phone string, cities []string, labels []string, limit int, offset int64) ([]*model.BusinessUnit, int64, error) {
	if phone == "" {
		return nil, 0, apperrors.InvalidInput("Phone number cannot be empty")
	}
	phone = sanitizer.SanitizePhone(phone)
	labels, cities = s.sanitizeSearchRequest(labels, cities)

	var count int64
	var units []*model.BusinessUnit
	var errCount, errFind error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error
		count, err = s.repo.CountByPhone(ctx, phone, cities, labels)
		if err != nil {
			s.cfg.Log.Error("Failed to count business units by phone", "phone", phone, "error", err)
			errCount = apperrors.Internal("Failed to count business units by phone", err)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		units, err = s.repo.GetByPhone(ctx, phone, cities, labels, limit, offset)
		if err != nil {
			s.cfg.Log.Error("Failed to get business units by phone",
				"phone", phone,
				"limit", limit,
				"offset", offset,
				"error", err,
			)
			errFind = apperrors.Internal("Failed to retrieve business units by phone", err)
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

func (s *businessUnitService) Search(ctx context.Context, cities []string, labels []string, limit int, offset int64) ([]*model.BusinessUnit, int64, error) {
	if len(cities) == 0 || len(labels) == 0 {
		return nil, 0, apperrors.InvalidInput("Both search criteria (cities and labels) must be provided")
	}

	originalCities := append([]string(nil), cities...)
	originalLabels := append([]string(nil), labels...)

	labels, cities = s.sanitizeSearchRequest(labels, cities)

	if len(cities) == 0 || len(labels) == 0 {
		s.cfg.Log.Warn("Search criteria normalized to empty",
			"original_cities", originalCities,
			"original_labels", originalLabels,
			"normalized_cities", cities,
			"normalized_labels", labels,
		)
		return nil, 0, apperrors.InvalidInput("Search criteria resulted in no valid items after normalization")
	}

	pairs := make([]string, 0, len(cities)*len(labels))
	for _, city := range cities {
		for _, label := range labels {
			pairs = append(pairs, fmt.Sprintf("%s|%s", city, label))
		}
	}

	var count int64
	var units []*model.BusinessUnit
	var errCount, errFind error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error
		count, err = s.repo.CountByCityLabelPairs(ctx, pairs)
		if err != nil {
			s.cfg.Log.Error("Failed to count business units by city_label_pairs",
				"cities", cities,
				"labels", labels,
				"pairs", pairs,
				"error", err,
			)
			errCount = apperrors.Internal("Failed to count business units", err)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		units, err = s.repo.SearchByCityLabelPairs(ctx, pairs, limit, offset)
		if err != nil {
			s.cfg.Log.Error("Failed to search business units by city_label_pairs",
				"cities", cities,
				"labels", labels,
				"pairs", pairs,
				"limit", limit,
				"offset", offset,
				"error", err,
			)
			errFind = apperrors.Internal("Failed to search business units", err)
		}
	}()

	wg.Wait()

	if errCount != nil {
		return nil, 0, errCount
	}
	if errFind != nil {
		return nil, 0, errFind
	}

	s.cfg.Log.Debug("Business units search completed",
		"cities", cities,
		"labels", labels,
		"pairs_count", len(pairs),
		"results_count", len(units),
		"total_count", count,
	)

	return units, count, nil
}

func (s *businessUnitService) applyDefaults(bu *model.BusinessUnit) {
	if bu.TimeZone == "" {
		bu.TimeZone = locale.InferTimezoneFromPhone(bu.AdminPhone)
	}
	if bu.Priority == 0 {
		bu.Priority = int64(s.cfg.DefaultBusinessPriority)
	}
	if bu.Maintainers == nil {
		bu.Maintainers = map[string]string{}
	}
	if bu.WebsiteURLs == nil {
		bu.WebsiteURLs = []string{}
	}
}

func (s *businessUnitService) sanitize(bu *model.BusinessUnit) {
	bu.Name = sanitizer.SanitizeNameOrAddress(bu.Name)
	bu.AdminPhone = sanitizer.SanitizePhone(bu.AdminPhone)
	bu.Cities = sanitizer.SanitizeSlice(bu.Cities, sanitizer.SanitizeCityOrLabel)
	bu.Labels = sanitizer.SanitizeSlice(bu.Labels, sanitizer.SanitizeCityOrLabel)
	bu.Maintainers = sanitizer.SanitizeMaintainersMap(bu.Maintainers, bu.AdminPhone)
	bu.WebsiteURLs = sanitizer.SanitizeSlice(bu.WebsiteURLs, sanitizer.SanitizeURL)
	bu.Priority = sanitizer.SanitizePriority(s.cfg, bu.Priority)
}

func (s *businessUnitService) sanitizeSearchRequest(labels, cities []string) (l []string, c []string) {
	labels = sanitizer.SanitizeSlice(labels, sanitizer.SanitizeCityOrLabel)
	cities = sanitizer.SanitizeSlice(cities, sanitizer.SanitizeCityOrLabel)
	return labels, cities
}

func (s *businessUnitService) validate(bu *model.BusinessUnit) error {
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
	return nil
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

	if updates.WebsiteURLs != nil {
		merged.WebsiteURLs = *updates.WebsiteURLs
	}

	merged.ID = existing.ID
	merged.CreatedAt = existing.CreatedAt

	return &merged
}

func (s *businessUnitService) isDuplicate(newBU, existingBU *model.BusinessUnit) bool {
	if sanitizer.SanitizeNameOrAddress(newBU.Name) != sanitizer.SanitizeNameOrAddress(existingBU.Name) {
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

func (s *businessUnitService) populateCityLabelPairs(bu *model.BusinessUnit) {
	pairs := make([]string, 0, len(bu.Cities)*len(bu.Labels))
	for _, city := range bu.Cities {
		for _, label := range bu.Labels {
			pairs = append(pairs, fmt.Sprintf("%s|%s", city, label))
		}
	}
	bu.CityLabelPairs = pairs
}

func (s *businessUnitService) verifyDuplication(ctx context.Context, bu *model.BusinessUnit) (err error) {
	total, err := s.repo.CountByPhone(ctx, bu.AdminPhone, bu.Cities, bu.Labels)
	if err != nil {
		return err
	}
	var offset int64 = 0
	var chunk []*model.BusinessUnit
	for offset < total {
		chunk, err = s.repo.GetByPhone(ctx, bu.AdminPhone, bu.Cities, bu.Labels, config.DefaultPaginationLimit, offset)
		if err != nil {
			return fmt.Errorf("failed to check for duplicates: %w", err)
		}
		if len(chunk) == 0 {
			break
		}
		for _, existingBU := range chunk {
			if existingBU.ID == bu.ID {
				continue
			}
			if s.isDuplicate(bu, existingBU) {
				return apperrors.Conflict(fmt.Sprintf(
					"Business unit with similar details already exists (id: %s)",
					existingBU.ID,
				))
			}
		}
		offset += int64(len(chunk))
	}
	return nil
}

func (s *businessUnitService) verifyLimitPerPhoneAdmin(ctx context.Context, bu *model.BusinessUnit) (err error) {
	_, total, err := s.GetByPhone(ctx, bu.AdminPhone, bu.Cities, bu.Labels, 10, 0)
	if err != nil {
		return err
	}
	if total >= int64(config.DefaultMaxBusinessUnitsPerAdminPhone) {
		return apperrors.Conflict(fmt.Sprintf(
			"Phone num exceeded maximum business units allowed (%s)",
			bu.AdminPhone,
		))
	}
	return nil
}
