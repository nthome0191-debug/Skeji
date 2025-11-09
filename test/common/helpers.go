package common

import (
	"fmt"
	"testing"
)

func ClearTestData(t *testing.T, httpClient *Client, tableName string) {
	t.Helper()

	resp := httpClient.GET(t, fmt.Sprintf("/api/v1/%s?limit=1000&offset=0", tableName))
	if resp.StatusCode != 200 {
		return
	}

	var result struct {
		Data []map[string]any `json:"data"`
	}

	if err := resp.DecodeJSON(&result); err != nil {
		t.Logf("Failed to decode JSON for resource %s: %v", tableName, err)
		return
	}

	for _, item := range result.Data {
		id, ok := item["_id"].(string)
		if !ok || id == "" {
			continue
		}
		httpClient.DELETE(t, fmt.Sprintf("/api/v1/%s/id/%s", tableName, id))
	}
}
