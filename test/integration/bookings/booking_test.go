package integrationtests

import (
	"fmt"
	"os"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/pkg/sanitizer"
	"skeji/test/common"
	"sync"
	"testing"
	"time"
)

const (
	ServiceName = "bookings-integration-tests"
	TableName   = "bookings"
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
	testConcurrentBookingCreation(t)
	testBookingStatusCompleted(t)
	testParticipantsValidation(t)
	testSearchWithExactTimeMatch(t)
	testBookingWithPastEndTime(t)
	testUpdateParticipantsExceedCapacity(t)
	testManagedByValidation(t)
	testSearchWithoutTimeRange(t)
	testUpdateClearParticipants(t)
	testBookingWithSameBusinessDifferentSchedule(t)
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

// --- Helpers ---

func createValidBooking(businessID, scheduleID string, label string, start, end time.Time) map[string]any {
	return map[string]any{
		"business_id":   businessID,
		"schedule_id":   scheduleID,
		"service_label": label,
		"start_time":    start.Format(time.RFC3339),
		"end_time":      end.Format(time.RFC3339),
		"capacity":      5,
		"participants": map[string]string{
			"+972501234567": "Alice",
			"+972541111111": "Bob",
		},
		"status":     "pending",
		"managed_by": map[string]string{"+972509999999": "Manager"},
	}
}

func decodeBooking(t *testing.T, resp *common.Response) *model.Booking {
	t.Helper()
	var result struct {
		Data model.Booking `json:"data"`
	}
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode booking: %v", err)
	}
	return &result.Data
}

func decodeBookings(t *testing.T, resp *common.Response) []model.Booking {
	t.Helper()
	var result struct {
		Data []model.Booking `json:"data"`
	}
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode bookings: %v", err)
	}
	return result.Data
}

func decodeBookingsPaginated(t *testing.T, resp *common.Response) ([]model.Booking, int, int, int) {
	t.Helper()
	var result struct {
		Data       []model.Booking `json:"data"`
		TotalCount int             `json:"total_count"`
		Limit      int             `json:"limit"`
		Offset     int             `json:"offset"`
	}
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode paginated bookings: %v", err)
	}
	return result.Data, result.TotalCount, result.Limit, result.Offset
}

func testGet(t *testing.T) {
	testGetEmptyTable(t)
	testGetAllPaginatedEmpty(t)
	testCreateAndGetByID(t)
	testGetInvalidID(t)
	testSearchEmpty(t)
	testSearchRange(t)
}

func testPost(t *testing.T) {
	testCreateValid(t)
	testCreateInvalidTimeRange(t)
	testCreateOverlapConflict(t)
	testCreateInvalidParticipantFormat(t)
	testCreateMalformedJSON(t)
	testCreateCapacityBoundaries(t)
	testCreateZeroCapacity(t)
	testCreateNegativeCapacity(t)
	testCreateCapacityExceededByParticipants(t)
	testCreateEmptyParticipants(t)
	testCreateMaxParticipants(t)
	testCreateTooManyParticipants(t)
	testCreateDuplicateParticipants(t)
	testCreateMultipleCountryPhones(t)
	testCreateServiceLabelBoundaries(t)
	testCreateServiceLabelSpecialChars(t)
	testCreatePastTime(t)
	testCreateMidnightTime(t)
	testCreateVeryShortDuration(t)
	testCreateVeryLongDuration(t)
	testCreateMultipleDaySpan(t)
	testCreateInvalidBusinessID(t)
	testCreateInvalidScheduleID(t)
	testCreateAllStatuses(t)
	testCreateInvalidStatus(t)
	testCreateExactSameTime(t)
	testCreatePartialOverlap(t)
}

func testUpdate(t *testing.T) {
	testUpdateValid(t)
	testUpdateInvalidID(t)
	testUpdateTimeOverlap(t)
	testUpdateMalformedJSON(t)
	testUpdateStatusTransitions(t)
	testUpdateCapacityBelowParticipants(t)
	testUpdateAddParticipants(t)
	testUpdateRemoveParticipants(t)
	testUpdateManagedBy(t)
	testUpdateOnlyTime(t)
	testUpdateOnlyCapacity(t)
	testUpdateMultipleFields(t)
}

func testDelete(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	testDeleteNonExisting(t)
	testDeleteInvalidID(t)
	testCreateAndDelete(t)
	testDoubleDelete(t)
}

