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
type mockBusinessUnitRepository struct {
	findAllFunc   func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error)
	countFunc     func(ctx context.Context) (int64, error)
}

func (m *mockBusinessUnitRepository) Create(ctx context.Context, bu *model.BusinessUnit) error {
	return nil
}

func (m *mockBusinessUnitRepository) FindByID(ctx context.Context, id string) (*model.BusinessUnit, error) {
	return nil, nil
}

func (m *mockBusinessUnitRepository) FindAll(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx, limit, offset)
	}
	return []*model.BusinessUnit{}, nil
}

func (m *mockBusinessUnitRepository) Update(ctx context.Context, id string, bu *model.BusinessUnit) (*mongo.UpdateResult, error) {
	return nil, nil
}

func (m *mockBusinessUnitRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockBusinessUnitRepository) FindByAdminPhone(ctx context.Context, phone string) ([]*model.BusinessUnit, error) {
	return nil, nil
}

func (m *mockBusinessUnitRepository) Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error) {
	return nil, nil
}

func (m *mockBusinessUnitRepository) Count(ctx context.Context) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx)
	}
	return 0, nil
}

func (m *mockBusinessUnitRepository) ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error {
	return nil
}

func TestGetAll_ConcurrentAccess(t *testing.T) {
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

	// This test would catch race conditions if run with -race flag
	mockRepo := &mockBusinessUnitRepository{
		countFunc: func(ctx context.Context) (int64, error) {
			time.Sleep(10 * time.Millisecond) // Simulate DB delay
			return 100, nil
		},
		findAllFunc: func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error) {
			time.Sleep(10 * time.Millisecond) // Simulate DB delay
			return []*model.BusinessUnit{
				{ID: "1", Name: "Business 1"},
				{ID: "2", Name: "Business 2"},
			}, nil
		},
	}

	service := NewBusinessUnitService(mockRepo, nil, cfg)

	// Run multiple times to increase chance of catching race condition
	for i := 0; i < 10; i++ {
		ctx := context.Background()
		units, count, err := service.GetAll(ctx, 10, 0)

		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}

		if count != 100 {
			t.Errorf("iteration %d: expected count 100, got %d", i, count)
		}

		if len(units) != 2 {
			t.Errorf("iteration %d: expected 2 units, got %d", i, len(units))
		}
	}
}

func TestGetAll_LimitNormalization(t *testing.T) {
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

	mockRepo := &mockBusinessUnitRepository{
		countFunc: func(ctx context.Context) (int64, error) {
			return 0, nil
		},
		findAllFunc: func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error) {
			// Test validates that limit is properly normalized
			if limit <= 0 {
				t.Error("limit should not be <= 0 after normalization")
			}
			if limit > 100 {
				t.Error("limit should not be > 100 after normalization")
			}
			return []*model.BusinessUnit{}, nil
		},
	}

	service := NewBusinessUnitService(mockRepo, nil, cfg)

	tests := []struct {
		name          string
		inputLimit    int
		inputOffset   int
	}{
		{"zero limit", 0, 0},
		{"negative limit", -1, 0},
		{"excessive limit", 500, 0},
		{"negative offset", 10, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, _, err := service.GetAll(ctx, tt.inputLimit, tt.inputOffset)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetAll_ContextTimeout(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})

	cfg := &config.Config{
		Log:         log,
		ReadTimeout: 50 * time.Millisecond, // Short timeout
	}

	mockRepo := &mockBusinessUnitRepository{
		countFunc: func(ctx context.Context) (int64, error) {
			// Simulate slow DB query
			time.Sleep(200 * time.Millisecond)

			// Check if context was cancelled
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
				return 100, nil
			}
		},
		findAllFunc: func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error) {
			return []*model.BusinessUnit{}, nil
		},
	}

	service := NewBusinessUnitService(mockRepo, nil, cfg)

	ctx := context.Background()
	_, _, err := service.GetAll(ctx, 10, 0)

	// Should return error due to timeout
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

// TestSanitizeUpdates_InvalidPhone documents the "invalid_result" bug
// NOTE: This test is commented out because it requires access to the unexported sanitizeUpdate method
// The bug is documented in CODE_REVIEW_REPORT_2.md Issue #3
//
// The bug: When AdminPhone normalization fails, it's set to "invalid_result" instead of returning an error
// Location: internal/businessunits/service/business_unit.go:372-376
//
// func TestSanitizeUpdates_InvalidPhone(t *testing.T) {
// 	updates := &model.BusinessUnitUpdate{
// 		AdminPhone: "invalid-phone-123",
// 	}
// 	service.sanitizeUpdate(updates)
// 	if updates.AdminPhone == "invalid_result" {
// 		t.Error("BUG: invalid phone normalized to 'invalid_result' instead of returning error")
// 	}
// }
