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
	GetByUsername(username string) (*models.User, error)
}

// Service errors. Exposed as sentinel errors so handlers/tests can branch.
var (
	ErrRelayNotFound    = errors.New("newapi relay not found")
	ErrRelayInvalid     = errors.New("newapi relay registration is invalid")
	ErrUpstreamRejected = errors.New("newapi relay rejected the credential")
	ErrQuotaExhausted   = errors.New("newapi relay daily quota exhausted")
	ErrModuleNotReady   = errors.New("newapi module encryption key is not configured")
)

// Config carries the module's own settings (never part of the core Config).
type Config struct {
	EncryptionKey string
	JWTSecret     string
	JWTExpiry     time.Duration
}

// ExchangeResult is returned to the SSO handler. The relay credential is
// intentionally NOT included: it is the shared relay API key, stored encrypted
// and resolved server-side.
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

// Service implements the module's business logic.
type Service interface {
	CreateRelayKey(ctx context.Context, name, baseURL, relayKey string, dailyLimit int64, createdBy int) error
	ListRelayKeys() ([]RelayKeyView, error)
	DeleteRelayKey(id int) error
	ExchangeSSO(ctx context.Context, relayName, externalID string) (*ExchangeResult, error)
	ResolveCredential(ctx context.Context, userID int) (*integration.CredentialOverride, error)
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

func dayKey(t time.Time) string { return t.UTC().Format("20060102") }

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

// ExchangeSSO lazily provisions a clawmanager account for an external user
// (identified by a stable external handle such as an email) and links it to the
// requested relay. The link makes the gateway route the user's chat traffic
// through the relay account's shared API key, bounded by the module's per-user
// daily request breaker. The user's own New API credential is never required,
// stored, or minted.
func (s *service) ExchangeSSO(ctx context.Context, relayName, externalID string) (*ExchangeResult, error) {
	relayName = strings.TrimSpace(relayName)
	externalID = strings.TrimSpace(externalID)
	if relayName == "" {
		return nil, ErrRelayInvalid
	}
	if externalID == "" {
		return nil, fmt.Errorf("%w: external id (email) is required", ErrRelayInvalid)
	}

	relayKey, err := s.repo.GetRelayKeyByName(relayName)
	if err != nil {
		return nil, err
	}
	if relayKey == nil {
		return nil, ErrRelayNotFound
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
	if strings.Contains(externalID, "@") {
		userEmail = externalID
	} else {
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

	if err := s.repo.UpsertIdentityLink(&IdentityLink{
		UserID:         user.ID,
		RelayKeyID:     relayKey.ID,
		ExternalID:     externalID,
		UpstreamUserID: "",
		AccessTokenEnc: relayKey.RelayTokenEnc,
	}); err != nil {
		return nil, err
	}

	if err := s.repo.UpsertTrialQuota(&TrialQuota{
		UserID:     user.ID,
		RelayKeyID: relayKey.ID,
		DailyLimit: relayKey.DefaultDailyLimit,
		DayKey:     dayKey(time.Now()),
		UsedTokens: 0,
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
// shared relay API key for linked users and enforces the module's per-user
// daily request breaker before handing out the credential.
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

		today := dayKey(time.Now())
		quota, err := s.repo.GetTrialQuota(userID, relayKey.ID, today)
		if err == nil && quota != nil && quota.DailyLimit > 0 && quota.UsedTokens >= quota.DailyLimit {
			continue
		}
		if err == nil && quota != nil && quota.DailyLimit > 0 {
			if incErr := s.repo.IncrementTrialQuotaUsed(quota.ID); incErr == nil {
				_ = s.repo.TouchIdentityLink(link.ID)
			}
		}
		return &integration.CredentialOverride{
			BaseURL: strings.TrimRight(relayKey.BaseURL, "/") + "/v1",
			APIKey:  accessToken,
		}, nil
	}
	return nil, ErrQuotaExhausted
}
