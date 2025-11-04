package integration

import (
	"net/http"
	"net/url"
	"testing"

	"skeji/pkg/model"
	"skeji/test/integration/testutil"
)

func TestSearch_ValidCitiesAndLabels(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	businessUnits := []model.BusinessUnit{
		testutil.NewBusinessUnitBuilder().WithName("TLV Barbershop").WithCities("Tel Aviv").WithLabels("barbershop").WithAdminPhone("+972501111111").Build(),
		testutil.NewBusinessUnitBuilder().WithName("TLV Cafe").WithCities("Tel Aviv").WithLabels("cafe").WithAdminPhone("+972502222222").Build(),
		testutil.NewBusinessUnitBuilder().WithName("JLM Barbershop").WithCities("Jerusalem").WithLabels("barbershop").WithAdminPhone("+972503333333").Build(),
	}

	for _, bu := range businessUnits {
		resp := client.POST(t, "/api/v1/business-units", bu)
		testutil.AssertStatusCode(t, resp, http.StatusCreated)
	}

	searchURL := "/api/v1/business-units/search?cities=Tel%20Aviv&labels=barbershop"
	resp := client.GET(t, searchURL)

	testutil.AssertStatusCode(t, resp, http.StatusOK)

	var results []model.BusinessUnit
	if err := resp.DecodeJSON(&results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Name != "TLV Barbershop" {
		t.Errorf("expected 'TLV Barbershop', got %q", results[0].Name)
	}
}

func TestSearch_MultipleCitiesAndLabels(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	businessUnits := []model.BusinessUnit{
		testutil.NewBusinessUnitBuilder().WithName("TLV Multi").WithCities("Tel Aviv", "Haifa").WithLabels("barbershop", "salon").WithAdminPhone("+972501111111").Build(),
		testutil.NewBusinessUnitBuilder().WithName("JLM Single").WithCities("Jerusalem").WithLabels("barbershop").WithAdminPhone("+972502222222").Build(),
		testutil.NewBusinessUnitBuilder().WithName("Haifa Cafe").WithCities("Haifa").WithLabels("cafe").WithAdminPhone("+972503333333").Build(),
	}

	for _, bu := range businessUnits {
		resp := client.POST(t, "/api/v1/business-units", bu)
		testutil.AssertStatusCode(t, resp, http.StatusCreated)
	}

	searchURL := "/api/v1/business-units/search?cities=Tel%20Aviv,Haifa&labels=barbershop,salon"
	resp := client.GET(t, searchURL)

	testutil.AssertStatusCode(t, resp, http.StatusOK)

	var results []model.BusinessUnit
	if err := resp.DecodeJSON(&results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least 1 result, got 0")
	}
}

func TestSearch_NoResults(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := testutil.NewBusinessUnitBuilder().
		WithCities("Tel Aviv").
		WithLabels("barbershop").
		Build()
	resp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, resp, http.StatusCreated)

	searchURL := "/api/v1/business-units/search?cities=Eilat&labels=restaurant"
	resp = client.GET(t, searchURL)

	testutil.AssertStatusCode(t, resp, http.StatusOK)

	var results []model.BusinessUnit
	if err := resp.DecodeJSON(&results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_MissingCities(t *testing.T) {
	env := testutil.NewTestEnv()
	_, client := env.Setup(t)
	defer env.Cleanup(t, nil)

	searchURL := "/api/v1/business-units/search?labels=barbershop"
	resp := client.GET(t, searchURL)

	testutil.AssertStatusCode(t, resp, http.StatusBadRequest)
	testutil.AssertContains(t, resp, "cities")
}

func TestSearch_MissingLabels(t *testing.T) {
	env := testutil.NewTestEnv()
	_, client := env.Setup(t)
	defer env.Cleanup(t, nil)

	searchURL := "/api/v1/business-units/search?cities=Tel%20Aviv"
	resp := client.GET(t, searchURL)

	testutil.AssertStatusCode(t, resp, http.StatusBadRequest)
	testutil.AssertContains(t, resp, "labels")
}

func TestSearch_EmptyParameters(t *testing.T) {
	env := testutil.NewTestEnv()
	_, client := env.Setup(t)
	defer env.Cleanup(t, nil)

	testCases := []struct {
		name        string
		cities      string
		labels      string
		expectError bool
	}{
		{name: "empty cities", cities: "", labels: "barbershop", expectError: true},
		{name: "empty labels", cities: "Tel Aviv", labels: "", expectError: true},
		{name: "both empty", cities: "", labels: "", expectError: true},
		{name: "whitespace cities", cities: "   ", labels: "barbershop", expectError: true},
		{name: "whitespace labels", cities: "Tel Aviv", labels: "   ", expectError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			searchURL := "/api/v1/business-units/search?cities=" + url.QueryEscape(tc.cities) + "&labels=" + url.QueryEscape(tc.labels)
			resp := client.GET(t, searchURL)

			if tc.expectError {
				if resp.StatusCode == http.StatusOK {
					t.Error("expected error status, got 200")
				}
			}
		})
	}
}

func TestSearch_SortedByPriority(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	businessUnits := []model.BusinessUnit{
		testutil.NewBusinessUnitBuilder().WithName("Low Priority").WithPriority(10).WithCities("Tel Aviv").WithLabels("cafe").WithAdminPhone("+972501111111").Build(),
		testutil.NewBusinessUnitBuilder().WithName("High Priority").WithPriority(100).WithCities("Tel Aviv").WithLabels("cafe").WithAdminPhone("+972502222222").Build(),
		testutil.NewBusinessUnitBuilder().WithName("Medium Priority").WithPriority(50).WithCities("Tel Aviv").WithLabels("cafe").WithAdminPhone("+972503333333").Build(),
	}

	for _, bu := range businessUnits {
		resp := client.POST(t, "/api/v1/business-units", bu)
		testutil.AssertStatusCode(t, resp, http.StatusCreated)
	}

	searchURL := "/api/v1/business-units/search?cities=Tel%20Aviv&labels=cafe"
	resp := client.GET(t, searchURL)

	testutil.AssertStatusCode(t, resp, http.StatusOK)

	var results []model.BusinessUnit
	if err := resp.DecodeJSON(&results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Name != "High Priority" {
		t.Errorf("expected first result to be 'High Priority', got %q", results[0].Name)
	}
	if results[1].Name != "Medium Priority" {
		t.Errorf("expected second result to be 'Medium Priority', got %q", results[1].Name)
	}
	if results[2].Name != "Low Priority" {
		t.Errorf("expected third result to be 'Low Priority', got %q", results[2].Name)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	bu := testutil.NewBusinessUnitBuilder().
		WithCities("Tel Aviv").
		WithLabels("Barbershop").
		Build()
	resp := client.POST(t, "/api/v1/business-units", bu)
	testutil.AssertStatusCode(t, resp, http.StatusCreated)

	testCases := []struct {
		name   string
		cities string
		labels string
	}{
		{name: "lowercase", cities: "tel aviv", labels: "barbershop"},
		{name: "uppercase", cities: "TEL AVIV", labels: "BARBERSHOP"},
		{name: "mixed case", cities: "Tel AVIV", labels: "BarberSHOP"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			searchURL := "/api/v1/business-units/search?cities=" + url.QueryEscape(tc.cities) + "&labels=" + url.QueryEscape(tc.labels)
			resp := client.GET(t, searchURL)

			testutil.AssertStatusCode(t, resp, http.StatusOK)

			var results []model.BusinessUnit
			if err := resp.DecodeJSON(&results); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if len(results) != 1 {
				t.Errorf("expected 1 result for %q + %q, got %d", tc.cities, tc.labels, len(results))
			}
		})
	}
}

func TestSearch_CommaDelimitedValues(t *testing.T) {
	env := testutil.NewTestEnv()
	mongo, client := env.Setup(t)
	defer env.Cleanup(t, mongo)

	businessUnits := []model.BusinessUnit{
		testutil.NewBusinessUnitBuilder().WithName("TLV Barber").WithCities("Tel Aviv").WithLabels("barbershop").WithAdminPhone("+972501111111").Build(),
		testutil.NewBusinessUnitBuilder().WithName("JLM Barber").WithCities("Jerusalem").WithLabels("barbershop").WithAdminPhone("+972502222222").Build(),
		testutil.NewBusinessUnitBuilder().WithName("TLV Cafe").WithCities("Tel Aviv").WithLabels("cafe").WithAdminPhone("+972503333333").Build(),
	}

	for _, bu := range businessUnits {
		resp := client.POST(t, "/api/v1/business-units", bu)
		testutil.AssertStatusCode(t, resp, http.StatusCreated)
	}

	searchURL := "/api/v1/business-units/search?cities=Tel%20Aviv,Jerusalem&labels=barbershop"
	resp := client.GET(t, searchURL)

	testutil.AssertStatusCode(t, resp, http.StatusOK)

	var results []model.BusinessUnit
	if err := resp.DecodeJSON(&results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}
