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
	testServiceLabelNormalization(t)
	testServiceLabelEmptyString(t)
	testBookingWithMaxCapacity(t)
	testBookingWithSingleParticipant(t)
	testMultipleBookingsSameTimeSlot(t)
	testBookingAcrossTimeZones(t)
	testBulkBookingCreation(t)
	testPaginationWithLargeDataset(t)
	testSearchWithMultipleFilters(t)
	testSearchByParticipantPhone(t)
	testBookingStatusWorkflow(t)
	testUpdateBookingToConflictingTime(t)
	testUpdateBookingWithInvalidStatus(t)
	testBookingWithVeryLongServiceLabel(t)
	testBookingWithSpecialCharsInServiceLabel(t)
	testParticipantsWithInternationalPhones(t)
	testManagedByMultipleManagers(t)
	testManagedByEmptyMap(t)
	testConcurrentUpdatesToSameBooking(t)
	testBookingAtMidnight(t)
	testBookingSpanningMultipleDays(t)
	testSearchBookingsByDateRange(t)
	testSearchBookingsByBusinessAndSchedule(t)
	testUpdateCapacityToZero(t)
	testUpdateCapacityToMaximum(t)
	testBookingWithNegativeTimeDuration(t)
	testDeleteBookingAndRecreate(t)
	testGetBookingCreatedAt(t)
	testSearchPaginationEdgeCases(t)
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
			"Alice": "+972501234567",
			"Bob":   "+972541111111",
		},
		"status":     "pending",
		"managed_by": map[string]string{"Manager": "+972509999999"},
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
	p1Phone := "+972501234567"
	p2Phone := "+972541111111"
	p1Name := sanitizer.SanitizeNameOrAddress("Alice")
	p2Name := sanitizer.SanitizeNameOrAddress("Bob")
	if created.Participants[p1Name] != p1Phone {
		t.Errorf("expected participant %s -> %s, got %s",
			p1Phone, p1Name, created.Participants[p1Name])
	}
	if created.Participants[p2Name] != p2Phone {
		t.Errorf("expected participant %s -> %s, got %s",
			p2Phone, p2Name, created.Participants[p2Name])
	}

	if len(created.ManagedBy) != 1 {
		t.Errorf("expected 1 managed_by entry, got %d", len(created.ManagedBy))
	}
	mPhone := "+972509999999"
	mName := sanitizer.SanitizeNameOrAddress("Manager")
	if created.ManagedBy[mName] != mPhone {
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
	payload["participants"] = map[string]string{"Invalid": "notaphone"}

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
	payload["participants"] = map[string]string{"Alice": "+972501234567"}
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
		"Alice":   "+972501234567",
		"Bob":     "+972541111111",
		"Charlie": "+972542222222",
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
	common.AssertStatusCode(t, resp, 201)
}

