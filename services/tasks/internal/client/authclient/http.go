package authclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"pz1.2/shared/middleware"
)

type HTTPClient struct {
	httpClient *http.Client
	baseURL    string
	log        *zap.Logger
}

type VerifyResponse struct {
	Valid   bool   `json:"valid"`
	Subject string `json:"subject,omitempty"`
	Error   string `json:"error,omitempty"`
}

func NewHTTPClient(baseURL string, timeout time.Duration, log *zap.Logger) *HTTPClient {
	return &HTTPClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		log:     log.With(zap.String("component", "auth_client_http")),
	}
}

func (c *HTTPClient) Verify(ctx context.Context, token string) (*VerifyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.httpClient.Timeout)
	defer cancel()

	requestID := middleware.GetRequestID(ctx)
	l := c.log.With(zap.String("request_id", requestID))
	l.Debug("calling auth HTTP verify")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/auth/verify", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		l.Error("auth HTTP verify failed", zap.Error(err))
		return nil, fmt.Errorf("auth service unavailable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var verifyResp VerifyResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		l.Warn("auth HTTP verify: unauthorized")
		return &verifyResp, nil
	}

	if resp.StatusCode != http.StatusOK {
		l.Error("auth HTTP verify: unexpected status", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	l.Debug("auth HTTP verify: success", zap.String("subject", verifyResp.Subject))
	return &verifyResp, nil
}
