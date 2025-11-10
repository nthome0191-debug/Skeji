package integrationtests

import (
	"fmt"
	"os"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/test/common"
	"testing"
)

const (
	ServiceName = "business-units-integration-tests"
	TableName   = "business-units"
)

var (
	cfg        *config.Config
	httpClient *common.Client
)

func TestMain(t *testing.T) {
	setup()
	testGet(t)
	testPost(t)
	testUpdate(t)
	testDelete(t)
	teardown()
}

func setup() {
	cfg = config.Load(ServiceName)

	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	httpClient = common.NewClient(serverURL)
}

func teardown() {
	cfg.GracefulShutdown()
}

func createValidBusinessUnit(name, phone string) map[string]any {
	return map[string]any{
		"name":        name,
		"cities":      []string{"Tel Aviv", "Jerusalem"},
		"labels":      []string{"Haircut", "Styling"},
		"admin_phone": phone,
		"priority":    1,
		"time_zone":   "Asia/Jerusalem",
	}
}

func decodeBusinessUnit(t *testing.T, resp *common.Response) *model.BusinessUnit {
	t.Helper()
	var result struct {
		Data model.BusinessUnit `json:"data"`
	}
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return &result.Data
}

func decodeBusinessUnits(t *testing.T, resp *common.Response) []model.BusinessUnit {
	t.Helper()
	var result struct {
		Data []model.BusinessUnit `json:"data"`
	}
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return result.Data
}

func decodePaginated(t *testing.T, resp *common.Response) ([]model.BusinessUnit, int, int, int) {
	t.Helper()
	var result struct {
		Data       []model.BusinessUnit `json:"data"`
		TotalCount int                  `json:"total_count"`
		Limit      int                  `json:"limit"`
		Offset     int                  `json:"offset"`
	}
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return result.Data, result.TotalCount, result.Limit, result.Offset
}

func testGet(t *testing.T) {
	testGetByIdEmptyTable(t)
	testGetBySearchEmptyTable(t)
	testGetAllPaginatedEmptyTable(t)
	testGetValidIdExistingRecord(t)
	testGetInvalidIdExistingRecord(t)
	testGetValidSearchExistingRecords(t)
	testGetInvalidSearchExistingRecords(t)
	testGetValidPaginationExistingRecords(t)
	testGetInvalidPaginationExistingRecords(t)
	testGetVerifyCreatedAt(t)
	testGetSearchPriorityOrdering(t)
	testGetPaginationEdgeCases(t)
	testGetSearchNormalization(t)
	testGetSearchMultipleCitiesLabels(t)
}

func testPost(t *testing.T) {
	testPostValidRecord(t)
	testPostInvalidRecord(t)
	testPostWithExtraJsonKeys(t)
	testPostWithMissingRelevantKeys(t)
	testPostWithWebsiteURL(t)
	testPostWithMaintainers(t)
	testPostWithArrayMaxLengths(t)
	testPostWithNameBoundaries(t)
	testPostWithPriorityValues(t)
	testPostMalformedJSON(t)
	testPostWithUSPhoneNumber(t)
	testPostWithSpecialCharacters(t)
	testPostDuplicateDetection(t)
}

func testUpdate(t *testing.T) {
	testUpdateNonExistingRecord(t)
	testUpdateWithInvalidId(t)
	testUpdateDeletedRecord(t)
	testUpdateWithBadFormatKeys(t)
	testUpdateWithGoodFormatKeys(t)
	testUpdateWithEmptyJson(t)
	testUpdateWebsiteURL(t)
	testUpdateMaintainers(t)
	testUpdateArraysToMaxLength(t)
	testUpdatePriorityEdgeCases(t)
	testUpdateClearOptionalFields(t)
	testUpdateMalformedJSON(t)
}

func testDelete(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	testDeleteNonExistingRecord(t)
	testDeleteWithInvalidId(t)
	testDeletedRecord(t)

}

