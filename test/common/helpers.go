package common

import (
	"fmt"
	"testing"
)

type Entity interface {
	GetID() string
}

type SingleEntityResponse[T any] struct {
	Data T `json:"data"`
}

type EntityListResponse[T any] struct {
	Data []T `json:"data"`
}

type PaginatedResponse[T any] struct {
	Data       []T   `json:"data"`
	TotalCount int64 `json:"total_count"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
}

func DecodeSingleEntity[T any](t *testing.T, resp *Response) *T {
	t.Helper()
	var result SingleEntityResponse[T]
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode single entity response: %v", err)
	}
	return &result.Data
}

func DecodeEntityList[T any](t *testing.T, resp *Response) []T {
	t.Helper()
	var result EntityListResponse[T]
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode entity list response: %v", err)
	}
	return result.Data
}

func DecodePaginated[T any](t *testing.T, resp *Response) ([]T, int64, int, int) {
	t.Helper()
	var result PaginatedResponse[T]
	if err := resp.DecodeJSON(&result); err != nil {
		t.Fatalf("failed to decode paginated response: %v", err)
	}
	return result.Data, result.TotalCount, result.Limit, result.Offset
}

func ClearAllData[T Entity](t *testing.T, client *Client, listEndpoint string, deleteEndpointPattern string) {
	t.Helper()
	resp := client.GET(t, listEndpoint)
	if resp.StatusCode != 200 {
		return
	}

	entities := DecodeEntityList[T](t, resp)
	for _, entity := range entities {
		client.DELETE(t, fmt.Sprintf(deleteEndpointPattern, entity.GetID()))
	}
}
