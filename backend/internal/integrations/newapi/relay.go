package newapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RelayClient talks to a New API (One API) instance over its admin/user API.
// The exact wire contract lives here and nowhere else, so the rest of the
// module (and core) stays decoupled from upstream specifics.
type RelayClient interface {
	// FetchTokenSelf verifies an access token against the relay instance and
	// returns the upstream token metadata (id + name).
	FetchTokenSelf(ctx context.Context, baseURL, accessToken string) (TokenSelf, error)
	// MintToken creates a new scoped token under the relay account (using the
	// admin relay token) bounded by the given token budget.
	MintToken(ctx context.Context, baseURL, relayToken, name string, budget int64) (string, error)
}

// TokenSelf is the upstream metadata returned by GET /api/token/self.
type TokenSelf struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type httpRelayClient struct {
	httpClient *http.Client
}

// NewRelayClient creates a RelayClient backed by an HTTP client.
func NewRelayClient() RelayClient {
	return &httpRelayClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type newAPIEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

func (c *httpRelayClient) do(ctx context.Context, method, baseURL, path, token string, body interface{}, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, normalizeBaseURL(baseURL)+path, bodyReader)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("newapi relay request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}

	var envelope newAPIEnvelope
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		return fmt.Errorf("newapi relay returned malformed response (status %d): %w", resp.StatusCode, err)
	}
	if !envelope.Success || resp.StatusCode >= 400 {
		return fmt.Errorf("newapi relay error (status %d): %s", resp.StatusCode, strings.TrimSpace(envelope.Message))
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return fmt.Errorf("newapi relay data unmarshal failed: %w", err)
		}
	}
	return nil
}

func (c *httpRelayClient) FetchTokenSelf(ctx context.Context, baseURL, accessToken string) (TokenSelf, error) {
	var self TokenSelf
	if err := c.do(ctx, http.MethodGet, baseURL, "/api/token/self", accessToken, nil, &self); err != nil {
		return TokenSelf{}, err
	}
	return self, nil
}

type mintTokenRequest struct {
	Name        string `json:"name"`
	RemainQuota int64  `json:"remain_quota,omitempty"`
	Unlimited   bool   `json:"unlimited"`
}

type mintTokenData struct {
	Key string `json:"key"`
}

func (c *httpRelayClient) MintToken(ctx context.Context, baseURL, relayToken, name string, budget int64) (string, error) {
	body := mintTokenRequest{Name: name, RemainQuota: budget, Unlimited: false}
	if budget <= 0 {
		body.Unlimited = true
		body.RemainQuota = 0
	}
	var data mintTokenData
	if err := c.do(ctx, http.MethodPost, baseURL, "/api/token/", relayToken, &body, &data); err != nil {
		return "", err
	}
	if data.Key == "" {
		return "", fmt.Errorf("newapi relay returned a token without a key")
	}
	return data.Key, nil
}
