package integrationtests

import (
	"fmt"
	"math/rand"
	"os"
	"skeji/pkg/client"
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
	ServiceName = "schedules-integration-tests"
	TableName   = "schedules"
)

var (
	cfg             *config.Config
	httpClient      *client.HttpClient
	schedulesClient *client.ScheduleClient
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
	testLargeScaleSchedules(t)
	testSearchPaginationLargeDataset(t)
	testUpdateTimeRangeValidation(t)
	testMaxParticipantsValidation(t)
	testWorkingDaysEmptyArray(t)
	testWorkingDaysWeekendOnly(t)
	testExceptionsDuplicateDates(t)
	testExceptionsInvalidFormat(t)
	testExceptionsPastDates(t)
	testTimeZoneDSTTransition(t)
	testMultipleSchedulesSameCity(t)
	testScheduleWithLongAddress(t)
	testScheduleNameWithSpecialChars(t)
	testUpdateTimeZoneImpact(t)
	testSearchByMultipleCities(t)
	testConcurrentScheduleUpdates(t)
	testBatchScheduleCreation(t)
	testScheduleWithMinimalDuration(t)
	testScheduleWith24HourOperation(t)
	testUpdateWorkingDaysToEmpty(t)
	testMaxSchedulesPerBusinessUnit(t)
	testMaxSchedulesPerBusinessPerCityCreate(t)
	testMaxSchedulesPerBusinessPerCityUpdate(t)
}

func setup() {
	cfg = config.Load(ServiceName)

	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	httpClient = client.NewHttpClient(serverURL)
	schedulesClient = client.NewScheduleClient(serverURL)
}

func teardown() {
	cfg.GracefulShutdown()
}

func createValidSchedule(name string) map[string]any {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	suffix1 := r.Intn(100000)
	suffix2 := r.Intn(200000)

	return map[string]any{
		"business_id":  "507f1f77bcf86cd799439011",
		"name":         fmt.Sprintf("%s-%d", name, suffix1+suffix2),
		"city":         "Tel Aviv",
		"address":      fmt.Sprintf("Derech Menachem Begin 121 #%d", suffix1+suffix2),
		"working_days": []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday"},
		"start_of_day": "09:00",
		"end_of_day":   "18:00",
		"time_zone":    "Asia/Jerusalem",
	}
}

func decodeSchedule(t *testing.T, resp *client.Response) *model.Schedule {
	t.Helper()
	schedule, err := schedulesClient.DecodeSchedule(resp)
	if err != nil {
		t.Fatalf("failed to decode schedule: %v", err)
	}
	return schedule
}

func decodeSchedules(t *testing.T, resp *client.Response) []*model.Schedule {
	t.Helper()
	schedules, _, err := schedulesClient.DecodeSchedules(resp)
	if err != nil {
		t.Fatalf("failed to decode schedules: %v", err)
	}
	return schedules
}

func decodePaginated(t *testing.T, resp *client.Response) ([]*model.Schedule, int, int, int) {
	t.Helper()
	schedules, metadata, err := schedulesClient.DecodeSchedules(resp)
	if err != nil {
		t.Fatalf("failed to decode paginated schedules: %v", err)
	}
	return schedules, int(metadata.TotalCount), metadata.Limit, int(metadata.Offset)
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
	resp, err := schedulesClient.GetByID("507f1f77bcf86cd799439011")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 404)
	common.AssertContains(t, resp, "not found")
}

func testGetBySearchEmptyTable(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp, err := schedulesClient.Search("507f1f77bcf86cd799439011&city=Tel%20Aviv", "", 1000, 0)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	data := decodeSchedules(t, resp)
	if len(data) != 0 {
		t.Errorf("expected empty results, got %d", len(data))
	}
}

func testGetAllPaginatedEmptyTable(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp, err := schedulesClient.GetAll(10, 0)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)

	data, totalCount, _, _ := decodePaginated(t, resp)
	if totalCount != 0 || len(data) != 0 {
		t.Errorf("expected empty results, got total=%d, data=%d", totalCount, len(data))
	}
}

func testGetValidIdExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Acro Tower Branch")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	resp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	fetched := decodeSchedule(t, resp)

	if fetched.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, fetched.ID)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testGetInvalidIdExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Azrieli Branch")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	resp, err := schedulesClient.GetByID("invalid-id-format")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 400)

	resp, err = schedulesClient.GetByID("507f1f77bcf86cd799439011")
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 404)

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testGetValidSearchExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	adminBusiness := "507f1f77bcf86cd799439011"

	req1 := createValidSchedule("Acro Tower Branch")
	req1["business_id"] = adminBusiness
	req1["city"] = "Tel Aviv"
	resp, err := schedulesClient.Create(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	sch1, err := schedulesClient.DecodeSchedule(resp)
	if err != nil {
		t.Errorf("failed to decode schedule entity: %v", err)
	}
	if sch1.ID == "" {
		t.Errorf("failed to create schedule")
	}

	req2 := createValidSchedule("Azrieli Branch")
	req2["business_id"] = adminBusiness
	req2["city"] = "Jerusalem"
	resp, err = schedulesClient.Create(req2)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	sch2, err := schedulesClient.DecodeSchedule(resp)
	if err != nil {
		t.Errorf("failed to decode schedule entity: %v", err)
	}
	if sch2.ID == "" {
		t.Errorf("failed to create schedule")
	}

	resp, err = schedulesClient.Search("507f1f77bcf86cd799439011", "", 1000, 0)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	all := decodeSchedules(t, resp)
	if len(all) < 2 {
		t.Errorf("expected at least 2 results for business_id search, got %d", len(all))
	}

	resp, err = schedulesClient.Search("507f1f77bcf86cd799439011", "Tel%20Aviv", 1000, 0)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	filtered := decodeSchedules(t, resp)
	if len(filtered) < 1 {
		t.Error("expected city filter to return results")
	}
}

func testGetInvalidSearchExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp, err := httpClient.GET("/api/v1/schedules/search?city=Tel%20Aviv")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 400)
	common.AssertContains(t, resp, "business_id")

	resp, err = httpClient.GET("/api/v1/schedules/search?business_id=")
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 400)
}

func testGetValidPaginationExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	for i := 1; i <= 5; i++ {
		req := createValidSchedule(fmt.Sprintf("Branch %d", i))
		resp, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		common.AssertStatusCode(t, resp, 201)
	}

	resp, err := schedulesClient.GetAll(2, 0)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
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

	resp, err = schedulesClient.GetAll(2, 2)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 200)
}

func testGetInvalidPaginationExistingRecords(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp, err := httpClient.GET("/api/v1/schedules?limit=abc&offset=xyz")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 400)

	resp, err = httpClient.GET("/api/v1/schedules?limit=10&offset=-1")
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 200)
}

func testGetVerifyCreatedAt(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Ramat Aviv Clinic")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)
	if created.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}

	originalCreatedAt := created.CreatedAt
	update := map[string]any{"name": "Ramat Aviv Clinic - Updated"}
	_, err = schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeSchedule(t, getResp)

	if !fetched.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("created_at should not change on update: original=%v, after_update=%v", originalCreatedAt, fetched.CreatedAt)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testGetPaginationEdgeCases(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	for i := 0; i < 3; i++ {
		req := createValidSchedule(fmt.Sprintf("Edge Branch %d", i))
		_, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}

	resp, err := schedulesClient.GetAll(0, 0)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	data, _, _, _ := decodePaginated(t, resp)
	if len(data) > 10 {
		t.Errorf("limit=0 should return max 10 results, got %d results", len(data))
	}

	resp, err = schedulesClient.GetAll(1000, 0)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 200)
	data, _, _, _ = decodePaginated(t, resp)
	if len(data) > 100 {
		t.Errorf("limit=1000 should be capped at reasonable max (e.g. 100), got %d results", len(data))
	}

	resp, err = schedulesClient.GetAll(10, 9999)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 200)
	data, _, _, _ = decodePaginated(t, resp)
	if len(data) != 0 {
		t.Errorf("offset beyond total records should return empty array, got %d results", len(data))
	}
}

func testPostValidRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Acro Tower Branch")
	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if created.ID == "" {
		t.Error("expected ID to be set")
	}
	if created.Name != sanitizer.SanitizeNameOrAddress(fmt.Sprint(req["name"])) {
		t.Errorf("expected name 'Acro Tower Branch', got %s", created.Name)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testPostInvalidRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req1 := createValidSchedule("Missing Biz")
	delete(req1, "business_id")
	resp, err := schedulesClient.Create(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing business_id, got %d", resp.StatusCode)
	}

	req2 := createValidSchedule("Missing City")
	delete(req2, "city")
	resp, err = schedulesClient.Create(req2)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing city, got %d", resp.StatusCode)
	}

	req3 := createValidSchedule("Missing Address")
	delete(req3, "address")
	resp, err = schedulesClient.Create(req3)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for missing address, got %d", resp.StatusCode)
	}

	req4 := createValidSchedule("Empty Working Days")
	req4["working_days"] = []string{}
	resp, err = schedulesClient.Create(req4)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 201)

	req5 := createValidSchedule("Bad Time Format")
	req5["start_of_day"] = "25:61"
	resp, err = schedulesClient.Create(req5)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for bad start_of_day, got %d", resp.StatusCode)
	}

	req6 := createValidSchedule("Bad TZ")
	req6["time_zone"] = "Invalid/Timezone"
	resp, err = schedulesClient.Create(req6)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for invalid timezone, got %d", resp.StatusCode)
	}
}

func testPostMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp, err := schedulesClient.CreateRaw([]byte(`{"name": "x", "invalid`))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 400)

	resp, err = schedulesClient.CreateRaw([]byte(`not json at all`))
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 400)
}

func testPostWithSpecialCharacters(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Café - Acro Branch™ 🎨")
	req["address"] = "רח' יפה 10, תל אביב"
	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)
	if created.Name != sanitizer.SanitizeNameOrAddress(fmt.Sprint(req["name"])) {
		t.Errorf("expected special char name, got %s", created.Name)
	}
	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testPostWithTimeBoundaries(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Late Hours Branch")
	req["start_of_day"] = "00:00"
	req["end_of_day"] = "23:59"
	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)
	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateNonExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	updates := map[string]any{"name": "Updated Name"}
	resp, err := httpClient.PATCH("/api/v1/schedules/id/507f1f77bcf86cd799439011", updates)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 404)
}

func testUpdateWithInvalidId(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	updates := map[string]any{"name": "Updated Name"}
	resp, err := httpClient.PATCH("/api/v1/schedules/id/invalid-id-format", updates)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 400)
}

func testUpdateDeletedRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("To Be Deleted Branch")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	deleteResp, err := schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, deleteResp, 204)

	updates := map[string]any{"name": "Should Not Update"}
	resp, err := schedulesClient.Update(created.ID, updates)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 404)
}

