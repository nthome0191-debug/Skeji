package service

import (
	"context"
	"errors"
	"fmt"
	bookingserrors "skeji/internal/bookings/errors"
	"skeji/internal/bookings/repository"
	"skeji/internal/bookings/validator"
	"skeji/pkg/config"
	apperrors "skeji/pkg/errors"
	"skeji/pkg/model"
	"skeji/pkg/sanitizer"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type BookingService interface {
	Create(ctx context.Context, booking *model.Booking) error
	GetByID(ctx context.Context, id string) (*model.Booking, error)
	GetAll(ctx context.Context, limit int, offset int) ([]*model.Booking, int64, error)
	Update(ctx context.Context, id string, updates *model.BookingUpdate) error
	Delete(ctx context.Context, id string) error
	SearchBySchedule(ctx context.Context, businessID string, scheduleID string, startTime, endTime *time.Time) ([]*model.Booking, error)
}

type bookingService struct {
	repo      repository.BookingRepository
	validator *validator.BookingValidator
	cfg       *config.Config
}

func NewBookingService(
	repo repository.BookingRepository,
	validator *validator.BookingValidator,
	cfg *config.Config,
) BookingService {
	return &bookingService{
		repo:      repo,
		validator: validator,
		cfg:       cfg,
	}
}

func (s *bookingService) Create(ctx context.Context, booking *model.Booking) error {
	s.sanitize(booking)
	s.applyDefaults(booking)

	if err := s.validator.Validate(booking); err != nil {
		s.cfg.Log.Warn("Booking validation failed", "error", err)
		return apperrors.Validation("Booking validation failed", map[string]any{"error": err.Error()})
	}

	err := s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		existing, err := s.repo.FindByBusinessAndSchedule(sessCtx, booking.BusinessID, booking.ScheduleID, &booking.StartTime, &booking.EndTime)
		if err != nil {
			return apperrors.Internal("Failed to check existing bookings", err)
		}

		for _, b := range existing {
			if overlaps(b.StartTime, b.EndTime, booking.StartTime, booking.EndTime) {
				return apperrors.Conflict(fmt.Sprintf(
					"Booking time overlaps with existing booking (%s - %s)",
					b.StartTime.Format(time.RFC3339),
					b.EndTime.Format(time.RFC3339),
				))
			}
		}

		if err := s.repo.Create(sessCtx, booking); err != nil {
			return apperrors.Internal("Failed to create booking", err)
		}

		return nil
	})
	if err != nil {
		s.cfg.Log.Error("Failed to create booking", "error", err)
		return err
	}

	s.cfg.Log.Info("Booking created successfully",
		"id", booking.ID,
		"business_id", booking.BusinessID,
		"schedule_id", booking.ScheduleID,
		"start_time", booking.StartTime,
	)
	return nil
}

func (s *bookingService) GetByID(ctx context.Context, id string) (*model.Booking, error) {
	if id == "" {
		return nil, apperrors.InvalidInput("Booking ID cannot be empty")
	}

	booking, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, bookingserrors.ErrNotFound) {
			return nil, apperrors.NotFoundWithID("Booking", id)
		}
		if errors.Is(err, bookingserrors.ErrInvalidID) {
			return nil, apperrors.InvalidInput("Invalid booking ID format")
		}
		return nil, apperrors.Internal("Failed to retrieve booking", err)
	}

	return booking, nil
}

func (s *bookingService) GetAll(ctx context.Context, limit int, offset int) ([]*model.Booking, int64, error) {

	var count int64
	var bookings []*model.Booking
	var errCount, errFind error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		count, errCount = s.repo.Count(ctx)
		if errCount != nil {
			s.cfg.Log.Error("Failed to count bookings", "error", errCount)
			errCount = apperrors.Internal("Failed to count bookings", errCount)
		}
	}()

	go func() {
		defer wg.Done()
		bookings, errFind = s.repo.FindAll(ctx, limit, offset)
		if errFind != nil {
			s.cfg.Log.Error("Failed to list bookings", "error", errFind)
			errFind = apperrors.Internal("Failed to retrieve bookings", errFind)
		}
	}()

	wg.Wait()
	if errCount != nil {
		return nil, 0, errCount
	}
	if errFind != nil {
		return nil, 0, errFind
	}

	return bookings, count, nil
}