func testGetByIdEmptyTable(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/business-units/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
	common.AssertContains(t, resp, "not found")
}

func testGetBySearchEmptyTable(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=Tel%20Aviv&labels=Haircut")
	common.AssertStatusCode(t, resp, 200)

	data := decodeBusinessUnits(t, resp)
	if len(data) != 0 {
		t.Errorf("expected empty results, got %d", len(data))
	}
}

func testGetAllPaginatedEmptyTable(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/business-units?limit=10&offset=0")
	common.AssertStatusCode(t, resp, 200)

	data, totalCount, _, _ := decodePaginated(t, resp)
	if totalCount != 0 || len(data) != 0 {
		t.Errorf("expected empty results, got total=%d, data=%d", totalCount, len(data))
	}
}

func testGetValidIdExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Get Test Business", "+972541234567")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	resp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, resp, 200)
	result := decodeBusinessUnit(t, resp)

	if result.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, result.ID)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testGetInvalidIdExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Invalid ID Test", "+972541234567")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	resp := httpClient.GET(t, "/api/v1/business-units/id/invalid-id-format")
	common.AssertStatusCode(t, resp, 400)

	resp = httpClient.GET(t, "/api/v1/business-units/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testGetValidSearchExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu1 := createValidBusinessUnit("Tel Aviv Salon", "+972541234567")
	httpClient.POST(t, "/api/v1/business-units", bu1)

	bu2 := createValidBusinessUnit("Jerusalem Spa", "+972541234567")
	bu2["cities"] = []string{"Jerusalem"}
	bu2["labels"] = []string{"Massage"}
	httpClient.POST(t, "/api/v1/business-units", bu2)

	bu3 := createValidBusinessUnit("Haifa Barber", "+972541234567")
	httpClient.POST(t, "/api/v1/business-units", bu3)

	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=Tel%20Aviv&labels=Haircut")
	common.AssertStatusCode(t, resp, 200)
	data := decodeBusinessUnits(t, resp)

	if len(data) < 2 {
		t.Errorf("expected at least 2 results, got %d", len(data))
	}
}

func testGetInvalidSearchExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/business-units/search?labels=Haircut")
	common.AssertStatusCode(t, resp, 400)
	common.AssertContains(t, resp, "cities")

	resp = httpClient.GET(t, "/api/v1/business-units/search?cities=Tel%20Aviv")
	common.AssertStatusCode(t, resp, 400)
	common.AssertContains(t, resp, "labels")

	resp = httpClient.GET(t, "/api/v1/business-units/search?cities=&labels=")
	common.AssertStatusCode(t, resp, 400)
}

func testGetValidPaginationExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	for i := 1; i <= 5; i++ {
		bu := createValidBusinessUnit(fmt.Sprintf("Business %d", i), fmt.Sprintf("+97250%04d", 1120+i))
		httpClient.POST(t, "/api/v1/business-units", bu)
	}

	resp := httpClient.GET(t, "/api/v1/business-units?limit=2&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, totalCount, limit, offset := decodePaginated(t, resp)

	if totalCount < 5 {
		t.Errorf("expected at least 5 total, got %d", totalCount)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 items on first page, got %d", len(data))
	}
	if limit != 2 || offset != 0 {
		t.Errorf("expected limit=2 offset=0, got limit=%d offset=%d", limit, offset)
	}

	resp = httpClient.GET(t, "/api/v1/business-units?limit=2&offset=2")
	common.AssertStatusCode(t, resp, 200)
}

func testGetInvalidPaginationExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/business-units?limit=abc&offset=xyz")
	common.AssertStatusCode(t, resp, 200)

	resp = httpClient.GET(t, "/api/v1/business-units?limit=10&offset=-1")
	common.AssertStatusCode(t, resp, 200)
}

