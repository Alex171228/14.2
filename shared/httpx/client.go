package httpx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"pz1.2/shared/middleware"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, c.httpClient.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if requestID := middleware.GetRequestID(ctx); requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) DoWithAuth(ctx context.Context, method, path, token string, body io.Reader) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, c.httpClient.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if requestID := middleware.GetRequestID(ctx); requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	return c.httpClient.Do(req)
}
