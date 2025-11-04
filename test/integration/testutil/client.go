package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type Response struct {
	*http.Response
	Body []byte
}

func (r *Response) DecodeJSON(target interface{}) error {
	return json.Unmarshal(r.Body, target)
}

func (c *Client) GET(t *testing.T, path string) *Response {
	t.Helper()
	return c.request(t, http.MethodGet, path, nil, nil)
}

func (c *Client) POST(t *testing.T, path string, body interface{}) *Response {
	t.Helper()
	return c.request(t, http.MethodPost, path, body, nil)
}

func (c *Client) PATCH(t *testing.T, path string, body interface{}) *Response {
	t.Helper()
	return c.request(t, http.MethodPatch, path, body, nil)
}

func (c *Client) DELETE(t *testing.T, path string) *Response {
	t.Helper()
	return c.request(t, http.MethodDelete, path, nil, nil)
}

func (c *Client) POSTWithHeaders(t *testing.T, path string, body interface{}, headers map[string]string) *Response {
	t.Helper()
	return c.request(t, http.MethodPost, path, body, headers)
}

func (c *Client) request(t *testing.T, method, path string, body interface{}, headers map[string]string) *Response {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(context.Background(), method, url, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return &Response{
		Response: resp,
		Body:     respBody,
	}
}

func (c *Client) WaitForHealthy(t *testing.T, maxWait time.Duration) {
	t.Helper()

	deadline := time.Now().Add(maxWait)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		resp, err := c.HTTPClient.Get(c.BaseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			t.Log("Service is healthy")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		<-ticker.C
	}

	t.Fatalf("service did not become healthy within %v", maxWait)
}

func AssertStatusCode(t *testing.T, resp *Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Fatalf("expected status %d, got %d. Body: %s", expected, resp.StatusCode, string(resp.Body))
	}
}

func AssertContains(t *testing.T, resp *Response, substr string) {
	t.Helper()
	body := string(resp.Body)
	if !contains(body, substr) {
		t.Fatalf("response body does not contain %q. Body: %s", substr, body)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func PrintResponse(t *testing.T, resp *Response) {
	t.Helper()
	t.Logf("Status: %d", resp.StatusCode)
	t.Logf("Body: %s", string(resp.Body))
	t.Logf("Headers: %v", resp.Header)
}

func GetErrorMessage(t *testing.T, resp *Response) string {
	t.Helper()
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if err := resp.DecodeJSON(&errResp); err != nil {
		return fmt.Sprintf("failed to unmarshal error: %v", err)
	}
	if errResp.Message != "" {
		return errResp.Message
	}
	if errResp.Error != "" {
		return errResp.Error
	}
	return errResp.Code
}
