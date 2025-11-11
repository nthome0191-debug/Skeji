package integrationtests

import (
	"fmt"
	"math/rand"
	"os"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/test/common"
	"testing"
	"time"
)

const (
	ServiceName = "schedules-integration-tests"
	TableName   = "schedules"
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

func createValidSchedule(name string) map[string]any {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	suffix := r.Intn(100000)

	return map[string]any{
		"business_id":  "507f1f77bcf86cd799439011",
		"name":         fmt.Sprintf("%s-%d", name, suffix),
		"city":         "Tel Aviv",
		"address":      fmt.Sprintf("Derech Menachem Begin 121 #%d", suffix),
		"working_days": []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday"},
		"start_of_day": "09:00",
		"end_of_day":   "18:00",
		"time_zone":    "Asia/Jerusalem",
	}
}

func decodeSchedule(t *testing.T, resp *common.Response) *model.Schedule {
	t.Helper()
	var result struct {
		Data model.Schedule `json:"data"`
	}
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return &result.Data
}

func decodeSchedules(t *testing.T, resp *common.Response) []model.Schedule {
	t.Helper()
	var result struct {
		Data []model.Schedule `json:"data"`
	}
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return result.Data
}

func decodePaginated(t *testing.T, resp *common.Response) ([]model.Schedule, int, int, int) {
	t.Helper()
	var result struct {
		Data       []model.Schedule `json:"data"`
		TotalCount int              `json:"total_count"`
		Limit      int              `json:"limit"`
		Offset     int              `json:"offset"`
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
	testGetPaginationEdgeCases(t)
}

func testPost(t *testing.T) {
	testPostValidRecord(t)
	testPostInvalidRecord(t)
	testPostMalformedJSON(t)
	testPostWithSpecialCharacters(t)
	testPostWithTimeBoundaries(t)
}

func testUpdate(t *testing.T) {
	testUpdateNonExistingRecord(t)
	testUpdateWithInvalidId(t)
	testUpdateDeletedRecord(t)
	testUpdateWithBadFormatKeys(t)
	testUpdateWithGoodFormatKeys(t)
	testUpdateWithEmptyJson(t)
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
	resp := httpClient.GET(t, "/api/v1/schedules/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
	common.AssertContains(t, resp, "not found")
}

func testGetBySearchEmptyTable(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/schedules/search?business_id=507f1f77bcf86cd799439011&city=Tel%20Aviv")
	common.AssertStatusCode(t, resp, 200)
	data := decodeSchedules(t, resp)
	if len(data) != 0 {
		t.Errorf("expected empty results, got %d", len(data))
	}
}

func testGetAllPaginatedEmptyTable(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/schedules?limit=10&offset=0")
	common.AssertStatusCode(t, resp, 200)

	data, totalCount, _, _ := decodePaginated(t, resp)
	if totalCount != 0 || len(data) != 0 {
		t.Errorf("expected empty results, got total=%d, data=%d", totalCount, len(data))
	}
}

func testGetValidIdExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Acro Tower Branch")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	resp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	common.AssertStatusCode(t, resp, 200)
	fetched := decodeSchedule(t, resp)

	if fetched.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, fetched.ID)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testGetInvalidIdExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Azrieli Branch")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	resp := httpClient.GET(t, "/api/v1/schedules/id/invalid-id-format")
	common.AssertStatusCode(t, resp, 400)

	resp = httpClient.GET(t, "/api/v1/schedules/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testGetValidSearchExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	adminBusiness := "507f1f77bcf86cd799439011"

	req1 := createValidSchedule("Acro Tower Branch")
	req1["business_id"] = adminBusiness
	req1["city"] = "Tel Aviv"
	httpClient.POST(t, "/api/v1/schedules", req1)

	req2 := createValidSchedule("Azrieli Branch")
	req2["business_id"] = adminBusiness
	req2["city"] = "Jerusalem"
	httpClient.POST(t, "/api/v1/schedules", req2)

	resp := httpClient.GET(t, "/api/v1/schedules/search?business_id=507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 200)
	all := decodeSchedules(t, resp)
	if len(all) < 2 {
		t.Errorf("expected at least 2 results for business_id search, got %d", len(all))
	}

	resp = httpClient.GET(t, "/api/v1/schedules/search?business_id=507f1f77bcf86cd799439011&city=Tel%20Aviv")
	common.AssertStatusCode(t, resp, 200)
	filtered := decodeSchedules(t, resp)
	if len(filtered) < 1 {
		t.Error("expected city filter to return results")
	}
}

func testGetInvalidSearchExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/schedules/search?city=Tel%20Aviv")
	common.AssertStatusCode(t, resp, 400)
	common.AssertContains(t, resp, "business_id")

	resp = httpClient.GET(t, "/api/v1/schedules/search?business_id=")
	common.AssertStatusCode(t, resp, 400)
}

func testGetValidPaginationExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	for i := 1; i <= 5; i++ {
		req := createValidSchedule(fmt.Sprintf("Branch %d", i))
		resp := httpClient.POST(t, "/api/v1/schedules", req)
		common.AssertStatusCode(t, resp, 201)
	}

	resp := httpClient.GET(t, "/api/v1/schedules?limit=2&offset=0")
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

	resp = httpClient.GET(t, "/api/v1/schedules?limit=2&offset=2")
	common.AssertStatusCode(t, resp, 200)
}

func testGetInvalidPaginationExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/schedules?limit=abc&offset=xyz")
	common.AssertStatusCode(t, resp, 400)

	resp = httpClient.GET(t, "/api/v1/schedules?limit=10&offset=-1")
	common.AssertStatusCode(t, resp, 200)
}

