package integrationtests

import (
	"fmt"
	"math/rand"
	"os"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/test/common"
	"sync"
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
	testAdvanced(t)
	teardown()
}

func testAdvanced(t *testing.T) {
	testDuplicateScheduleDetection(t)
	testConcurrentScheduleCreation(t)
	testWorkingDaysNormalization(t)
	testExceptionsEdgeCases(t)
	testOptionalFieldsDefaults(t)
	testSearchWithOnlyBusinessID(t)
	testSearchWithCityFilter(t)
	testUpdatePartialFields(t)
	testCityNormalization(t)
	testBreakDurationZero(t)
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
	testPostWorkingDaysSingleDay(t)
	testPostWorkingDaysAllWeek(t)
	testPostWorkingDaysInvalidDay(t)
	testPostWorkingDaysDuplicates(t)
	testPostWorkingDaysCaseSensitivity(t)
	testPostTimeEqualStartEnd(t)
	testPostTimeEndBeforeStart(t)
	testPostTimeMinuteBoundaries(t)
	testPostOptionalFieldsBoundaries(t)
	testPostNameAndAddressLengths(t)
	testPostMultipleSchedulesSameBusiness(t)
	testPostExceptionsArray(t)
	testPostAllOptionalFields(t)
}

func testUpdate(t *testing.T) {
	testUpdateNonExistingRecord(t)
	testUpdateWithInvalidId(t)
	testUpdateDeletedRecord(t)
	testUpdateWithBadFormatKeys(t)
	testUpdateWithGoodFormatKeys(t)
	testUpdateWithEmptyJson(t)
	testUpdateMalformedJSON(t)
	testUpdateWorkingDays(t)
	testUpdateTimeZone(t)
	testUpdateAddExceptions(t)
	testUpdateRemoveExceptions(t)
	testUpdateAllFieldsAtOnce(t)
	testUpdateOnlyName(t)
	testUpdateOnlyTimeRange(t)
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
	common.AssertStatusCode(t, resp, 201)

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

func testPostWorkingDaysSingleDay(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Single Day Branch")
	req["working_days"] = []string{"Monday"}

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)
	if len(created.WorkingDays) != 1 {
		t.Errorf("expected 1 working day, got %d", len(created.WorkingDays))
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testPostWorkingDaysAllWeek(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("All Week Branch")
	req["working_days"] = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)
	if len(created.WorkingDays) != 7 {
		t.Errorf("expected 7 working days, got %d", len(created.WorkingDays))
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testPostWorkingDaysInvalidDay(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Invalid Day Branch")
	req["working_days"] = []string{"InvalidDay", "Monday"}

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for invalid day, got %d", resp.StatusCode)
	}
}

func testPostWorkingDaysDuplicates(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Duplicate Days Branch")
	req["working_days"] = []string{"Monday", "Tuesday", "Monday"}

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("unexpected status code for duplicate days: %d", resp.StatusCode)
	}
}

func testPostWorkingDaysCaseSensitivity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Case Sensitivity Branch")
	req["working_days"] = []string{"monday", "TUESDAY", "Wednesday"}

	resp := httpClient.POST(t, "/api/v1/schedules", req)

	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("unexpected status code for case sensitivity: %d", resp.StatusCode)
	}
}

func testPostTimeEqualStartEnd(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Zero Hours Branch")
	req["start_of_day"] = "10:00"
	req["end_of_day"] = "10:00"

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for start=end, got %d", resp.StatusCode)
	}
}

func testPostTimeEndBeforeStart(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Reverse Time Branch")
	req["start_of_day"] = "18:00"
	req["end_of_day"] = "09:00"

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for end<start, got %d", resp.StatusCode)
	}
}

