package client

import (
	"fmt"
	"net/url"
	"strings"
)

type BusinessUnitClient struct {
	httpClient *HttpClient
}

func NewBusinessUnitClient(baseUrl string) *BusinessUnitClient {
	return &BusinessUnitClient{
		httpClient: NewHttpClient(baseUrl),
	}
}

func (c *BusinessUnitClient) Create(body any) (*Response, error) {
	return c.httpClient.POST("/api/v1/business-units", body)
}

func (c *BusinessUnitClient) GetAll(limit int, offset int64) (*Response, error) {
	path := fmt.Sprintf("/api/v1/business-units?limit=%d&offset=%d", limit, offset)
	return c.httpClient.GET(path)
}

func (c *BusinessUnitClient) Search(cities []string, labels []string, limit int, offset int64) (*Response, error) {
	q := url.Values{}
	q.Set("cities", strings.Join(cities, ","))
	q.Set("labels", strings.Join(labels, ","))
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))

	path := "/api/v1/business-units/search?" + q.Encode()
	return c.httpClient.GET(path)
}

func (c *BusinessUnitClient) GetByPhone(phone string, limit int, offset int64) (*Response, error) {
	path := fmt.Sprintf(
		"/api/v1/business-units/phone/%s?limit=%d&offset=%d",
		url.PathEscape(phone),
		limit,
		offset,
	)
	return c.httpClient.GET(path)
}

func (c *BusinessUnitClient) GetByID(id string) (*Response, error) {
	path := "/api/v1/business-units/id/" + url.PathEscape(id)
	return c.httpClient.GET(path)
}

func (c *BusinessUnitClient) Update(id string, body any) (*Response, error) {
	path := "/api/v1/business-units/id/" + url.PathEscape(id)
	return c.httpClient.PATCH(path, body)
}

func (c *BusinessUnitClient) Delete(id string) (*Response, error) {
	path := "/api/v1/business-units/id/" + url.PathEscape(id)
	return c.httpClient.DELETE(path)
}

func (c *BusinessUnitClient) CreateRaw(rawBody []byte) (*Response, error) {
	return c.httpClient.POSTRaw("/api/v1/business-units", rawBody)
}

func (c *BusinessUnitClient) UpdateRaw(id string, rawBody []byte) (*Response, error) {
	path := "/api/v1/business-units/id/" + url.PathEscape(id)
	return c.httpClient.PATCHRaw(path, rawBody)
}
