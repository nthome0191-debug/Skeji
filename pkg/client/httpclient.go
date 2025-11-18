package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HttpClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewHttpClient(baseURL string) *HttpClient {
	return &HttpClient{
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

func (r *Response) DecodeJSON(target any) error {
	return json.Unmarshal(r.Body, target)
}

func (c *HttpClient) GET(path string) (*Response, error) {
	return c.request(http.MethodGet, path, nil, nil)
}

func (c *HttpClient) POST(path string, body any) (*Response, error) {
	return c.request(http.MethodPost, path, body, nil)
}

func (c *HttpClient) PATCH(path string, body any) (*Response, error) {
	return c.request(http.MethodPatch, path, body, nil)
}

func (c *HttpClient) DELETE(path string) (*Response, error) {
	return c.request(http.MethodDelete, path, nil, nil)
}

func (c *HttpClient) POSTWithHeaders(path string, body any, headers map[string]string) (*Response, error) {
	return c.request(http.MethodPost, path, body, headers)
}

func (c *HttpClient) POSTRaw(path string, rawBody []byte) (*Response, error) {
	return c.requestRaw(http.MethodPost, path, rawBody, nil)
}

func (c *HttpClient) PATCHRaw(path string, rawBody []byte) (*Response, error) {
	return c.requestRaw(http.MethodPatch, path, rawBody, nil)
}

func (c *HttpClient) request(method, path string, body any, headers map[string]string) (*Response, error) {
	var reqBody io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	return c.do(method, path, reqBody, body != nil, headers)
}

func (c *HttpClient) requestRaw(method, path string, rawBody []byte, headers map[string]string) (*Response, error) {
	var reqBody io.Reader
	if rawBody != nil {
		reqBody = bytes.NewBuffer(rawBody)
	}
	return c.do(method, path, reqBody, rawBody != nil, headers)
}

func (c *HttpClient) do(method, path string, reqBody io.Reader, hasBody bool, headers map[string]string) (*Response, error) {
	url := c.BaseURL + path

	req, err := http.NewRequestWithContext(context.Background(), method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		Response: resp,
		Body:     respBody,
	}, nil
}

func (c *HttpClient) WaitForHealthy(maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		resp, err := c.HTTPClient.Get(c.BaseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		<-ticker.C
	}

	return fmt.Errorf("service did not become healthy within %v", maxWait)
}

func GetErrorMessage(resp *Response) string {
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