func testPostTimeMinuteBoundaries(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	times := []struct {
		start string
		end   string
	}{
		{"09:00", "17:00"},
		{"09:15", "17:15"},
		{"09:30", "17:30"},
		{"09:45", "17:45"},
	}

	for i, tc := range times {
		req := createValidSchedule(fmt.Sprintf("Minute Boundary %d", i))
		req["start_of_day"] = tc.start
		req["end_of_day"] = tc.end

		resp := httpClient.POST(t, "/api/v1/schedules", req)
		common.AssertStatusCode(t, resp, 201)
		created := decodeSchedule(t, resp)
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	}
}

func testPostOptionalFieldsBoundaries(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req1 := createValidSchedule("Meeting Duration Min")
	req1["default_meeting_duration_min"] = 1
	resp1 := httpClient.POST(t, "/api/v1/schedules", req1)
	common.AssertStatusCode(t, resp1, 422)

	req1 = createValidSchedule("Meeting Duration Min")
	req1["default_meeting_duration_min"] = 5
	resp1 = httpClient.POST(t, "/api/v1/schedules", req1)
	common.AssertStatusCode(t, resp1, 201)

	req2 := createValidSchedule("Meeting Duration Max")
	req2["default_meeting_duration_min"] = 480
	resp2 := httpClient.POST(t, "/api/v1/schedules", req2)
	common.AssertStatusCode(t, resp2, 201)

	req3 := createValidSchedule("Break Duration")
	req3["default_break_duration_min"] = 15
	resp3 := httpClient.POST(t, "/api/v1/schedules", req3)
	common.AssertStatusCode(t, resp3, 201)

	req4 := createValidSchedule("Max Participants Min")
	req4["max_participants_per_slot"] = 1
	resp4 := httpClient.POST(t, "/api/v1/schedules", req4)
	common.AssertStatusCode(t, resp4, 201)

	req5 := createValidSchedule("Max Participants Large")
	req5["max_participants_per_slot"] = 100
	resp5 := httpClient.POST(t, "/api/v1/schedules", req5)
	common.AssertStatusCode(t, resp5, 201)
}

func testPostNameAndAddressLengths(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req1 := createValidSchedule("AB")
	resp1 := httpClient.POST(t, "/api/v1/schedules", req1)
	common.AssertStatusCode(t, resp1, 201)

	req2 := createValidSchedule("A")
	req2["name"] = "A"
	resp2 := httpClient.POST(t, "/api/v1/schedules", req2)
	if resp2.StatusCode != 422 && resp2.StatusCode != 400 {
		t.Errorf("expected validation error for 1-char name, got %d", resp2.StatusCode)
	}

	longName := ""
	for range 90 {
		longName += "A"
	}
	req3 := createValidSchedule(longName)
	resp3 := httpClient.POST(t, "/api/v1/schedules", req3)
	common.AssertStatusCode(t, resp3, 201)

	tooLongName := longName + "AAAAAAAAAAAAAAAAAAAAAAA"
	req4 := createValidSchedule(tooLongName)
	resp4 := httpClient.POST(t, "/api/v1/schedules", req4)
	if resp4.StatusCode != 422 && resp4.StatusCode != 400 {
		t.Errorf("expected validation error for 101-char name, got %d", resp4.StatusCode)
	}

	longAddr := ""
	for range 200 {
		longAddr += "Long Address Street "
	}
	req5 := createValidSchedule("Long Address Branch")
	req5["address"] = longAddr
	resp5 := httpClient.POST(t, "/api/v1/schedules", req5)
	if resp5.StatusCode != 201 && resp5.StatusCode != 422 && resp5.StatusCode != 400 {
		t.Errorf("unexpected status for very long address: %d", resp5.StatusCode)
	}
}

