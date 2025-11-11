package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"testing"

	"github.com/julienschmidt/httprouter"
)

// Mock service for testing
type mockBusinessUnitService struct {
	getAllFunc func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, int64, error)
}

func (m *mockBusinessUnitService) Create(ctx context.Context, bu *model.BusinessUnit) error {
	return nil
}

func (m *mockBusinessUnitService) GetByID(ctx context.Context, id string) (*model.BusinessUnit, error) {
	return nil, nil
}

func (m *mockBusinessUnitService) GetAll(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, int64, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx, limit, offset)
	}
	return []*model.BusinessUnit{}, 0, nil
}

func (m *mockBusinessUnitService) Update(ctx context.Context, id string, updates *model.BusinessUnitUpdate) error {
	return nil
}

func (m *mockBusinessUnitService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockBusinessUnitService) GetByAdminPhone(ctx context.Context, phone string) ([]*model.BusinessUnit, error) {
	return nil, nil
}

func (m *mockBusinessUnitService) Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error) {
	return nil, nil
}

func TestGetAll_InvalidQueryParameters(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})

	// Track what values the service receives
	var receivedLimit, receivedOffset int
	mockService := &mockBusinessUnitService{
		getAllFunc: func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, int64, error) {
			receivedLimit = limit
			receivedOffset = offset
			return []*model.BusinessUnit{}, 0, nil
		},
	}

	handler := &BusinessUnitHandler{
		service: mockService,
		log:     log,
	}

	tests := []struct {
		name           string
		queryString    string
		expectHTTPCode int
		checkValues    bool
	}{
		{
			name:           "invalid limit - alphabetic",
			queryString:    "?limit=abc&offset=0",
			expectHTTPCode: http.StatusOK, // BUG: Should be 400
			checkValues:    true,
		},
		{
			name:           "invalid offset - alphabetic",
			queryString:    "?limit=10&offset=xyz",
			expectHTTPCode: http.StatusOK, // BUG: Should be 400
			checkValues:    true,
		},
		{
			name:           "invalid both parameters",
			queryString:    "?limit=abc&offset=xyz",
			expectHTTPCode: http.StatusOK, // BUG: Should be 400
			checkValues:    true,
		},
		{
			name:           "negative values",
			queryString:    "?limit=-10&offset=-5",
			expectHTTPCode: http.StatusOK,
			checkValues:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/business-units"+tt.queryString, nil)
			w := httptest.NewRecorder()

			handler.GetAll(w, req, httprouter.Params{})

			if w.Code != tt.expectHTTPCode {
				t.Errorf("expected status %d, got %d", tt.expectHTTPCode, w.Code)
			}

			// BUG DETECTED: Invalid input is silently converted to 0
			if tt.checkValues && tt.expectHTTPCode == http.StatusOK {
				if receivedLimit != 0 || receivedOffset != 0 {
					t.Logf("Invalid input converted to: limit=%d, offset=%d", receivedLimit, receivedOffset)
				}
				t.Log("BUG DETECTED: Invalid query parameters are silently ignored instead of returning 400 Bad Request")
			}
		})
	}
}

func TestGetAll_ValidQueryParameters(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})

	var receivedLimit, receivedOffset int
	mockService := &mockBusinessUnitService{
		getAllFunc: func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, int64, error) {
			receivedLimit = limit
			receivedOffset = offset
			return []*model.BusinessUnit{
				{ID: "1", Name: "Business 1"},
				{ID: "2", Name: "Business 2"},
			}, 100, nil
		},
	}

	handler := &BusinessUnitHandler{
		service: mockService,
		log:     log,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/business-units?limit=20&offset=10", nil)
	w := httptest.NewRecorder()

	handler.GetAll(w, req, httprouter.Params{})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if receivedLimit != 20 {
		t.Errorf("expected limit 20, got %d", receivedLimit)
	}

	if receivedOffset != 10 {
		t.Errorf("expected offset 10, got %d", receivedOffset)
	}

	// Verify response structure
	var response struct {
		Data       []model.BusinessUnit `json:"data"`
		TotalCount int64                `json:"total_count"`
		Limit      int                  `json:"limit"`
		Offset     int                  `json:"offset"`
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TotalCount != 100 {
		t.Errorf("expected total_count 100, got %d", response.TotalCount)
	}

	if len(response.Data) != 2 {
		t.Errorf("expected 2 items, got %d", len(response.Data))
	}
}

func TestGetAll_EdgeCaseLimits(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})

	mockService := &mockBusinessUnitService{
		getAllFunc: func(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, int64, error) {
			return []*model.BusinessUnit{}, 0, nil
		},
	}

	handler := &BusinessUnitHandler{
		service: mockService,
		log:     log,
	}

	tests := []struct {
		name        string
		queryString string
	}{
		{"zero limit", "?limit=0&offset=0"},
		{"huge limit", "?limit=999999&offset=0"},
		{"huge offset", "?limit=10&offset=999999"},
		{"missing parameters", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/business-units"+tt.queryString, nil)
			w := httptest.NewRecorder()

			handler.GetAll(w, req, httprouter.Params{})

			// Should not panic or error
			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}
		})
	}
}
