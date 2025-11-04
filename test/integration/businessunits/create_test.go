package businessunits

import (
	"net/http"
	"testing"

	"skeji/pkg/model"
	"skeji/test/integration/common"
)

func TestCreate_ValidBusinessUnit(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := ValidBusinessUnit()

	resp := client.POST(t, "/api/v1/business-units", bu)

	common.AssertStatusCode(t, resp, http.StatusCreated)

	var created model.BusinessUnit
	if err := resp.DecodeJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if created.ID == "" {
		t.Error("expected ID to be set")
	}
	if created.Name != bu.Name {
		t.Errorf("expected name %q, got %q", bu.Name, created.Name)
	}
	if len(created.Cities) == 0 {
		t.Error("expected cities to be set")
	}
	if created.AdminPhone != bu.AdminPhone {
		t.Errorf("expected admin_phone %q, got %q", bu.AdminPhone, created.AdminPhone)
	}

	count := mongo.CountDocuments(t, common.BusinessUnitsCollection)
	if count != 1 {
		t.Errorf("expected 1 document in DB, got %d", count)
	}
}

func TestCreate_MinimalBusinessUnit(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := MinimalBusinessUnit()

	resp := client.POST(t, "/api/v1/business-units", bu)

	common.AssertStatusCode(t, resp, http.StatusCreated)

	var created model.BusinessUnit
	if err := resp.DecodeJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if created.Priority == 0 {
		t.Error("expected default priority to be set")
	}
	if created.TimeZone == "" {
		t.Error("expected timezone to be set")
	}
}

func TestCreate_EmptyBusinessUnit(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := EmptyBusinessUnit()

	resp := client.POST(t, "/api/v1/business-units", bu)

	common.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
	common.AssertContains(t, resp, "validation")

	count := mongo.CountDocuments(t, common.BusinessUnitsCollection)
	if count != 0 {
		t.Errorf("expected 0 documents in DB, got %d", count)
	}
}

func TestCreate_MissingRequiredFields(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	testCases := []struct {
		name string
		bu   model.BusinessUnit
		want string
	}{
		{
			name: "missing name",
			bu: model.BusinessUnit{
				Cities:     []string{"Tel Aviv"},
				Labels:     []string{"cafe"},
				AdminPhone: "+972501234567",
			},
			want: "name",
		},
		{
			name: "missing cities",
			bu: model.BusinessUnit{
				Name:       "Test Business",
				Labels:     []string{"cafe"},
				AdminPhone: "+972501234567",
			},
			want: "cities",
		},
		{
			name: "missing labels",
			bu: model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{"Tel Aviv"},
				AdminPhone: "+972501234567",
			},
			want: "labels",
		},
		{
			name: "missing admin_phone",
			bu: model.BusinessUnit{
				Name:   "Test Business",
				Cities: []string{"Tel Aviv"},
				Labels: []string{"cafe"},
			},
			want: "admin_phone",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mongo.CleanCollection(t, common.BusinessUnitsCollection)

			resp := client.POST(t, "/api/v1/business-units", tc.bu)

			common.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
			common.AssertContains(t, resp, tc.want)

			count := mongo.CountDocuments(t, common.BusinessUnitsCollection)
			if count != 0 {
				t.Errorf("expected 0 documents in DB, got %d", count)
			}
		})
	}
}

func TestCreate_EmptyCitiesOrLabels(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	testCases := []struct {
		name string
		bu   model.BusinessUnit
	}{
		{
			name: "empty cities array",
			bu: model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{},
				Labels:     []string{"cafe"},
				AdminPhone: "+972501234567",
			},
		},
		{
			name: "empty labels array",
			bu: model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{"Tel Aviv"},
				Labels:     []string{},
				AdminPhone: "+972501234567",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mongo.CleanCollection(t, common.BusinessUnitsCollection)

			resp := client.POST(t, "/api/v1/business-units", tc.bu)

			common.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)

			count := mongo.CountDocuments(t, common.BusinessUnitsCollection)
			if count != 0 {
				t.Errorf("expected 0 documents in DB, got %d", count)
			}
		})
	}
}

func TestCreate_InvalidPhoneFormat(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	testCases := []struct {
		name  string
		phone string
	}{
		{name: "no plus sign", phone: "972501234567"},
		{name: "letters", phone: "+97250ABC1234"},
		{name: "too short", phone: "+972123"},
		{name: "no country code", phone: "0501234567"},
		{name: "spaces", phone: "+972 50 123 4567"},
		{name: "dashes", phone: "+972-50-123-4567"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mongo.CleanCollection(t, common.BusinessUnitsCollection)

			bu := NewBusinessUnitBuilder().
				WithAdminPhone(tc.phone).
				Build()

			resp := client.POST(t, "/api/v1/business-units", bu)

			common.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
			common.AssertContains(t, resp, "phone")

			count := mongo.CountDocuments(t, common.BusinessUnitsCollection)
			if count != 0 {
				t.Errorf("expected 0 documents in DB, got %d", count)
			}
		})
	}
}

func TestCreate_UnsupportedCountry(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	testCases := []struct {
		name    string
		phone   string
		country string
	}{
		{name: "UK", phone: "+441234567890", country: "UK"},
		{name: "Germany", phone: "+491234567890", country: "Germany"},
		{name: "France", phone: "+331234567890", country: "France"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mongo.CleanCollection(t, common.BusinessUnitsCollection)

			bu := NewBusinessUnitBuilder().
				WithAdminPhone(tc.phone).
				Build()

			resp := client.POST(t, "/api/v1/business-units", bu)

			common.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
			common.AssertContains(t, resp, "supported country")

			count := mongo.CountDocuments(t, common.BusinessUnitsCollection)
			if count != 0 {
				t.Errorf("expected 0 documents in DB, got %d", count)
			}
		})
	}
}

func TestCreate_MultipleBusinessUnits(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	businessUnits := []model.BusinessUnit{
		NewBusinessUnitBuilder().WithName("Business 1").WithAdminPhone("+972501111111").Build(),
		NewBusinessUnitBuilder().WithName("Business 2").WithAdminPhone("+972502222222").Build(),
		NewBusinessUnitBuilder().WithName("Business 3").WithAdminPhone("+972503333333").Build(),
	}

	for _, bu := range businessUnits {
		resp := client.POST(t, "/api/v1/business-units", bu)
		common.AssertStatusCode(t, resp, http.StatusCreated)
	}

	count := mongo.CountDocuments(t, common.BusinessUnitsCollection)
	if count != int64(len(businessUnits)) {
		t.Errorf("expected %d documents in DB, got %d", len(businessUnits), count)
	}
}

func TestCreate_SameAdminPhoneMultipleBusinesses(t *testing.T) {
	env := common.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	adminPhone := "+972501234567"
	bu1 := NewBusinessUnitBuilder().
		WithName("Business 1").
		WithAdminPhone(adminPhone).
		WithCities("Tel Aviv").
		Build()

	bu2 := NewBusinessUnitBuilder().
		WithName("Business 2").
		WithAdminPhone(adminPhone).
		WithCities("Jerusalem").
		Build()

	resp1 := client.POST(t, "/api/v1/business-units", bu1)
	resp2 := client.POST(t, "/api/v1/business-units", bu2)

	common.AssertStatusCode(t, resp1, http.StatusCreated)
	common.AssertStatusCode(t, resp2, http.StatusCreated)

	count := mongo.CountDocuments(t, common.BusinessUnitsCollection)
	if count != 2 {
		t.Errorf("expected 2 documents in DB, got %d", count)
	}
}