func testPostMultipleSchedulesSameBusiness(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	businessID := "507f1f77bcf86cd799439011"

	req1 := createValidSchedule("Tel Aviv Branch")
	req1["business_id"] = businessID
	req1["city"] = "Tel Aviv"
	resp1 := httpClient.POST(t, "/api/v1/schedules", req1)
	common.AssertStatusCode(t, resp1, 201)

	req2 := createValidSchedule("Jerusalem Branch")
	req2["business_id"] = businessID
	req2["city"] = "Jerusalem"
	resp2 := httpClient.POST(t, "/api/v1/schedules", req2)
	common.AssertStatusCode(t, resp2, 201)

	req3 := createValidSchedule("Tel Aviv North")
	req3["business_id"] = businessID
	req3["city"] = "Tel Aviv"
	resp3 := httpClient.POST(t, "/api/v1/schedules", req3)
	common.AssertStatusCode(t, resp3, 201)

	searchResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/search?business_id=%s", businessID))
	common.AssertStatusCode(t, searchResp, 200)
	results := decodeSchedules(t, searchResp)
	if len(results) < 3 {
		t.Errorf("expected at least 3 schedules for business, got %d", len(results))
	}
}

func testPostExceptionsArray(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Exceptions Branch")
	req["exceptions"] = []string{"2025-12-25", "2025-12-26", "2026-01-01"}

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)
	if len(created.Exceptions) != 3 {
		t.Errorf("expected 3 exceptions, got %d", len(created.Exceptions))
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))

	req2 := createValidSchedule("No Exceptions Branch")
	req2["exceptions"] = []string{}
	resp2 := httpClient.POST(t, "/api/v1/schedules", req2)
	common.AssertStatusCode(t, resp2, 201)

	req3 := createValidSchedule("Invalid Exception")
	req3["exceptions"] = []string{"not-a-date"}
	resp3 := httpClient.POST(t, "/api/v1/schedules", req3)
	if resp3.StatusCode != 422 && resp3.StatusCode != 400 && resp3.StatusCode != 201 {
		t.Errorf("unexpected status for invalid exception date: %d", resp3.StatusCode)
	}
}

