package integrationtests

import (
	"fmt"
	"os"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/test/common"
	"testing"
	"time"
)

const (
	BookingsServiceName = "bookings-integration-tests"
	BookingsTableName   = "bookings"
)

var (
	bookingsCfg        *config.Config
	bookingsHTTPClient *common.Client
)

func TestMain(t *testing.T) {
	setupBookings()
	testBookingsGet(t)
	testBookingsPost(t)
	testBookingsUpdate(t)
	testBookingsDelete(t)
	teardownBookings()
}

// --- Setup / Teardown ---

func setupBookings() {
	bookingsCfg = config.Load(BookingsServiceName)

	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	bookingsHTTPClient = common.NewClient(serverURL)
}

func teardownBookings() {
	bookingsCfg.GracefulShutdown()
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

// --- Tests ---

func testBookingsGet(t *testing.T) {
	testBookingGetEmptyTable(t)
	testBookingGetAllPaginatedEmpty(t)
	testBookingCreateAndGetByID(t)
	testBookingGetInvalidID(t)
	testBookingSearchEmpty(t)
	testBookingSearchRange(t)
}

func testBookingsPost(t *testing.T) {
	testBookingCreateValid(t)
	testBookingCreateInvalidTimeRange(t)
	testBookingCreateOverlapConflict(t)
	testBookingCreateInvalidParticipantFormat(t)
	testBookingCreateMalformedJSON(t)
}

func testBookingsUpdate(t *testing.T) {
	testBookingUpdateValid(t)
	testBookingUpdateInvalidID(t)
	testBookingUpdateTimeOverlap(t)
	testBookingUpdateMalformedJSON(t)
}

func testBookingsDelete(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	testBookingDeleteNonExisting(t)
	testBookingDeleteInvalidID(t)
	testBookingCreateAndDelete(t)
	testBookingDoubleDelete(t)
}

// --- Individual Test Cases ---

func testBookingGetEmptyTable(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	resp := bookingsHTTPClient.GET(t, "/api/v1/bookings?id=507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
}

func testBookingGetAllPaginatedEmpty(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	resp := bookingsHTTPClient.GET(t, "/api/v1/bookings?limit=10&offset=0")
	common.AssertStatusCode(t, resp, 200)
	data, total, _, _ := decodeBookingsPaginated(t, resp)
	if total != 0 || len(data) != 0 {
		t.Errorf("expected empty table, got total=%d len=%d", total, len(data))
	}
}

func testBookingCreateValid(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Haircut", start, end)

	resp := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 201)
	created := decodeBooking(t, resp)
	if created.ID == "" {
		t.Error("expected booking ID to be set")
	}
	if created.ServiceLabel != "Haircut" {
		t.Errorf("expected service_label 'Haircut', got %s", created.ServiceLabel)
	}
	bookingsHTTPClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingCreateInvalidTimeRange(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	start := time.Now().Add(2 * time.Hour)
	end := start.Add(-1 * time.Hour)
	payload := createValidBooking("507f1f77bcf86cd799439011", "507f1f77bcf86cd799439012", "Invalid Time", start, end)

	resp := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, resp, 422)
	common.AssertContains(t, resp, "end_time")
}

func testBookingCreateOverlapConflict(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	now := time.Now().Add(1 * time.Hour)
	payload1 := createValidBooking("b1", "s1", "Haircut", now, now.Add(1*time.Hour))
	payload2 := createValidBooking("b1", "s1", "Haircut 2", now.Add(30*time.Minute), now.Add(2*time.Hour))

	resp1 := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)
	resp2 := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload2)
	if resp2.StatusCode != 409 && resp2.StatusCode != 400 {
		t.Errorf("expected conflict or validation error, got %d", resp2.StatusCode)
	}
}

func testBookingCreateInvalidParticipantFormat(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("b1", "s1", "Bad Participants", start, end)
	payload["participants"] = map[string]string{
		"notaphone": "Invalid Name",
	}

	resp := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload)
	if resp.StatusCode != 422 && resp.StatusCode != 400 {
		t.Errorf("expected 400/422 for invalid participant phone, got %d", resp.StatusCode)
	}
}

func testBookingCreateMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	resp := bookingsHTTPClient.POSTRaw(t, "/api/v1/bookings", []byte(`{"bad": json`))
	common.AssertStatusCode(t, resp, 400)
}

func testBookingCreateAndGetByID(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("b1", "s1", "Massage", start, end)

	createResp := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	getResp := bookingsHTTPClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	common.AssertStatusCode(t, getResp, 200)
	fetched := decodeBooking(t, getResp)
	if fetched.ID != created.ID {
		t.Errorf("expected same ID, got %s != %s", fetched.ID, created.ID)
	}
	bookingsHTTPClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
}