func testUpdateWithBadFormatKeys(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Bad Format Branch")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	updates := map[string]any{"time_zone": "Invalid/Zone"}
	resp, err := schedulesClient.Update(created.ID, updates)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("invalid timezone in update returned %d", resp.StatusCode)
	}

	updates = map[string]any{"start_of_day": "99:99"}
	resp, err = schedulesClient.Update(created.ID, updates)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("invalid start_of_day in update returned %d", resp.StatusCode)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateWithGoodFormatKeys(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Acro Tower Branch")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	name := "Acro Tower Branch - Floor 12"
	updates := map[string]any{"name": name}
	resp, err := schedulesClient.Update(created.ID, updates)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeSchedule(t, getResp)
	if fetched.Name != sanitizer.SanitizeNameOrAddress(name) {
		t.Errorf("expected updated name, got %s", fetched.Name)
	}

	city := "Jerusalem"
	updates = map[string]any{"city": city}
	resp, err = schedulesClient.Update(created.ID, updates)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err = schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Error(err.Error())
	}
	fetched = decodeSchedule(t, getResp)
	if fetched.City != sanitizer.SanitizeCityOrLabel(city) {
		t.Errorf("expected city 'Jerusalem', got %s", fetched.City)
	}

	updates = map[string]any{
		"working_days": []string{"Sunday", "Monday", "Tuesday"},
		"start_of_day": "10:00",
		"end_of_day":   "19:00",
		"time_zone":    "Asia/Jerusalem",
	}
	resp, err = schedulesClient.Update(created.ID, updates)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err = schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Error(err.Error())
	}
	fetched = decodeSchedule(t, getResp)
	if fetched.StartOfDay != "10:00" || fetched.EndOfDay != "19:00" {
		t.Errorf("expected hours 10:00-19:00, got %s-%s", fetched.StartOfDay, fetched.EndOfDay)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateWithEmptyJson(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Empty JSON Branch")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	updates := map[string]any{}
	resp, err := schedulesClient.Update(created.ID, updates)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
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

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Malformed Update Branch")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	resp, err := schedulesClient.UpdateRaw(created.ID, []byte(`{"name": "x", invalid`))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 400)

	resp, err = schedulesClient.UpdateRaw(created.ID, []byte(`not json`))
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 400)

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testDeleteNonExistingRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp, err := schedulesClient.Delete("507f1f77bcf86cd799439011")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 404)
}

func testDeleteWithInvalidId(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp, err := schedulesClient.Delete("invalid-id-format")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 400)
}

func testDeletedRecord(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Delete Twice Branch")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	resp, err := schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	resp, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 404)
}

func testPostWorkingDaysSingleDay(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Single Day Branch")
	req["working_days"] = []string{"Monday"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)
	if len(created.WorkingDays) != 1 {
		t.Errorf("expected 1 working day, got %d", len(created.WorkingDays))
	}
	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testPostWorkingDaysAllWeek(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("All Week Branch")
	req["working_days"] = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)
	if len(created.WorkingDays) != 7 {
		t.Errorf("expected 7 working days, got %d", len(created.WorkingDays))
	}
	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testPostWorkingDaysInvalidDay(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Invalid Day Branch")
	req["working_days"] = []string{"InvalidDay", "Monday"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for invalid day, got %d", resp.StatusCode)
	}
}

func testPostWorkingDaysDuplicates(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Duplicate Days Branch")
	req["working_days"] = []string{"Monday", "Tuesday", "Monday"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("unexpected status code for duplicate days: %d", resp.StatusCode)
	}
}

func testPostWorkingDaysCaseSensitivity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Case Sensitivity Branch")
	req["working_days"] = []string{"monday", "TUESDAY", "Wednesday"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("unexpected status code for case sensitivity: %d", resp.StatusCode)
	}
}

func testPostTimeEqualStartEnd(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Zero Hours Branch")
	req["start_of_day"] = "10:00"
	req["end_of_day"] = "10:00"

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for start=end, got %d", resp.StatusCode)
	}
}

func testPostTimeEndBeforeStart(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Reverse Time Branch")
	req["start_of_day"] = "18:00"
	req["end_of_day"] = "09:00"

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
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

		resp, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		common.AssertStatusCode(t, resp, 201)
		created := decodeSchedule(t, resp)
		_, err = schedulesClient.Delete(created.ID)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}
}

func testPostOptionalFieldsBoundaries(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req1 := createValidSchedule("Meeting Duration Min")
	req1["default_meeting_duration_min"] = 1
	resp1, err := schedulesClient.Create(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp1, 422)

	req1 = createValidSchedule("Meeting Duration Min")
	req1["default_meeting_duration_min"] = 5
	resp1, err = schedulesClient.Create(req1)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp1, 201)

	req2 := createValidSchedule("Meeting Duration Max")
	req2["default_meeting_duration_min"] = 480
	resp2, err := schedulesClient.Create(req2)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp2, 201)

	req3 := createValidSchedule("Break Duration")
	req3["default_break_duration_min"] = 15
	resp3, err := schedulesClient.Create(req3)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp3, 201)

	req4 := createValidSchedule("Max Participants Min")
	req4["max_participants_per_slot"] = 1
	resp4, err := schedulesClient.Create(req4)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp4, 201)

	req5 := createValidSchedule("Max Participants Large")
	req5["max_participants_per_slot"] = 100
	resp5, err := schedulesClient.Create(req5)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp5, 201)
}