func testPostAllOptionalFields(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("All Optional Fields")
	req["default_meeting_duration_min"] = 30
	req["default_break_duration_min"] = 10
	req["max_participants_per_slot"] = 5
	req["exceptions"] = []string{"2025-12-25"}

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if created.DefaultMeetingDurationMin != 30 {
		t.Errorf("expected meeting duration 30, got %d", created.DefaultMeetingDurationMin)
	}
	if created.DefaultBreakDurationMin != 10 {
		t.Errorf("expected break duration 10, got %d", created.DefaultBreakDurationMin)
	}
	if created.MaxParticipantsPerSlot != 5 {
		t.Errorf("expected max participants 5, got %d", created.MaxParticipantsPerSlot)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateWorkingDays(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Update Working Days")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{
		"working_days": []string{"Friday", "Saturday"},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	fetched := decodeSchedule(t, getResp)
	if len(fetched.WorkingDays) != 2 {
		t.Errorf("expected 2 working days after update, got %d", len(fetched.WorkingDays))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateTimeZone(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Update Timezone")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{"time_zone": "America/New_York"}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeSchedule(t, getResp)
	if fetched.TimeZone != "America/New_York" {
		t.Errorf("expected timezone America/New_York, got %s", fetched.TimeZone)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateAddExceptions(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Add Exceptions")
	req["exceptions"] = []string{}
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{
		"exceptions": []string{"2025-12-25", "2025-12-26"},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	fetched := decodeSchedule(t, getResp)
	if len(fetched.Exceptions) != 2 {
		t.Errorf("expected 2 exceptions after update, got %d", len(fetched.Exceptions))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateRemoveExceptions(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Remove Exceptions")
	req["exceptions"] = []string{"2025-12-25", "2025-12-26"}
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{
		"exceptions": []string{},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	fetched := decodeSchedule(t, getResp)
	if len(fetched.Exceptions) != 0 {
		t.Errorf("expected 0 exceptions after update, got %d", len(fetched.Exceptions))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateAllFieldsAtOnce(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Update All Fields")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{
		"name":                         "Completely Updated Branch",
		"city":                         "Haifa",
		"address":                      "New Address 123",
		"working_days":                 []string{"Sunday", "Monday"},
		"start_of_day":                 "08:00",
		"end_of_day":                   "20:00",
		"time_zone":                    "America/New_York",
		"default_meeting_duration_min": 45,
		"default_break_duration_min":   15,
		"max_participants_per_slot":    10,
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	fetched := decodeSchedule(t, getResp)
	if fetched.Name != "Completely Updated Branch" {
		t.Errorf("expected updated name, got %s", fetched.Name)
	}
	if fetched.City != "Haifa" {
		t.Errorf("expected city Haifa, got %s", fetched.City)
	}
	if fetched.StartOfDay != "08:00" {
		t.Errorf("expected start 08:00, got %s", fetched.StartOfDay)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateOnlyName(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Original Name")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{"name": "New Name Only"}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	fetched := decodeSchedule(t, getResp)
	if fetched.Name != "New Name Only" {
		t.Errorf("expected name 'New Name Only', got %s", fetched.Name)
	}

	if fetched.City != created.City {
		t.Errorf("city should not change, was %s, now %s", created.City, fetched.City)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testUpdateOnlyTimeRange(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Time Range Update")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{
		"start_of_day": "07:00",
		"end_of_day":   "21:00",
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	fetched := decodeSchedule(t, getResp)
	if fetched.StartOfDay != "07:00" || fetched.EndOfDay != "21:00" {
		t.Errorf("expected time range 07:00-21:00, got %s-%s", fetched.StartOfDay, fetched.EndOfDay)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testDuplicateScheduleDetection(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"
	address := "Unique Address 123"

	req1 := createValidSchedule("Branch A")
	req1["business_id"] = businessID
	req1["address"] = address
	resp1 := httpClient.POST(t, "/api/v1/schedules", req1)
	common.AssertStatusCode(t, resp1, 201)
	created1 := decodeSchedule(t, resp1)

	req2 := createValidSchedule("Branch A")
	req2["business_id"] = businessID
	req2["address"] = address
	resp2 := httpClient.POST(t, "/api/v1/schedules", req2)
	if resp2.StatusCode == 201 {
		created2 := decodeSchedule(t, resp2)
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created2.ID))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created1.ID))
}

func testConcurrentScheduleCreation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	var wg sync.WaitGroup
	results := make([]int, 5)
	ids := make([]string, 5)

	conc := 5

	for i := range conc {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			req := createValidSchedule(fmt.Sprintf("Concurrent Schedule %d", index))
			resp := httpClient.POST(t, "/api/v1/schedules", req)
			results[index] = resp.StatusCode
			if resp.StatusCode == 201 {
				created := decodeSchedule(t, resp)
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

	if successCount != conc {
		t.Errorf("Concurrent schedule creation: %d/5 succeeded", successCount)
	}

	for _, id := range ids {
		if id != "" {
			httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", id))
		}
	}
}

func testWorkingDaysNormalization(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Working Days Norm Test")
	req["working_days"] = []string{"sunday", "MONDAY", "TuEsDaY"}

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	if resp.StatusCode == 201 {
		created := decodeSchedule(t, resp)
		for _, wd := range created.WorkingDays {
			// if not spaces_trimmed and lowercased - err
		}
		t.Logf("Working days: %v", created.WorkingDays)
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	} else {
		t.Logf("Mixed case working days returned status %d", resp.StatusCode)
	}
}

func testExceptionsEdgeCases(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Many Exceptions")
	exceptions := []string{}
	for i := 1; i <= 365; i++ {
		exceptions = append(exceptions, fmt.Sprintf("2025-%02d-%02d", (i%12)+1, (i%28)+1))
	}
	req["exceptions"] = exceptions

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	if resp.StatusCode == 201 {
		created := decodeSchedule(t, resp)
		t.Logf("Created schedule with %d exceptions", len(created.Exceptions))
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	}

	// Test with duplicate exceptions
	req2 := createValidSchedule("Duplicate Exceptions")
	req2["exceptions"] = []string{"2025-12-25", "2025-12-25", "2025-12-26"}
	resp = httpClient.POST(t, "/api/v1/schedules", req2)
	if resp.StatusCode == 201 {
		created := decodeSchedule(t, resp)
		t.Logf("Duplicate exceptions test: got %d exceptions", len(created.Exceptions))
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	}
}

func testOptionalFieldsDefaults(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create schedule without optional fields
	req := createValidSchedule("Defaults Test")
	delete(req, "default_meeting_duration_min")
	delete(req, "default_break_duration_min")
	delete(req, "max_participants_per_slot")
	delete(req, "exceptions")

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	t.Logf("Default meeting duration: %d", created.DefaultMeetingDurationMin)
	t.Logf("Default break duration: %d", created.DefaultBreakDurationMin)
	t.Logf("Max participants: %d", created.MaxParticipantsPerSlot)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testSearchWithOnlyBusinessID(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"

	// Create multiple schedules for same business
	for i := 0; i < 3; i++ {
		req := createValidSchedule(fmt.Sprintf("Search Test %d", i))
		req["business_id"] = businessID
		req["city"] = fmt.Sprintf("City%d", i)
		httpClient.POST(t, "/api/v1/schedules", req)
	}

	// Search by business_id only (no city filter)
	resp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/search?business_id=%s", businessID))
	common.AssertStatusCode(t, resp, 200)
	results := decodeSchedules(t, resp)

	if len(results) < 3 {
		t.Errorf("expected at least 3 schedules for business, got %d", len(results))
	}
}

func testSearchWithCityFilter(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"

	// Create schedules in different cities
	req1 := createValidSchedule("Tel Aviv Branch")
	req1["business_id"] = businessID
	req1["city"] = "Tel Aviv"
	httpClient.POST(t, "/api/v1/schedules", req1)

	req2 := createValidSchedule("Jerusalem Branch")
	req2["business_id"] = businessID
	req2["city"] = "Jerusalem"
	httpClient.POST(t, "/api/v1/schedules", req2)

	// Search with city filter
	resp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/search?business_id=%s&city=Tel%%20Aviv", businessID))
	common.AssertStatusCode(t, resp, 200)
	results := decodeSchedules(t, resp)

	if len(results) < 1 {
		t.Error("expected at least 1 schedule in Tel Aviv")
	}

	for _, schedule := range results {
		if schedule.City != "Tel Aviv" {
			t.Errorf("expected city 'Tel Aviv', got %s", schedule.City)
		}
	}
}

func testUpdatePartialFields(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Partial Update Test")
	createResp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	originalCity := created.City
	originalAddress := created.Address

	// Update only name
	update := map[string]any{"name": "New Name Only"}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	// Verify other fields unchanged
	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	fetched := decodeSchedule(t, getResp)

	if fetched.Name != "New Name Only" {
		t.Errorf("expected updated name, got %s", fetched.Name)
	}
	if fetched.City != originalCity {
		t.Errorf("city should not change, was %s, now %s", originalCity, fetched.City)
	}
	if fetched.Address != originalAddress {
		t.Errorf("address should not change, was %s, now %s", originalAddress, fetched.Address)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}

func testCityNormalization(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Test various city formats
	cities := []string{
		"Tel Aviv",
		"TEL AVIV",
		"tel-aviv",
		"Tel_Aviv",
	}

	for _, city := range cities {
		req := createValidSchedule(fmt.Sprintf("City Test %s", city))
		req["city"] = city
		resp := httpClient.POST(t, "/api/v1/schedules", req)
		common.AssertStatusCode(t, resp, 201)
		created := decodeSchedule(t, resp)
		t.Logf("Input city: '%s', Stored city: '%s'", city, created.City)
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
	}
}

func testBreakDurationZero(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Zero Break Duration")
	req["default_break_duration_min"] = 0

	resp := httpClient.POST(t, "/api/v1/schedules", req)
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if created.DefaultBreakDurationMin != 0 {
		t.Errorf("expected break duration 0, got %d", created.DefaultBreakDurationMin)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/schedules/id/%s", created.ID))
}
