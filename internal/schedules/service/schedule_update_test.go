package service

import (
	"context"
	"skeji/internal/schedules/validator"
	"skeji/pkg/config"
	mongotx "skeji/pkg/db/mongo"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Mock repository for update tests
type mockScheduleRepositoryForUpdate struct {
	findByIDFunc         func(ctx context.Context, id string) (*model.Schedule, error)
	searchFunc           func(ctx context.Context, businessID string, city string) ([]*model.Schedule, error)
	capturedSchedule     *model.Schedule // Capture updated schedule
	executeTransactionFunc func(ctx context.Context, fn mongotx.TransactionFunc) error
}

func (m *mockScheduleRepositoryForUpdate) FindByID(ctx context.Context, id string) (*model.Schedule, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockScheduleRepositoryForUpdate) Search(ctx context.Context, businessID string, city string) ([]*model.Schedule, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, businessID, city)
	}
	return []*model.Schedule{}, nil
}

func (m *mockScheduleRepositoryForUpdate) Update(ctx context.Context, id string, schedule *model.Schedule) (*mongo.UpdateResult, error) {
	m.capturedSchedule = schedule // Capture for verification
	return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}

func (m *mockScheduleRepositoryForUpdate) ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error {
	if m.executeTransactionFunc != nil {
		return m.executeTransactionFunc(ctx, fn)
	}
	// Execute the function with a fake session context
	sessCtx := mongo.NewSessionContext(ctx, nil)
	return fn(sessCtx)
}

// Implement other required repository methods
func (m *mockScheduleRepositoryForUpdate) Create(ctx context.Context, schedule *model.Schedule) error {
	return nil
}

func (m *mockScheduleRepositoryForUpdate) FindAll(ctx context.Context, limit int, offset int) ([]*model.Schedule, error) {
	return nil, nil
}

func (m *mockScheduleRepositoryForUpdate) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockScheduleRepositoryForUpdate) Count(ctx context.Context) (int64, error) {
	return 0, nil
}

// TestUpdate_ExceptionDatesValidation tests Issue #1 fix
// Verifies that invalid exception dates are filtered out during update
func TestUpdate_ExceptionDatesValidation(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})

	cfg := &config.Config{
		Log:                     log,
		DefaultMeetingDurationMin: 30,
		DefaultBreakDurationMin:   10,
		DefaultMaxParticipantsPerSlot: 5,
		DefaultStartOfDay:       "09:00",
		DefaultEndOfDay:         "17:00",
		DefaultWorkingDaysIsrael: []config.Weekday{"Sunday", "Monday"},
		DefaultWorkingDaysUs:    []config.Weekday{"Monday", "Tuesday"},
	}

	v := validator.NewScheduleValidator(log)

	existing := &model.Schedule{
		ID:                        "507f1f77bcf86cd799439011",
		BusinessID:                "507f1f77bcf86cd799439012", // Valid MongoDB ObjectID
		Name:                      "Original Schedule",
		City:                      "Tel Aviv",
		Address:                   "123 Main St",
		StartOfDay:                "09:00",
		EndOfDay:                  "17:00",
		WorkingDays:               []config.Weekday{"Sunday", "Monday"},
		DefaultMeetingDurationMin: 30,
		DefaultBreakDurationMin:   10,
		MaxParticipantsPerSlot:    5,
		TimeZone:                  "Asia/Jerusalem",
		Exceptions:                []string{}, // Initially no exceptions
		CreatedAt:                 time.Now(),
	}

	mockRepo := &mockScheduleRepositoryForUpdate{
		findByIDFunc: func(ctx context.Context, id string) (*model.Schedule, error) {
			return existing, nil
		},
		searchFunc: func(ctx context.Context, businessID string, city string) ([]*model.Schedule, error) {
			return []*model.Schedule{}, nil // No conflicts
		},
	}

	service := NewScheduleService(mockRepo, v, cfg)

	// Test with mixed valid and invalid exception dates
	invalidExceptions := []string{
		"2024-12-25",    // Valid
		"invalid-date",  // Invalid format
		"2024-13-01",    // Invalid month
		"2025-01-15",    // Valid
		"1800-01-01",    // Out of range (too old)
		"2200-01-01",    // Out of range (too new)
		"2024/12/31",    // Invalid format (slash instead of dash)
	}

	updates := &model.ScheduleUpdate{
		Exceptions: &invalidExceptions,
	}

	ctx := context.Background()
	err := service.Update(ctx, "507f1f77bcf86cd799439011", updates)

	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify only valid dates were kept
	if mockRepo.capturedSchedule == nil {
		t.Fatal("Expected schedule to be updated, got nil")
	}

	expectedValidDates := []string{"2024-12-25", "2025-01-15"}
	if len(mockRepo.capturedSchedule.Exceptions) != len(expectedValidDates) {
		t.Errorf("Expected %d valid exception dates, got %d", len(expectedValidDates), len(mockRepo.capturedSchedule.Exceptions))
		t.Logf("Actual exceptions: %v", mockRepo.capturedSchedule.Exceptions)
	}

	// Verify the exact dates
	validDatesMap := make(map[string]bool)
	for _, date := range mockRepo.capturedSchedule.Exceptions {
		validDatesMap[date] = true
	}

	for _, expectedDate := range expectedValidDates {
		if !validDatesMap[expectedDate] {
			t.Errorf("Expected valid date %s to be in exceptions, but it wasn't", expectedDate)
		}
	}

	// Verify invalid dates were filtered out
	invalidDates := []string{"invalid-date", "2024-13-01", "1800-01-01", "2200-01-01", "2024/12/31"}
	for _, invalidDate := range invalidDates {
		if validDatesMap[invalidDate] {
			t.Errorf("Invalid date %s should have been filtered out, but it's in exceptions", invalidDate)
		}
	}
}
