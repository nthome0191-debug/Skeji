package integrationtests

import (
	"fmt"
	"os"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/pkg/sanitizer"
	"skeji/test/common"
	"strings"
	"sync"
	"testing"
	"time"
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
	testAdvanced(t)
	teardown()
}

func testAdvanced(t *testing.T) {
	testPhoneNumberEdgeCases(t)
	testConcurrentCreation(t)
	testConcurrentUpdates(t)
	testSearchPartialMatches(t)
	testMaintainersEdgeCases(t)
	testInternationalPhoneNumbers(t)
	testCityLabelNormalizationEdgeCases(t)
	testMaxLimitPagination(t)
	testPriorityRangeValidation(t)
	testTimezoneBoundaries(t)
	testLargeScaleBusinessUnits(t)
	testSearchWithManyFilters(t)
	testSearchPerformance(t)
	testComplexPriorityOrdering(t)
	testURLDeduplicationComplex(t)
	testMaintainersMaxLimit(t)
	testUpdateWithPartialOverlap(t)
	testSearchCaseSensitivity(t)
	testBatchDeletion(t)
	testConcurrentSearches(t)
	testUpdateAllFieldsSimultaneously(t)
	testCitiesLabelsIntersection(t)
	testGetByPhone(t)
	testEmptySearchResults(t)
	testSearchWithInvalidPriority(t)
	testUpdateToExistingData(t)
	testCreateWithMinimalFields(t)
	testUpdatePriorityImpactOnSearch(t)
	testMultipleCitiesSingleLabel(t)
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

func decodePaginated(t *testing.T, resp *common.Response) (bu_model []model.BusinessUnit, count int, limit int, offset int) {
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

func letterSequence(n int) string {
	s := ""
	for n >= 0 {
		s = string(rune('A'+(n%26))) + s
		n = n/26 - 1
	}
	return s
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
	testPostWithMultipleSocialMediaURLs(t)
	testPostWithURLNormalization(t)
	testPostWithDuplicateURLs(t)
	testPostWithMixedValidInvalidURLs(t)
	testPostWithMaintainers(t)
	testPostWithArrayMaxLengths(t)
	testPostWithNameBoundaries(t)
	testPostWithPriorityValues(t)
	testPostMalformedJSON(t)
	testPostWithUSPhoneNumber(t)
	testPostWithSpecialCharacters(t)
	testPostDuplicateDetection(t)
	testPostAdminPhoneValidation(t)
}

func testUpdate(t *testing.T) {
	testUpdateNonExistingRecord(t)
	testUpdateWithInvalidId(t)
	testUpdateDeletedRecord(t)
	testUpdateWithBadFormatKeys(t)
	testUpdateWithGoodFormatKeys(t)
	testUpdateWithEmptyJson(t)
	testUpdateWebsiteURL(t)
	testUpdateAddURLs(t)
	testUpdateRemoveAllURLs(t)
	testUpdateReplaceURLs(t)
	testUpdateMaintainers(t)
	testUpdateArraysToMaxLength(t)
	testUpdatePriorityEdgeCases(t)
	testUpdateClearOptionalFields(t)
	testUpdateMalformedJSON(t)
	testUpdateAdminPhoneValidation(t)
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
	common.AssertStatusCode(t, resp, 400)

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
	resp0 := httpClient.POST(t, "/api/v1/business-units", bu1)
	common.AssertStatusCode(t, resp0, 201)

	bu2 := createValidBusinessUnit("Priority 5", "+972523355")
	bu2["priority"] = 5
	resp1 := httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp1, 201)

	bu3 := createValidBusinessUnit("Priority 3", "+972523356")
	bu3["priority"] = 3
	resp2 := httpClient.POST(t, "/api/v1/business-units", bu3)
	common.AssertStatusCode(t, resp2, 201)

	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=tel_aviv&labels=haircut")
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
	sanitized := sanitizer.SanitizeNameOrAddress(fmt.Sprint(bu["name"]))
	if created.Name != sanitized {
		t.Errorf("expected name %s, got %s", sanitized, created.Name)
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
	bu["website_urls"] = []string{"https://example.com", "https://facebook.com/page"}
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if len(created.WebsiteURLs) != 2 {
		t.Errorf("expected 2 website_urls, got %d", len(created.WebsiteURLs))
	}
	if created.WebsiteURLs[0] != "https://example.com" {
		t.Errorf("expected first URL 'https://example.com', got %s", created.WebsiteURLs[0])
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	bu2 := createValidBusinessUnit("Too Many URLs Test", "+972523337")
	bu2["website_urls"] = []string{"https://example1.com", "https://example2.com", "https://example3.com", "https://example4.com", "https://example5.com", "https://example6.com"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for too many URLs, got %d", resp.StatusCode)
	}

	// Test with invalid URL
	bu3 := createValidBusinessUnit("Invalid URL Test", "+972523338")
	bu3["website_urls"] = []string{"http://example.com"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for non-https URL, got %d", resp.StatusCode)
	}

	bu4 := createValidBusinessUnit("Malformed URL Test", "+972523339")
	bu4["website_urls"] = []string{"not-a-url"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu4)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for malformed URL, got %d", resp.StatusCode)
	}
}

func testPostWithMaintainers(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Maintainers Test", "+97252333944")
	bu["maintainers"] = map[string]string{"+97254111133": "lala", "+97254222233": "lele"}
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if len(created.Maintainers) != 2 {
		t.Errorf("expected 2 maintainers, got %d", len(created.Maintainers))
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	bu2 := createValidBusinessUnit("Invalid Maintainer Test", "+972523340")
	bu2["maintainers"] = map[string]string{"not-a-phone": "lala", "+97254222233": "lele"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp, 422)
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostWithArrayMaxLengths(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Max Cities Test", "+97252533341")
	cities := make([]string, 51)
	for i := 0; i < 51; i++ {
		cities[i] = fmt.Sprintf("City%v", letterSequence(i))
	}
	bu["cities"] = cities
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for 51 cities (max 50), got %d", resp.StatusCode)
	}

	bu2 := createValidBusinessUnit("Max Labels Test", "+97252533341")
	labels := make([]string, 11)
	for i := 0; i < 11; i++ {
		labels[i] = fmt.Sprintf("Label%v", letterSequence(i))
	}
	bu2["labels"] = labels
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for 11 labels (max 10), got %d", resp.StatusCode)
	}

	bu3 := createValidBusinessUnit("Exactly Max Test", "+97252533341")
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

	bu2 := createValidBusinessUnit("X", "+972525333415")
	bu2["name"] = "X"
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for 1-char name (min 2), got %d", resp.StatusCode)
	}

	name100 := string(make([]byte, 100))
	for i := 0; i < 100; i++ {
		name100 = name100[:i] + "a"
	}
	bu3 := createValidBusinessUnit(name100, "+972525333415")
	bu3["name"] = name100
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	common.AssertStatusCode(t, resp, 201)
	created = decodeBusinessUnit(t, resp)
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))

	name101 := name100 + "a"
	bu4 := createValidBusinessUnit(name101, "+972525333415")
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
	bu := createValidBusinessUnit("Update Bad Format Test", "+97252323332")
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

	if fetched.Name != sanitizer.SanitizeNameOrAddress(fmt.Sprint(updates["name"])) {
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

	// getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	// common.AssertStatusCode(t, getResp, 200)
	// fetched = decodeBusinessUnit(t, getResp)

	// if fetched.TimeZone != "America/New_York" {
	// 	t.Errorf("expected time_zone 'America/New_York', got %s", fetched.TimeZone)
	// }

	// updates = map[string]any{
	// 	"cities": []string{"Haifa", "Eilat"},
	// 	"labels": []string{"Massage", "Spa"},
	// }
	// resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	// common.AssertStatusCode(t, resp, 204)

	// getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	// common.AssertStatusCode(t, getResp, 200)
	// fetched = decodeBusinessUnit(t, getResp)

	// if len(fetched.Cities) != 2 || fetched.Cities[0] != "haifa" || fetched.Cities[1] != "eilat" {
	// 	t.Errorf("expected cities [haifa, eilat], got %v", fetched.Cities)
	// }
	// if len(fetched.Labels) != 2 || fetched.Labels[0] != "massage" || fetched.Labels[1] != "spa" {
	// 	t.Errorf("expected labels [massage, spa], got %v", fetched.Labels)
	// }

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
		"website_urls": []string{"https://newexample.com", "https://instagram.com/profile"},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeBusinessUnit(t, getResp)

	if len(fetched.WebsiteURLs) != 2 {
		t.Errorf("expected 2 website_urls, got %d", len(fetched.WebsiteURLs))
	}
	if fetched.WebsiteURLs[0] != "https://newexample.com" {
		t.Errorf("expected first URL 'https://newexample.com', got %s", fetched.WebsiteURLs[0])
	}

	updates = map[string]any{
		"website_urls": []string{"http://invalid.com"},
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
		"maintainers": map[string]string{"+972543333333": "baba", "+972544444444": "yababa"},
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

	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=tel_aviv&labels=haircut")
	common.AssertStatusCode(t, resp, 200)
	data := decodeBusinessUnits(t, resp)
	if len(data) < 1 {
		t.Error("search should find business unit with normalized lowercase city/label")
	}

	resp = httpClient.GET(t, "/api/v1/business-units/search?cities=tel_aviv&labels=HAIRCUT")
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

	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=tel_aviv,haifa&labels=haircut,massage")
	common.AssertStatusCode(t, resp, 200)
	data := decodeBusinessUnits(t, resp)
	if len(data) < 1 {
		t.Error("search should support multiple cities and labels")
	}

	resp = httpClient.GET(t, "/api/v1/business-units/search?cities=tel_aviv,jerusalem&labels=haircut,spa")
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

	if created.Name != sanitizer.SanitizeNameOrAddress(fmt.Sprint(bu["name"])) {
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
	bu8["cities"] = []string{"tel_aviv"}
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
		cities51[i] = fmt.Sprintf("City%v", letterSequence(i))
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
		labels11[i] = fmt.Sprintf("Label%v", letterSequence(i))
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
	if resp.StatusCode != 204 {
		t.Errorf("expected 204 for negative prioriyty, got %d", resp.StatusCode)
	}
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
	bu["website_urls"] = []string{"https://example.com"}
	bu["maintainers"] = map[string]string{"+972541111111": "shalom"}
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	if len(created.WebsiteURLs) == 0 {
		t.Error("expected website_urls to be set")
	}
	if len(created.Maintainers) == 0 {
		t.Error("expected maintainers to be set")
	}

	updates := map[string]any{
		"website_urls": []string{},
		"maintainers":  map[string]string{},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	fetched := decodeBusinessUnit(t, getResp)

	if len(fetched.WebsiteURLs) != 0 {
		t.Errorf("Note: website_urls has %d items, expected 0 after clearing", len(fetched.WebsiteURLs))
	}
	if len(fetched.Maintainers) != 0 {
		t.Errorf("Note: maintainers has %d items, expected 0 after clearing with null", len(fetched.Maintainers))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostWithMultipleSocialMediaURLs(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Social Media Test", "+972523369")
	bu["website_urls"] = []string{
		"https://example.com",
		"https://facebook.com/businesspage",
		"https://instagram.com/businessprofile",
		"https://twitter.com/businesshandle",
		"https://linkedin.com/company/business",
	}
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if len(created.WebsiteURLs) != 5 {
		t.Errorf("expected 5 URLs, got %d", len(created.WebsiteURLs))
	}

	expectedURLs := map[string]bool{
		"https://example.com":                   true,
		"https://facebook.com/businesspage":     true,
		"https://instagram.com/businessprofile": true,
		"https://twitter.com/businesshandle":    true,
		"https://linkedin.com/company/business": true,
	}

	for _, url := range created.WebsiteURLs {
		if !expectedURLs[url] {
			t.Errorf("unexpected URL: %s", url)
		}
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostWithURLNormalization(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("URL Normalization Test", "+972523370")
	bu["website_urls"] = []string{
		"https://Example.COM/path",
		"https://www.example.com",
		"https://example.com/path?utm_source=test&param=val",
		"https://example.com/path/",
	}
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if len(created.WebsiteURLs) < 1 {
		t.Errorf("expected at least 1 normalized URL, got %d", len(created.WebsiteURLs))
	}

	for _, url := range created.WebsiteURLs {
		if strings.Contains(url, "www.") {
			t.Errorf("URL should not contain 'www.': %s", url)
		}
		if strings.Contains(url, "utm_") {
			t.Errorf("URL should not contain UTM parameters: %s", url)
		}
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostWithDuplicateURLs(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Duplicate URLs Test", "+972523371")
	bu["website_urls"] = []string{
		"https://example.com",
		"https://example.com",
		"https://Example.com",
		"https://www.example.com",
		"https://facebook.com/page",
	}
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if len(created.WebsiteURLs) > 3 {
		t.Logf("Note: Expected deduplication to reduce URLs from 5 to ~2, got %d URLs", len(created.WebsiteURLs))
		t.Logf("URLs: %v", created.WebsiteURLs)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateAddURLs(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Add URLs Test", "+972523372")
	bu["website_urls"] = []string{"https://example.com"}
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	if len(created.WebsiteURLs) != 1 {
		t.Errorf("expected 1 initial URL, got %d", len(created.WebsiteURLs))
	}

	updates := map[string]any{
		"website_urls": []string{
			"https://example.com",
			"https://facebook.com/page",
			"https://instagram.com/profile",
		},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	fetched := decodeBusinessUnit(t, getResp)

	if len(fetched.WebsiteURLs) != 3 {
		t.Errorf("expected 3 URLs after update, got %d", len(fetched.WebsiteURLs))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateRemoveAllURLs(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Remove URLs Test", "+972523373")
	bu["website_urls"] = []string{
		"https://example.com",
		"https://facebook.com/page",
	}
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	if len(created.WebsiteURLs) != 2 {
		t.Errorf("expected 2 initial URLs, got %d", len(created.WebsiteURLs))
	}

	updates := map[string]any{
		"website_urls": []string{},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	fetched := decodeBusinessUnit(t, getResp)

	if len(fetched.WebsiteURLs) != 0 {
		t.Errorf("expected 0 URLs after clearing, got %d", len(fetched.WebsiteURLs))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdateReplaceURLs(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Replace URLs Test", "+972523374")
	bu["website_urls"] = []string{
		"https://oldexample.com",
		"https://facebook.com/oldpage",
	}
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	updates := map[string]any{
		"website_urls": []string{
			"https://newexample.com",
			"https://instagram.com/newprofile",
		},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	fetched := decodeBusinessUnit(t, getResp)

	if len(fetched.WebsiteURLs) != 2 {
		t.Errorf("expected 2 URLs after replacement, got %d", len(fetched.WebsiteURLs))
	}

	urlMap := make(map[string]bool)
	for _, url := range fetched.WebsiteURLs {
		urlMap[url] = true
	}

	if urlMap["https://oldexample.com"] {
		t.Error("old URL should have been replaced")
	}
	if urlMap["https://facebook.com/oldpage"] {
		t.Error("old URL should have been replaced")
	}
	if !urlMap["https://newexample.com"] {
		t.Error("new URL should be present")
	}
	if !urlMap["https://instagram.com/newprofile"] {
		t.Error("new URL should be present")
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPostWithMixedValidInvalidURLs(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	bu := createValidBusinessUnit("Mixed URLs Test", "+972523375")
	bu["website_urls"] = []string{
		"https://example.com",
		"http://invalid.com",
		"https://valid.com",
		"not-a-url",
	}
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for mixed valid/invalid URLs, got %d", resp.StatusCode)
	}
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

func testPostAdminPhoneValidation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	payload := map[string]any{
		"name":   "Business Without Phone",
		"cities": []string{"Tel Aviv"},
		"labels": []string{"Haircut"},
	}
	resp := httpClient.POST(t, "/api/v1/business-units", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error (400 or 422) for missing admin_phone, got %d", resp.StatusCode)
	}
	common.AssertContains(t, resp, "AdminPhone")

	payload2 := map[string]any{
		"name":        "Business With Empty Phone",
		"cities":      []string{"Tel Aviv"},
		"labels":      []string{"Haircut"},
		"admin_phone": "",
	}
	resp = httpClient.POST(t, "/api/v1/business-units", payload2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error (400 or 422) for empty admin_phone, got %d", resp.StatusCode)
	}

	payload3 := map[string]any{
		"name":        "Business With Whitespace Phone",
		"cities":      []string{"Tel Aviv"},
		"labels":      []string{"Haircut"},
		"admin_phone": "   ",
	}
	resp = httpClient.POST(t, "/api/v1/business-units", payload3)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error (400 or 422) for whitespace-only admin_phone, got %d", resp.StatusCode)
	}
}

func testUpdateAdminPhoneValidation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("Admin Phone Update Test", "+972523371")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	updates3 := map[string]any{
		"admin_phone": "invalid-phone",
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates3)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error (400 or 422) when updating admin_phone to invalid format, got %d", resp.StatusCode)
	}

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeBusinessUnit(t, getResp)

	if fetched.AdminPhone != "+972523371" {
		t.Errorf("expected admin_phone to remain unchanged at '+972523371', got %s", fetched.AdminPhone)
	}

	updates4 := map[string]any{
		"admin_phone": "+972501234567",
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), updates4)
	common.AssertStatusCode(t, resp, 204)

	getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched = decodeBusinessUnit(t, getResp)

	if fetched.AdminPhone != "+972501234567" {
		t.Errorf("expected admin_phone to be updated to '+972501234567', got %s", fetched.AdminPhone)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testPhoneNumberEdgeCases(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("Phone With Spaces", "+972 50 1234567")
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 422)
	created := decodeBusinessUnit(t, resp)
	if strings.Contains(created.AdminPhone, " ") {
		t.Logf("Admin phone wasn't sanitized/normalized: %s", created.AdminPhone)
	}

	bu2 := createValidBusinessUnit("Phone With Dashes", "+972-50-1234567")
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp, 422)
	created = decodeBusinessUnit(t, resp)
	if strings.Contains(created.AdminPhone, "-") {
		t.Logf("Admin phone wasn't sanitized/normalized: %s", created.AdminPhone)
	}

	bu3 := createValidBusinessUnit("Min Phone", "+9728")
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	if resp.StatusCode == 201 {
		created := decodeBusinessUnit(t, resp)
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	}

	bu4 := createValidBusinessUnit("Max Phone", "+123456789012345")
	resp = httpClient.POST(t, "/api/v1/business-units", bu4)
	if resp.StatusCode == 201 {
		created := decodeBusinessUnit(t, resp)
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	}
}

func testConcurrentCreation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	var wg sync.WaitGroup
	results := make([]int, 5)
	ids := make([]string, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			bu := createValidBusinessUnit("Concurrent Business", fmt.Sprintf("+97250%07d", 2000000+index))
			bu["cities"] = []string{"Tel Aviv"}
			bu["labels"] = []string{"Test"}
			resp := httpClient.POST(t, "/api/v1/business-units", bu)
			results[index] = resp.StatusCode
			if resp.StatusCode == 201 {
				created := decodeBusinessUnit(t, resp)
				ids[index] = created.ID
			}
		}(i)
	}

	wg.Wait()

	successCount := 0
	for _, status := range results {
		if status == 201 {
			successCount++
		}
	}

	if successCount != 5 {
		t.Logf("Concurrent creation: %d/5 succeeded (expected all to succeed with different phones)", successCount)
	}

	// Cleanup
	for _, id := range ids {
		if id != "" {
			httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", id))
		}
	}
}

func testConcurrentUpdates(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("Concurrent Update Test", "+972502000000")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	var wg sync.WaitGroup
	results := make([]int, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			update := map[string]any{
				"name": fmt.Sprintf("Updated Name %d", index),
			}
			resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)
			results[index] = resp.StatusCode
		}(i)
	}

	wg.Wait()

	successCount := 0
	for _, status := range results {
		if status == 204 {
			successCount++
		}
	}

	if successCount != 10 {
		t.Logf("Concurrent updates: %d/10 succeeded", successCount)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testSearchPartialMatches(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu1 := createValidBusinessUnit("Tel Aviv Salon", "+972502000001")
	bu1["cities"] = []string{"Tel Aviv", "Tel Aviv-Yafo"}
	bu1["labels"] = []string{"Haircut", "Hairstyling"}
	httpClient.POST(t, "/api/v1/business-units", bu1)

	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=tel_aviv&labels=haircut")
	common.AssertStatusCode(t, resp, 200)
	results := decodeBusinessUnits(t, resp)
	if len(results) < 1 {
		t.Error("Expected to find business unit with city match")
	}

	resp = httpClient.GET(t, "/api/v1/business-units/search?cities=tel_aviv,jerusalem&labels=haircut,massage")
	common.AssertStatusCode(t, resp, 200)
	results = decodeBusinessUnits(t, resp)
	if len(results) < 1 {
		t.Error("Expected to find business unit with partial match")
	}
}

func testMaintainersEdgeCases(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu2 := createValidBusinessUnit("Duplicate Maintainers", "+972502000011")
	bu2["maintainers"] = map[string]string{"+972541111111": "sh", "+972542222222": "mma"}
	resp := httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp, 201)
	created2 := decodeBusinessUnit(t, resp)
	if len(created2.Maintainers) > 2 {
		t.Logf("Expected deduplication of maintainers, got %d maintainers", len(created2.Maintainers))
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created2.ID))

	bu3 := createValidBusinessUnit("Admin As Maintainer", "+972502000012")
	bu3["maintainers"] = map[string]string{"+972502000012": "ya", "+972541111111": "da"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	common.AssertStatusCode(t, resp, 201)
	created3 := decodeBusinessUnit(t, resp)
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created3.ID))
}

func testInternationalPhoneNumbers(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	testCases := []struct {
		name       string
		phone      string
		shouldPass bool
	}{
		{"US Number", "+12125551234", true},
		{"Canada Number", "+14165551234", true},
		{"Israel Number", "+972501234567", true},
		{"UK Number", "+447700900123", false},
		{"France Number", "+33612345678", false},
		{"Invalid Prefix", "+999123456789", false},
	}

	for _, tc := range testCases {
		bu := createValidBusinessUnit(tc.name, tc.phone)
		resp := httpClient.POST(t, "/api/v1/business-units", bu)

		if tc.shouldPass {
			if resp.StatusCode == 201 {
				created := decodeBusinessUnit(t, resp)
				httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
			} else {
				t.Errorf("%s: expected 201, got %d", tc.name, resp.StatusCode)
			}
		} else {
			if resp.StatusCode != 422 && resp.StatusCode != 400 {
				t.Logf("%s: expected validation error, got %d", tc.name, resp.StatusCode)
			}
		}
	}
}

func testCityLabelNormalizationEdgeCases(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("Special Chars", "+972502000030")
	bu["cities"] = []string{"Tel-Aviv", "Tel Aviv", "TEL_AVIV"}
	bu["labels"] = []string{"Hair-Cut", "Hair Cut", "HAIR_CUT"}
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	if len(created.Cities) != 1 || created.Cities[0] != "tel_aviv" {
		t.Errorf("Cities are not normalized: %v", created.Cities)
	}
	if len(created.Labels) != 1 || created.Labels[0] != "hair_cut" {
		t.Errorf("Labels are not normalized: %v", created.Labels)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testMaxLimitPagination(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	for i := range 5 {
		bu := createValidBusinessUnit(fmt.Sprintf("Pagination Test %d", i), fmt.Sprintf("+97250%07d", 3000000+i))
		httpClient.POST(t, "/api/v1/business-units", bu)
	}

	resp := httpClient.GET(t, "/api/v1/business-units?limit=10000&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, count, limit, _ := decodePaginated(t, resp)

	if limit > config.DefaultPaginationLimit {
		t.Errorf("Expected limit to be capped at 100, got %d", limit)
	}

	if len(data) != count {
		t.Errorf("Requested count=%d, data count=%d", count, len(data))
	}
}

func testPriorityRangeValidation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	testPriorities := []int64{0, 1, 50, 100, 1000, 10000}
	for _, priority := range testPriorities {
		bu := createValidBusinessUnit(fmt.Sprintf("Priority %d", priority), fmt.Sprintf("+97250%07d", 5000000+int(priority)))
		bu["priority"] = priority
		resp := httpClient.POST(t, "/api/v1/business-units", bu)
		common.AssertStatusCode(t, resp, 201)
		created := decodeBusinessUnit(t, resp)
		if (priority <= 0 && created.Priority != config.DefaultDefaultBusinessPriority) ||
			(priority > config.DefaultMaxBusinessPriority && created.Priority != config.DefaultMaxBusinessPriority) ||
			(priority > 0 && priority < config.DefaultMaxBusinessPriority && created.Priority != priority) {
			t.Errorf("Requested priority=%d, got priority=%d", priority, created.Priority)
		}
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	}
}

func testTimezoneBoundaries(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	timezones := []string{
		// "UTC",
		// "GMT",
		"Asia/Jerusalem",
		"America/New_York",
		// "Europe/London",
		// "Pacific/Auckland",
		// "Asia/Tokyo",
	}

	for i, tz := range timezones {
		bu := createValidBusinessUnit(fmt.Sprintf("TZ Test %s", tz), fmt.Sprintf("+97250%07d", 6000000+i))
		bu["time_zone"] = tz
		resp := httpClient.POST(t, "/api/v1/business-units", bu)
		common.AssertStatusCode(t, resp, 201)
		created := decodeBusinessUnit(t, resp)

		if created.TimeZone != tz {
			t.Errorf("Expected timezone %s, got %s", tz, created.TimeZone)
		}

		httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	}
}

// ========== ENRICHED TESTS ==========

func testLargeScaleBusinessUnits(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create 100 business units
	createdIDs := []string{}
	for i := 0; i < 100; i++ {
		bu := createValidBusinessUnit(fmt.Sprintf("Large Scale BU %d", i), fmt.Sprintf("+97250%07d", 8000000+i))
		bu["cities"] = []string{"Tel Aviv"}
		bu["labels"] = []string{"Service"}
		bu["priority"] = i % 10

		resp := httpClient.POST(t, "/api/v1/business-units", bu)
		if resp.StatusCode == 201 {
			created := decodeBusinessUnit(t, resp)
			createdIDs = append(createdIDs, created.ID)
		}
	}

	// Verify pagination works with large dataset
	resp := httpClient.GET(t, "/api/v1/business-units?limit=50&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, total, _, _ := decodePaginated(t, resp)

	if total < 100 {
		t.Errorf("expected at least 100 business units, got %d", total)
	}
	if len(data) != 50 {
		t.Errorf("expected 50 items per page, got %d", len(data))
	}

	// Cleanup
	for _, id := range createdIDs {
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", id))
	}
}

func testSearchWithManyFilters(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create business units with various combinations
	bu1 := createValidBusinessUnit("Multi Filter Test 1", "+97250800001")
	bu1["cities"] = []string{"Tel Aviv", "Haifa", "Jerusalem"}
	bu1["labels"] = []string{"Haircut", "Massage", "Spa"}
	bu1["priority"] = 5
	httpClient.POST(t, "/api/v1/business-units", bu1)

	bu2 := createValidBusinessUnit("Multi Filter Test 2", "+97250800002")
	bu2["cities"] = []string{"Beer Sheva", "Eilat"}
	bu2["labels"] = []string{"Styling", "Nails"}
	bu2["priority"] = 8
	httpClient.POST(t, "/api/v1/business-units", bu2)

	// Search with multiple cities and labels
	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=tel_aviv,haifa,beer_sheva&labels=haircut,styling")
	common.AssertStatusCode(t, resp, 200)
	results := decodeBusinessUnits(t, resp)

	if len(results) < 2 {
		t.Errorf("expected at least 2 results for multi-filter search, got %d", len(results))
	}
}

func testSearchPerformance(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create 50 business units for performance testing
	for i := 0; i < 50; i++ {
		bu := createValidBusinessUnit(fmt.Sprintf("Perf Test %d", i), fmt.Sprintf("+97250%07d", 8100000+i))
		bu["cities"] = []string{fmt.Sprintf("City%d", i%10)}
		bu["labels"] = []string{fmt.Sprintf("Label%d", i%5)}
		httpClient.POST(t, "/api/v1/business-units", bu)
	}

	// Measure search time
	searchURL := "/api/v1/business-units/search?cities=city0,city1,city2&labels=label0,label1"

	start := time.Now()
	resp := httpClient.GET(t, searchURL)
	duration := time.Since(start)

	common.AssertStatusCode(t, resp, 200)

	if duration > 2*time.Second {
		t.Logf("Search took %v (warning: might be slow)", duration)
	}
}

func testComplexPriorityOrdering(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	priorities := []int{10, 1, 5, 8, 3, 7, 2, 9, 4, 6}
	for i, priority := range priorities {
		bu := createValidBusinessUnit(fmt.Sprintf("Priority %d", priority), fmt.Sprintf("+97250%07d", 8200000+i))
		bu["priority"] = priority
		bu["cities"] = []string{"TestCity"}
		bu["labels"] = []string{"TestLabel"}
		httpClient.POST(t, "/api/v1/business-units", bu)
	}

	// Search and verify ordering
	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=testcity&labels=testlabel")
	common.AssertStatusCode(t, resp, 200)
	results := decodeBusinessUnits(t, resp)

	if len(results) < 10 {
		t.Errorf("expected 10 results, got %d", len(results))
	}

	// Verify descending priority order
	for i := 1; i < len(results); i++ {
		if results[i-1].Priority < results[i].Priority {
			t.Errorf("results not ordered by priority descending: %d < %d at positions %d and %d",
				results[i-1].Priority, results[i].Priority, i-1, i)
		}
	}
}

func testURLDeduplicationComplex(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("URL Dedup Complex", "+972508300001")
	bu["website_urls"] = []string{
		"https://example.com",
		"https://Example.com",
		"https://EXAMPLE.COM",
		"https://www.example.com",
		"https://example.com/",
		"https://example.com/?utm_source=test",
	}

	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	// Should deduplicate all variations to single URL
	if len(created.WebsiteURLs) > 2 {
		t.Logf("URL deduplication: expected <= 2 URLs, got %d: %v", len(created.WebsiteURLs), created.WebsiteURLs)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testMaintainersMaxLimit(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("Max Maintainers", "+972508400001")

	// Try to add maximum number of maintainers
	maintainers := make(map[string]string)
	for i := 0; i < 100; i++ {
		maintainers[fmt.Sprintf("+97250%07d", 8400000+i)] = fmt.Sprintf("Maintainer%d", i)
	}
	bu["maintainers"] = maintainers

	resp := httpClient.POST(t, "/api/v1/business-units", bu)

	// Should either accept all or cap at maximum
	if resp.StatusCode == 201 {
		created := decodeBusinessUnit(t, resp)
		t.Logf("Created with %d maintainers", len(created.Maintainers))
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	} else if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("unexpected status for max maintainers: %d", resp.StatusCode)
	}
}

func testUpdateWithPartialOverlap(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	adminPhone := "+972508500001"

	// Create business unit with cities A, B, C and labels X, Y, Z
	bu1 := createValidBusinessUnit("Overlap Test 1", adminPhone)
	bu1["cities"] = []string{"CityA", "CityB", "CityC"}
	bu1["labels"] = []string{"LabelX", "LabelY", "LabelZ"}
	resp1 := httpClient.POST(t, "/api/v1/business-units", bu1)
	common.AssertStatusCode(t, resp1, 201)
	created1 := decodeBusinessUnit(t, resp1)

	// Try to update to partially overlap: cities B, C, D and labels Y, Z, W
	update := map[string]any{
		"cities": []string{"CityB", "CityC", "CityD"},
		"labels": []string{"LabelY", "LabelZ", "LabelW"},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created1.ID), update)

	// Should accept partial overlap for same business
	if resp.StatusCode != 204 && resp.StatusCode != 409 {
		t.Logf("Partial overlap update returned status %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created1.ID))
}

func testSearchCaseSensitivity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("Case Test", "+972508600001")
	bu["cities"] = []string{"Tel Aviv"}
	bu["labels"] = []string{"Haircut"}
	httpClient.POST(t, "/api/v1/business-units", bu)

	// Search with different case variations
	testCases := []string{
		"/api/v1/business-units/search?cities=tel_aviv&labels=haircut",
		"/api/v1/business-units/search?cities=TEL_AVIV&labels=HAIRCUT",
		"/api/v1/business-units/search?cities=Tel_Aviv&labels=Haircut",
	}

	for _, searchURL := range testCases {
		resp := httpClient.GET(t, searchURL)
		common.AssertStatusCode(t, resp, 200)
		results := decodeBusinessUnits(t, resp)

		if len(results) < 1 {
			t.Errorf("case variation %s returned no results", searchURL)
		}
	}
}

func testBatchDeletion(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create 20 business units
	createdIDs := []string{}
	for i := 0; i < 20; i++ {
		bu := createValidBusinessUnit(fmt.Sprintf("Batch Delete %d", i), fmt.Sprintf("+97250%07d", 8700000+i))
		resp := httpClient.POST(t, "/api/v1/business-units", bu)
		common.AssertStatusCode(t, resp, 201)
		created := decodeBusinessUnit(t, resp)
		createdIDs = append(createdIDs, created.ID)
	}

	// Delete all at once
	for _, id := range createdIDs {
		resp := httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", id))
		common.AssertStatusCode(t, resp, 204)
	}

	// Verify all deleted
	for _, id := range createdIDs {
		resp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", id))
		common.AssertStatusCode(t, resp, 404)
	}
}

func testConcurrentSearches(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create some business units
	for i := 0; i < 10; i++ {
		bu := createValidBusinessUnit(fmt.Sprintf("Concurrent Search %d", i), fmt.Sprintf("+97250%07d", 8800000+i))
		bu["cities"] = []string{"TestCity"}
		bu["labels"] = []string{"TestLabel"}
		httpClient.POST(t, "/api/v1/business-units", bu)
	}

	// Run concurrent searches
	var wg sync.WaitGroup
	results := make([]int, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			resp := httpClient.GET(t, "/api/v1/business-units/search?cities=testcity&labels=testlabel")
			results[index] = resp.StatusCode
		}(i)
	}

	wg.Wait()

	for i, status := range results {
		if status != 200 {
			t.Errorf("concurrent search %d returned status %d", i, status)
		}
	}
}

func testUpdateAllFieldsSimultaneously(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("Update All Fields", "+972508900001")
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	// Update every possible field
	update := map[string]any{
		"name":         "Completely New Name",
		"cities":       []string{"New City1", "New City2"},
		"labels":       []string{"New Label1", "New Label2"},
		"admin_phone":  "+972509000000",
		"priority":     9,
		"time_zone":    "America/New_York",
		"website_urls": []string{"https://newsite.com"},
		"maintainers":  map[string]string{"+972509000001": "NewManager"},
	}

	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	// Verify all fields updated
	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	updated := decodeBusinessUnit(t, getResp)

	if updated.AdminPhone != "+972509000000" {
		t.Errorf("admin_phone not updated")
	}
	if updated.Priority != 9 {
		t.Errorf("priority not updated")
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testCitiesLabelsIntersection(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create BU with cities [A, B] and labels [X, Y]
	bu1 := createValidBusinessUnit("Intersection Test 1", "+972509100001")
	bu1["cities"] = []string{"CityA", "CityB"}
	bu1["labels"] = []string{"LabelX", "LabelY"}
	httpClient.POST(t, "/api/v1/business-units", bu1)

	// Create BU with cities [B, C] and labels [Y, Z]
	bu2 := createValidBusinessUnit("Intersection Test 2", "+972509100002")
	bu2["cities"] = []string{"CityB", "CityC"}
	bu2["labels"] = []string{"LabelY", "LabelZ"}
	httpClient.POST(t, "/api/v1/business-units", bu2)

	// Search for intersection (city B, label Y) should find both
	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=cityb&labels=labely")
	common.AssertStatusCode(t, resp, 200)
	results := decodeBusinessUnits(t, resp)

	if len(results) < 2 {
		t.Errorf("expected at least 2 results for intersection, got %d", len(results))
	}
}

func testGetByPhone(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Test 1: Get by admin phone
	adminPhone := "+972509300001"
	bu1 := createValidBusinessUnit("Admin Phone Test", adminPhone)
	resp := httpClient.POST(t, "/api/v1/business-units", bu1)
	common.AssertStatusCode(t, resp, 201)
	created1 := decodeBusinessUnit(t, resp)

	searchURL := fmt.Sprintf("/api/v1/business-units/phone/%s", adminPhone)
	searchResp := httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, searchResp, 200)
	results := decodeBusinessUnits(t, searchResp)

	if len(results) != 1 {
		t.Errorf("expected 1 business unit for admin phone, got %d", len(results))
	}
	if len(results) > 0 && results[0].ID != created1.ID {
		t.Errorf("expected business unit ID %s, got %s", created1.ID, results[0].ID)
	}

	// Test 2: Get by maintainer phone
	maintainerPhone := "+972509300002"
	bu2 := createValidBusinessUnit("Maintainer Phone Test", "+972509300000")
	bu2["maintainers"] = map[string]string{maintainerPhone: "TestMaintainer"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp, 201)
	created2 := decodeBusinessUnit(t, resp)

	searchURL = fmt.Sprintf("/api/v1/business-units/phone/%s", maintainerPhone)
	searchResp = httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, searchResp, 200)
	results = decodeBusinessUnits(t, searchResp)

	if len(results) != 1 {
		t.Errorf("expected 1 business unit for maintainer phone, got %d", len(results))
	}
	if len(results) > 0 && results[0].ID != created2.ID {
		t.Errorf("expected business unit ID %s, got %s", created2.ID, results[0].ID)
	}

	// Test 3: Admin phone in maintainers should be sanitized
	adminPhone3 := "+972509300003"
	bu3 := createValidBusinessUnit("Admin in Maintainers Test", adminPhone3)
	bu3["maintainers"] = map[string]string{
		adminPhone3:         "ShouldBeRemoved",
		"+972509300004": "ShouldStay",
	}
	resp = httpClient.POST(t, "/api/v1/business-units", bu3)
	common.AssertStatusCode(t, resp, 201)
	created3 := decodeBusinessUnit(t, resp)

	searchURL = fmt.Sprintf("/api/v1/business-units/phone/%s", adminPhone3)
	searchResp = httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, searchResp, 200)
	results = decodeBusinessUnits(t, searchResp)

	if len(results) != 1 {
		t.Errorf("expected 1 business unit, got %d", len(results))
	}
	if len(results) > 0 {
		// Admin phone should be removed from maintainers
		if _, exists := results[0].Maintainers[adminPhone3]; exists {
			t.Errorf("admin phone should be removed from maintainers, but found it")
		}
		// Other maintainer should still be there
		if _, exists := results[0].Maintainers["+972509300004"]; !exists {
			t.Errorf("other maintainer should still be present")
		}
		if len(results[0].Maintainers) != 1 {
			t.Errorf("expected 1 maintainer after sanitization, got %d", len(results[0].Maintainers))
		}
	}

	// Test 4: Multiple business units with same maintainer
	sharedMaintainer := "+972509300005"
	bu4 := createValidBusinessUnit("Shared Maintainer 1", "+972509300006")
	bu4["maintainers"] = map[string]string{sharedMaintainer: "SharedPerson"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu4)
	common.AssertStatusCode(t, resp, 201)
	created4 := decodeBusinessUnit(t, resp)

	bu5 := createValidBusinessUnit("Shared Maintainer 2", "+972509300007")
	bu5["maintainers"] = map[string]string{sharedMaintainer: "SharedPerson"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu5)
	common.AssertStatusCode(t, resp, 201)
	created5 := decodeBusinessUnit(t, resp)

	searchURL = fmt.Sprintf("/api/v1/business-units/phone/%s", sharedMaintainer)
	searchResp = httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, searchResp, 200)
	results = decodeBusinessUnits(t, searchResp)

	if len(results) != 2 {
		t.Errorf("expected 2 business units for shared maintainer, got %d", len(results))
	}

	foundIDs := make(map[string]bool)
	for _, bu := range results {
		foundIDs[bu.ID] = true
	}
	if !foundIDs[created4.ID] || !foundIDs[created5.ID] {
		t.Errorf("expected to find both business units with shared maintainer")
	}

	// Test 5: Phone that is admin in one unit and maintainer in another
	dualPhone := "+972509300008"
	bu6 := createValidBusinessUnit("Dual Role Admin", dualPhone)
	resp = httpClient.POST(t, "/api/v1/business-units", bu6)
	common.AssertStatusCode(t, resp, 201)
	created6 := decodeBusinessUnit(t, resp)

	bu7 := createValidBusinessUnit("Dual Role Maintainer", "+972509300009")
	bu7["maintainers"] = map[string]string{dualPhone: "DualPerson"}
	resp = httpClient.POST(t, "/api/v1/business-units", bu7)
	common.AssertStatusCode(t, resp, 201)
	created7 := decodeBusinessUnit(t, resp)

	searchURL = fmt.Sprintf("/api/v1/business-units/phone/%s", dualPhone)
	searchResp = httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, searchResp, 200)
	results = decodeBusinessUnits(t, searchResp)

	if len(results) != 2 {
		t.Errorf("expected 2 business units (admin + maintainer roles), got %d", len(results))
	}

	foundIDs = make(map[string]bool)
	for _, bu := range results {
		foundIDs[bu.ID] = true
	}
	if !foundIDs[created6.ID] || !foundIDs[created7.ID] {
		t.Errorf("expected to find both business units where phone is admin and maintainer")
	}

	// Test 6: Non-existent phone should return empty array
	nonExistentPhone := "+972509399999"
	searchURL = fmt.Sprintf("/api/v1/business-units/phone/%s", nonExistentPhone)
	searchResp = httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, searchResp, 200)
	results = decodeBusinessUnits(t, searchResp)

	if len(results) != 0 {
		t.Errorf("expected 0 business units for non-existent phone, got %d", len(results))
	}

	// Cleanup
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created1.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created2.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created3.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created4.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created5.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created6.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created7.ID))
}

func testEmptySearchResults(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Search for non-existent combination
	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=nonexistentcity&labels=nonexistentlabel")
	common.AssertStatusCode(t, resp, 200)
	results := decodeBusinessUnits(t, resp)

	if len(results) != 0 {
		t.Errorf("expected 0 results for non-existent search, got %d", len(results))
	}
}

func testSearchWithInvalidPriority(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Try search with invalid priority filter (if API supports it)
	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=telaviv&labels=haircut&priority=-1")

	// Should either ignore invalid priority or return 400
	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		t.Logf("Search with invalid priority returned status %d", resp.StatusCode)
	}
}

func testUpdateToExistingData(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("Update To Existing", "+972509400001")
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	// Update to exact same data
	update := map[string]any{
		"name":        created.Name,
		"cities":      created.Cities,
		"labels":      created.Labels,
		"admin_phone": created.AdminPhone,
	}

	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testCreateWithMinimalFields(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create with only required fields
	minimal := map[string]any{
		"name":        "Minimal BU",
		"cities":      []string{"Tel Aviv"},
		"labels":      []string{"Service"},
		"admin_phone": "+972509500001",
	}

	resp := httpClient.POST(t, "/api/v1/business-units", minimal)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	// Verify defaults were applied
	if created.Priority != config.DefaultDefaultBusinessPriority {
		t.Errorf("expected default priority %d, got %d", config.DefaultDefaultBusinessPriority, created.Priority)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}

func testUpdatePriorityImpactOnSearch(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create two business units with different priorities
	bu1 := createValidBusinessUnit("Priority Impact 1", "+972509600001")
	bu1["priority"] = 1
	bu1["cities"] = []string{"TestCity"}
	bu1["labels"] = []string{"TestLabel"}
	resp1 := httpClient.POST(t, "/api/v1/business-units", bu1)
	common.AssertStatusCode(t, resp1, 201)
	created1 := decodeBusinessUnit(t, resp1)

	bu2 := createValidBusinessUnit("Priority Impact 2", "+972509600002")
	bu2["priority"] = 5
	bu2["cities"] = []string{"TestCity"}
	bu2["labels"] = []string{"TestLabel"}
	resp2 := httpClient.POST(t, "/api/v1/business-units", bu2)
	common.AssertStatusCode(t, resp2, 201)
	created2 := decodeBusinessUnit(t, resp2)

	// Search - bu2 should come first (higher priority)
	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=testcity&labels=testlabel")
	common.AssertStatusCode(t, resp, 200)
	results := decodeBusinessUnits(t, resp)

	if len(results) >= 2 && results[0].Priority < results[1].Priority {
		t.Error("results not ordered by priority (higher first)")
	}

	// Update bu1 to have higher priority
	update := map[string]any{"priority": 10}
	httpClient.PATCH(t, fmt.Sprintf("/api/v1/business-units/id/%s", created1.ID), update)

	// Search again - bu1 should now come first
	resp = httpClient.GET(t, "/api/v1/business-units/search?cities=testcity&labels=testlabel")
	common.AssertStatusCode(t, resp, 200)
	results = decodeBusinessUnits(t, resp)

	if len(results) >= 2 && results[0].ID != created1.ID {
		t.Logf("Priority update didn't affect search ordering as expected")
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created1.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created2.ID))
}

func testMultipleCitiesSingleLabel(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	bu := createValidBusinessUnit("Multi City Single Label", "+972509700001")
	bu["cities"] = []string{"Tel Aviv", "Haifa", "Jerusalem", "Beer Sheva", "Eilat"}
	bu["labels"] = []string{"Haircut"}
	resp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBusinessUnit(t, resp)

	// Search for any of the cities with the label
	searchTests := []string{
		"/api/v1/business-units/search?cities=tel_aviv&labels=haircut",
		"/api/v1/business-units/search?cities=haifa&labels=haircut",
		"/api/v1/business-units/search?cities=jerusalem&labels=haircut",
	}

	for _, searchURL := range searchTests {
		resp := httpClient.GET(t, searchURL)
		common.AssertStatusCode(t, resp, 200)
		results := decodeBusinessUnits(t, resp)

		if len(results) < 1 {
			t.Errorf("search %s didn't find business unit", searchURL)
		}
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
}