func testGetVerifyCreatedAt(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("CreatedAt Test", "+972523353")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	if created.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}

	originalCreatedAt := created.CreatedAt

	updates := map[string]any{
		"name": "Updated Name",
	}
	httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeBusinessUnit(t, getResp)

	if !fetched.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("created_at should not change on update: original=%v, after_update=%v", originalCreatedAt, fetched.CreatedAt)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testGetSearchPriorityOrdering(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu1 := createValidBusinessUnit("Priority 1", "+972523354")
	bu1["priority"] = 1
	httpClient.POST(t, "/api/v1/business-units", bu1)

	bu2 := createValidBusinessUnit("Priority 5", "+972523355")
	bu2["priority"] = 5
	httpClient.POST(t, "/api/v1/business-units", bu2)

	bu3 := createValidBusinessUnit("Priority 3", "+972523356")
	bu3["priority"] = 3
	httpClient.POST(t, "/api/v1/business-units", bu3)

	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=telaviv&labels=haircut")
	common.AssertStatusCode(t, resp, 200)
	results := decodeBusinessUnits(t, resp)

	if len(results) < 3 {
		t.Errorf("expected at least 3 results, got %d", len(results))
	}

	for i := 1; i < len(results); i++ {
		if results[i-1].Priority < results[i].Priority {
			t.Errorf("results not ordered by priority descending: %d < %d at positions %d and %d",
				results[i-1].Priority, results[i].Priority, i-1, i)
		}
	}
}

func testGetPaginationEdgeCases(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	for i := 0; i < 3; i++ {
		bu := createValidBusinessUnit(fmt.Sprintf("Pagination Test %d", i), fmt.Sprintf("+97252335%d", 7+i))
		httpClient.POST(t, "/api/v1/business-units", bu)
	}

	resp := httpClient.GET(t, "/api/v1/business-units?limit=0&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, _, _, _ := decodePaginated(t, resp)
	if len(data) > 10 {
		t.Errorf("limit=0 should return max 10 results, got %d results", len(data))
	}

	resp = httpClient.GET(t, "/api/v1/business-units?limit=1000&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, _, _, _ = decodePaginated(t, resp)
	if len(data) > 100 {
		t.Errorf("limit=1000 should be capped at reasonable max (e.g. 100), got %d results", len(data))
	}

	resp = httpClient.GET(t, "/api/v1/business-units?limit=10&offset=9999")
	common.AssertStatusCode(t, resp, 200)
	data, _, _, _ = decodePaginated(t, resp)
	if len(data) != 0 {
		t.Errorf("offset beyond total records should return empty array, got %d results", len(data))
	}
}

func testPostValidRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Valid Business", "+972512221")
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)

	created := decodeBusinessUnit(t, resp)
	if created.ID == "" {
		t.Error("expected ID to be set")
	}
	if created.Name != "Valid Business" {
		t.Errorf("expected name 'Valid Business', got %s", created.Name)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostInvalidRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Invalid Phone", "not-a-phone")
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected status 422 or 400 for invalid phone, got %d", resp.StatusCode)
	}

	bu2 := createValidBusinessUnit("Invalid Timezone", "+972512222")
	bu2["time_zone"] = "Invalid/Timezone"
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected status 422 or 400 for invalid timezone, got %d", resp.StatusCode)
	}

	bu3 := createValidBusinessUnit("A", "+972512223")
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected status 422 or 400 for short name, got %d", resp.StatusCode)
	}

	bu4 := createValidBusinessUnit("No Cities", "+972512224")
	bu4["cities"] = []string{}
	resp = httpClient.POST(t, "/api/v1/business-units", bu4)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected status 422 or 400 for empty cities, got %d", resp.StatusCode)
	}

	bu5 := createValidBusinessUnit("No Labels", "+972512225")
	bu5["labels"] = []string{}
	resp = httpClient.POST(t, "/api/v1/business-units", bu5)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected status 422 or 400 for empty labels, got %d", resp.StatusCode)
	}
}