func (s *bookingService) Update(ctx context.Context, id string, updates *model.BookingUpdate) error {
	if id == "" {
		return apperrors.InvalidInput("Booking ID cannot be empty")
	}

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, bookingserrors.ErrNotFound) {
			return apperrors.NotFoundWithID("Booking", id)
		}
		if errors.Is(err, bookingserrors.ErrInvalidID) {
			return apperrors.InvalidInput("Invalid booking ID format")
		}
		return apperrors.Internal("Failed to check booking existence", err)
	}

	if err := s.validator.ValidateUpdate(updates); err != nil {
		s.cfg.Log.Warn("Booking update validation failed", "id", id, "error", err)
		return apperrors.Validation("Invalid update input", map[string]any{"error": err.Error()})
	}

	merged := s.mergeBookingUpdates(existing, updates)

	if err := s.validator.Validate(merged); err != nil {
		s.cfg.Log.Warn("Merged booking validation failed", "id", id, "error", err)
		return apperrors.Validation("Booking validation failed", map[string]any{"error": err.Error()})
	}

	err = s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {

		existingBookings, err := s.repo.FindByBusinessAndSchedule(sessCtx, merged.BusinessID, merged.ScheduleID, &merged.StartTime, &merged.EndTime)
		if err != nil {
			return apperrors.Internal("Failed to check for overlapping bookings", err)
		}

		for _, b := range existingBookings {
			if b.ID == merged.ID {
				continue
			}
			if overlaps(b.StartTime, b.EndTime, merged.StartTime, merged.EndTime) {
				return apperrors.Conflict("Updated booking overlaps with another booking")
			}
		}

		if _, err := s.repo.Update(sessCtx, id, merged); err != nil {
			return apperrors.Internal("Failed to update booking", err)
		}

		return nil
	})

	if err != nil {
		s.cfg.Log.Error("Failed to update booking", "id", id, "error", err)
		return err
	}

	s.cfg.Log.Info("Booking updated successfully", "id", id)
	return nil
}

func (s *bookingService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return apperrors.InvalidInput("Booking ID cannot be empty")
	}

	err := s.repo.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
		if err := s.repo.Delete(sessCtx, id); err != nil {
			if errors.Is(err, bookingserrors.ErrNotFound) {
				return apperrors.NotFoundWithID("Booking", id)
			}
			if errors.Is(err, bookingserrors.ErrInvalidID) {
				return apperrors.InvalidInput("Invalid booking ID format")
			}
			return apperrors.Internal("Failed to delete booking", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.cfg.Log.Info("Booking deleted successfully", "id", id)
	return nil
}

func (s *bookingService) SearchBySchedule(ctx context.Context, businessID string, scheduleID string, startTime, endTime *time.Time) ([]*model.Booking, error) {
	if businessID == "" || scheduleID == "" {
		return nil, apperrors.InvalidInput("BusinessID and ScheduleID are required")
	}

	bookings, err := s.repo.FindByBusinessAndSchedule(ctx, businessID, scheduleID, startTime, endTime)
	if err != nil {
		s.cfg.Log.Error("Failed to search bookings", "error", err)
		return nil, apperrors.Internal("Failed to search bookings", err)
	}

	s.cfg.Log.Debug("Booking search completed",
		"business_id", businessID,
		"schedule_id", scheduleID,
		"count", len(bookings),
	)
	return bookings, nil
}

// --- Helpers ---

func overlaps(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}

func (s *bookingService) sanitize(b *model.Booking) {
	b.ServiceLabel = sanitizer.Normalize(b.ServiceLabel)
	for k, v := range b.Participants {
		b.Participants[sanitizer.NormalizePhone(k)] = sanitizer.Normalize(v)
	}
	for k, v := range b.ManagedBy {
		b.ManagedBy[sanitizer.NormalizePhone(k)] = sanitizer.Normalize(v)
	}
}

func (s *bookingService) applyDefaults(b *model.Booking) {
	if b.Status == "" {
		b.Status = config.Pending
	}
	if b.Capacity == 0 {
		b.Capacity = max(len(b.Participants), 1)
	}
}

func (s *bookingService) mergeBookingUpdates(existing *model.Booking, updates *model.BookingUpdate) *model.Booking {
	merged := *existing

	if updates.ServiceLabel != "" {
		merged.ServiceLabel = updates.ServiceLabel
	}
	if updates.StartTime != nil {
		merged.StartTime = *updates.StartTime
	}
	if updates.EndTime != nil {
		merged.EndTime = *updates.EndTime
	}
	if updates.Capacity != nil {
		merged.Capacity = *updates.Capacity
	}
	if updates.Participants != nil {
		merged.Participants = *updates.Participants
	}
	if updates.Status != "" {
		merged.Status = updates.Status
	}
	if updates.ManagedBy != nil {
		merged.ManagedBy = updates.ManagedBy
	}

	return &merged
}
