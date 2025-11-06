package businessunits

import (
	"fmt"
	"os"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/test/common"
	"testing"
)

const ServiceName = "business-units-integration-tests"

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

func clearTestData(t *testing.T) {
	t.Helper()
	resp := httpClient.GET(t, "/api/v1/business-units?limit=1000&offset=0")
	if resp.StatusCode != 200 {
		return
	}

	var result struct {
		Data []model.BusinessUnit `json:"data"`
	}
	if err := resp.DecodeJSON(&result); err != nil {
		return
	}

	for _, bu := range result.Data {
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", bu.ID))
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
	clearTestData(t)
}

func testPost(t *testing.T) {
	testPostValidRecord(t)
	testPostInvalidRecord(t)
	testPostWithExtraJsonKeys(t)
	testPostWithMissingRelevantKeys(t)
	clearTestData(t)
}

func testUpdate(t *testing.T) {
	testUpdateNonExistingRecord(t)
	testUpdateWithInvalidId(t)
	testUpdateDeletedRecord(t)
	testUpdateWithBadFormatKeys(t)
	testUpdateWithEmptyJson(t)
	clearTestData(t)
}

func testDelete(t *testing.T) {
	testDeleteNonExistingRecord(t)
	testDeleteWithInvalidId(t)
	testDeletedRecord(t)
	clearTestData(t)
}

func testGetByIdEmptyTable(t *testing.T) {
	clearTestData(t)
	resp := httpClient.GET(t, "/api/v1/business-units/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
	common.AssertContains(t, resp, "not found")
}

func testGetBySearchEmptyTable(t *testing.T) {
	clearTestData(t)
	resp := httpClient.GET(t, "/api/v1/business-units/search?cities=Tel%20Aviv&labels=Haircut")
	common.AssertStatusCode(t, resp, 200)

	data := decodeBusinessUnits(t, resp)
	if len(data) != 0 {
		t.Errorf("expected empty results, got %d", len(data))
	}
}

func testGetAllPaginatedEmptyTable(t *testing.T) {
	clearTestData(t)
	resp := httpClient.GET(t, "/api/v1/business-units?limit=10&offset=0")
	common.AssertStatusCode(t, resp, 200)

	data, totalCount, _, _ := decodePaginated(t, resp)
	if totalCount != 0 || len(data) != 0 {
		t.Errorf("expected empty results, got total=%d, data=%d", totalCount, len(data))
	}
}

func testGetValidIdExistingRecord(t *testing.T) {
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
	resp := httpClient.GET(t, "/api/v1/business-units?limit=abc&offset=xyz")
	common.AssertStatusCode(t, resp, 200)

	resp = httpClient.GET(t, "/api/v1/business-units?limit=10&offset=-1")
	common.AssertStatusCode(t, resp, 200)
}

func testPostValidRecord(t *testing.T) {
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
}

func testPostWithExtraJsonKeys(t *testing.T) {
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

func testUpdateNonExistingRecord(t *testing.T) {
	updates := map[string]any{
		"name": "Updated Name",
	}
	resp := httpClient.PATCH(t, "/api/v1/business-units/id/507f1f77bcf86cd799439011", updates)
	common.AssertStatusCode(t, resp, 404)
}

func testUpdateWithInvalidId(t *testing.T) {
	updates := map[string]any{
		"name": "Updated Name",
	}
	resp := httpClient.PATCH(t, "/api/v1/business-units/id/invalid-id-format", updates)
	common.AssertStatusCode(t, resp, 400)
}

func testUpdateDeletedRecord(t *testing.T) {
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

func testUpdateWithEmptyJson(t *testing.T) {
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

func testDeleteNonExistingRecord(t *testing.T) {
	resp := httpClient.DELETE(t, "/api/v1/business-units/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
}

func testDeleteWithInvalidId(t *testing.T) {
	resp := httpClient.DELETE(t, "/api/v1/business-units/id/invalid-id-format")
	common.AssertStatusCode(t, resp, 400)
}

func testDeletedRecord(t *testing.T) {
	bu := createValidBusinessUnit("Delete Twice Test", "+972523334")
	createResp := httpClient.POST(t, "/api/v1/business-units", bu)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBusinessUnit(t, createResp)

	resp := httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, resp, 204)

	resp = httpClient.DELETE(t, fmt.Sprintf("/api/v1/business-units/id/%s", created.ID))
	common.AssertStatusCode(t, resp, 404)
}