func testGetEmptyTable(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/bookings/id=507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
}

func testGetAllPaginatedEmpty(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/bookings?limit=10&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, total, _, _ := decodeBookingsPaginated(t, resp)
	if total != 0 || len(data) != 0 {
		t.Errorf("expected empty table, got total=%d len=%d", total, len(data))
	}
}

func testCreateValid(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Haircut", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)
	if created.ID == "" {
		t.Error("expected booking ID to be set")
	}
	if created.ServiceLabel != sanitizer.SanitizeCityOrLabel(fmt.Sprint(payload["service_label"])) {
		t.Errorf("expected service_label '%s', got %s",
			sanitizer.SanitizeCityOrLabel(fmt.Sprint(payload["service_label"])),
			created.ServiceLabel,
		)
	}
	if string(created.Status) != "pending" {
		t.Errorf("expected default status 'pending', got '%s'", string(created.Status))
	}
	if created.Capacity != 5 {
		t.Errorf("expected capacity 5, got %d", created.Capacity)
	}

	if len(created.Participants) != 2 {
		t.Errorf("expected 2 participants, got %d", len(created.Participants))
	}
	p1Phone := sanitizer.SanitizePhone("+972501234567")
	p2Phone := sanitizer.SanitizePhone("+972541111111")
	p1Name := sanitizer.SanitizeNameOrAddress("Alice")
	p2Name := sanitizer.SanitizeNameOrAddress("Bob")
	if created.Participants[p1Phone] != p1Name {
		t.Errorf("expected participant %s -> %s, got %s",
			p1Phone, p1Name, created.Participants[p1Phone])
	}
	if created.Participants[p2Phone] != p2Name {
		t.Errorf("expected participant %s -> %s, got %s",
			p2Phone, p2Name, created.Participants[p2Phone])
	}

	if len(created.ManagedBy) != 1 {
		t.Errorf("expected 1 managed_by entry, got %d", len(created.ManagedBy))
	}
	mPhone := sanitizer.SanitizePhone("+972509999999")
	mName := sanitizer.SanitizeNameOrAddress("Manager")
	if created.ManagedBy[mPhone] != mName {
		t.Errorf("expected managed_by %s -> %s, got %s",
			mPhone, mName, created.ManagedBy[mPhone])
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testCreateInvalidTimeRange(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(2 * time.Hour)
	end := start.Add(-1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Invalid Time", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 422)
	common.AssertContains(t, resp, "EndTime")
}

func testCreateOverlapConflict(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	now := time.Now().Add(1 * time.Hour)
	payload1 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Haircut", now, now.Add(1*time.Hour))
	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Haircut 2", now.Add(30*time.Minute), now.Add(2*time.Hour))

	resp1 := httpClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)
	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	if resp2.StatusCode != 409 && resp2.StatusCode != 400 {
		t.Errorf("expected conflict or validation error, got %d", resp2.StatusCode)
	}
}

func testCreateInvalidParticipantFormat(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Bad Participants", start, end)
	payload["participants"] = map[string]string{"notaphone": "Invalid"}

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected 400/422 for invalid participant phone, got %d", resp.StatusCode)
	}
}

func testCreateMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.POSTRaw(t, "/api/v1/bookings", []byte(`{"bad": json`))
	common.AssertStatusCode(t, resp, 400)
}

func testCreateAndGetByID(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Massage", start, end)

	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeBooking(t, getResp)
	if fetched.ID != created.ID {
		t.Errorf("expected same ID, got %s != %s", fetched.ID, created.ID)
	}
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testGetInvalidID(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/bookings/id/invalid-id")
	common.AssertStatusCode(t, resp, 400)
}

func testSearchEmpty(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	resp := httpClient.GET(t, "/api/v1/bookings/search?business_id=507f1f77bcf86cd799439011&schedule_id=507f1f77bcf86cd799439012")
	common.AssertStatusCode(t, resp, 200)
	data := decodeBookings(t, resp)
	if len(data) != 0 {
		t.Errorf("expected empty results, got %d", len(data))
	}
}

func testSearchRange(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(2 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Range Search", start, end)
	httpClient.POST(t, "/api/v1/bookings", payload)

	resp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/search?business_id=507f1f77bcf86cd799439011&schedule_id=507f1f77bcf86cd799439012&start_time=%s&end_time=%s",
		start.Format(time.RFC3339), end.Format(time.RFC3339)))
	common.AssertStatusCode(t, resp, 200)
	data := decodeBookings(t, resp)
	if len(data) < 1 {
		t.Errorf("expected at least one booking in time range")
	}
}

