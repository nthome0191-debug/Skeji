package businessunits

import (
	"fmt"
	"net/http"
	"testing"

	"skeji/pkg/model"
	"skeji/test/integration/common"
)

func TestUpdate_ValidUpdate(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.DecodeJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	update := ValidBusinessUnitUpdate()

	updateResp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)

	common.AssertStatusCode(t, updateResp, http.StatusNoContent)

	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, http.StatusOK)

	var updated model.BusinessUnit
	if err := getResp.DecodeJSON(&updated); err != nil {
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
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.DecodeJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	update := PartialBusinessUnitUpdate()

	updateResp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)

	common.AssertStatusCode(t, updateResp, http.StatusNoContent)

	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, http.StatusOK)

	var updated model.BusinessUnit
	if err := getResp.DecodeJSON(&updated); err != nil {
		t.Fatalf("failed to unmarshal get response: %v", err)
	}

	if updated.Name != update.Name {
		t.Errorf("expected name %q, got %q", update.Name, updated.Name)
	}

	if updated.AdminPhone != created.AdminPhone {
		t.Errorf("expected admin_phone to remain %q, got %q", created.AdminPhone, updated.AdminPhone)
	}
}

func TestUpdate_EmptyUpdate(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.DecodeJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	update := EmptyBusinessUnitUpdate()

	updateResp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)

	common.AssertStatusCode(t, updateResp, http.StatusNoContent)

	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, http.StatusOK)

	var updated model.BusinessUnit
	if err := getResp.DecodeJSON(&updated); err != nil {
		t.Fatalf("failed to unmarshal get response: %v", err)
	}

	if updated.Name != created.Name {
		t.Error("expected name to remain unchanged")
	}
}

func TestUpdate_NonExistentID(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	nonExistentID := "507f1f77bcf86cd799439011"
	update := ValidBusinessUnitUpdate()

	resp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", nonExistentID), update)

	common.AssertStatusCode(t, resp, http.StatusNotFound)
	common.AssertContains(t, resp, "not found")
}

func TestUpdate_InvalidID(t *testing.T) {
	env := common.NewTestEnv()
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
			update := ValidBusinessUnitUpdate()

			resp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", tc.id), update)

			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
				t.Errorf("expected status 400 or 404, got %d", resp.StatusCode)
			}
		})
	}
}

func TestUpdate_InvalidPhoneFormat(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.DecodeJSON(&created); err != nil {
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

			update := model.BusinessUnitUpdate{
				AdminPhone: tc.phone,
			}

			resp := client.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)

			common.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
			common.AssertContains(t, resp, "phone")
		})
	}
}

func TestDelete_ExistingBusinessUnit(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.DecodeJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	initialCount := mongo.CountDocuments(t, common.BusinessUnitsCollection)
	if initialCount != 1 {
		t.Fatalf("expected 1 document before delete, got %d", initialCount)
	}

	deleteResp := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	common.AssertStatusCode(t, deleteResp, http.StatusNoContent)

	count := mongo.CountDocuments(t, common.BusinessUnitsCollection)
	if count != 0 {
		t.Errorf("expected 0 documents after delete, got %d", count)
	}

	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, http.StatusNotFound)
}

func TestDelete_NonExistentID(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	nonExistentID := "507f1f77bcf86cd799439011"
	resp := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", nonExistentID))

	common.AssertStatusCode(t, resp, http.StatusNotFound)
	common.AssertContains(t, resp, "not found")
}

func TestDelete_InvalidID(t *testing.T) {
	env := common.NewTestEnv()
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
			resp := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", tc.id))

			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
				t.Errorf("expected status 400 or 404, got %d", resp.StatusCode)
			}
		})
	}
}

func TestDelete_DeletedIDCannotBeRetrieved(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.DecodeJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	deleteResp := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, deleteResp, http.StatusNoContent)

	getResp := client.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	common.AssertStatusCode(t, getResp, http.StatusNotFound)
}

func TestDelete_DoubleDelete(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := ValidBusinessUnit()
	createResp := client.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, http.StatusCreated)

	var created model.BusinessUnit
	if err := createResp.DecodeJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	deleteResp1 := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, deleteResp1, http.StatusNoContent)

	deleteResp2 := client.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	common.AssertStatusCode(t, deleteResp2, http.StatusNotFound)
}