func testPostWithExtraJsonKeys(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	payload := map[string]any{
		"name":        "Extra Fields Test",
		"cities":      []string{"Tel Aviv"},
		"labels":      []string{"Haircut"},
		"admin_phone": "+972512226",
		"priority":    1,
		"extra_field": "should be ignored",
		"another_key": 12345,
		"random_data": map[string]any{"nested": "value"},
	}

	resp := httpClient.POST(t, "/api/v1/business-units", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostWithMissingRelevantKeys(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	payload := map[string]any{
		"cities":      []string{"Tel Aviv"},
		"labels":      []string{"Haircut"},
		"admin_phone": "+972512227",
	}
	resp := httpClient.POST(t, "/api/v1/business-units", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing name, got %d", resp.StatusCode)
	}

	payload2 := map[string]any{
		"name":        "Test",
		"labels":      []string{"Haircut"},
		"admin_phone": "+972512228",
	}
	resp = httpClient.POST(t, "/api/v1/business-units", payload2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing cities, got %d", resp.StatusCode)
	}

	payload3 := map[string]any{
		"name":        "Test",
		"cities":      []string{"Tel Aviv"},
		"admin_phone": "+972512229",
	}
	resp = httpClient.POST(t, "/api/v1/business-units", payload3)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing labels, got %d", resp.StatusCode)
	}

	payload4 := map[string]any{
		"name":   "Test",
		"cities": []string{"Tel Aviv"},
		"labels": []string{"Haircut"},
	}
	resp = httpClient.POST(t, "/api/v1/business-units", payload4)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing admin_phone, got %d", resp.StatusCode)
	}
}

func testPostWithWebsiteURL(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Website Test", "+972523336")
	bu["website_url"] = "https://example.com"
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if created.WebsiteURL != "https://example.com" {
		t.Errorf("expected website_url 'https://example.com', got %s", created.WebsiteURL)
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	bu2 := createValidBusinessUnit("Invalid URL Test", "+972523337")
	bu2["website_url"] = "http://example.com"
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for non-https URL, got %d", resp.StatusCode)
	}

	bu3 := createValidBusinessUnit("Malformed URL Test", "+972523338")
	bu3["website_url"] = "not-a-url"
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for malformed URL, got %d", resp.StatusCode)
	}
}

func testPostWithMaintainers(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Maintainers Test", "+972523339")
	bu["maintainers"] = []string{"+972541111111", "+972542222222"}
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if len(created.Maintainers) != 2 {
		t.Errorf("expected 2 maintainers, got %d", len(created.Maintainers))
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	bu2 := createValidBusinessUnit("Invalid Maintainer Test", "+972523340")
	bu2["maintainers"] = []string{"not-a-phone", "+972541111111"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp, 201)
	created = decodeBusinessUnit(t, resp)

	if len(created.Maintainers) != 1 {
		t.Errorf("expected 1 valid maintainer (invalid one filtered), got %d", len(created.Maintainers))
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostWithArrayMaxLengths(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Max Cities Test", "+972523341")
	cities := make([]string, 51)
	for i := 0; i < 51; i++ {
		cities[i] = fmt.Sprintf("City%d", i)
	}
	bu["cities"] = cities
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for 51 cities (max 50), got %d", resp.StatusCode)
	}

	bu2 := createValidBusinessUnit("Max Labels Test", "+972523342")
	labels := make([]string, 11)
	for i := 0; i < 11; i++ {
		labels[i] = fmt.Sprintf("Label%d", i)
	}
	bu2["labels"] = labels
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for 11 labels (max 10), got %d", resp.StatusCode)
	}

	bu3 := createValidBusinessUnit("Exactly Max Test", "+972523343")
	cities50 := make([]string, 50)
	for i := 0; i < 50; i++ {
		cities50[i] = fmt.Sprintf("City%d", i)
	}
	bu3["cities"] = cities50
	labels10 := make([]string, 10)
	for i := 0; i < 10; i++ {
		labels10[i] = fmt.Sprintf("Label%d", i)
	}
	bu3["labels"] = labels10
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostWithNameBoundaries(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("AB", "+972523344")
	bu["name"] = "AB"
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	bu2 := createValidBusinessUnit("X", "+972523345")
	bu2["name"] = "X"
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for 1-char name (min 2), got %d", resp.StatusCode)
	}

	name100 := string(make([]byte, 100))
	for i := 0; i < 100; i++ {
		name100 = name100[:i] + "a"
	}
	bu3 := createValidBusinessUnit(name100, "+972523346")
	bu3["name"] = name100
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	common.AssertStatusCode(t, resp, 201)
	created = decodeBusinessUnit(t, resp)
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	name101 := name100 + "a"
	bu4 := createValidBusinessUnit(name101, "+972523347")
	bu4["name"] = name101
	resp = httpClient.POST(t, "/api/v1/business-units", bu4)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for 101-char name (max 100), got %d", resp.StatusCode)
	}
}

