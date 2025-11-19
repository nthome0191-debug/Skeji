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
	var schedule *model.Schedule
	err := resp.DecodeJSON(&schedule)
	if err != nil {
		return nil, fmt.Errorf("could not decode schedule json:\n%+v\n%s", resp, err)
	}
	return schedule, nil
}

func (c *ScheduleClient) DecodeSchedules(resp *Response) ([]*model.Schedule, *Metadata, error) {
	var paginated map[string]any
	err := resp.DecodeJSON(&paginated)
	if err != nil {
		return nil, nil, fmt.Errorf("could not decode paginated resp:\n%+v\n%s", resp, err)
	}

	byteArr, err := json.Marshal(paginated["data"])
	if err != nil {
		return nil, nil, fmt.Errorf("could not encode schedules json:\n%+v\n%s", resp, err)
	}

	var schedules []*model.Schedule
	err = json.Unmarshal(byteArr, &schedules)
	if err != nil {
		return nil, nil, fmt.Errorf("could not decode schedule list:\n%+v\n%s", resp, err)
	}

	metadata := &Metadata{
		TotalCount: int64(paginated["total_count"].(float64)),
		Limit:      int(paginated["limit"].(float64)),
		Offset:     int64(paginated["offset"].(float64)),
	}

	return schedules, metadata, nil
}
