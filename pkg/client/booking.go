package client

import (
	"fmt"
	"net/url"
	"skeji/pkg/model"
)

type BookingClient struct {
	httpClient *HttpClient
}

func NewBookingClient(baseUrl string) *BookingClient {
	return &BookingClient{
		httpClient: NewHttpClient(baseUrl),
	}
}

func (c *BookingClient) Create(body any) (*Response, error) {
	return c.httpClient.POST("/api/v1/bookings", body)
}

func (c *BookingClient) GetAll(limit int, offset int64) (*Response, error) {
	path := fmt.Sprintf("/api/v1/bookings?limit=%d&offset=%d", limit, offset)
	return c.httpClient.GET(path)
}

func (c *BookingClient) Search(businessID string, scheduleID string, startTime string, endTime string, limit int, offset int64) (*Response, error) {
	q := url.Values{}
	q.Set("business_id", businessID)
	q.Set("schedule_id", scheduleID)

	if startTime != "" {
		q.Set("start_time", startTime)
	}
	if endTime != "" {
		q.Set("end_time", endTime)
	}

	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))

	path := "/api/v1/bookings/search?" + q.Encode()
	return c.httpClient.GET(path)
}

func (c *BookingClient) GetByID(id string) (*Response, error) {
	path := "/api/v1/bookings/id/" + url.PathEscape(id)
	return c.httpClient.GET(path)
}

func (c *BookingClient) Update(id string, body any) (*Response, error) {
	path := "/api/v1/bookings/id/" + url.PathEscape(id)
	return c.httpClient.PATCH(path, body)
}

func (c *BookingClient) Delete(id string) (*Response, error) {
	path := "/api/v1/bookings/id/" + url.PathEscape(id)
	return c.httpClient.DELETE(path)
}

func (c *BookingClient) DecodeBooking(resp *Response) (*model.Booking, error) {
	var booking *model.Booking
	err := resp.DecodeJSON(booking)
	if err != nil {
		return nil, fmt.Errorf("coulf not decode booking json:\n%+v\n%s", resp, err)
	}
	return booking, nil
}

func (c *BookingClient) DecodeBookings(resp *Response) ([]*model.Booking, error) {
	var bookings []*model.Booking
	err := resp.DecodeJSON(bookings)
	if err != nil {
		return nil, fmt.Errorf("coulf not decode bookings json:\n%+v\n%s", resp, err)
	}
	return bookings, nil
}

func (c *BookingClient) CreateRaw(rawBody []byte) (*Response, error) {
	return c.httpClient.POSTRaw("/api/v1/bookings", rawBody)
}

func (c *BookingClient) UpdateRaw(id string, rawBody []byte) (*Response, error) {
	path := "/api/v1/bookings/id/" + url.PathEscape(id)
	return c.httpClient.PATCHRaw(path, rawBody)
}