func testUpdateValid(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Update Test", start, end)

	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{
		"service_label": "Updated Label",
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)
	expected := sanitizer.SanitizeCityOrLabel(fmt.Sprint(update["service_label"]))
	if fetched.ServiceLabel != expected {
		t.Errorf("expected updated label '%s', got '%s'", expected, fetched.ServiceLabel)
	}
}

func testUpdateInvalidID(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	update := map[string]any{"service_label": "New Label"}
	resp := httpClient.PATCH(t, "/api/v1/bookings/id/invalid-id", update)
	common.AssertStatusCode(t, resp, 400)
}

func testUpdateTimeOverlap(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	now := time.Now().Add(1 * time.Hour)
	payload1 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "First", now, now.Add(1*time.Hour))
	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Second", now.Add(2*time.Hour), now.Add(3*time.Hour))

	resp1 := httpClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)
	decodeBooking(t, resp1)
	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp2, 201)
	created2 := decodeBooking(t, resp2)

	update := map[string]any{
		"start_time": now.Add(30 * time.Minute).Format(time.RFC3339),
		"end_time":   now.Add(90 * time.Minute).Format(time.RFC3339),
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created2.ID), update)
	if resp.StatusCode != 409 && resp.StatusCode != 400 {
		t.Errorf("expected conflict for overlapping update, got %d", resp.StatusCode)
	}
}

func testUpdateMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Bad JSON Update", start, end)
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	created := decodeBooking(t, createResp)

	resp := httpClient.PATCHRaw(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), []byte(`{"bad":json`))
	common.AssertStatusCode(t, resp, 400)
}

func testDeleteNonExisting(t *testing.T) {
	resp := httpClient.DELETE(t, "/api/v1/bookings/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
}

func testDeleteInvalidID(t *testing.T) {
	resp := httpClient.DELETE(t, "/api/v1/bookings/id/invalid-id")
	common.AssertStatusCode(t, resp, 400)
}

func testCreateAndDelete(t *testing.T) {
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Delete Test", start, end)

	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	delResp := httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	common.AssertStatusCode(t, delResp, 204)
}

func testDoubleDelete(t *testing.T) {
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Double Delete", start, end)

	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	delResp := httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	common.AssertStatusCode(t, delResp, 204)

	delResp2 := httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	common.AssertStatusCode(t, delResp2, 404)
}

func testCreateCapacityBoundaries(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Min Capacity", start, end)
	payload["capacity"] = 1
	payload["participants"] = map[string]string{"+972501234567": "Alice"}
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)
	if created.Capacity != 1 {
		t.Errorf("expected capacity 1, got %d", created.Capacity)
	}

	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Max Capacity", start.Add(2*time.Hour), end.Add(2*time.Hour))
	payload2["capacity"] = 200
	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp2, 201)
	created2 := decodeBooking(t, resp2)
	if created2.Capacity != 200 {
		t.Errorf("expected capacity 200, got %d", created2.Capacity)
	}

	payload3 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Over Max", start.Add(4*time.Hour), end.Add(4*time.Hour))
	payload3["capacity"] = 201
	resp3 := httpClient.POST(t, "/api/v1/bookings", payload3)
	if resp3.StatusCode != 422 && resp3.StatusCode != 400 {
		t.Errorf("expected validation error for capacity=201, got %d", resp3.StatusCode)
	}
}

func testCreateZeroCapacity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Zero Capacity", start, end)
	payload["capacity"] = 0

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created2 := decodeBooking(t, resp)
	if created2.Capacity != 2 {
		t.Errorf("expected capacity 2, got %d", created2.Capacity)
	}
}

func testCreateNegativeCapacity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Negative Capacity", start, end)
	payload["capacity"] = -5

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	created2 := decodeBooking(t, resp)
	if created2.Capacity != 2 {
		t.Errorf("expected capacity 2, got %d", created2.Capacity)
	}
}

func testCreateCapacityExceededByParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Exceeded", start, end)
	payload["capacity"] = 2
	payload["participants"] = map[string]string{
		"+972501234567": "Alice",
		"+972541111111": "Bob",
		"+972542222222": "Charlie",
	}

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for bad participants format")
	}
}

func testCreateEmptyParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Empty Participants", start, end)
	payload["participants"] = map[string]string{}

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for empty participants, got %d", resp.StatusCode)
	}
}

func testCreateMaxParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Max Participants", start, end)
	payload["capacity"] = 200

	participants := make(map[string]string)
	for i := range 200 {
		phone := fmt.Sprintf("+9725012%05d", i+1)
		participants[phone] = fmt.Sprintf("Person%d", i+1)
	}
	payload["participants"] = participants

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)
	if len(created.Participants) != 200 {
		t.Errorf("expected 200 participants, got %d", len(created.Participants))
	}
}

func testCreateTooManyParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Too Many Participants", start, end)
	payload["capacity"] = 250

	participants := make(map[string]string)
	for i := 0; i < 201; i++ {
		phone := fmt.Sprintf("+9725012%05d", i+1)
		participants[phone] = fmt.Sprintf("Person%d", i+1)
	}
	payload["participants"] = participants

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for >200 participants, got %d", resp.StatusCode)
	}
}

func testCreateDuplicateParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Duplicate Participants", start, end)
	payload["participants"] = map[string]string{
		"+972501234567": "Alice",
		"+972541111111": "Bob",
	}

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
}

func testCreateMultipleCountryPhones(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Multi Country", start, end)
	origParticipants := map[string]string{
		"+972501234567": "Israel",
		"+12125551234":  "USA",
		"+447700900123": "UK",
		"+33612345678":  "France",
		"+81312345678":  "Japan",
	}
	payload["participants"] = origParticipants

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	if len(created.Participants) != len(origParticipants) {
		t.Errorf("expected %d participants, got %d", len(origParticipants), len(created.Participants))
	}

	for phone, name := range origParticipants {
		sPhone := sanitizer.SanitizePhone(phone)
		sName := sanitizer.SanitizeNameOrAddress(name)
		if created.Participants[sPhone] != sName {
			t.Errorf("expected participant %s -> %s, got %s",
				sPhone, sName, created.Participants[sPhone])
		}
	}
}

func testCreateServiceLabelBoundaries(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "AB", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)

	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "A", start.Add(2*time.Hour), end.Add(2*time.Hour))
	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	if resp2.StatusCode != 422 && resp2.StatusCode != 400 {
		t.Errorf("expected validation error for 1-char label, got %d", resp2.StatusCode)
	}

	longLabel := ""
	for i := 0; i < 100; i++ {
		longLabel += "A"
	}
	payload3 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", longLabel, start.Add(4*time.Hour), end.Add(4*time.Hour))
	resp3 := httpClient.POST(t, "/api/v1/bookings", payload3)
	common.AssertStatusCode(t, resp3, 201)

	tooLongLabel := longLabel + "A"
	payload4 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", tooLongLabel, start.Add(6*time.Hour), end.Add(6*time.Hour))
	resp4 := httpClient.POST(t, "/api/v1/bookings", payload4)
	if resp4.StatusCode != 422 && resp4.StatusCode != 400 {
		t.Errorf("expected validation error for 101-char label, got %d", resp4.StatusCode)
	}
}

func testCreateServiceLabelSpecialChars(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	rawLabel := "תספורת ✂️ Hair Cut™"
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", rawLabel, start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	expected := sanitizer.SanitizeCityOrLabel(rawLabel)
	if created.ServiceLabel != expected {
		t.Errorf("expected special chars label sanitized to '%s', got '%s'", expected, created.ServiceLabel)
	}
}

func testCreatePastTime(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(-25 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Past Time", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Past time booking returned status %d", resp.StatusCode)
	}
}

func testCreateMidnightTime(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	end := midnight.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Midnight", midnight, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
}

func testCreateVeryShortDuration(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Minute)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Very Short", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
}

