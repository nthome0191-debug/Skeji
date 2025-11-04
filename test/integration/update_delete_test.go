package integration

import (
	"fmt"
	"net/http"
	"testing"

	"skeji/pkg/model"
	"skeji/test/integration/testutil"
)

// ============================================================================
// UPDATE Tests
// ============================================================================

func TestUpdate_ValidUpdate(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create a business unit
	bu := testutil.ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	// Prepare update
	update := testutil.ValidBusinessUnitUpdate()

	// Act
	updateResp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)

	// Assert
	testutil.AssertStatusCode(t, updateResp, http.StatusNoContent)

	// Verify the update
	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	testutil.AssertStatusCode(t, getResp, http.StatusOK)

	var updated model.BusinessUnit
	if err := getResp.UnmarshalJSON(&updated); err != nil {
		t.Fatalf("failed to unmarshal get response: %v", err)
	}

	if updated.Name != update.Name {
		t.Errorf("expected name %q, got %q", update.Name, updated.Name)
	}
	if updated.AdminPhone != update.AdminPhone {
		t.Errorf("expected admin_phone %q, got %q", update.AdminPhone, updated.AdminPhone)
	}
}

func TestUpdate_PartialUpdate(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create a business unit
	bu := testutil.ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	// Prepare partial update (only name)
	update := testutil.PartialBusinessUnitUpdate()

	// Act
	updateResp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)

	// Assert
	testutil.AssertStatusCode(t, updateResp, http.StatusNoContent)

	// Verify the update
	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	testutil.AssertStatusCode(t, getResp, http.StatusOK)

	var updated model.BusinessUnit
	if err := getResp.UnmarshalJSON(&updated); err != nil {
		t.Fatalf("failed to unmarshal get response: %v", err)
	}

	// Name should be updated
	if updated.Name != update.Name {
		t.Errorf("expected name %q, got %q", update.Name, updated.Name)
	}
	// Other fields should remain unchanged
	if updated.AdminPhone != created.AdminPhone {
		t.Errorf("expected admin_phone to remain %q, got %q", created.AdminPhone, updated.AdminPhone)
	}
}

func TestUpdate_EmptyUpdate(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create a business unit
	bu := testutil.ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	// Prepare empty update
	update := testutil.EmptyBusinessUnitUpdate()

	// Act
	updateResp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)

	// Assert - Empty update should succeed (no changes)
	testutil.AssertStatusCode(t, updateResp, http.StatusNoContent)

	// Verify nothing changed
	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	testutil.AssertStatusCode(t, getResp, http.StatusOK)

	var updated model.BusinessUnit
	if err := getResp.UnmarshalJSON(&updated); err != nil {
		t.Fatalf("failed to unmarshal get response: %v", err)
	}

	if updated.Name != created.Name {
		t.Error("expected name to remain unchanged")
	}
}

func TestUpdate_NonExistentID(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange
	nonExistentID := "507f1f77bcf86cd799439011"
	update := testutil.ValidBusinessUnitUpdate()

	// Act
	resp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", nonExistentID), update)

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusNotFound)
	testutil.AssertContains(t, resp, "not found")
}

func TestUpdate_InvalidID(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	testCases := []struct {
		name string
		id   string
	}{
		{name: "empty string", id: ""},
		{name: "invalid format", id: "invalid-id"},
		{name: "special characters", id: "abc@#$%"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			update := testutil.ValidBusinessUnitUpdate()

			// Act
			resp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", tc.id), update)

			// Assert - Expecting either 400 or 404
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
				t.Errorf("expected status 400 or 404, got %d", resp.StatusCode)
			}
		})
	}
}

func TestUpdate_InvalidPhoneFormat(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create a business unit
	bu := testutil.ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	testCases := []struct {
		name  string
		phone string
	}{
		{name: "invalid format", phone: "invalid-phone"},
		{name: "no plus", phone: "972501234567"},
		{name: "letters", phone: "+97250ABC1234"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			update := model.BusinessUnitUpdate{
				AdminPhone: tc.phone,
			}

			// Act
			resp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)

			// Assert
			testutil.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
			testutil.AssertContains(t, resp, "phone")
		})
	}
}

// ============================================================================
// DELETE Tests
// ============================================================================

func TestDelete_ExistingBusinessUnit(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create a business unit
	bu := testutil.ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	// Verify it exists
	initialCount := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
	if initialCount != 1 {
		t.Fatalf("expected 1 document before delete, got %d", initialCount)
	}

	// Act
	deleteResp := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	// Assert
	testutil.AssertStatusCode(t, deleteResp, http.StatusNoContent)

	// Verify it's deleted
	count := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
	if count != 0 {
		t.Errorf("expected 0 documents after delete, got %d", count)
	}

	// Verify GET returns 404
	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	testutil.AssertStatusCode(t, getResp, http.StatusNotFound)
}

func TestDelete_NonExistentID(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Act
	nonExistentID := "507f1f77bcf86cd799439011"
	resp := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", nonExistentID))

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusNotFound)
	testutil.AssertContains(t, resp, "not found")
}

func TestDelete_InvalidID(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	testCases := []struct {
		name string
		id   string
	}{
		{name: "empty string", id: ""},
		{name: "invalid format", id: "invalid-id"},
		{name: "special characters", id: "abc@#$%"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			resp := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", tc.id))

			// Assert - Expecting either 400 or 404
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
				t.Errorf("expected status 400 or 404, got %d", resp.StatusCode)
			}
		})
	}
}

func TestDelete_DeletedIDCannotBeRetrieved(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create and delete a business unit
	bu := testutil.ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	deleteResp := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	testutil.AssertStatusCode(t, deleteResp, http.StatusNoContent)

	// Act - Try to get the deleted business unit
	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	// Assert
	testutil.AssertStatusCode(t, getResp, http.StatusNotFound)
}

func TestDelete_DoubleDelete(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create a business unit
	bu := testutil.ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	// Act - Delete once
	deleteResp1 := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	testutil.AssertStatusCode(t, deleteResp1, http.StatusNoContent)

	// Try to delete again
	deleteResp2 := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	// Assert - Second delete should return 404
	testutil.AssertStatusCode(t, deleteResp2, http.StatusNotFound)
}
