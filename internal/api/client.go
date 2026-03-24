package api

import (
	"fmt"
	"net/http"
	"time"
)

// Client is a thin HTTP client for the Guild API.
// It attaches Bearer token auth and JSON accept headers to every request.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a Client targeting the given base URL with the provided auth token.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Do executes an HTTP request with Bearer token and Accept: application/json headers.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api request %s %s: %w", req.Method, req.URL.Path, err)
	}
	return resp, nil
}
