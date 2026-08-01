package newapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RelayClient talks to a New API (One API) instance. The wire contract lives
// here and nowhere else, so the rest of the module (and core) stays decoupled
// from upstream specifics.
//
// The only upstream call the shared-relay model needs is credential
// validation: New API serves /v1/models for any valid API key (sk-...) and
// rejects invalid keys with 401, which doubles as a reachability check and an
// identity-ownership check. Token minting (POST /api/token/) is NOT used: on
// real deployments it requires a short-lived dashboard JWT and does not return
// the minted key, so the module shares the relay account's API key instead and
// enforces per-user limits with its own request breaker.
type RelayClient interface {
	// ValidateCredential verifies that apiKey is a working credential on the
	// given base URL. Returns nil when the key is valid (HTTP 200 from
	// GET /v1/models), an error otherwise.
	ValidateCredential(ctx context.Context, baseURL, apiKey string) error
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
