package service

import (
	"context"
	"skeji/pkg/config"
	mongotx "skeji/pkg/db/mongo"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Mock repository for testing
type mockScheduleRepository struct {
	findAllFunc   func(ctx context.Context, limit int, offset int) ([]*model.Schedule, error)
	countFunc     func(ctx context.Context) (int64, error)
	searchFunc    func(ctx context.Context, businessId string, city string) ([]*model.Schedule, error)
}

func (m *mockScheduleRepository) Create(ctx context.Context, sc *model.Schedule) error {
	return nil
}

func (m *mockScheduleRepository) FindByID(ctx context.Context, id string) (*model.Schedule, error) {
	return nil, nil
}

func (m *mockScheduleRepository) FindAll(ctx context.Context, limit int, offset int) ([]*model.Schedule, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx, limit, offset)
	}
	return []*model.Schedule{}, nil
}

func (m *mockScheduleRepository) Update(ctx context.Context, id string, sc *model.Schedule) (*mongo.UpdateResult, error) {
	return nil, nil
}

func (m *mockScheduleRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockScheduleRepository) Search(ctx context.Context, businessId string, city string) ([]*model.Schedule, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, businessId, city)
	}
	return []*model.Schedule{}, nil
}

func (m *mockScheduleRepository) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

func (m *mockScheduleRepository) ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error {
	return nil
}

func TestGetAll_RaceCondition(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})

	cfg := &config.Config{
		Log:         log,
		ReadTimeout: 5 * time.Second,
	}

	mockRepo := &mockScheduleRepository{
		countFunc: func(ctx context.Context) (int64, error) {
			time.Sleep(10 * time.Millisecond)
			return 50, nil
		},
		findAllFunc: func(ctx context.Context, limit int, offset int) ([]*model.Schedule, error) {
			time.Sleep(10 * time.Millisecond)
			return []*model.Schedule{
				{ID: "1", Name: "Schedule 1"},
			}, nil
		},
	}

	service := NewScheduleService(mockRepo, nil, cfg)

	// Run with -race flag to detect the race condition
	for i := 0; i < 20; i++ {
		ctx := context.Background()
		schedules, count, err := service.GetAll(ctx, 10, 0)

		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}

		if count != 50 {
			t.Errorf("iteration %d: expected count 50, got %d", i, count)
		}

		if len(schedules) != 1 {
			t.Errorf("iteration %d: expected 1 schedule, got %d", i, len(schedules))
		}
	}
}

func TestSearch_ReDoSVulnerability(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})

	cfg := &config.Config{
		Log:         log,
		ReadTimeout: 5 * time.Second,
	}

	// Track what city parameter is passed to repository
	var capturedCity string
	mockRepo := &mockScheduleRepository{
		searchFunc: func(ctx context.Context, businessId string, city string) ([]*model.Schedule, error) {
			capturedCity = city
			// In real code, this unsanitized city goes directly into MongoDB regex
			// Allowing patterns like "(a+)+b" that cause exponential backtracking
			return []*model.Schedule{}, nil
		},
	}

	service := NewScheduleService(mockRepo, nil, cfg)

	// Test with malicious regex pattern
	maliciousCity := "(a+)+b"
	ctx := context.Background()
	_, err := service.Search(ctx, "business-123", maliciousCity)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// This test documents the vulnerability - the malicious pattern passes through
	if capturedCity != maliciousCity {
		t.Error("city parameter was modified, expected it to pass through as-is")
	}

	// BUG DETECTED: Repository receives unescaped regex pattern
	t.Logf("VULNERABILITY: Malicious regex pattern '%s' passed to repository", capturedCity)
}

func TestGetAll_BothGoroutinesTimeout(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})

	cfg := &config.Config{
		Log:         log,
		ReadTimeout: 50 * time.Millisecond,
	}

	mockRepo := &mockScheduleRepository{
		countFunc: func(ctx context.Context) (int64, error) {
			time.Sleep(200 * time.Millisecond)
			return 0, ctx.Err()
		},
		findAllFunc: func(ctx context.Context, limit int, offset int) ([]*model.Schedule, error) {
			time.Sleep(200 * time.Millisecond)
			return nil, ctx.Err()
		},
	}

	service := NewScheduleService(mockRepo, nil, cfg)

	// Create context with short timeout to simulate timeout scenario
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _, err := service.GetAll(ctx, 10, 0)
	elapsed := time.Since(start)

	// Should timeout
	if err == nil {
		t.Error("expected timeout error, got nil")
	}

	// Function should fail relatively quickly when context times out
	if elapsed > 300*time.Millisecond {
		t.Errorf("function took %v, expected faster failure on timeout", elapsed)
	}
}

// TestSanitizeUpdates_NilPointerRisk documents the exceptions validation bug
// NOTE: This test is commented out because the mergeScheduleUpdates method doesn't return a map
// The bug is documented in CODE_REVIEW_REPORT_2.md Issue #5
//
// The bug: Exceptions field is merged without validating that dates are valid YYYY-MM-DD format
// Location: internal/schedules/service/schedule.go:370-372
//
// func TestSanitizeUpdates_NilPointerRisk(t *testing.T) {
// 	emptySlice := []string{"invalid-date", "not-a-date"}
// 	updates := &model.ScheduleUpdate{
// 		Exceptions: &emptySlice,
// 	}
// 	// These invalid dates would be merged without validation
// 	t.Log("BUG: Exceptions are merged without date format validation")
// }