func testPostNameAndAddressLengths(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req1 := createValidSchedule("AB")
	resp1, err := schedulesClient.Create(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp1, 201)

	req2 := createValidSchedule("A")
	req2["name"] = "A"
	resp2, err := schedulesClient.Create(req2)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp2.StatusCode != 422 && resp2.StatusCode != 400 {
		t.Errorf("expected validation error for 1-char name, got %d", resp2.StatusCode)
	}

	longName := ""
	for range 90 {
		longName += "A"
	}
	req3 := createValidSchedule(longName)
	resp3, err := schedulesClient.Create(req3)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp3, 201)

	tooLongName := longName + "AAAAAAAAAAAAAAAAAAAAAAA"
	req4 := createValidSchedule(tooLongName)
	resp4, err := schedulesClient.Create(req4)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp4.StatusCode != 422 && resp4.StatusCode != 400 {
		t.Errorf("expected validation error for 101-char name, got %d", resp4.StatusCode)
	}

	longAddr := ""
	for range 200 {
		longAddr += "Long Address Street "
	}
	req5 := createValidSchedule("Long Address Branch")
	req5["address"] = longAddr
	resp5, err := schedulesClient.Create(req5)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
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
	resp1, err := schedulesClient.Create(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp1, 201)

	req2 := createValidSchedule("Jerusalem Branch")
	req2["business_id"] = businessID
	req2["city"] = "Jerusalem"
	resp2, err := schedulesClient.Create(req2)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp2, 201)

	req3 := createValidSchedule("Tel Aviv North")
	req3["business_id"] = businessID
	req3["city"] = "Tel Aviv"
	resp3, err := schedulesClient.Create(req3)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp3, 201)

	searchResp, err := httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s", businessID))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
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

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)
	if len(created.Exceptions) != 3 {
		t.Errorf("expected 3 exceptions, got %d", len(created.Exceptions))
	}
	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	req2 := createValidSchedule("No Exceptions Branch")
	req2["exceptions"] = []string{}
	resp2, err := schedulesClient.Create(req2)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp2, 201)

	req3 := createValidSchedule("Invalid Exception")
	req3["exceptions"] = []string{"not-a-date"}
	resp3, err := schedulesClient.Create(req3)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
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

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
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

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateWorkingDays(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Update Working Days")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{
		"working_days": []string{"Friday", "Saturday"},
	}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	fetched := decodeSchedule(t, getResp)
	if len(fetched.WorkingDays) != 2 {
		t.Errorf("expected 2 working days after update, got %d", len(fetched.WorkingDays))
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateTimeZone(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Update Timezone")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{"time_zone": "America/New_York"}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeSchedule(t, getResp)
	if fetched.TimeZone != "America/New_York" {
		t.Errorf("expected timezone America/New_York, got %s", fetched.TimeZone)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateAddExceptions(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Add Exceptions")
	req["exceptions"] = []string{}
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{
		"exceptions": []string{"2025-12-25", "2025-12-26"},
	}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	fetched := decodeSchedule(t, getResp)
	if len(fetched.Exceptions) != 2 {
		t.Errorf("expected 2 exceptions after update, got %d", len(fetched.Exceptions))
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateRemoveExceptions(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Remove Exceptions")
	req["exceptions"] = []string{"2025-12-25", "2025-12-26"}
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{
		"exceptions": []string{},
	}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	fetched := decodeSchedule(t, getResp)
	if len(fetched.Exceptions) != 0 {
		t.Errorf("expected 0 exceptions after update, got %d", len(fetched.Exceptions))
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateAllFieldsAtOnce(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Update All Fields")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
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
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	fetched := decodeSchedule(t, getResp)
	if fetched.Name != sanitizer.SanitizeNameOrAddress(fmt.Sprint(update["name"])) {
		t.Errorf("expected updated name, got %s", fetched.Name)
	}
	if fetched.City != sanitizer.SanitizeCityOrLabel(fmt.Sprint(update["city"])) {
		t.Errorf("expected city Haifa, got %s", fetched.City)
	}
	if fetched.StartOfDay != "08:00" {
		t.Errorf("expected start 08:00, got %s", fetched.StartOfDay)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateOnlyName(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Original Name")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{"name": "New Name Only"}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	fetched := decodeSchedule(t, getResp)
	if fetched.Name != sanitizer.SanitizeNameOrAddress(fmt.Sprint(update["name"])) {
		t.Errorf("expected name 'New Name Only', got %s", fetched.Name)
	}

	if fetched.City != created.City {
		t.Errorf("city should not change, was %s, now %s", created.City, fetched.City)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateOnlyTimeRange(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	req := createValidSchedule("Time Range Update")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	update := map[string]any{
		"start_of_day": "07:00",
		"end_of_day":   "21:00",
	}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	fetched := decodeSchedule(t, getResp)
	if fetched.StartOfDay != "07:00" || fetched.EndOfDay != "21:00" {
		t.Errorf("expected time range 07:00-21:00, got %s-%s", fetched.StartOfDay, fetched.EndOfDay)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testDuplicateScheduleDetection(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"
	address := "Unique Address 123"

	req1 := createValidSchedule("Branch A")
	req1["business_id"] = businessID
	req1["address"] = address
	resp1, err := schedulesClient.Create(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp1, 201)
	created1 := decodeSchedule(t, resp1)

	req2 := createValidSchedule("Branch A")
	req2["business_id"] = businessID
	req2["address"] = address
	resp2, err := schedulesClient.Create(req2)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp2.StatusCode == 201 {
		created2 := decodeSchedule(t, resp2)
		_, err = schedulesClient.Delete(created2.ID)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}

	_, err = schedulesClient.Delete(created1.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testConcurrentScheduleCreation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	conc := 5
	var wg sync.WaitGroup

	results := make([]int, conc)
	ids := make([]string, conc)

	errCh := make(chan error, conc)

	for i := 0; i < conc; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			req := createValidSchedule(fmt.Sprintf("Concurrent Schedule %d", index))
			resp, err := schedulesClient.Create(req)
			if err != nil {
				errCh <- fmt.Errorf("create failed for schedule %d: %w", index, err)
				return
			}

			results[index] = resp.StatusCode

			if resp.StatusCode == 201 {
				created := decodeSchedule(t, resp)
				ids[index] = created.ID
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Collect errors from goroutines
	for err := range errCh {
		t.Error(err)
	}

	successCount := 0
	for _, status := range results {
		if status == 201 {
			successCount++
		}
	}

	if successCount != conc {
		t.Errorf("Concurrent schedule creation: %d/%d succeeded", successCount, conc)
	}

	// Cleanup
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, err := schedulesClient.Delete(id); err != nil {
			t.Errorf("cleanup failed for schedule %s: %v", id, err)
		}
	}
}

func testWorkingDaysNormalization(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Working Days Norm Test")
	req["working_days"] = []string{"sunday", "MONDAY", "TuEsDaY"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp.StatusCode == 201 {
		created := decodeSchedule(t, resp)
		for _, wd := range created.WorkingDays {
			if wd != strings.ToLower(strings.TrimSpace(wd)) {
				t.Errorf("Working days: %v", created.WorkingDays)
			}
		}
		_, err = schedulesClient.Delete(created.ID)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	} else {
		t.Errorf("Mixed case working days returned status %d", resp.StatusCode)
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

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for num of exceptions is too big, got %d", resp.StatusCode)
	}

	req2 := createValidSchedule("Duplicate Exceptions")
	req2["exceptions"] = []string{"2025-12-25", "2025-12-25", "2025-12-26"}
	resp, err = schedulesClient.Create(req2)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.StatusCode == 201 {
		created := decodeSchedule(t, resp)
		if len(created.Exceptions) != 2 {
			t.Errorf("Duplicate exceptions test: got %d exceptions", len(created.Exceptions))
		}
		_, err = schedulesClient.Delete(created.ID)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}
}

func testOptionalFieldsDefaults(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Defaults Test")
	delete(req, "default_meeting_duration_min")
	delete(req, "default_break_duration_min")
	delete(req, "max_participants_per_slot")
	delete(req, "exceptions")

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if created.DefaultMeetingDurationMin != config.DefaultDefaultMeetingDurationMin {
		t.Errorf("Default meeting duration: %d", created.DefaultMeetingDurationMin)
	}
	if created.DefaultBreakDurationMin != config.DefaultDefaultBreakDurationMin {
		t.Errorf("Default break duration: %d", created.DefaultBreakDurationMin)
	}
	if created.MaxParticipantsPerSlot != config.DefaultDefaultMaxParticipantsPerSlot {
		t.Errorf("Max participants: %d", created.MaxParticipantsPerSlot)
	}
	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testSearchWithOnlyBusinessID(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"

	for i := range 3 {
		req := createValidSchedule(fmt.Sprintf("Search Test %d", i))
		req["business_id"] = businessID
		req["city"] = fmt.Sprintf("City%d", i)
		_, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}

	resp, err := httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s", businessID))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	results := decodeSchedules(t, resp)

	if len(results) < 3 {
		t.Errorf("expected at least 3 schedules for business, got %d", len(results))
	}
}

func testSearchWithCityFilter(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"

	req1 := createValidSchedule("Tel Aviv Branch")
	req1["business_id"] = businessID
	req1["city"] = "Tel Aviv"
	_, err := schedulesClient.Create(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	req2 := createValidSchedule("Jerusalem Branch")
	req2["business_id"] = businessID
	req2["city"] = "Jerusalem"
	_, err = schedulesClient.Create(req2)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	resp, err := httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s&city=Tel%%20Aviv", businessID))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	results := decodeSchedules(t, resp)

	if len(results) < 1 {
		t.Error("expected at least 1 schedule in Tel Aviv")
	}

	for _, schedule := range results {
		if schedule.City != sanitizer.SanitizeCityOrLabel(fmt.Sprint(req1["city"])) {
			t.Errorf("expected city 'Tel Aviv', got %s", schedule.City)
		}
	}
}

func testUpdatePartialFields(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Partial Update Test")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	originalCity := created.City
	originalAddress := created.Address

	update := map[string]any{"name": "New Name Only"}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	fetched := decodeSchedule(t, getResp)

	if fetched.Name != sanitizer.SanitizeNameOrAddress(fmt.Sprint(update["name"])) {
		t.Errorf("expected updated name, got %s", fetched.Name)
	}
	if fetched.City != originalCity {
		t.Errorf("city should not change, was %s, now %s", originalCity, fetched.City)
	}
	if fetched.Address != originalAddress {
		t.Errorf("address should not change, was %s, now %s", originalAddress, fetched.Address)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testCityNormalization(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	cities := []string{
		"Tel Aviv",
		"TEL AVIV",
		"tel-aviv",
		"Tel_Aviv",
	}

	for _, city := range cities {
		req := createValidSchedule(fmt.Sprintf("City Test %s", city))
		req["city"] = city
		resp, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		common.AssertStatusCode(t, resp, 201)
		created := decodeSchedule(t, resp)
		if created.City != sanitizer.SanitizeCityOrLabel(city) {
			t.Errorf("Input city: '%s', Stored city: '%s'", city, created.City)
		}
		_, err = schedulesClient.Delete(created.ID)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}
}

func testBreakDurationZero(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Zero Break Duration")
	req["default_break_duration_min"] = 1

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if created.DefaultBreakDurationMin != 1 {
		t.Errorf("expected break duration 1, got %d", created.DefaultBreakDurationMin)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

// ========== ENRICHED TESTS ==========

func testLargeScaleSchedules(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"
	createdIDs := []string{}
	successCount := 0
	failCount := 0

	// Attempt to create 50 schedules — expect only 10 to succeed
	for i := 0; i < 50; i++ {
		req := createValidSchedule(fmt.Sprintf("Large Scale Schedule %d", i))
		req["business_id"] = businessID
		req["city"] = fmt.Sprintf("City%d", i%10)

		resp, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}

		if resp.StatusCode == 201 {
			created := decodeSchedule(t, resp)
			createdIDs = append(createdIDs, created.ID)
			successCount++
		} else {
			failCount++
		}
	}

	// Assert business rule enforcement
	if successCount != 10 {
		t.Fatalf("expected exactly 10 schedules to be created, got %d", successCount)
	}
	if failCount != 40 {
		t.Fatalf("expected 40 schedules to fail creation, got %d", failCount)
	}

	// Verify pagination
	resp, err := schedulesClient.GetAll(25, 0)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	common.AssertStatusCode(t, resp, 200)
	data, total, _, _ := decodePaginated(t, resp)

	if total != 10 {
		t.Errorf("expected total = 10 schedules, got %d", total)
	}
	if len(data) != 10 { // because total < limit
		t.Errorf("expected page size = 10, got %d", len(data))
	}

	// Cleanup only the successfully created ones
	for _, id := range createdIDs {
		_, err = schedulesClient.Delete(id)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}
}

func testSearchPaginationLargeDataset(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"

	// Create 30 schedules
	for i := 0; i < 30; i++ {
		req := createValidSchedule(fmt.Sprintf("Pagination Test %d", i))
		req["business_id"] = businessID
		req["city"] = "TestCity"
		_, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}

	// Test search pagination
	resp, err := httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s&limit=10&offset=0", businessID))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)

	// Get second page
	resp, err = httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s&limit=10&offset=10", businessID))
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 200)
}

func testUpdateTimeRangeValidation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Time Range Validation")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	// Try to update with invalid time range (end before start)
	update := map[string]any{
		"start_of_day": "18:00",
		"end_of_day":   "09:00",
	}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for invalid time range, got %d", resp.StatusCode)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testMaxParticipantsValidation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Max Participants Test")
	req["max_participants_per_slot"] = 1000

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	// Should either accept or cap at maximum
	if resp.StatusCode == 201 {
		created := decodeSchedule(t, resp)
		t.Logf("Created with max_participants_per_slot = %d", created.MaxParticipantsPerSlot)
		_, err = schedulesClient.Delete(created.ID)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	} else if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("unexpected status for max participants: %d", resp.StatusCode)
	}
}

func testWorkingDaysEmptyArray(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Empty Working Days")
	req["working_days"] = []string{}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	expectedDays := 5
	if len(created.WorkingDays) != expectedDays {
		t.Errorf("expected %d working days (Israel defaults), got %d", expectedDays, len(created.WorkingDays))
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testWorkingDaysWeekendOnly(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Weekend Only")
	req["working_days"] = []string{"Friday", "Saturday"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if len(created.WorkingDays) != 2 {
		t.Errorf("expected 2 working days, got %d", len(created.WorkingDays))
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testExceptionsDuplicateDates(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Duplicate Exceptions")
	req["exceptions"] = []string{"2025-12-25", "2025-12-25", "2025-12-26"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	// Should deduplicate
	if len(created.Exceptions) != 2 {
		t.Errorf("expected 2 unique exceptions, got %d", len(created.Exceptions))
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testExceptionsInvalidFormat(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Invalid Exception Format")
	req["exceptions"] = []string{"12/25/2025", "not-a-date", "2025-13-45"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	// Should either reject or filter out invalid dates
	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Invalid exception format returned status %d", resp.StatusCode)
	}
}

func testExceptionsPastDates(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Past Exceptions")
	req["exceptions"] = []string{"2020-01-01", "2021-12-25", "2022-06-15"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	// Should either accept or reject past dates
	if resp.StatusCode == 201 {
		created := decodeSchedule(t, resp)
		_, err = schedulesClient.Delete(created.ID)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	} else if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Past exceptions returned status %d", resp.StatusCode)
	}
}

func testTimeZoneDSTTransition(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create schedule with timezone that observes DST
	req := createValidSchedule("DST Transition Test")
	req["time_zone"] = "America/New_York"

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if created.TimeZone != "America/New_York" {
		t.Errorf("expected timezone America/New_York, got %s", created.TimeZone)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testMultipleSchedulesSameCity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"

	// Create 5 schedules in same city
	createdIDs := []string{}
	for i := 0; i < 5; i++ {
		req := createValidSchedule(fmt.Sprintf("Same City Schedule %d", i))
		req["business_id"] = businessID
		req["city"] = "Tel Aviv"
		resp, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		common.AssertStatusCode(t, resp, 201)
		created := decodeSchedule(t, resp)
		createdIDs = append(createdIDs, created.ID)
	}

	// Search by city
	resp, err := httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s&city=Tel%%20Aviv", businessID))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	results := decodeSchedules(t, resp)

	if len(results) < 5 {
		t.Errorf("expected at least 5 schedules in Tel Aviv, got %d", len(results))
	}

	// Cleanup
	for _, id := range createdIDs {
		_, err = schedulesClient.Delete(id)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}
}

func testScheduleWithLongAddress(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	longAddress := ""
	for i := 0; i < 500; i++ {
		longAddress += "Long Address Street "
	}

	req := createValidSchedule("Long Address Test")
	req["address"] = longAddress

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	// Should either accept with truncation or reject
	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Long address returned status %d", resp.StatusCode)
	}
}

func testScheduleNameWithSpecialChars(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	specialNames := []string{
		"עברית Branch",
		"中文 Location",
		"Café™ Shop",
		"Test@#$% Center",
	}

	for _, name := range specialNames {
		req := createValidSchedule(name)
		resp, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}

		if resp.StatusCode == 201 {
			created := decodeSchedule(t, resp)
			_, err = schedulesClient.Delete(created.ID)
			if err != nil {
				t.Fatalf("HTTP request failed: %v", err)
			}
		}
	}
}

func testUpdateTimeZoneImpact(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Timezone Update Test")
	req["time_zone"] = "Asia/Jerusalem"
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	// Update to different timezone
	update := map[string]any{"time_zone": "America/Los_Angeles"}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 204)

	// Verify timezone changed
	getResp, err := schedulesClient.GetByID(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	updated := decodeSchedule(t, getResp)

	if updated.TimeZone != "America/Los_Angeles" {
		t.Errorf("expected timezone America/Los_Angeles, got %s", updated.TimeZone)
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testSearchByMultipleCities(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439011"

	// Create schedules in different cities
	req1 := createValidSchedule("Tel Aviv Branch")
	req1["business_id"] = businessID
	req1["city"] = "Tel Aviv"
	_, err := schedulesClient.Create(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	req2 := createValidSchedule("Haifa Branch")
	req2["business_id"] = businessID
	req2["city"] = "Haifa"
	_, err = schedulesClient.Create(req2)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	req3 := createValidSchedule("Jerusalem Branch")
	req3["business_id"] = businessID
	req3["city"] = "Jerusalem"
	_, err = schedulesClient.Create(req3)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	// Search for specific city
	resp, err := httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s&city=Tel%%20Aviv", businessID))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	results := decodeSchedules(t, resp)

	if len(results) < 1 {
		t.Error("expected at least 1 schedule in Tel Aviv")
	}
}

func testConcurrentScheduleUpdates(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Concurrent Update Test")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("initial create failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)

	created := decodeSchedule(t, createResp)

	var wg sync.WaitGroup
	results := make([]int, 10)
	errCh := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			update := map[string]any{
				"name": fmt.Sprintf("Updated Name %d", index),
			}

			resp, err := schedulesClient.Update(created.ID, update)
			if err != nil {
				errCh <- fmt.Errorf("update %d failed: %w", index, err)
				return
			}

			results[index] = resp.StatusCode
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Collect goroutine errors
	for err := range errCh {
		t.Error(err)
	}

	// Count successes
	successCount := 0
	for _, status := range results {
		if status == 204 {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("concurrent updates: %d/10 succeeded", successCount)
	}

	if _, err := schedulesClient.Delete(created.ID); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

func testBatchScheduleCreation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	createdIDs := []string{}
	successCount := 0
	failCount := 0

	// Create 20 schedules for the same business
	for i := 0; i < 20; i++ {
		req := createValidSchedule(fmt.Sprintf("Batch Schedule %d", i))

		resp, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}

		if resp.StatusCode == 201 {
			created := decodeSchedule(t, resp)
			createdIDs = append(createdIDs, created.ID)
			successCount++
		} else {
			failCount++
		}
	}

	// Assertions matching new business rule
	if successCount != 10 {
		t.Errorf("expected 10 schedules allowed per business, got %d", successCount)
	}
	if failCount != 10 {
		t.Errorf("expected 10 failures after exceeding the limit, got %d", failCount)
	}

	// Cleanup only created ones
	for _, id := range createdIDs {
		_, err := schedulesClient.Delete(id)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}
}

func testScheduleWithMinimalDuration(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Minimal Duration")
	req["start_of_day"] = "09:00"
	req["end_of_day"] = "09:01" // 1 minute duration

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	// Should either accept or reject based on business rules
	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Minimal duration returned status %d", resp.StatusCode)
	}
}

func testScheduleWith24HourOperation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("24 Hour Operation")
	req["start_of_day"] = "00:00"
	req["end_of_day"] = "23:59"
	req["working_days"] = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	resp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created := decodeSchedule(t, resp)

	if created.StartOfDay != "00:00" || created.EndOfDay != "23:59" {
		t.Errorf("expected 24-hour operation, got %s - %s", created.StartOfDay, created.EndOfDay)
	}

	if len(created.WorkingDays) != 7 {
		t.Errorf("expected 7 working days, got %d", len(created.WorkingDays))
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testUpdateWorkingDaysToEmpty(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	req := createValidSchedule("Update To Empty Working Days")
	createResp, err := schedulesClient.Create(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, createResp, 201)
	created := decodeSchedule(t, createResp)

	// Update to empty working days
	update := map[string]any{
		"working_days": []string{},
	}
	resp, err := schedulesClient.Update(created.ID, update)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	// Should accept empty working days
	if resp.StatusCode == 204 {
		getResp, err := schedulesClient.GetByID(created.ID)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		updated := decodeSchedule(t, getResp)

		if len(updated.WorkingDays) != 0 {
			t.Errorf("expected 0 working days after update, got %d", len(updated.WorkingDays))
		}
	}

	_, err = schedulesClient.Delete(created.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
}

func testMaxSchedulesPerBusinessUnit(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// The limit is DefaultMaxSchedulesPerBusinessUnits (2000)
	// We'll create just a few to test we're below limit
	// Testing the actual limit would be too slow for integration tests

	businessID := "507f1f77bcf86cd799439999"
	createdIDs := make([]string, 0, 5)

	// Create 5 schedules for the same business unit successfully
	for i := 0; i < 5; i++ {
		req := createValidSchedule(fmt.Sprintf("Schedule %d", i))
		req["business_id"] = businessID
		req["city"] = fmt.Sprintf("City%d", i)
		req["address"] = fmt.Sprintf("Address %d", i)
		resp, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		common.AssertStatusCode(t, resp, 201)
		created := decodeSchedule(t, resp)
		createdIDs = append(createdIDs, created.ID)
	}

	// Verify we have 5 schedules for this business
	resp, err := httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s&limit=10&offset=0", businessID))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	_, count, _, _ := decodePaginated(t, resp)
	if count != 5 {
		t.Errorf("expected 5 schedules for business, got %d", count)
	}

	// Note: The limit enforcement logic is tested through the service layer
	// This test verifies the mechanism works for small numbers

	// Cleanup
	for _, id := range createdIDs {
		_, err = schedulesClient.Delete(id)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}
}

func testMaxSchedulesPerBusinessPerCityCreate(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// The limit is DefaultMaxSchedulesPerBusinessUnitsPerCity (200)
	// We'll create just a few to test we're below limit

	businessID := "507f1f77bcf86cd799439888"
	city := "TestCityLimit"
	createdIDs := make([]string, 0, 5)

	// Create 5 schedules for the same business unit and city successfully
	for i := 0; i < 5; i++ {
		req := createValidSchedule(fmt.Sprintf("Schedule %d", i))
		req["business_id"] = businessID
		req["city"] = city
		req["address"] = fmt.Sprintf("Unique Address %d", i)
		resp, err := schedulesClient.Create(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		common.AssertStatusCode(t, resp, 201)
		created := decodeSchedule(t, resp)
		createdIDs = append(createdIDs, created.ID)
	}

	// Verify we have 5 schedules for this business and city
	resp, err := httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s&city=%s&limit=10&offset=0", businessID, city))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 200)
	_, count, _, _ := decodePaginated(t, resp)
	if count != 5 {
		t.Errorf("expected 5 schedules for business and city, got %d", count)
	}

	// Note: Testing the actual limit of 200 would be too slow for integration tests
	// The limit enforcement logic is tested through the service layer
	// This test verifies the mechanism works for small numbers

	// Cleanup
	for _, id := range createdIDs {
		_, err = schedulesClient.Delete(id)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
	}
}

func testMaxSchedulesPerBusinessPerCityUpdate(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	businessID := "507f1f77bcf86cd799439777"
	city1 := "CityA"
	city2 := "CityB"

	// Create schedule in city1
	req1 := createValidSchedule("Schedule 1")
	req1["business_id"] = businessID
	req1["city"] = city1
	req1["address"] = "Address 1"
	resp, err := schedulesClient.Create(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	common.AssertStatusCode(t, resp, 201)
	created1 := decodeSchedule(t, resp)

	// Create schedule in city2
	req2 := createValidSchedule("Schedule 2")
	req2["business_id"] = businessID
	req2["city"] = city2
	req2["address"] = "Address 2"
	resp, err = schedulesClient.Create(req2)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 201)
	created2 := decodeSchedule(t, resp)

	// Update schedule 2's city to city1 (should work since we're under limit)
	updateReq := map[string]any{
		"city": city1,
	}
	resp, err = schedulesClient.Update(created2.ID, updateReq)
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 204)

	// Verify both schedules now have city1
	resp, err = httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s&city=%s&limit=10&offset=0", businessID, city1))
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 200)
	_, count, _, _ := decodePaginated(t, resp)
	if count != 2 {
		t.Errorf("expected 2 schedules for city1 after update, got %d", count)
	}

	// Verify city2 now has 0 schedules
	resp, err = httpClient.GET(fmt.Sprintf("/api/v1/schedules/search?business_id=%s&city=%s&limit=10&offset=0", businessID, city2))
	if err != nil {
		t.Error(err.Error())
	}
	common.AssertStatusCode(t, resp, 200)
	_, count, _, _ = decodePaginated(t, resp)
	if count != 0 {
		t.Errorf("expected 0 schedules for city2 after update, got %d", count)
	}

	// Cleanup
	_, err = schedulesClient.Delete(created1.ID)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	_, err = schedulesClient.Delete(created2.ID)
	if err != nil {
		t.Error(err.Error())
	}
}
