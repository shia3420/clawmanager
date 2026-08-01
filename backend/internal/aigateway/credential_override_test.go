package aigateway

import (
	"context"
	"errors"
	"testing"

	"clawreef/internal/integration"
	"clawreef/internal/models"
)

type stubCredentialResolver struct {
	override *integration.CredentialOverride
	err      error
	calls    int
}

func (s *stubCredentialResolver) ResolveCredential(ctx context.Context, userID int) (*integration.CredentialOverride, error) {
	s.calls++
	return s.override, s.err
}

func TestCredentialOverrideIsNilSafeByDefault(t *testing.T) {
	svc := &service{}
	prepared := &preparedChatRequest{
		userID: 1,
		resolvedModel: &models.LLMModel{
			BaseURL:           "https://default.example.com/v1",
			ProviderModelName: "gpt-4o",
		},
	}
	svc.applyCredentialOverride(prepared)
	if prepared.resolvedModel.BaseURL != "https://default.example.com/v1" {
		t.Fatalf("default model endpoint must be preserved, got %q", prepared.resolvedModel.BaseURL)
	}
	if svc.credentialResolver != nil {
		t.Fatal("resolver must be nil by default")
	}
}

func TestCredentialOverrideAppliesRelayCredential(t *testing.T) {
	key := "sk-relayed"
	svc := &service{
		credentialResolver: &stubCredentialResolver{
			override: &integration.CredentialOverride{
				BaseURL: "https://relay.example.com/v1",
				APIKey:  key,
			},
		},
	}
	original := &models.LLMModel{
		BaseURL:           "https://default.example.com/v1",
		ProviderModelName: "gpt-4o",
	}
	prepared := &preparedChatRequest{userID: 3, resolvedModel: original}
	svc.applyCredentialOverride(prepared)

	if prepared.resolvedModel.BaseURL != "https://relay.example.com/v1" {
		t.Fatalf("expected relay base URL, got %q", prepared.resolvedModel.BaseURL)
	}
	if prepared.resolvedModel.APIKey == nil || *prepared.resolvedModel.APIKey != key {
		t.Fatalf("expected relay API key to be injected")
	}
	// the original model object must not be mutated
	if original.BaseURL != "https://default.example.com/v1" {
		t.Fatalf("original model must not be mutated")
	}
	if prepared.resolvedModel.ProviderModelName != "gpt-4o" {
		t.Fatalf("provider model name must be preserved, got %q", prepared.resolvedModel.ProviderModelName)
	}
}

func TestCredentialOverrideFallsBackOnResolverError(t *testing.T) {
	svc := &service{
		credentialResolver: &stubCredentialResolver{err: errors.New("boom")},
	}
	prepared := &preparedChatRequest{
		userID:        3,
		resolvedModel: &models.LLMModel{BaseURL: "https://default.example.com/v1"},
	}
	svc.applyCredentialOverride(prepared)
	if prepared.resolvedModel.BaseURL != "https://default.example.com/v1" {
		t.Fatalf("resolver error must fall back to model credentials, got %q", prepared.resolvedModel.BaseURL)
	}
}
