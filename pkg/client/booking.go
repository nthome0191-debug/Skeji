package client

import (
	"encoding/json"
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

func (c *BookingClient) CreateRaw(rawBody []byte) (*Response, error) {
	return c.httpClient.POSTRaw("/api/v1/bookings", rawBody)
}

func (c *BookingClient) UpdateRaw(id string, rawBody []byte) (*Response, error) {
	path := "/api/v1/bookings/id/" + url.PathEscape(id)
	return c.httpClient.PATCHRaw(path, rawBody)
}

func (c *BookingClient) DecodeBooking(resp *Response) (*model.Booking, error) {
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(resp.Body, &wrapper); err != nil {
		return nil, fmt.Errorf("could not decode booking wrapper:\n%+v\n%s", resp.ToString(), err)
	}

	var booking model.Booking
	if err := json.Unmarshal(wrapper.Data, &booking); err != nil {
		return nil, fmt.Errorf("could not decode booking json:\n%+v\n%s", resp.ToString(), err)
	}

	return &booking, nil
}

func (c *BookingClient) DecodeBookings(resp *Response) ([]*model.Booking, *Metadata, error) {
	var wrapper struct {
		Data       json.RawMessage `json:"data"`
		TotalCount int64           `json:"total_count"`
		Limit      int             `json:"limit"`
		Offset     int64           `json:"offset"`
	}

	if err := json.Unmarshal(resp.Body, &wrapper); err != nil {
		return nil, nil, fmt.Errorf("could not decode paginated resp:\n%+v\n%s", resp.ToString(), err)
	}

	var bookings []*model.Booking
	if err := json.Unmarshal(wrapper.Data, &bookings); err != nil {
		return nil, nil, fmt.Errorf("could not decode booking list:\n%+v\n%s", resp.ToString(), err)
	}

	metadata := &Metadata{
		TotalCount: wrapper.TotalCount,
		Limit:      wrapper.Limit,
		Offset:     wrapper.Offset,
	}

	return bookings, metadata, nil
}