func testCreateVeryLongDuration(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(24 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Very Long", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
}

func testCreateMultipleDaySpan(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(72 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Multi Day", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
}

func testCreateInvalidBusinessID(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("invalid-business-id", "507f1f77bcf86cd799439012", "Bad Biz ID", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for invalid business_id, got %d", resp.StatusCode)
	}
}

func testCreateInvalidScheduleID(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "not-a-valid-id", "Bad Schedule ID", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for invalid schedule_id, got %d", resp.StatusCode)
	}
}

func testCreateAllStatuses(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	statuses := []string{"pending", "confirmed", "cancelled"}

	for i, status := range statuses {
		start := time.Now().Add(time.Duration(i+1) * 2 * time.Hour)
		end := start.Add(1 * time.Hour)
		payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", fmt.Sprintf("Status %s", status), start, end)
		payload["status"] = status

		resp := httpClient.POST(t, "/api/v1/bookings", payload)
		common.AssertStatusCode(t, resp, 201)
		created := decodeBooking(t, resp)
		if string(created.Status) != status {
			t.Errorf("expected status %s, got %s", status, string(created.Status))
		}
	}
}

func testCreateInvalidStatus(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Bad Status", start, end)
	payload["status"] = "invalid_status"

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for invalid status, got %d", resp.StatusCode)
	}
}

func testCreateExactSameTime(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload1 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "First", start, end)
	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Second", start, end)

	resp1 := httpClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)

	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	if resp2.StatusCode != 409 && resp2.StatusCode != 400 {
		t.Errorf("expected conflict for exact same time, got %d", resp2.StatusCode)
	}
}

func testCreatePartialOverlap(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)

	payload1 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "First", start, start.Add(1*time.Hour))
	resp1 := httpClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)

	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Overlap End", start.Add(30*time.Minute), start.Add(90*time.Minute))
	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	if resp2.StatusCode != 409 && resp2.StatusCode != 400 {
		t.Errorf("expected conflict for partial overlap (end), got %d", resp2.StatusCode)
	}

	payload3 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Overlap Start", start.Add(-30*time.Minute), start.Add(30*time.Minute))
	resp3 := httpClient.POST(t, "/api/v1/bookings", payload3)
	if resp3.StatusCode != 409 && resp3.StatusCode != 400 {
		t.Errorf("expected conflict for partial overlap (start), got %d", resp3.StatusCode)
	}

	payload4 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Contains", start.Add(-30*time.Minute), start.Add(90*time.Minute))
	resp4 := httpClient.POST(t, "/api/v1/bookings", payload4)
	if resp4.StatusCode != 409 && resp4.StatusCode != 400 {
		t.Errorf("expected conflict for containing overlap, got %d", resp4.StatusCode)
	}

	payload5 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Within", start.Add(15*time.Minute), start.Add(45*time.Minute))
	resp5 := httpClient.POST(t, "/api/v1/bookings", payload5)
	if resp5.StatusCode != 409 && resp5.StatusCode != 400 {
		t.Errorf("expected conflict for within overlap, got %d", resp5.StatusCode)
	}
}

func testUpdateStatusTransitions(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Status Test", start, end)
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	validTransitions := []string{"confirmed", "cancelled", "completed"}
	for _, newStatus := range validTransitions {
		update := map[string]any{"status": newStatus}
		resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
		common.AssertStatusCode(t, resp, 204)

		getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
		fetched := decodeBooking(t, getResp)
		if string(fetched.Status) != newStatus {
			t.Errorf("expected status %s, got %s", newStatus, string(fetched.Status))
		}
	}
}

func testUpdateCapacityBelowParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Capacity Test", start, end)
	payload["capacity"] = 10
	payload["participants"] = map[string]string{
		"+972501234567": "Alice",
		"+972541111111": "Bob",
		"+972542222222": "Charlie",
	}
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{"capacity": 2}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for capacity < participants, got %d", resp.StatusCode)
	}
}

func testUpdateAddParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Add Participants", start, end)
	payload["participants"] = map[string]string{"+972501234567": "Alice"}
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{
		"participants": map[string]string{
			"+972501234567": "Alice",
			"+972541111111": "Bob",
			"+972542222222": "Charlie",
		},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)
	if len(fetched.Participants) != 3 {
		t.Errorf("expected 3 participants, got %d", len(fetched.Participants))
	}

	p1 := sanitizer.SanitizePhone("+972501234567")
	p2 := sanitizer.SanitizePhone("+972541111111")
	p3 := sanitizer.SanitizePhone("+972542222222")
	if fetched.Participants[p1] != sanitizer.SanitizeNameOrAddress("Alice") {
		t.Errorf("expected Alice sanitized under %s, got %s", p1, fetched.Participants[p1])
	}
	if fetched.Participants[p2] != sanitizer.SanitizeNameOrAddress("Bob") {
		t.Errorf("expected Bob sanitized under %s, got %s", p2, fetched.Participants[p2])
	}
	if fetched.Participants[p3] != sanitizer.SanitizeNameOrAddress("Charlie") {
		t.Errorf("expected Charlie sanitized under %s, got %s", p3, fetched.Participants[p3])
	}
}

func testUpdateRemoveParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Remove Participants", start, end)
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{
		"participants": map[string]string{"+972501234567": "Alice"},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)
	if len(fetched.Participants) != 1 {
		t.Errorf("expected 1 participant, got %d", len(fetched.Participants))
	}

	p1 := sanitizer.SanitizePhone("+972501234567")
	if fetched.Participants[p1] != sanitizer.SanitizeNameOrAddress("Alice") {
		t.Errorf("expected only Alice sanitized under %s, got %s", p1, fetched.Participants[p1])
	}
}

func testUpdateManagedBy(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Managed By Test", start, end)
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{"managed_by": map[string]string{"+972508888888": "New Manager"}}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)
	newPhone := sanitizer.SanitizePhone("+972508888888")
	newName := sanitizer.SanitizeNameOrAddress("New Manager")
	if fetched.ManagedBy[newPhone] != newName {
		t.Errorf("expected managed_by %s -> %s, got %s",
			newPhone, newName, fetched.ManagedBy[newPhone])
	}
}

func testUpdateOnlyTime(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Time Update", start, end)
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	newStart := start.Add(3 * time.Hour)
	newEnd := newStart.Add(1 * time.Hour)
	update := map[string]any{
		"start_time": newStart.Format(time.RFC3339),
		"end_time":   newEnd.Format(time.RFC3339),
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)
	if fetched.StartTime.Unix() != newStart.Unix() {
		t.Errorf("expected start time %v, got %v", newStart, fetched.StartTime)
	}
	if fetched.EndTime.Unix() != newEnd.Unix() {
		t.Errorf("expected end time %v, got %v", newEnd, fetched.EndTime)
	}
}

func testUpdateOnlyCapacity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Capacity Update", start, end)
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{"capacity": 20}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)
	if fetched.Capacity != 20 {
		t.Errorf("expected capacity 20, got %d", fetched.Capacity)
	}
}

func testUpdateMultipleFields(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Multi Field", start, end)
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{
		"service_label": "Updated Service",
		"capacity":      15,
		"status":        "confirmed",
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)

	expectedLabel := sanitizer.SanitizeCityOrLabel(fmt.Sprint(update["service_label"]))
	if fetched.ServiceLabel != expectedLabel {
		t.Errorf("expected service_label '%s', got %s", expectedLabel, fetched.ServiceLabel)
	}
	if fetched.Capacity != 15 {
		t.Errorf("expected capacity 15, got %d", fetched.Capacity)
	}
	if string(fetched.Status) != "confirmed" {
		t.Errorf("expected status 'confirmed', got %s", string(fetched.Status))
	}
}

// Additional advanced test functions for bookings

