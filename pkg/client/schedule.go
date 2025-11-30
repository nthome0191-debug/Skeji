package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"skeji/pkg/model"
)

type ScheduleClient struct {
	httpClient *HttpClient
}

func NewScheduleClient(baseUrl string) *ScheduleClient {
	return &ScheduleClient{
		httpClient: NewHttpClient(baseUrl),
	}
}

func (c *ScheduleClient) Create(body any) (*Response, error) {
	return c.httpClient.POST("/api/v1/schedules", body)
}

func (c *ScheduleClient) GetAll(limit int, offset int64) (*Response, error) {
	path := fmt.Sprintf("/api/v1/schedules?limit=%d&offset=%d", limit, offset)
	return c.httpClient.GET(path)
}

func (c *ScheduleClient) Search(businessID string, city string, limit int, offset int64) (*Response, error) {
	q := url.Values{}
	q.Set("business_id", businessID)

	if city != "" {
		q.Set("city", city)
	}

	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))

	path := "/api/v1/schedules/search?" + q.Encode()
	return c.httpClient.GET(path)
}

func (c *ScheduleClient) BatchSearch(businessID string, cities []string, limit int, offset int64) (*Response, error) {
	q := url.Values{}
	q.Set("business_id", businessID)

	// Join cities with comma for the query parameter
	if len(cities) > 0 {
		citiesStr := ""
		for i, city := range cities {
			if i > 0 {
				citiesStr += ","
			}
			citiesStr += city
		}
		q.Set("cities", citiesStr)
	}

	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))

	path := "/api/v1/schedules/batch-search?" + q.Encode()
	return c.httpClient.GET(path)
}

func (c *ScheduleClient) GetByID(id string) (*Response, error) {
	path := "/api/v1/schedules/id/" + url.PathEscape(id)
	return c.httpClient.GET(path)
}

func (c *ScheduleClient) Update(id string, body any) (*Response, error) {
	path := "/api/v1/schedules/id/" + url.PathEscape(id)
	return c.httpClient.PATCH(path, body)
}

func (c *ScheduleClient) Delete(id string) (*Response, error) {
	path := "/api/v1/schedules/id/" + url.PathEscape(id)
	return c.httpClient.DELETE(path)
}

func (c *ScheduleClient) CreateRaw(rawBody []byte) (*Response, error) {
	return c.httpClient.POSTRaw("/api/v1/schedules", rawBody)
}

func (c *ScheduleClient) UpdateRaw(id string, rawBody []byte) (*Response, error) {
	path := "/api/v1/schedules/id/" + url.PathEscape(id)
	return c.httpClient.PATCHRaw(path, rawBody)
}

func (c *ScheduleClient) DecodeSchedule(resp *Response) (*model.Schedule, error) {
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(resp.Body, &wrapper); err != nil {
		return nil, fmt.Errorf("could not decode schedule wrapper:\n%+v\n%s", resp.ToString(), err)
	}

	var schedule model.Schedule
	if err := json.Unmarshal(wrapper.Data, &schedule); err != nil {
		return nil, fmt.Errorf("could not decode schedule json:\n%+v\n%s", resp.ToString(), err)
	}

	return &schedule, nil
}

func (c *ScheduleClient) DecodeSchedules(resp *Response) ([]*model.Schedule, *Metadata, error) {
	var wrapper struct {
		Data       json.RawMessage `json:"data"`
		TotalCount int64           `json:"total_count"`
		Limit      int             `json:"limit"`
		Offset     int64           `json:"offset"`
	}

	if err := json.Unmarshal(resp.Body, &wrapper); err != nil {
		return nil, nil, fmt.Errorf("could not decode paginated resp:\n%+v\n%s", resp.ToString(), err)
	}

	var schedules []*model.Schedule
	if err := json.Unmarshal(wrapper.Data, &schedules); err != nil {
		return nil, nil, fmt.Errorf("could not decode schedule list:\n%+v\n%s", resp.ToString(), err)
	}

	metadata := &Metadata{
		TotalCount: wrapper.TotalCount,
		Limit:      wrapper.Limit,
		Offset:     wrapper.Offset,
	}

	return schedules, metadata, nil
}
