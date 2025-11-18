package client

import (
	"fmt"
	"net/url"
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

func (c *ScheduleClient) Search(cities []string, labels []string, limit int, offset int64) (*Response, error) {
	q := url.Values{}
	for _, cty := range cities {
		q.Add("cities", cty)
	}
	for _, label := range labels {
		q.Add("labels", label)
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