func testGetVerifyCreatedAt(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Ramat Aviv Clinic")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)
	if created.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}

	originalCreatedAt := created.CreatedAt
	update := map[string]any{"name": "Ramat Aviv Clinic - Updated"}
	httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), update)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeSchedule(t, getResp)

	if !fetched.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("created_at should not change on update: original=%v, after_update=%v", originalCreatedAt, fetched.CreatedAt)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testGetPaginationEdgeCases(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	for i := 0; i < 3; i++ {
		req := createValidSchedule(fmt.Sprintf("Edge Branch %d", i))
		httpClient.POST(t, "/api/v1/schedules", req)
	}

	resp := httpClient.GET(t, "/api/v1/schedules?limit=0&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, _, _, _ := decodePaginated(t, resp)
	if len(data) > 10 {
		t.Errorf("limit=0 should return max 10 results, got %d results", len(data))
	}

	resp = httpClient.GET(t, "/api/v1/schedules?limit=1000&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, _, _, _ = decodePaginated(t, resp)
	if len(data) > 100 {
		t.Errorf("limit=1000 should be capped at reasonable max (e.g. 100), got %d results", len(data))
	}

	resp = httpClient.GET(t, "/api/v1/schedules?limit=10&offset=9999")
	common.AssertStatusCode(t, resp, 200)
	data, _, _, _ = decodePaginated(t, resp)
	if len(data) != 0 {
		t.Errorf("offset beyond total records should return empty array, got %d results", len(data))
	}
}

func testPostValidRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Acro Tower Branch")
	resp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if created.ID == "" {
		t.Error("expected ID to be set")
	}
	if created.Name != req["name"] {
		t.Errorf("expected name 'Acro Tower Branch', got %s", created.Name)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testPostInvalidRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req1 := createValidSchedule("Missing Biz")
	delete(req1, "business_id")
	resp := httpClient.POST(t, "/api/v1/schedules", req1)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing business_id, got %d", resp.StatusCode)
	}

	req2 := createValidSchedule("Missing City")
	delete(req2, "city")
	resp = httpClient.POST(t, "/api/v1/schedules", req2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing city, got %d", resp.StatusCode)
	}

	req3 := createValidSchedule("Missing Address")
	delete(req3, "address")
	resp = httpClient.POST(t, "/api/v1/schedules", req3)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing address, got %d", resp.StatusCode)
	}

	req4 := createValidSchedule("Empty Working Days")
	req4["working_days"] = []string{}
	resp = httpClient.POST(t, "/api/v1/schedules", req4)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for empty working_days, got %d", resp.StatusCode)
	}

	req5 := createValidSchedule("Bad Time Format")
	req5["start_of_day"] = "25:61"
	resp = httpClient.POST(t, "/api/v1/schedules", req5)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for bad start_of_day, got %d", resp.StatusCode)
	}

	req6 := createValidSchedule("Bad TZ")
	req6["time_zone"] = "Invalid/Timezone"
	resp = httpClient.POST(t, "/api/v1/schedules", req6)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for invalid timezone, got %d", resp.StatusCode)
	}
}

func testPostMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.POSTRaw(t, "/api/v1/schedules", []byte(`{"name": "x", "invalid`))
	common.AssertStatusCode(t, resp, 400)

	resp = httpClient.POSTRaw(t, "/api/v1/schedules", []byte(`not json at all`))
	common.AssertStatusCode(t, resp, 400)
}

func testPostWithSpecialCharacters(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Café - Acro Branch™ 🎨")
	req["address"] = "רח' יפה 10, תל אביב"
	resp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if created.Name != req["name"] {
		t.Errorf("expected special char name, got %s", created.Name)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testPostWithTimeBoundaries(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Late Hours Branch")
	req["start_of_day"] = "00:00"
	req["end_of_day"] = "23:59"
	resp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateNonExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	updates := map[string]any{"name": "Updated Name"}
	resp := httpClient.PATCH(t, "/api/v1/schedules/id/507f1f77bcf86cd799439011", updates)
	common.AssertStatusCode(t, resp, 404)
}

func testUpdateWithInvalidId(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	updates := map[string]any{"name": "Updated Name"}
	resp := httpClient.PATCH(t, "/api/v1/schedules/id/invalid-id-format", updates)
	common.AssertStatusCode(t, resp, 400)
}

func testUpdateDeletedRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("To Be Deleted Branch")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	deleteResp := httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	common.AssertStatusCode(t, deleteResp, 204)

	updates := map[string]any{"name": "Should Not Update"}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 404)
}

func testUpdateWithBadFormatKeys(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Bad Format Branch")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	updates := map[string]any{"time_zone": "Invalid/Zone"}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("invalid timezone in update returned %d", resp.StatusCode)
	}

	updates = map[string]any{"start_of_day": "99:99"}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), updates)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("invalid start_of_day in update returned %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateWithGoodFormatKeys(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Acro Tower Branch")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	updates := map[string]any{"name": "Acro Tower Branch - Floor 12"}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeSchedule(t, getResp)
	if fetched.Name != "Acro Tower Branch - Floor 12" {
		t.Errorf("expected updated name, got %s", fetched.Name)
	}

	updates = map[string]any{"city": "Jerusalem"}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	fetched = decodeSchedule(t, getResp)
	if fetched.City != "Jerusalem" {
		t.Errorf("expected city 'Jerusalem', got %s", fetched.City)
	}

	updates = map[string]any{
		"working_days": []string{"Sunday", "Monday", "Tuesday"},
		"start_of_day": "10:00",
		"end_of_day":   "19:00",
		"time_zone":    "Asia/Jerusalem",
	}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp = httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	fetched = decodeSchedule(t, getResp)
	if fetched.StartOfDay != "10:00" || fetched.EndOfDay != "19:00" {
		t.Errorf("expected hours 10:00-19:00, got %s-%s", fetched.StartOfDay, fetched.EndOfDay)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateWithEmptyJson(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Empty JSON Branch")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	updates := map[string]any{}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), updates)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeSchedule(t, getResp)

	if fetched.Name != created.Name {
		t.Errorf("expected name %s, got %s", created.Name, fetched.Name)
	}
	if fetched.City != created.City {
		t.Errorf("expected city %s, got %s", created.City, fetched.City)
	}
	if fetched.Address != created.Address {
		t.Errorf("expected address %s, got %s", created.Address, fetched.Address)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Malformed Update Branch")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	resp := httpClient.PATCHRaw(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), []byte(`{"name": "x", invalid`))
	common.AssertStatusCode(t, resp, 400)

	resp = httpClient.PATCHRaw(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), []byte(`not json`))
	common.AssertStatusCode(t, resp, 400)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testDeleteNonExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.DELETE(t, "/api/v1/schedules/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
}

func testDeleteWithInvalidId(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.DELETE(t, "/api/v1/schedules/id/invalid-id-format")
	common.AssertStatusCode(t, resp, 400)
}

func testDeletedRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Delete Twice Branch")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	resp := httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	common.AssertStatusCode(t, resp, 204)

	resp = httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	common.AssertStatusCode(t, resp, 404)
}