func testPostWithPriorityValues(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Priority Test", "+972523348")
	bu["priority"] = 0
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if created.Priority != config.DefaultDefaultBusinessPriority {
		t.Errorf("expected priority %d, got %d", config.DefaultDefaultBusinessPriority, created.Priority)
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	bu2 := createValidBusinessUnit("Negative Priority Test", "+972523349")
	bu2["priority"] = -1
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp, 201)
	created = decodeBusinessUnit(t, resp)

	if created.Priority < 0 {
		t.Errorf("expected normalization error for negative priority, got %d", created.Priority)
	}

	bu3 := createValidBusinessUnit("High Priority Test", "+972523350")
	bu3["priority"] = 9999
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	common.AssertStatusCode(t, resp, 201)
	created = decodeBusinessUnit(t, resp)

	if created.Priority > config.DefaultMaxBusinessPriority {
		t.Errorf("expected priority %d, got %d", config.DefaultMaxBusinessPriority, created.Priority)
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateNonExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	updates := map[string]any{
		"name": "Updated Name",
	}
	resp := httpClient.PATCH(t, "/api/v1/business-units/id/507f1f77bcf86cd799439011", updates)
	common.AssertStatusCode(t, resp, 404)
}

func testUpdateWithInvalidId(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	updates := map[string]any{
		"name": "Updated Name",
	}
	resp := httpClient.PATCH(t, "/api/v1/business-units/id/invalid-id-format", updates)
	common.AssertStatusCode(t, resp, 400)
}

func testUpdateDeletedRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Update Deleted Test", "+972523331")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	deleteResp := httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, deleteResp, 204)

	updates := map[string]any{
		"name": "Should Not Update",
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 404)
}

