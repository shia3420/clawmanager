// Package integration defines the seams through which optional integration
// modules (e.g. a New API token relay / SSO bridge) plug into the clawmanager
// core without the core importing them. Interfaces live in this leaf package so
// that both the core and the modules only depend on it, keeping the direction
// of dependencies one-way and allowing modules to be disabled without touching
// core code.
package integration

import "context"

// CredentialOverride describes an upstream LLM endpoint override resolved for a
// specific user. When the gateway resolves an LLM model for a chat request it
// first consults the registered CredentialResolver; if it returns a non-nil
// override, the gateway routes to override.BaseURL using override.APIKey
// instead of the model's configured endpoint/credential.
type CredentialOverride struct {
	// BaseURL is the upstream base URL (e.g. https://relay.example.com/v1).
	BaseURL string
	// APIKey is the upstream bearer credential to use for this user.
	APIKey string
}

// CredentialResolver resolves a per-user upstream LLM credential override.
// Implementations must return (nil, nil) to fall back to the model's configured
// credentials. The gateway calls this opportunistically and ignores errors
// (falling back to the model's configured credentials) so a resolver outage can
// never take down chat traffic.
type CredentialResolver interface {
	ResolveCredential(ctx context.Context, userID int) (*CredentialOverride, error)
}
