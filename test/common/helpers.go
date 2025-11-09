package common

import (
	"fmt"
	"testing"
)

func ClearTestData(t *testing.T, httpClient *Client, tableName string) {
	t.Helper()
	totalCount := 1000
	for totalCount > 0 {
		resp := httpClient.GET(t, fmt.Sprintf("/api/v1/%s?limit=1000&offset=0", tableName))
		if resp.StatusCode != 200 {
			t.Fatalf("Failed to clear table data %s, %d\n", tableName, resp.StatusCode)
		}

		var result struct {
			Data []map[string]any `json:"data"`
		}

		if err := resp.DecodeJSON(&result); err != nil {
			t.Logf("Failed to decode JSON for resource %s: %v", tableName, err)
			return
		}

		for _, item := range result.Data {
			id, ok := item["id"].(string)
			if !ok || id == "" {
				continue
			}
			httpClient.DELETE(t, fmt.Sprintf("/api/v1/%s/id/%s", tableName, id))
		}
		resp = httpClient.GET(t, fmt.Sprintf("/api/v1/%s?limit=10&offset=0", tableName))
		if resp.StatusCode != 200 {
			t.Fatalf("Failed to clear table data %s, %d\n", tableName, resp.StatusCode)
		}
		var res struct {
			Data       []map[string]any `json:"data"`
			TotalCount int              `json:"total_count"`
			Limit      int              `json:"limit"`
			Offset     int              `json:"offset"`
		}
		if err := resp.DecodeJSON(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		totalCount = res.TotalCount
	}
}
