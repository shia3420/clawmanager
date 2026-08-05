package newapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"clawreef/internal/integration"
	"clawreef/internal/models"
)

// userStore is the minimal slice of the core user repository the module needs.
// repository.UserRepository satisfies it; keeping the dependency narrow makes
// the module testable and its coupling to core explicit.
type userStore interface {
	Create(user *models.User) error
	GetByID(id int) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
}

// Service errors. Exposed as sentinel errors so handlers/tests can branch.
var (
	ErrRelayNotFound        = errors.New("newapi relay not found")
	ErrRelayInvalid         = errors.New("newapi relay registration is invalid")
	ErrUpstreamRejected     = errors.New("newapi relay rejected the credential")
	ErrModuleNotReady       = errors.New("newapi module encryption key is not configured")
	ErrIdentityLinkNotFound = errors.New("newapi identity link not found")
)

// Config carries the module's own settings (never part of the core Config).
type Config struct {
	EncryptionKey string
	JWTSecret     string
	JWTExpiry     time.Duration
}

// ExchangeResult is returned to the SSO handler. The user's own relay
// credential is intentionally NOT included: it is minted on the upstream
// instance during the exchange, stored encrypted, and resolved server-side.
type ExchangeResult struct {
	UserID       int
	RelayName    string
	RelayBaseURL string
	CreatedUser  bool
}

// RelayKeyView is a masked view of a relay registration for admin listings.
type RelayKeyView struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	BaseURL     string    `json:"base_url"`
	DailyLimit  int64     `json:"daily_limit"`
	MaskedToken string    `json:"masked_token"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// IdentityLinkView is an admin-facing view of a user-level gateway binding.
// It joins the linked clawmanager user, its relay and the today's trial-quota
// usage. The per-user credential is never exposed: only its presence and the
// upstream token name (so an admin can revoke it in the New API console).
type IdentityLinkView struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	RelayKeyID     int        `json:"relay_key_id"`
	RelayName      string     `json:"relay_name"`
	RelayBaseURL   string     `json:"relay_base_url"`
	ExternalID     string     `json:"external_id"`
	UpstreamUserID string     `json:"upstream_user_id"`
	TokenName      string     `json:"token_name"`
	HasCredential  bool       `json:"has_credential"`
	TodayUsed      int64      `json:"today_used"`
	TodayLimit     int64      `json:"today_limit"`
	CreatedAt      time.Time  `json:"created_at"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
}

// Service implements the module's business logic.
type Service interface {
	CreateRelayKey(ctx context.Context, name, baseURL, relayKey string, dailyLimit int64, createdBy int) error
	ListRelayKeys() ([]RelayKeyView, error)
	DeleteRelayKey(id int) error
	ExchangeSSO(ctx context.Context, relayName, dashboardToken string) (*ExchangeResult, error)
	ResolveCredential(ctx context.Context, userID int) (*integration.CredentialOverride, error)
	ListIdentityLinks() ([]IdentityLinkView, error)
	UnlinkIdentityLink(id int) error
}

type service struct {
	repo     Repository
	userRepo userStore
	relay    RelayClient
	cipher   *newAPICipher
	config   Config
}