func testCreateMaxParticipants(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Max Participants", start, end)
	payload["capacity"] = 200

	participants := make(map[string]string)
	for i := range 200 {
		name := fmt.Sprintf("Person%d", i+1)
		phone := fmt.Sprintf("+9725012%05d", i+1)
		participants[name] = phone
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
		name := fmt.Sprintf("Person%d", i+1)
		phone := fmt.Sprintf("+9725012%05d", i+1)
		participants[name] = phone
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

	payload := createValidBooking(
		"507f1f77bcf86cd799439011",
		"507f1f77bcf86cd799439012",
		"Duplicate Participants",
		start, end,
	)

	payload["participants"] = map[string]string{
		"Alice": "+972501234567",
		"Bob":   "+972541111111",
	}

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
}

func testCreateMultipleCountryPhones(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking(
		"507f1f77bcf86cd799439011",
		"507f1f77bcf86cd799439012",
		"Multi Country",
		start, end,
	)

	origParticipants := map[string]string{
		"Israel": "+972501234567",
		"USA":    "+12125551234",
		"UK":     "+447700900123",
		"France": "+33612345678",
		"Japan":  "+81312345678",
	}

	payload["participants"] = origParticipants

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	if len(created.Participants) != len(origParticipants) {
		t.Errorf("expected %d participants, got %d", len(origParticipants), len(created.Participants))
	}

	for name, phone := range origParticipants {
		sName := sanitizer.SanitizeNameOrAddress(name)
		if created.Participants[sName] != phone {
			t.Errorf("expected participant %s -> %s, got %s",
				sName, phone, created.Participants[sName])
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

	validTransitions := []string{"confirmed", "cancelled"}
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
		"Alice":   "+972501234567",
		"Bob":     "+972541111111",
		"Charlie": "+972542222222",
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
	payload["participants"] = map[string]string{"Alice": "+972501234567"}
	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{
		"participants": map[string]string{
			"Alice":   "+972501234567",
			"Bob":     "+972541111111",
			"Charlie": "+972542222222",
		},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)
	if len(fetched.Participants) != 3 {
		t.Errorf("expected 3 participants, got %d", len(fetched.Participants))
	}

	if fetched.Participants["alice"] != "+972501234567" {
		t.Errorf("expected Alice sanitized under key 'alice'")
	}
	if fetched.Participants["bob"] != "+972541111111" {
		t.Errorf("expected Bob sanitized under key 'bob'")
	}
	if fetched.Participants["charlie"] != "+972542222222" {
		t.Errorf("expected Charlie sanitized under key 'charlie'")
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
		"participants": map[string]string{"Alice": "+972501234567"},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)
	if len(fetched.Participants) != 1 {
		t.Errorf("expected 1 participant, got %d", len(fetched.Participants))
	}

	if fetched.Participants["alice"] != "+972501234567" {
		t.Errorf("expected only Alice sanitized")
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

	manager_name := "New Manager"
	manager_phone := "+972508888888"
	update := map[string]any{"managed_by": map[string]string{manager_name: manager_phone}}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)

	if fetched.ManagedBy[sanitizer.SanitizeNameOrAddress(manager_name)] != manager_phone {
		t.Errorf("expected managed_by to be updated")
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
func testConcurrentBookingCreation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	var wg sync.WaitGroup
	results := make([]int, 5)
	ids := make([]string, 5)

	conc := 5
	for i := 0; i < conc; i++ {
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

	if successCount != conc {
		t.Errorf("Concurrent booking creation: %d/5 succeeded", successCount)
	}

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

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Empty Name Test", start, end)
	payload["participants"] = map[string]string{"": "+972501234567"}
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Empty participant name returned status %d", resp.StatusCode)
	}

	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Special Chars Name Test", start.Add(2*time.Hour), end.Add(2*time.Hour))
	payload2["participants"] = map[string]string{
		"José María": "+972501234567",
		"张伟":         "+972541111111",
		"Владимир":   "+972542222222",
	}
	resp = httpClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	expectedParticipants := map[string]string{
		sanitizer.SanitizeNameOrAddress("José María"): "+972501234567",
		sanitizer.SanitizeNameOrAddress("张伟"):         "+972541111111",
		sanitizer.SanitizeNameOrAddress("Владимир"):   "+972542222222",
	}
	if len(created.Participants) != len(expectedParticipants) {
		t.Errorf("expected %d participants, got %d", len(expectedParticipants), len(created.Participants))
	}
	for name, phone := range expectedParticipants {
		if created.Participants[name] != phone {
			t.Errorf("expected participant %s -> %s, got %s",
				name, phone, created.Participants[name])
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

	start := time.Now().Add(-12 * time.Hour)
	end := time.Now().Add(-1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Past Booking", start, end)

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode == 201 {
		created := decodeBooking(t, resp)
		t.Errorf("Past booking created with ID %s", created.ID)
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	}
}

func testUpdateParticipantsExceedCapacity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Capacity Test", start, end)
	payload["capacity"] = 2
	payload["participants"] = map[string]string{"Alice": "+972501234567"}

	createResp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{
		"participants": map[string]string{
			"Alice":   "+972501234567",
			"Bob":     "+972541111111",
			"Charlie": "+972542222222",
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

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Invalid Managed By", start, end)
	payload["managed_by"] = map[string]string{"Manager": "invalid-phone"}
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Invalid managed_by phone returned status %d", resp.StatusCode)
	}

	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Empty Managed By", start.Add(2*time.Hour), end.Add(2*time.Hour))
	payload2["managed_by"] = map[string]string{}
	resp = httpClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp, 201)
}

func testSearchWithoutTimeRange(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "No Time Range Search", start, end)
	httpClient.POST(t, "/api/v1/bookings", payload)

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

	update := map[string]any{
		"participants": map[string]string{},
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingWithSameBusinessDifferentSchedule(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload1 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Schedule 1", start, end)
	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439013", "Schedule 2", start, end)

	resp1 := httpClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)
	created1 := decodeBooking(t, resp1)

	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp2, 201)
	created2 := decodeBooking(t, resp2)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created1.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created2.ID))
}

// ========== ENRICHED TESTS ==========

func testServiceLabelNormalization(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	// Test with mixed case and special characters
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Hair-Cut & Styling™", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	if created.ServiceLabel != sanitizer.SanitizeCityOrLabel("Hair-Cut & Styling™") {
		t.Errorf("expected normalized service label, got %s", created.ServiceLabel)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testServiceLabelEmptyString(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)

	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for empty service label, got %d", resp.StatusCode)
	}
}

func testBookingWithMaxCapacity(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Large Event", start, end)
	payload["capacity"] = 100
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	if created.Capacity != 100 {
		t.Errorf("expected capacity 100, got %d", created.Capacity)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingWithSingleParticipant(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Solo Session", start, end)
	payload["participants"] = map[string]string{"Alice": "+972501234567"}
	payload["capacity"] = 1

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	if len(created.Participants) != 1 {
		t.Errorf("expected 1 participant, got %d", len(created.Participants))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testMultipleBookingsSameTimeSlot(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	// Create booking for schedule 1
	payload1 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Service A", start, end)
	resp1 := httpClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)
	created1 := decodeBooking(t, resp1)

	// Create booking for schedule 2 (same business, different schedule, same time)
	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439013", "Service B", start, end)
	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp2, 201)
	created2 := decodeBooking(t, resp2)

	// Both should succeed as they're on different schedules
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created1.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created2.ID))
}

func testBookingAcrossTimeZones(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create booking with UTC time
	startUTC := time.Now().UTC().Add(2 * time.Hour)
	endUTC := startUTC.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Cross TZ Meeting", startUTC, endUTC)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	// Verify times are stored correctly
	if created.StartTime.IsZero() || created.EndTime.IsZero() {
		t.Error("expected valid start and end times")
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBulkBookingCreation(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create 20 bookings
	createdIDs := []string{}
	for i := 0; i < 20; i++ {
		start := time.Now().Add(time.Duration(i+1) * time.Hour)
		end := start.Add(30 * time.Minute)

		payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", fmt.Sprintf("Bulk Booking %d", i), start, end)
		resp := httpClient.POST(t, "/api/v1/bookings", payload)
		common.AssertStatusCode(t, resp, 201)
		created := decodeBooking(t, resp)
		createdIDs = append(createdIDs, created.ID)
	}

	// Verify all were created
	resp := httpClient.GET(t, "/api/v1/bookings?limit=25&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, total, _, _ := decodeBookingsPaginated(t, resp)

	if total < 20 {
		t.Errorf("expected at least 20 bookings, got %d", total)
	}
	if len(data) < 20 {
		t.Errorf("expected at least 20 bookings in data, got %d", len(data))
	}

	// Cleanup
	for _, id := range createdIDs {
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", id))
	}
}

func testPaginationWithLargeDataset(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create 50 bookings
	for i := 0; i < 50; i++ {
		start := time.Now().Add(time.Duration(i+1) * time.Hour)
		end := start.Add(30 * time.Minute)

		payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", fmt.Sprintf("Pagination Test %d", i), start, end)
		httpClient.POST(t, "/api/v1/bookings", payload)
	}

	// Test pagination
	resp := httpClient.GET(t, "/api/v1/bookings?limit=10&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, total, limit, offset := decodeBookingsPaginated(t, resp)

	if total < 50 {
		t.Errorf("expected at least 50 total, got %d", total)
	}
	if len(data) != 10 {
		t.Errorf("expected 10 items on page, got %d", len(data))
	}
	if limit != 10 || offset != 0 {
		t.Errorf("expected limit=10 offset=0, got limit=%d offset=%d", limit, offset)
	}

	// Test second page
	resp = httpClient.GET(t, "/api/v1/bookings?limit=10&offset=10")
	common.AssertStatusCode(t, resp, 200)
	data, _, limit, offset = decodeBookingsPaginated(t, resp)

	if len(data) != 10 {
		t.Errorf("expected 10 items on second page, got %d", len(data))
	}
	if limit != 10 || offset != 10 {
		t.Errorf("expected limit=10 offset=10, got limit=%d offset=%d", limit, offset)
	}
}

func testSearchWithMultipleFilters(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Multi Filter Test", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	// Search with business_id, schedule_id, and time range
	searchURL := fmt.Sprintf("/api/v1/bookings/search?business_id=%s&schedule_id=%s&start_time=%s&end_time=%s",
		"507f1f77bcf86cd799439011",
		"507f1f77bcf86cd799439012",
		start.Add(-10*time.Minute).Format(time.RFC3339),
		end.Add(10*time.Minute).Format(time.RFC3339))

	searchResp := httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, searchResp, 200)
	results := decodeBookings(t, searchResp)

	if len(results) < 1 {
		t.Error("expected at least 1 booking in search results")
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testSearchByParticipantPhone(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	testPhone := "+972501234567"
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Participant Search", start, end)
	payload["participants"] = map[string]string{"Alice": testPhone}

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	// Search by participant phone (if supported by API)
	searchURL := fmt.Sprintf("/api/v1/bookings/search?participant_phone=%s", testPhone)
	searchResp := httpClient.GET(t, searchURL)

	// API may or may not support this - just test it doesn't crash
	if searchResp.StatusCode != 200 && searchResp.StatusCode != 400 {
		t.Logf("Participant phone search returned status %d", searchResp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingStatusWorkflow(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	// Create as pending
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Status Workflow", start, end)
	payload["status"] = "pending"
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	// Transition to confirmed
	update := map[string]any{"status": "confirmed"}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	// Verify status changed
	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	updated := decodeBooking(t, getResp)

	if updated.Status != "confirmed" {
		t.Errorf("expected status 'confirmed', got %s", updated.Status)
	}

	// Transition to cancelled
	update = map[string]any{"status": "cancelled"}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testUpdateBookingToConflictingTime(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start1 := time.Now().Add(1 * time.Hour)
	end1 := start1.Add(1 * time.Hour)

	start2 := start1.Add(2 * time.Hour)
	end2 := start2.Add(1 * time.Hour)

	// Create first booking
	payload1 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Booking 1", start1, end1)
	resp1 := httpClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)
	created1 := decodeBooking(t, resp1)

	// Create second booking
	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Booking 2", start2, end2)
	resp2 := httpClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp2, 201)
	created2 := decodeBooking(t, resp2)

	// Try to update booking2 to overlap with booking1
	update := map[string]any{
		"start_time": start1.Format(time.RFC3339),
		"end_time":   end1.Format(time.RFC3339),
	}
	resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created2.ID), update)

	if resp.StatusCode != 409 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Expected conflict error, got %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created1.ID))
	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created2.ID))
}

func testUpdateBookingWithInvalidStatus(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Invalid Status", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	// Try to update with invalid status
	update := map[string]any{"status": "invalid_status"}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)

	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for invalid status, got %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingWithVeryLongServiceLabel(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	longLabel := ""
	for i := 0; i < 150; i++ {
		longLabel += "A"
	}

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", longLabel, start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)

	// Should either accept with truncation or reject
	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Long service label returned unexpected status %d", resp.StatusCode)
	}
}

func testBookingWithSpecialCharsInServiceLabel(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	specialLabels := []string{
		"עברית",         // Hebrew
		"中文",            // Chinese
		"Café & Spa™",   // Special chars
		"Test@#$%",      // Symbols
		"Multi  Spaces", // Multiple spaces
		"Tab\tChar",     // Tab character
		"New\nLine",     // Newline
	}

	for _, label := range specialLabels {
		payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", label, start.Add(time.Minute), end.Add(time.Minute))
		resp := httpClient.POST(t, "/api/v1/bookings", payload)

		if resp.StatusCode == 201 {
			created := decodeBooking(t, resp)
			httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
		}
	}
}

func testParticipantsWithInternationalPhones(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "International", start, end)
	payload["participants"] = map[string]string{
		"US":     "+12125551234",
		"Canada": "+14165551234",
		"Israel": "+972501234567",
	}

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	if len(created.Participants) != 3 {
		t.Errorf("expected 3 participants, got %d", len(created.Participants))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testManagedByMultipleManagers(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Multi Manager", start, end)
	payload["managed_by"] = map[string]string{
		"Manager1": "+972501111111",
		"Manager2": "+972502222222",
		"Manager3": "+972503333333",
	}

	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	if len(created.ManagedBy) != 3 {
		t.Errorf("expected 3 managers, got %d", len(created.ManagedBy))
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testManagedByEmptyMap(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Empty Manager", start, end)
	payload["managed_by"] = map[string]string{}

	resp := httpClient.POST(t, "/api/v1/bookings", payload)

	// Should either accept empty map or require at least one manager
	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Empty managed_by returned status %d", resp.StatusCode)
	}
}

func testConcurrentUpdatesToSameBooking(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Concurrent Update", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	var wg sync.WaitGroup
	results := make([]int, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			update := map[string]any{
				"service_label": fmt.Sprintf("Updated Label %d", index),
			}
			resp := httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
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

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingAtMidnight(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	end := midnight.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Midnight Session", midnight, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingSpanningMultipleDays(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(24 * time.Hour)
	end := start.Add(36 * time.Hour) // Spans into next day

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Multi-Day Event", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)

	// Should either accept or reject based on business rules
	if resp.StatusCode != 201 && resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Logf("Multi-day booking returned status %d", resp.StatusCode)
	}
}

func testSearchBookingsByDateRange(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	// Create bookings across different days
	for i := 0; i < 5; i++ {
		start := time.Now().Add(time.Duration(24*(i+1)) * time.Hour)
		end := start.Add(1 * time.Hour)

		payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", fmt.Sprintf("Day %d", i), start, end)
		httpClient.POST(t, "/api/v1/bookings", payload)
	}

	// Search for bookings in specific date range
	searchStart := time.Now().Add(24 * time.Hour)
	searchEnd := searchStart.Add(72 * time.Hour)

	searchURL := fmt.Sprintf("/api/v1/bookings/search?business_id=%s&schedule_id=%s&start_time=%s&end_time=%s",
		"507f1f77bcf86cd799439011",
		"507f1f77bcf86cd799439012",
		searchStart.Format(time.RFC3339),
		searchEnd.Format(time.RFC3339))

	resp := httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, resp, 200)
	results := decodeBookings(t, resp)

	if len(results) < 3 {
		t.Errorf("expected at least 3 bookings in date range, got %d", len(results))
	}
}

func testSearchBookingsByBusinessAndSchedule(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	// Create multiple bookings
	payload1 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Service 1", start, end)
	httpClient.POST(t, "/api/v1/bookings", payload1)

	payload2 := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439013", "Service 2", start.Add(time.Hour), end.Add(time.Hour))
	httpClient.POST(t, "/api/v1/bookings", payload2)

	// Search by business_id only
	resp := httpClient.GET(t, "/api/v1/bookings/search?business_id=507f1f77bcf86cd799439011")
	if resp.StatusCode == 200 {
		results := decodeBookings(t, resp)
		if len(results) < 2 {
			t.Errorf("expected at least 2 bookings for business, got %d", len(results))
		}
	}

	// Search by business_id and schedule_id
	resp = httpClient.GET(t, "/api/v1/bookings/search?business_id=507f1f77bcf86cd799439011&schedule_id=507f1f77bcf86cd799439012")
	if resp.StatusCode == 200 {
		results := decodeBookings(t, resp)
		if len(results) < 1 {
			t.Errorf("expected at least 1 booking for specific schedule, got %d", len(results))
		}
	}
}

func testUpdateCapacityToZero(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Capacity Zero", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	// Try to update capacity to zero
	update := map[string]any{"capacity": 0}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)

	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for zero capacity, got %d", resp.StatusCode)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testUpdateCapacityToMaximum(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Max Capacity", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	// Update to maximum capacity
	update := map[string]any{"capacity": 1000}
	resp = httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)

	// Should either accept or cap at max value
	if resp.StatusCode == 204 {
		getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
		updated := decodeBooking(t, getResp)
		t.Logf("Capacity updated to %d", updated.Capacity)
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingWithNegativeTimeDuration(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(-30 * time.Minute) // End before start

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Negative Duration", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)

	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected validation error for negative duration, got %d", resp.StatusCode)
	}
}

func testDeleteBookingAndRecreate(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	// Create booking
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Delete and Recreate", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)
	firstID := created.ID

	// Delete it
	resp = httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", firstID))
	common.AssertStatusCode(t, resp, 204)

	// Recreate same booking
	resp = httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	recreated := decodeBooking(t, resp)

	if recreated.ID == firstID {
		t.Error("expected different ID for recreated booking")
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", recreated.ID))
}

func testGetBookingCreatedAt(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)

	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "CreatedAt Test", start, end)
	resp := httpClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)

	if created.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}

	originalCreatedAt := created.CreatedAt

	// Update booking
	update := map[string]any{"service_label": "Updated Label"}
	httpClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)

	// Verify created_at didn't change
	getResp := httpClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	updated := decodeBooking(t, getResp)

	if !updated.CreatedAt.Equal(originalCreatedAt) {
		t.Error("created_at should not change on update")
	}

	httpClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testSearchPaginationEdgeCases(t *testing.T) {
	defer common.ClearTestData(t, httpClient, TableName)

	searchURL := "/api/v1/bookings/search?business_id=507f1f77bcf86cd799439011&schedule_id=507f1f77bcf86cd799439012&limit=0&offset=0"
	resp := httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, resp, 200)

	searchURL = "/api/v1/bookings/search?business_id=507f1f77bcf86cd799439011&schedule_id=507f1f77bcf86cd799439012&limit=10000&offset=0"
	resp = httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, resp, 200)

	searchURL = "/api/v1/bookings/search?business_id=507f1f77bcf86cd799439011&schedule_id=507f1f77bcf86cd799439012&limit=10&offset=999999"
	resp = httpClient.GET(t, searchURL)
	common.AssertStatusCode(t, resp, 200)
}
