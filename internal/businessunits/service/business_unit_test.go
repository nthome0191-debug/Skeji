package service

import (
	"context"
	"fmt"
	"skeji/pkg/config"
	mongotx "skeji/pkg/db/mongo"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"sort"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// ────────────────────────────────────────────────
// Mock repository for testing
// ────────────────────────────────────────────────

type mockBusinessUnitRepository struct {
	findAllFunc            func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error)
	countFunc              func(ctx context.Context) (int64, error)
	searchByCityLabelPairs func(ctx context.Context, pairs []string) ([]*model.BusinessUnit, error)
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

func (m *mockBusinessUnitRepository) SearchByCityLabelPairs(ctx context.Context, pairs []string) ([]*model.BusinessUnit, error) {
	if m.searchByCityLabelPairs != nil {
		return m.searchByCityLabelPairs(ctx, pairs)
	}
	return []*model.BusinessUnit{}, nil
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

// ────────────────────────────────────────────────
// Tests for GetAll()
// ────────────────────────────────────────────────

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

	mockRepo := &mockBusinessUnitRepository{
		countFunc: func(ctx context.Context) (int64, error) {
			time.Sleep(10 * time.Millisecond)
			return 100, nil
		},
		findAllFunc: func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error) {
			time.Sleep(10 * time.Millisecond)
			return []*model.BusinessUnit{
				{ID: "1", Name: "Business 1"},
				{ID: "2", Name: "Business 2"},
			}, nil
		},
	}

	service := NewBusinessUnitService(mockRepo, nil, cfg)

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
		name        string
		inputLimit  int
		inputOffset int
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
		ReadTimeout: 50 * time.Millisecond,
	}

	mockRepo := &mockBusinessUnitRepository{
		countFunc: func(ctx context.Context) (int64, error) {
			time.Sleep(200 * time.Millisecond)
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

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := service.GetAll(ctx, 10, 0)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

// ────────────────────────────────────────────────
// Tests for SearchByCityLabelPairs()
// ────────────────────────────────────────────────

func TestSearchByCityLabelPairs(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "debug",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	cfg := &config.Config{
		Log:         log,
		ReadTimeout: 5 * time.Second,
	}

	expectedUnits := []*model.BusinessUnit{
		{ID: "1", Name: "Spa Tel Aviv"},
		{ID: "2", Name: "Massage Haifa"},
	}

	mockRepo := &mockBusinessUnitRepository{
		searchByCityLabelPairs: func(ctx context.Context, pairs []string) ([]*model.BusinessUnit, error) {
			expectedPairs := []string{"tel aviv|massage", "tel aviv|spa", "haifa|massage", "haifa|spa"}
			for _, exp := range expectedPairs {
				found := false
				for _, given := range pairs {
					if given == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pair %s not found in pairs: %v", exp, pairs)
				}
			}
			return expectedUnits, nil
		},
	}

	service := NewBusinessUnitService(mockRepo, nil, cfg)

	cities := []string{"Tel Aviv", "Haifa"}
	labels := []string{"Massage", "Spa"}

	units, err := service.Search(context.Background(), cities, labels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(units) != len(expectedUnits) {
		t.Errorf("expected %d results, got %d", len(expectedUnits), len(units))
	}
}

func TestSearchByCityLabelPairs_RepoError(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "debug",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	cfg := &config.Config{
		Log:         log,
		ReadTimeout: 5 * time.Second,
	}

	mockRepo := &mockBusinessUnitRepository{
		searchByCityLabelPairs: func(ctx context.Context, pairs []string) ([]*model.BusinessUnit, error) {
			return nil, fmt.Errorf("DB failure")
		},
	}

	service := NewBusinessUnitService(mockRepo, nil, cfg)

	_, err := service.Search(context.Background(), []string{"Tel Aviv"}, []string{"Spa"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSearchByCityLabelPairs_SortedByPriority(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "debug",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	cfg := &config.Config{
		Log:         log,
		ReadTimeout: 5 * time.Second,
	}

	mockRepo := &mockBusinessUnitRepository{
		searchByCityLabelPairs: func(ctx context.Context, pairs []string) ([]*model.BusinessUnit, error) {
			return []*model.BusinessUnit{
				{ID: "1", Name: "Low", Priority: 1},
				{ID: "2", Name: "Medium", Priority: 5},
				{ID: "3", Name: "High", Priority: 9},
			}, nil
		},
	}

	service := NewBusinessUnitService(mockRepo, nil, cfg)

	units, err := service.Search(context.Background(), []string{"Tel Aviv"}, []string{"Spa"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(units) != 3 {
		t.Fatalf("expected 3 results, got %d", len(units))
	}

	if !sort.SliceIsSorted(units, func(i, j int) bool { return units[i].Priority > units[j].Priority }) {
		t.Errorf("expected results sorted by descending priority, got: [%d, %d, %d]",
			units[0].Priority, units[1].Priority, units[2].Priority)
	}
}