// NewService creates the module service. The cipher is built from the module's
// dedicated encryption key.
func NewService(repo Repository, userRepo userStore, relay RelayClient, config Config) (Service, error) {
	if strings.TrimSpace(config.EncryptionKey) == "" {
		return nil, ErrModuleNotReady
	}
	cipher, err := newCipher(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("newapi module cipher init failed: %w", err)
	}
	return &service{repo: repo, userRepo: userRepo, relay: relay, cipher: cipher, config: config}, nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

func randomToken(nBytes int) string {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func (s *service) CreateRelayKey(ctx context.Context, name, baseURL, relayKey string, dailyLimit int64, createdBy int) error {
	name = strings.TrimSpace(name)
	baseURL = strings.TrimSpace(baseURL)
	relayKey = strings.TrimSpace(relayKey)
	if name == "" || baseURL == "" || relayKey == "" {
		return ErrRelayInvalid
	}
	if existing, err := s.repo.GetRelayKeyByName(name); err != nil {
		return err
	} else if existing != nil {
		return fmt.Errorf("newapi relay name already exists")
	}
	if err := s.relay.ValidateCredential(ctx, baseURL, relayKey); err != nil {
		return fmt.Errorf("%w: %v", ErrUpstreamRejected, err)
	}
	encrypted, err := s.cipher.Encrypt(relayKey)
	if err != nil {
		return err
	}
	return s.repo.CreateRelayKey(&RelayKey{
		Name:              name,
		BaseURL:           baseURL,
		RelayTokenEnc:     encrypted,
		DefaultDailyLimit: dailyLimit,
		CreatedBy:         createdBy,
	})
}

func (s *service) ListRelayKeys() ([]RelayKeyView, error) {
	keys, err := s.repo.ListRelayKeys()
	if err != nil {
		return nil, err
	}
	views := make([]RelayKeyView, 0, len(keys))
	for _, key := range keys {
		views = append(views, RelayKeyView{
			ID:          key.ID,
			Name:        key.Name,
			BaseURL:     key.BaseURL,
			DailyLimit:  key.DefaultDailyLimit,
			MaskedToken: maskToken("relay"),
			CreatedBy:   key.CreatedBy,
			CreatedAt:   key.CreatedAt,
		})
	}
	return views, nil
}

func (s *service) DeleteRelayKey(id int) error {
	return s.repo.DeleteRelayKey(id)
}

// mintedTokenName returns the upstream token name created for a clawmanager
// user during ExchangeSSO. It is derived from the user id, never persisted
// separately, and is how an admin can locate the key in the New API console.
func mintedTokenName(userID int) string {
	return fmt.Sprintf("clawmanager-u%d", userID)
}

// ListIdentityLinks returns the user-level gateway bindings for admin review,
// joined with the linked user, relay and today's trial-quota usage.
func (s *service) ListIdentityLinks() ([]IdentityLinkView, error) {
	links, err := s.repo.ListIdentityLinks()
	if err != nil {
		return nil, err
	}
	relays := map[int]*RelayKey{}
	if keys, err := s.repo.ListRelayKeys(); err == nil {
		for i := range keys {
			relays[keys[i].ID] = &keys[i]
		}
	}
	day := time.Now().Format("2006-01-02")
	views := make([]IdentityLinkView, 0, len(links))
	for _, link := range links {
		view := IdentityLinkView{
			ID:             link.ID,
			UserID:         link.UserID,
			RelayKeyID:     link.RelayKeyID,
			ExternalID:     link.ExternalID,
			UpstreamUserID: link.UpstreamUserID,
			TokenName:      mintedTokenName(link.UserID),
			HasCredential:  strings.TrimSpace(link.AccessTokenEnc) != "",
			CreatedAt:      link.CreatedAt,
			LastUsedAt:     link.LastUsedAt,
		}
		if user, err := s.userRepo.GetByID(link.UserID); err == nil && user != nil {
			view.Username = user.Username
			view.Email = user.Email
			view.Role = user.Role
		}
		if relay, ok := relays[link.RelayKeyID]; ok {
			view.RelayName = relay.Name
			view.RelayBaseURL = relay.BaseURL
			view.TodayLimit = relay.DefaultDailyLimit
		}
		if quota, err := s.repo.GetTrialQuota(link.UserID, link.RelayKeyID, day); err == nil && quota != nil {
			view.TodayUsed = quota.UsedTokens
		}
		views = append(views, view)
	}
	return views, nil
}

// UnlinkIdentityLink removes a user-level gateway binding locally. After this
// the gateway stops using that user's per-user upstream credential and falls
// back to the model's configured endpoint.
func (s *service) UnlinkIdentityLink(id int) error {
	return s.repo.DeleteIdentityLink(id)
}

// ExchangeSSO provisions a clawmanager account for a New API user (identified
// by their dashboard token) and links it to the requested relay. During the
// exchange the module mints a dedicated API token on the upstream instance for
// that user, retrieves its full key on demand, and stores it encrypted. The
// gateway later uses this per-user credential for the user's chat traffic, so
// every user consumes their own upstream quota and can be revoked
// independently. The dashboard token itself is never persisted.
func (s *service) ExchangeSSO(ctx context.Context, relayName, dashboardToken string) (*ExchangeResult, error) {
	relayName = strings.TrimSpace(relayName)
	dashboardToken = strings.TrimSpace(dashboardToken)
	if relayName == "" {
		return nil, ErrRelayInvalid
	}
	if dashboardToken == "" {
		return nil, fmt.Errorf("%w: a newapi dashboard token is required", ErrRelayInvalid)
	}

	relayKey, err := s.repo.GetRelayKeyByName(relayName)
	if err != nil {
		return nil, err
	}
	if relayKey == nil {
		return nil, ErrRelayNotFound
	}

	// Validate the dashboard token and derive the stable external identity
	// from the authenticated upstream user.
	upstream, err := s.relay.FetchSelf(ctx, relayKey.BaseURL, dashboardToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamRejected, err)
	}
	externalID := strings.TrimSpace(upstream.Email)
	if externalID == "" {
		externalID = strings.TrimSpace(upstream.Username)
	}
	if externalID == "" {
		externalID = fmt.Sprintf("uid-%d", upstream.ID)
	}

	// Idempotent re-login: reuse an existing link for this external handle.
	existing, err := s.repo.GetIdentityLinkByExternal(relayKey.ID, externalID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		_ = s.repo.TouchIdentityLink(existing.ID)
		return &ExchangeResult{
			UserID:       existing.UserID,
			RelayName:    relayKey.Name,
			RelayBaseURL: relayKey.BaseURL,
			CreatedUser:  false,
		}, nil
	}

	username := fmt.Sprintf("np_%d_%s", relayKey.ID, randomToken(4))
	userEmail := externalID
	if !strings.Contains(externalID, "@") {
		userEmail = username + "@relay.local"
	}
	user := &models.User{
		Username:     username,
		Email:        userEmail,
		PasswordHash: randomToken(24),
		Role:         "user",
		IsActive:     true,
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// Mint a dedicated upstream token for this user and retrieve its full key.
	tokenName := fmt.Sprintf("clawmanager-u%d", user.ID)
	tokenID, err := s.relay.MintToken(ctx, relayKey.BaseURL, dashboardToken, tokenName, upstream.Group)
	if err != nil {
		return nil, fmt.Errorf("newapi token minting failed: %w", err)
	}
	rawKey, err := s.relay.FetchTokenKey(ctx, relayKey.BaseURL, dashboardToken, tokenID)
	if err != nil {
		return nil, fmt.Errorf("newapi token key retrieval failed: %w", err)
	}
	encrypted, err := s.cipher.Encrypt("sk-" + rawKey)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpsertIdentityLink(&IdentityLink{
		UserID:         user.ID,
		RelayKeyID:     relayKey.ID,
		ExternalID:     externalID,
		UpstreamUserID: fmt.Sprintf("%d", upstream.ID),
		AccessTokenEnc: encrypted,
	}); err != nil {
		return nil, err
	}

	return &ExchangeResult{
		UserID:       user.ID,
		RelayName:    relayKey.Name,
		RelayBaseURL: relayKey.BaseURL,
		CreatedUser:  true,
	}, nil
}

// ResolveCredential implements integration.CredentialResolver. It resolves the
// user's own minted upstream API key and hands it out to the gateway for their
// chat traffic. Users without a link (or with an undecryptable credential)
// fall back to the model's configured endpoint, so a missing personal key can
// never take down chat traffic.
func (s *service) ResolveCredential(ctx context.Context, userID int) (*integration.CredentialOverride, error) {
	if userID <= 0 {
		return nil, nil
	}
	links, err := s.repo.ListIdentityLinksByUser(userID)
	if err != nil {
		return nil, nil
	}
	if len(links) == 0 {
		return nil, nil
	}
	for _, link := range links {
		relayKey, err := s.repo.GetRelayKeyByID(link.RelayKeyID)
		if err != nil || relayKey == nil {
			continue
		}
		accessToken, err := s.cipher.Decrypt(link.AccessTokenEnc)
		if err != nil {
			continue
		}
		return &integration.CredentialOverride{
			BaseURL: strings.TrimRight(relayKey.BaseURL, "/") + "/v1",
			APIKey:  accessToken,
		}, nil
	}
	return nil, nil
}
