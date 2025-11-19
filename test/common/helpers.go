package common

import (
	"encoding/json"
	"fmt"
	"skeji/pkg/client"
	"testing"
)

func AssertStatusCode(t *testing.T, resp *client.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("expected status code %d, got %d", expected, resp.StatusCode)
	}
}

func AssertContains(t *testing.T, resp *client.Response, substring string) {
	t.Helper()
	body := string(resp.Body)
	if len(body) == 0 {
		t.Errorf("expected response body to contain '%s', but body was empty", substring)
		return
	}
	if !contains(body, substring) {
		t.Errorf("expected response body to contain '%s', but it didn't. Body: %s", substring, body)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func ClearTestData(t *testing.T, httpClient *client.HttpClient, tableName string) {
	t.Helper()
	totalCount := 1000
	for totalCount > 0 {
		resp, err := httpClient.GET(fmt.Sprintf("/api/v1/%s?limit=1000&offset=0", tableName))
		if err != nil {
			t.Fatalf("Failed to get table data %v\n", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("Failed to clear table data %s, %d\n", tableName, resp.StatusCode)
		}

		var result struct {
			Data []map[string]any `json:"data"`
		}

		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Logf("Failed to decode JSON for resource %s: %v", tableName, err)
			return
		}

		for _, item := range result.Data {
			id, ok := item["id"].(string)
			if !ok || id == "" {
				continue
			}
			httpClient.DELETE(fmt.Sprintf("/api/v1/%s/id/%s", tableName, id))
		}
		resp, err = httpClient.GET(fmt.Sprintf("/api/v1/%s?limit=10&offset=0", tableName))
		if err != nil {
			t.Fatalf("Failed to get table data %v\n", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("Failed to clear table data %s, %d\n", tableName, resp.StatusCode)
		}
		var res struct {
			Data       []map[string]any `json:"data"`
			TotalCount int              `json:"total_count"`
			Limit      int              `json:"limit"`
			Offset     int              `json:"offset"`
		}
		if err := json.Unmarshal(resp.Body, &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		totalCount = res.TotalCount
	}
}
