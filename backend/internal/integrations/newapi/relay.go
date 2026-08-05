package newapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RelayClient talks to a New API (One API) instance. The wire contract lives
// here and nowhere else, so the rest of the module (and core) stays decoupled
// from upstream specifics.
//
// Two credential kinds are used:
//
//   - A relay API key (sk-...) for the admin-managed relay registration check.
//     New API serves GET /v1/models for any valid API key and rejects invalid
//     keys with 401, which doubles as a reachability check.
//   - A per-user dashboard token (short-lived JWT, or a long-lived "system
//     access token" from the user's security settings) used to mint and
//     retrieve the user's own dedicated API key: POST /api/token/ creates the
//     token and POST /api/token/{id}/key returns its full key on demand. The
//     created key belongs to the token's user, so each clawmanager user ends up
//     with their own independent upstream credential.
type RelayClient interface {
	// ValidateCredential verifies that apiKey is a working credential on the
	// given base URL. Returns nil when the key is valid (HTTP 200 from
	// GET /v1/models), an error otherwise.
	ValidateCredential(ctx context.Context, baseURL, apiKey string) error

	// FetchSelf validates a dashboard token and returns the identity of the
	// authenticated upstream user.
	FetchSelf(ctx context.Context, baseURL, dashboardToken string) (*UpstreamUser, error)

	// MintToken creates a dedicated API token for the authenticated user and
	// returns its id. The minted key itself is not part of the create response;
	// call FetchTokenKey with the returned id to retrieve it.
	MintToken(ctx context.Context, baseURL, dashboardToken, name, group string) (int, error)

	// FetchTokenKey returns the full (un-prefixed) key of a token owned by the
	// authenticated user. Callers prefix "sk-" before using it against /v1.
	FetchTokenKey(ctx context.Context, baseURL, dashboardToken string, tokenID int) (string, error)
}

// UpstreamUser is the identity returned by GET /api/user/self on a New API
// instance.
type UpstreamUser struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Group       string `json:"group"`
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

func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

func (c *httpRelayClient) ValidateCredential(ctx context.Context, baseURL, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalizeBaseURL(baseURL)+"/v1/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("newapi relay validation request failed: %w", err)
	}
	defer resp.Body.Close()

	// Drain a bounded amount so the connection can be reused.
	if _, err := io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20)); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("newapi relay rejected the credential (status %d)", resp.StatusCode)
	}
	return nil
}

// newAPIResponse is the generic envelope used by the dashboard management API.
type newAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Code    string          `json:"code"`
	Data    json.RawMessage `json:"data"`
}

func (c *httpRelayClient) doDashboardRequest(ctx context.Context, method, baseURL, dashboardToken, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, normalizeBaseURL(baseURL)+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(dashboardToken))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("newapi dashboard request failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("newapi dashboard token was rejected (status %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("newapi dashboard request failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	return payload, nil
}

func (c *httpRelayClient) FetchSelf(ctx context.Context, baseURL, dashboardToken string) (*UpstreamUser, error) {
	payload, err := c.doDashboardRequest(ctx, http.MethodGet, baseURL, dashboardToken, "/api/user/self", nil)
	if err != nil {
		return nil, err
	}
	var env newAPIResponse
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, fmt.Errorf("newapi self response is invalid: %w", err)
	}
	if !env.Success {
		return nil, fmt.Errorf("newapi self request failed: %s", env.Message)
	}
	var user UpstreamUser
	if err := json.Unmarshal(env.Data, &user); err != nil {
		return nil, fmt.Errorf("newapi self payload is invalid: %w", err)
	}
	if user.ID <= 0 {
		return nil, fmt.Errorf("newapi self payload missing user id")
	}
	return &user, nil
}

func (c *httpRelayClient) MintToken(ctx context.Context, baseURL, dashboardToken, name, group string) (int, error) {
	if strings.TrimSpace(group) == "" {
		group = "default"
	}
	body, err := json.Marshal(map[string]interface{}{
		"name":                 name,
		"remain_quota":         0,
		"expired_time":         -1,
		"unlimited_quota":      true,
		"model_limits_enabled": false,
		"allow_ips":            "",
		"group":                group,
	})
	if err != nil {
		return 0, err
	}
	if _, err := c.doDashboardRequest(ctx, http.MethodPost, baseURL, dashboardToken, "/api/token/", body); err != nil {
		return 0, err
	}

	// The create response does not carry the token id, so look it up by name in
	// the current user's token list. The name is unique per user.
	page := 1
	for page <= 3 {
		id, lookupErr := c.findTokenIDByName(ctx, baseURL, dashboardToken, name, page)
		if lookupErr != nil {
			return 0, lookupErr
		}
		if id > 0 {
			return id, nil
		}
		page++
	}
	return 0, fmt.Errorf("newapi minted token was not found in token list")
}

func (c *httpRelayClient) findTokenIDByName(ctx context.Context, baseURL, dashboardToken, name string, page int) (int, error) {
	path := "/api/token/?p=" + url.QueryEscape(fmt.Sprintf("%d", page)) + "&page_size=50"
	payload, err := c.doDashboardRequest(ctx, http.MethodGet, baseURL, dashboardToken, path, nil)
	if err != nil {
		return 0, err
	}
	var env struct {
		Success bool `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Items []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return 0, fmt.Errorf("newapi token list response is invalid: %w", err)
	}
	if !env.Success {
		return 0, fmt.Errorf("newapi token list failed: %s", env.Message)
	}
	for _, item := range env.Data.Items {
		if item.Name == name {
			return item.ID, nil
		}
	}
	return 0, nil
}

func (c *httpRelayClient) FetchTokenKey(ctx context.Context, baseURL, dashboardToken string, tokenID int) (string, error) {
	if tokenID <= 0 {
		return "", fmt.Errorf("newapi token id is invalid")
	}
	payload, err := c.doDashboardRequest(ctx, http.MethodPost, baseURL, dashboardToken, fmt.Sprintf("/api/token/%d/key", tokenID), nil)
	if err != nil {
		return "", err
	}
	var env struct {
		Success bool `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Key string `json:"key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return "", fmt.Errorf("newapi token key response is invalid: %w", err)
	}
	if !env.Success {
		return "", fmt.Errorf("newapi token key request failed: %s", env.Message)
	}
	if strings.TrimSpace(env.Data.Key) == "" {
		return "", fmt.Errorf("newapi token key request returned an empty key")
	}
	return env.Data.Key, nil
}
