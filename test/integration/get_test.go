package integration

import (
	"fmt"
	"net/http"
	"testing"

	"skeji/pkg/model"
	"skeji/test/integration/testutil"
)

func TestGetByID_ExistingBusinessUnit(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create a business unit first
	bu := testutil.ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	// Act
	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	// Assert
	testutil.AssertStatusCode(t, getResp, http.StatusOK)

	var fetched model.BusinessUnit
	if err := getResp.UnmarshalJSON(&fetched); err != nil {
		t.Fatalf("failed to unmarshal get response: %v", err)
	}

	if fetched.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, fetched.ID)
	}
	if fetched.Name != created.Name {
		t.Errorf("expected name %q, got %q", created.Name, fetched.Name)
	}
	if fetched.AdminPhone != created.AdminPhone {
		t.Errorf("expected admin_phone %q, got %q", created.AdminPhone, fetched.AdminPhone)
	}
}

func TestGetByID_NonExistentID(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Act - Try to get a business unit with a valid ObjectID format but doesn't exist
	nonExistentID := "507f1f77bcf86cd799439011" // Valid MongoDB ObjectID format
	resp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", nonExistentID))

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusNotFound)
	testutil.AssertContains(t, resp, "not found")
}

func TestGetByID_InvalidIDFormat(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	testCases := []struct {
		name string
		id   string
	}{
		{name: "empty string", id: ""},
		{name: "invalid hex", id: "invalid-id-format"},
		{name: "too short", id: "123"},
		{name: "special characters", id: "abc@#$%"},
		{name: "spaces", id: "507f 1f77 bcf8"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			resp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", tc.id))

			// Assert - Expecting either 400 or 404 depending on validation
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
				t.Errorf("expected status 400 or 404, got %d", resp.StatusCode)
			}
		})
	}
}

func TestGetAll_EmptyDatabase(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Act
	resp := client.GET(t, "/api/v1/business-units")

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusOK)

	var response struct {
		Data       []model.BusinessUnit `json:"data"`
		TotalCount int64                `json:"total_count"`
		Limit      int                  `json:"limit"`
		Offset     int                  `json:"offset"`
	}
	if err := resp.UnmarshalJSON(&response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.TotalCount != 0 {
		t.Errorf("expected total_count 0, got %d", response.TotalCount)
	}
	if len(response.Data) != 0 {
		t.Errorf("expected empty data array, got %d items", len(response.Data))
	}
}

func TestGetAll_MultipleBusinessUnits(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create multiple business units with different priorities
	businessUnits := []model.BusinessUnit{
		testutil.NewBusinessUnitBuilder().WithName("Low Priority").WithPriority(5).WithAdminPhone("+972501111111").Build(),
		testutil.NewBusinessUnitBuilder().WithName("High Priority").WithPriority(100).WithAdminPhone("+972502222222").Build(),
		testutil.NewBusinessUnitBuilder().WithName("Medium Priority").WithPriority(50).WithAdminPhone("+972503333333").Build(),
	}

	for _, bu := range businessUnits {
		resp := client.POST(t, "/api/v1/business-units", bu)
		testutil.AssertStatusCode(t, resp, http.StatusCreated)
	}

	// Act
	resp := client.GET(t, "/api/v1/business-units")

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusOK)

	var response struct {
		Data       []model.BusinessUnit `json:"data"`
		TotalCount int64                `json:"total_count"`
	}
	if err := resp.UnmarshalJSON(&response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.TotalCount != 3 {
		t.Errorf("expected total_count 3, got %d", response.TotalCount)
	}
	if len(response.Data) != 3 {
		t.Errorf("expected 3 business units, got %d", len(response.Data))
	}

	// Verify sorting by priority (descending)
	if response.Data[0].Priority < response.Data[1].Priority {
		t.Error("expected results to be sorted by priority descending")
	}
}

func TestGetAll_Pagination(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create 5 business units
	for i := 1; i <= 5; i++ {
		bu := testutil.NewBusinessUnitBuilder().
			WithName(fmt.Sprintf("Business %d", i)).
			WithAdminPhone(fmt.Sprintf("+97250%07d", i)).
			Build()
		resp := client.POST(t, "/api/v1/business-units", bu)
		testutil.AssertStatusCode(t, resp, http.StatusCreated)
	}

	testCases := []struct {
		name           string
		limit          int
		offset         int
		expectedCount  int
		expectedTotal  int64
	}{
		{name: "first page", limit: 2, offset: 0, expectedCount: 2, expectedTotal: 5},
		{name: "second page", limit: 2, offset: 2, expectedCount: 2, expectedTotal: 5},
		{name: "last page", limit: 2, offset: 4, expectedCount: 1, expectedTotal: 5},
		{name: "all items", limit: 10, offset: 0, expectedCount: 5, expectedTotal: 5},
		{name: "beyond total", limit: 5, offset: 10, expectedCount: 0, expectedTotal: 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			url := fmt.Sprintf("/api/v1/business-units?limit=%d&offset=%d", tc.limit, tc.offset)
			resp := client.GET(t, url)

			// Assert
			testutil.AssertStatusCode(t, resp, http.StatusOK)

			var response struct {
				Data       []model.BusinessUnit `json:"data"`
				TotalCount int64                `json:"total_count"`
				Limit      int                  `json:"limit"`
				Offset     int                  `json:"offset"`
			}
			if err := resp.UnmarshalJSON(&response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if response.TotalCount != tc.expectedTotal {
				t.Errorf("expected total_count %d, got %d", tc.expectedTotal, response.TotalCount)
			}
			if len(response.Data) != tc.expectedCount {
				t.Errorf("expected %d items, got %d", tc.expectedCount, len(response.Data))
			}
			if response.Limit != tc.limit {
				t.Errorf("expected limit %d, got %d", tc.limit, response.Limit)
			}
			if response.Offset != tc.offset {
				t.Errorf("expected offset %d, got %d", tc.offset, response.Offset)
			}
		})
	}
}

func TestGetAll_DefaultPagination(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create 15 business units
	for i := 1; i <= 15; i++ {
		bu := testutil.NewBusinessUnitBuilder().
			WithName(fmt.Sprintf("Business %d", i)).
			WithAdminPhone(fmt.Sprintf("+97250%07d", i)).
			Build()
		resp := client.POST(t, "/api/v1/business-units", bu)
		testutil.AssertStatusCode(t, resp, http.StatusCreated)
	}

	// Act - Request without limit/offset
	resp := client.GET(t, "/api/v1/business-units")

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusOK)

	var response struct {
		Data       []model.BusinessUnit `json:"data"`
		TotalCount int64                `json:"total_count"`
		Limit      int                  `json:"limit"`
	}
	if err := resp.UnmarshalJSON(&response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.TotalCount != 15 {
		t.Errorf("expected total_count 15, got %d", response.TotalCount)
	}
	// Default limit should be 10
	if len(response.Data) != 10 {
		t.Errorf("expected 10 items (default limit), got %d", len(response.Data))
	}
}

func TestGetAll_MaxLimit(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Act - Request with limit > 100 (should be capped at 100)
	resp := client.GET(t, "/api/v1/business-units?limit=200")

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusOK)

	var response struct {
		Limit int `json:"limit"`
	}
	if err := resp.UnmarshalJSON(&response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Limit should be capped at 100
	if response.Limit != 100 {
		t.Errorf("expected limit to be capped at 100, got %d", response.Limit)
	}
}
