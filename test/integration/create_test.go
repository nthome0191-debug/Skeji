package integration

import (
	"net/http"
	"testing"

	"skeji/pkg/model"
	"skeji/test/integration/testutil"
)

func TestCreate_ValidBusinessUnit(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange
	bu := testutil.ValidBusinessUnit()

	// Act
	resp := client.POST(t, "/api/v1/business-units", bu)

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusCreated)

	var created model.BusinessUnit
	if err := resp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify response contains all fields
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

	// Verify it's actually in the database
	count := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
	if count != 1 {
		t.Errorf("expected 1 document in DB, got %d", count)
	}
}

func TestCreate_MinimalBusinessUnit(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange
	bu := testutil.MinimalBusinessUnit()

	// Act
	resp := client.POST(t, "/api/v1/business-units", bu)

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusCreated)

	var created model.BusinessUnit
	if err := resp.UnmarshalJSON(&created); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify defaults are applied
	if created.Priority == 0 {
		t.Error("expected default priority to be set")
	}
	if created.TimeZone == "" {
		t.Error("expected timezone to be set")
	}
}

func TestCreate_EmptyBusinessUnit(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange
	bu := testutil.EmptyBusinessUnit()

	// Act
	resp := client.POST(t, "/api/v1/business-units", bu)

	// Assert
	testutil.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
	testutil.AssertContains(t, resp, "validation")

	// Verify nothing was created
	count := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
	if count != 0 {
		t.Errorf("expected 0 documents in DB, got %d", count)
	}
}

func TestCreate_MissingRequiredFields(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	testCases := []struct {
		name string
		bu   model.BusinessUnit
		want string // expected error substring
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
			// Clean before each subtest
			mongo.CleanCollection(t, testutil.BusinessUnitsCollection)

			// Act
			resp := client.POST(t, "/api/v1/business-units", tc.bu)

			// Assert
			testutil.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
			testutil.AssertContains(t, resp, tc.want)

			// Verify nothing was created
			count := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
			if count != 0 {
				t.Errorf("expected 0 documents in DB, got %d", count)
			}
		})
	}
}

func TestCreate_EmptyCitiesOrLabels(t *testing.T) {
	env := testutil.NewTestEnv()
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
			// Clean before each subtest
			mongo.CleanCollection(t, testutil.BusinessUnitsCollection)

			// Act
			resp := client.POST(t, "/api/v1/business-units", tc.bu)

			// Assert
			testutil.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)

			// Verify nothing was created
			count := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
			if count != 0 {
				t.Errorf("expected 0 documents in DB, got %d", count)
			}
		})
	}
}

func TestCreate_InvalidPhoneFormat(t *testing.T) {
	env := testutil.NewTestEnv()
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
			// Clean before each subtest
			mongo.CleanCollection(t, testutil.BusinessUnitsCollection)

			// Arrange
			bu := testutil.NewBusinessUnitBuilder().
				WithAdminPhone(tc.phone).
				Build()

			// Act
			resp := client.POST(t, "/api/v1/business-units", bu)

			// Assert
			testutil.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
			testutil.AssertContains(t, resp, "phone")

			// Verify nothing was created
			count := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
			if count != 0 {
				t.Errorf("expected 0 documents in DB, got %d", count)
			}
		})
	}
}

func TestCreate_UnsupportedCountry(t *testing.T) {
	env := testutil.NewTestEnv()
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
			// Clean before each subtest
			mongo.CleanCollection(t, testutil.BusinessUnitsCollection)

			// Arrange
			bu := testutil.NewBusinessUnitBuilder().
				WithAdminPhone(tc.phone).
				Build()

			// Act
			resp := client.POST(t, "/api/v1/business-units", bu)

			// Assert
			testutil.AssertStatusCode(t, resp, http.StatusUnprocessableEntity)
			testutil.AssertContains(t, resp, "supported country")

			// Verify nothing was created
			count := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
			if count != 0 {
				t.Errorf("expected 0 documents in DB, got %d", count)
			}
		})
	}
}

func TestCreate_MultipleBusinessUnits(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Create multiple business units
	businessUnits := []model.BusinessUnit{
		testutil.NewBusinessUnitBuilder().WithName("Business 1").WithAdminPhone("+972501111111").Build(),
		testutil.NewBusinessUnitBuilder().WithName("Business 2").WithAdminPhone("+972502222222").Build(),
		testutil.NewBusinessUnitBuilder().WithName("Business 3").WithAdminPhone("+972503333333").Build(),
	}

	// Act - Create each business unit
	for _, bu := range businessUnits {
		resp := client.POST(t, "/api/v1/business-units", bu)
		testutil.AssertStatusCode(t, resp, http.StatusCreated)
	}

	// Assert - Verify all were created
	count := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
	if count != int64(len(businessUnits)) {
		t.Errorf("expected %d documents in DB, got %d", len(businessUnits), count)
	}
}

func TestCreate_SameAdminPhoneMultipleBusinesses(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	// Arrange - Same admin manages multiple businesses
	adminPhone := "+972501234567"
	bu1 := testutil.NewBusinessUnitBuilder().
		WithName("Business 1").
		WithAdminPhone(adminPhone).
		WithCities("Tel Aviv").
		Build()

	bu2 := testutil.NewBusinessUnitBuilder().
		WithName("Business 2").
		WithAdminPhone(adminPhone).
		WithCities("Jerusalem").
		Build()

	// Act
	resp1 := client.POST(t, "/api/v1/business-units", bu1)
	resp2 := client.POST(t, "/api/v1/business-units", bu2)

	// Assert - Both should succeed
	testutil.AssertStatusCode(t, resp1, http.StatusCreated)
	testutil.AssertStatusCode(t, resp2, http.StatusCreated)

	// Verify both were created
	count := mongo.CountDocuments(t, testutil.BusinessUnitsCollection)
	if count != 2 {
		t.Errorf("expected 2 documents in DB, got %d", count)
	}
}