func testUpdateWithBadFormatKeys(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Update Bad Format Test", "+972523332")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	updates := map[string]any{
		"admin_phone": "not-a-phone",
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("Note: invalid phone in update returned %d", resp.StatusCode)
	}

	updates = map[string]any{
		"time_zone": "Invalid/Zone",
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("Note: invalid timezone in update returned %d", resp.StatusCode)
	}

	updates = map[string]any{
		"name": "A",
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("Note: short name in update returned %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateWithGoodFormatKeys(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Original Name", "+972523335")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	updates := map[string]any{
		"name": "Updated Name",
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeBusinessUnit(t, getResp)

	if fetched.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", fetched.Name)
	}

	updates = map[string]any{
		"admin_phone": "+972546789012",
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched = decodeBusinessUnit(t, getResp)

	if fetched.AdminPhone != "+972546789012" {
		t.Errorf("expected admin_phone '+972546789012', got %s", fetched.AdminPhone)
	}

	updates = map[string]any{
		"time_zone": "America/New_York",
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched = decodeBusinessUnit(t, getResp)

	if fetched.TimeZone != "America/New_York" {
		t.Errorf("expected time_zone 'America/New_York', got %s", fetched.TimeZone)
	}

	updates = map[string]any{
		"cities": []string{"Haifa", "Eilat"},
		"labels": []string{"Massage", "Spa"},
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched = decodeBusinessUnit(t, getResp)

	if len(fetched.Cities) != 2 || fetched.Cities[0] != "haifa" || fetched.Cities[1] != "eilat" {
		t.Errorf("expected cities [haifa, eilat], got %v", fetched.Cities)
	}
	if len(fetched.Labels) != 2 || fetched.Labels[0] != "massage" || fetched.Labels[1] != "spa" {
		t.Errorf("expected labels [massage, spa], got %v", fetched.Labels)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateWithEmptyJson(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Update Empty Test", "+972523333")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	updates := map[string]any{}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeBusinessUnit(t, getResp)

	if fetched.Name != created.Name {
		t.Errorf("expected name %s, got %s", created.Name, fetched.Name)
	}
	if fetched.AdminPhone != created.AdminPhone {
		t.Errorf("expected admin_phone %s, got %s", created.AdminPhone, fetched.AdminPhone)
	}
	if len(fetched.Cities) != len(created.Cities) {
		t.Errorf("expected %d cities, got %d", len(created.Cities), len(fetched.Cities))
	}
	if len(fetched.Labels) != len(created.Labels) {
		t.Errorf("expected %d labels, got %d", len(created.Labels), len(fetched.Labels))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateWebsiteURL(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("URL Update Test", "+972523351")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	updates := map[string]any{
		"website_url": "https://newexample.com",
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeBusinessUnit(t, getResp)

	if fetched.WebsiteURL != "https://newexample.com" {
		t.Errorf("expected website_url 'https://newexample.com', got %s", fetched.WebsiteURL)
	}

	updates = map[string]any{
		"website_url": "http://invalid.com",
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for non-https URL, got %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateMaintainers(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Maintainers Update Test", "+972523352")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	updates := map[string]any{
		"maintainers": []string{"+972543333333", "+972544444444"},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeBusinessUnit(t, getResp)

	if len(fetched.Maintainers) != 2 {
		t.Errorf("expected 2 maintainers, got %d", len(fetched.Maintainers))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testDeleteNonExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.DELETE(t, "/api/v1/business-units/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
}

func testDeleteWithInvalidId(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.DELETE(t, "/api/v1/business-units/id/invalid-id-format")
	common.AssertStatusCode(t, resp, 400)
}

func testDeletedRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Delete Twice Test", "+972523334")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	resp := httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, resp, 204)

	resp = httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, resp, 404)
}

func testGetSearchNormalization(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Normalization Test", "+972523361")
	bu["cities"] = []string{"Tel Aviv", "JERUSALEM"}
	bu["labels"] = []string{"Haircut", "MASSAGE"}
	httpClient.POST(t, "/api/v1/business-units", bu)

	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=telaviv&labels=haircut")
	common.AssertStatusCode(t, resp, 200)
	data := decodeBusinessUnits(t, resp)
	if len(data) < 1 {
		t.Error("search should find business unit with normalized lowercase city/label")
	}

	resp = httpClient.GET(t, "/api/v1/business-units/search?cities=TELAVIV&labels=HAIRCUT")
	common.AssertStatusCode(t, resp, 200)
	data = decodeBusinessUnits(t, resp)
	if len(data) < 1 {
		t.Error("search should find business unit with normalized uppercase city/label")
	}

	resp = httpClient.GET(t, "/api/v1/business-units/search?cities=jerusalem&labels=massage")
	common.AssertStatusCode(t, resp, 200)
	data = decodeBusinessUnits(t, resp)
	if len(data) < 1 {
		t.Error("search should find business unit with mixed case normalization")
	}
}

func testGetSearchMultipleCitiesLabels(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu1 := createValidBusinessUnit("Multi Search 1", "+972523362")
	bu1["cities"] = []string{"Tel Aviv", "Haifa"}
	bu1["labels"] = []string{"Haircut", "Massage"}
	httpClient.POST(t, "/api/v1/business-units", bu1)

	bu2 := createValidBusinessUnit("Multi Search 2", "+972523363")
	bu2["cities"] = []string{"Jerusalem", "Eilat"}
	bu2["labels"] = []string{"Spa", "Styling"}
	httpClient.POST(t, "/api/v1/business-units", bu2)

	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=telaviv,haifa&labels=haircut,massage")
	common.AssertStatusCode(t, resp, 200)
	data := decodeBusinessUnits(t, resp)
	if len(data) < 1 {
		t.Error("search should support multiple cities and labels")
	}

	resp = httpClient.GET(t, "/api/v1/business-units/search?cities=telaviv,jerusalem&labels=haircut,spa")
	common.AssertStatusCode(t, resp, 200)
	data = decodeBusinessUnits(t, resp)
	if len(data) < 2 {
		t.Error("search should return results matching any city and any label")
	}
}

func testPostMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.POSTRaw(t, "/api/v1/business-units", []byte(`{"name": "test", "invalid json`))
	common.AssertStatusCode(t, resp, 400)

	resp = httpClient.POSTRaw(t, "/api/v1/business-units", []byte(`not json at all`))
	common.AssertStatusCode(t, resp, 400)
}

func testPostWithUSPhoneNumber(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("US Phone Test", "+12125551234")
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if created.AdminPhone != "+12125551234" {
		t.Errorf("expected admin_phone '+12125551234', got %s", created.AdminPhone)
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostWithSpecialCharacters(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Café & Spa™", "+972523364")
	bu["name"] = "Café & Spa™ 🎨"
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if created.Name != "Café & Spa™ 🎨" {
		t.Errorf("expected name with special chars, got %s", created.Name)
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	bu2 := createValidBusinessUnit("Hebrew Test", "+972523365")
	bu2["cities"] = []string{"תל אביב", "ירושלים"}
	bu2["labels"] = []string{"תספורת", "עיצוב"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp, 201)
	created = decodeBusinessUnit(t, resp)
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostDuplicateDetection(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	adminPhone := "+972523370"

	bu1 := createValidBusinessUnit("My Salon", adminPhone)
	bu1["cities"] = []string{"Tel Aviv"}
	bu1["labels"] = []string{"Haircut"}
	resp := httpClient.POST(t, "/api/v1/business-units", bu1)
	common.AssertStatusCode(t, resp, 201)
	created1 := decodeBusinessUnit(t, resp)

	bu2 := createValidBusinessUnit("Different Salon", adminPhone)
	bu2["cities"] = []string{"Tel Aviv"}
	bu2["labels"] = []string{"Haircut"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp, 201)
	created2 := decodeBusinessUnit(t, resp)

	bu3 := createValidBusinessUnit("My Salon", adminPhone)
	bu3["cities"] = []string{"Haifa"}
	bu3["labels"] = []string{"Haircut"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	common.AssertStatusCode(t, resp, 201)
	created3 := decodeBusinessUnit(t, resp)

	bu4 := createValidBusinessUnit("My Salon", adminPhone)
	bu4["cities"] = []string{"Tel Aviv"}
	bu4["labels"] = []string{"Massage"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu4)
	common.AssertStatusCode(t, resp, 201)
	created4 := decodeBusinessUnit(t, resp)

	bu5 := createValidBusinessUnit("My Salon", adminPhone)
	bu5["cities"] = []string{"Tel Aviv"}
	bu5["labels"] = []string{"Haircut"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu5)
	if resp.StatusCode != 409 && resp.StatusCode != 400 {
		t.Errorf("expected conflict for exact duplicate, got %d", resp.StatusCode)
	}

	bu6 := createValidBusinessUnit("My Salon", adminPhone)
	bu6["cities"] = []string{"Tel Aviv", "Haifa"}
	bu6["labels"] = []string{"Haircut"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu6)
	if resp.StatusCode != 409 && resp.StatusCode != 400 {
		t.Errorf("expected conflict for cities overlap (subset), got %d", resp.StatusCode)
	}

	bu7 := createValidBusinessUnit("My Salon", adminPhone)
	bu7["cities"] = []string{"Tel Aviv"}
	bu7["labels"] = []string{"Haircut", "Massage"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu7)
	if resp.StatusCode != 409 && resp.StatusCode != 400 {
		t.Errorf("expected conflict for labels overlap (subset), got %d", resp.StatusCode)
	}

	bu8 := createValidBusinessUnit("my salon", adminPhone)
	bu8["cities"] = []string{"telaviv"}
	bu8["labels"] = []string{"haircut"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu8)
	if resp.StatusCode != 409 && resp.StatusCode != 400 {
		t.Errorf("expected conflict for case-insensitive duplicate, got %d", resp.StatusCode)
	}

	bu9 := createValidBusinessUnit("My Salon", adminPhone)
	bu9["cities"] = []string{"Eilat", "Netanya"}
	bu9["labels"] = []string{"Spa", "Styling"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu9)
	common.AssertStatusCode(t, resp, 201)
	created9 := decodeBusinessUnit(t, resp)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created1.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created2.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created3.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created4.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created9.ID))
}

func testUpdateArraysToMaxLength(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Array Max Test", "+972523366")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	cities51 := make([]string, 51)
	for i := 0; i < 51; i++ {
		cities51[i] = fmt.Sprintf("City%d", i)
	}
	updates := map[string]any{
		"cities": cities51,
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for 51 cities, got %d", resp.StatusCode)
	}

	labels11 := make([]string, 11)
	for i := 0; i < 11; i++ {
		labels11[i] = fmt.Sprintf("Label%d", i)
	}
	updates = map[string]any{
		"labels": labels11,
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for 11 labels, got %d", resp.StatusCode)
	}

	updates = map[string]any{
		"cities": []string{},
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for empty cities, got %d", resp.StatusCode)
	}

	updates = map[string]any{
		"labels": []string{},
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for empty labels, got %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdatePriorityEdgeCases(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Priority Update Test", "+972523367")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	updates := map[string]any{
		"priority": 0,
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	fetched := decodeBusinessUnit(t, getResp)
	if fetched.Priority != 0 {
		t.Errorf("expected priority 0, got %d", fetched.Priority)
	}

	updates = map[string]any{
		"priority": -5,
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)
	getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	fetched = decodeBusinessUnit(t, getResp)
	if fetched.Priority < 0 {
		t.Errorf("expected priority >= 0 after normalization, got %d", fetched.Priority)
	}

	updates = map[string]any{
		"priority": 10000,
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)
	getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	fetched = decodeBusinessUnit(t, getResp)
	if fetched.Priority > config.DefaultMaxBusinessPriority {
		t.Errorf("expected priority <= %d, got %d", config.DefaultMaxBusinessPriority, fetched.Priority)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateClearOptionalFields(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Clear Fields Test", "+972523368")
	bu["website_url"] = "https://example.com"
	bu["maintainers"] = []string{"+972541111111"}
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	if created.WebsiteURL == "" {
		t.Error("expected website_url to be set")
	}
	if len(created.Maintainers) == 0 {
		t.Error("expected maintainers to be set")
	}

	updates := map[string]any{
		"website_url": "",
		"maintainers": []string{},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	fetched := decodeBusinessUnit(t, getResp)

	if fetched.WebsiteURL != "" {
		t.Errorf("Note: website_url was '%s', expected empty after clearing with null", fetched.WebsiteURL)
	}
	if len(fetched.Maintainers) != 0 {
		t.Errorf("Note: maintainers has %d items, expected 0 after clearing with null", len(fetched.Maintainers))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Malformed Update Test", "+972523369")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	resp := httpClient.PATCHRaw(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), []byte(`{"name": "test", invalid`))
	common.AssertStatusCode(t, resp, 400)

	resp = httpClient.PATCHRaw(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), []byte(`not json`))
	common.AssertStatusCode(t, resp, 400)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}