func testBookingGetInvalidID(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	resp := bookingsHTTPClient.GET(t, "/api/v1/bookings/id/invalid-id")
	common.AssertStatusCode(t, resp, 400)
}

func testBookingSearchEmpty(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	resp := bookingsHTTPClient.GET(t, "/api/v1/bookings/search?business_id=b1&schedule_id=s1")
	common.AssertStatusCode(t, resp, 200)
	data := decodeBookings(t, resp)
	if len(data) != 0 {
		t.Errorf("expected empty results, got %d", len(data))
	}
}

func testBookingSearchRange(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(2 * time.Hour)
	payload := createValidBooking("b1", "s1", "Range Search", start, end)
	bookingsHTTPClient.POST(t, "/api/v1/bookings", payload)

	resp := bookingsHTTPClient.GET(t, fmt.Sprintf("/api/v1/bookings/search?business_id=b1&schedule_id=s1&start_time=%s&end_time=%s",
		start.Format(time.RFC3339), end.Format(time.RFC3339)))
	common.AssertStatusCode(t, resp, 200)
	data := decodeBookings(t, resp)
	if len(data) < 1 {
		t.Errorf("expected at least one booking in time range")
	}
}

func testBookingUpdateValid(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("b1", "s1", "Update Test", start, end)

	createResp := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	update := map[string]any{
		"service_label": "Updated Label",
	}
	resp := bookingsHTTPClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), update)
	common.AssertStatusCode(t, resp, 204)

	getResp := bookingsHTTPClient.GET(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	fetched := decodeBooking(t, getResp)
	if fetched.ServiceLabel != "Updated Label" {
		t.Errorf("expected updated label, got %s", fetched.ServiceLabel)
	}
}

func testBookingUpdateInvalidID(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	update := map[string]any{"service_label": "New Label"}
	resp := bookingsHTTPClient.PATCH(t, "/api/v1/bookings/id/invalid-id", update)
	common.AssertStatusCode(t, resp, 400)
}

func testBookingUpdateTimeOverlap(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	now := time.Now().Add(1 * time.Hour)
	payload1 := createValidBooking("b1", "s1", "First", now, now.Add(1*time.Hour))
	payload2 := createValidBooking("b1", "s1", "Second", now.Add(2*time.Hour), now.Add(3*time.Hour))

	resp1 := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload1)
	common.AssertStatusCode(t, resp1, 201)
	decodeBooking(t, resp1)
	resp2 := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload2)
	common.AssertStatusCode(t, resp2, 201)
	created2 := decodeBooking(t, resp2)

	update := map[string]any{
		"start_time": now.Add(30 * time.Minute).Format(time.RFC3339),
		"end_time":   now.Add(90 * time.Minute).Format(time.RFC3339),
	}
	resp := bookingsHTTPClient.PATCH(t, fmt.Sprintf("/api/v1/bookings/id/%s", created2.ID), update)
	if resp.StatusCode != 409 && resp.StatusCode != 400 {
		t.Errorf("expected conflict for overlapping update, got %d", resp.StatusCode)
	}
}

func testBookingUpdateMalformedJSON(t *testing.T) {
	defer common.ClearTestData(t, bookingsHTTPClient, BookingsTableName)
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("b1", "s1", "Bad JSON Update", start, end)
	createResp := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload)
	created := decodeBooking(t, createResp)

	resp := bookingsHTTPClient.PATCHRaw(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID), []byte(`{"bad":json`))
	common.AssertStatusCode(t, resp, 400)
}

func testBookingDeleteNonExisting(t *testing.T) {
	resp := bookingsHTTPClient.DELETE(t, "/api/v1/bookings/id/507f1f77bcf86cd799439011")
	common.AssertStatusCode(t, resp, 404)
}

func testBookingDeleteInvalidID(t *testing.T) {
	resp := bookingsHTTPClient.DELETE(t, "/api/v1/bookings/id/invalid-id")
	common.AssertStatusCode(t, resp, 400)
}

func testBookingCreateAndDelete(t *testing.T) {
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("b1", "s1", "Delete Test", start, end)

	createResp := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	delResp := bookingsHTTPClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	common.AssertStatusCode(t, delResp, 204)
}

func testBookingDoubleDelete(t *testing.T) {
	start := time.Now().Add(1 * time.Hour)
	end := start.Add(1 * time.Hour)
	payload := createValidBooking("b1", "s1", "Double Delete", start, end)

	createResp := bookingsHTTPClient.POST(t, "/api/v1/bookings", payload)
	common.AssertStatusCode(t, createResp, 201)
	created := decodeBooking(t, createResp)

	delResp := bookingsHTTPClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	common.AssertStatusCode(t, delResp, 204)

	delResp2 := bookingsHTTPClient.DELETE(t, fmt.Sprintf("/api/v1/bookings/id/%s", created.ID))
	common.AssertStatusCode(t, delResp2, 404)
}