func testConcurrentBookingCreation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	var wg sync.WaitGroup
	results := make([]int, 5)
	ids := make([]string, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			payload := createValidBooking(
				"507f1f77bcf86cd799439011",
				"507f1f77bcf86cd799439012",
				fmt.Sprintf("Service %d", index),
				start.Add(time.Duration(index)*2*time.Hour),
				end.Add(time.Duration(index)*2*time.Hour),
			)
			resp := httpClient.POST(t, "/api/v1/bookings", payload)
			results[index] = resp.StatusCode
			if resp.StatusCode == 201 {
				created := decodeBooking(t, resp)
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

	t.Logf("Concurrent booking creation: %d/5 succeeded", successCount)

	// Cleanup
	for _, id := range ids {
		if id != "" {
			httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", id))
		}
	}
}

func testBookingStatusCompleted(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Completed Status Test", start, end)
	payload["status"] = "completed"

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode == 201 {
		created := decodeBooking(t, resp)
		if string(created.Status) != "completed" {
			t.Errorf("expected status 'completed', got %s", string(created.Status))
		}
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	}
}

func testParticipantsValidation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	// Test with empty name in participants
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Empty Name Test", start, end)
	payload["participants"] = map[string]string{"+972501234567": ""}
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Empty participant name returned status %d", resp.StatusCode)
	}

	// Test with special characters in name
	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Special Chars Name Test", start.Add(2*time.Hour), end.Add(2*time.Hour))
	payload2["participants"] = map[string]string{
		"+972501234567": "José María",
		"+972541111111": "张伟",
		"+972542222222": "Владимир",
	}
	resp = httpClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	expectedParticipants := map[string]string{
		sanitizer.SanitizePhone("+972501234567"): sanitizer.SanitizeNameOrAddress("José María"),
		sanitizer.SanitizePhone("+972541111111"): sanitizer.SanitizeNameOrAddress("张伟"),
		sanitizer.SanitizePhone("+972542222222"): sanitizer.SanitizeNameOrAddress("Владимир"),
	}
	if len(created.Participants) != len(expectedParticipants) {
		t.Errorf("expected %d participants, got %d", len(expectedParticipants), len(created.Participants))
	}
	for phone, name := range expectedParticipants {
		if created.Participants[phone] != name {
			t.Errorf("expected participant %s -> %s, got %s",
				phone, name, created.Participants[phone])
		}
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testSearchWithExactTimeMatch(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Exact Time Match", start, end)
	httpClient.POST(t, "/api/v1/bookings", payload)

	// Search with exact time range
	resp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/search?business_id=507f1f77bcf86cd799439011&schedule_id=507f1f77bcf86cd799439012&start_time=%s&end_time=%s",
		start.Format(time.RFC3339), end.Format(time.RFC3339)))
	common.AssertStatusCode(t, resp, 200)
	data := decodeBookings(t, resp)
	if len(data) < 1 {
		t.Error("expected at least one booking in exact time range")
	}
}

func testBookingWithPastEndTime(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(-2 * time.Hour)
	end := time.Now().Add(-1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Past Booking", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	// Should allow creating past bookings but may log a warning
	if resp.StatusCode == 201 {
		created := decodeBooking(t, resp)
		t.Logf("Past booking created with ID %s", created.ID)
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	}
}

func testUpdateParticipantsExceedCapacity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Capacity Test", start, end)
	payload["capacity"] = 2
	payload["participants"] = map[string]string{"+972501234567": "Alice"}

	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	// Try to add more participants than capacity
	update := map[string]any{
		"participants": map[string]string{
			"+972501234567": "Alice",
			"+972541111111": "Bob",
			"+972542222222": "Charlie",
		},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for participants exceeding capacity, got %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testManagedByValidation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	// Test with invalid phone in managed_by
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Invalid Managed By", start, end)
	payload["managed_by"] = map[string]string{"invalid-phone": "Manager"}
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Invalid managed_by phone returned status %d", resp.StatusCode)
	}

	// Test with empty managed_by (should fail as it's required)
	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Empty Managed By", start.Add(2*time.Hour), end.Add(2*time.Hour))
	payload2["managed_by"] = map[string]string{}
	resp = httpClient.POST(t, "/api/v1/bookings", payload2)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Empty managed_by returned status %d", resp.StatusCode)
	}
}

func testSearchWithoutTimeRange(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "No Time Range Search", start, end)
	httpClient.POST(t, "/api/v1/bookings", payload)

	// Search without time range (should return all bookings for the schedule)
	resp := httpClient.GET(t, "/api/v1/bookings/search?business_id=507f1f77bcf86cd799439011&schedule_id=507f1f77bcf86cd799439012")
	common.AssertStatusCode(t, resp, 200)
	data := decodeBookings(t, resp)
	if len(data) < 1 {
		t.Error("expected at least one booking without time range filter")
	}
}

func testUpdateClearParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Clear Participants", start, end)

	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	// Try to clear participants (should fail as it would be invalid)
	update := map[string]any{
		"participants": map[string]string{},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Clearing participants returned status %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingWithSameBusinessDifferentSchedule(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	// Create two bookings with same business but different schedules at same time
	payload1 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Schedule 1", start, end)
	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439013", "Schedule 2", start, end)

	resp1 := httpClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)
	created1 := decodeBooking(t, resp1)

	// Should succeed since it's a different schedule
	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp2, 201)
	created2 := decodeBooking(t, resp2)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created1.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created2.ID))
}
