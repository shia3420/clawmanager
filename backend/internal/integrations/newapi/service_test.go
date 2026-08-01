package newapi

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"clawreef/internal/integration"
	"clawreef/internal/models"
)

// --- fakes ---

type fakeRepo struct {
	mu          sync.Mutex
	relays      map[int]*RelayKey
	links       map[int]*IdentityLink
	quotas      map[string]*TrialQuota
	nextRelayID int
	nextLinkID  int
	nextQuotaID int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		relays: map[int]*RelayKey{},
		links:  map[int]*IdentityLink{},
		quotas: map[string]*TrialQuota{},
	}
}

func (f *fakeRepo) CreateRelayKey(key *RelayKey) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextRelayID++
	key.ID = f.nextRelayID
	f.relays[key.ID] = key
	return nil
}
func (f *fakeRepo) GetRelayKeyByID(id int) (*RelayKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if k, ok := f.relays[id]; ok {
		cp := *k
		return &cp, nil
	}
	return nil, nil
}
func (f *fakeRepo) GetRelayKeyByName(name string) (*RelayKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range f.relays {
		if k.Name == name {
			cp := *k
			return &cp, nil
		}
	}
	return nil, nil
}
func (f *fakeRepo) ListRelayKeys() ([]RelayKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]RelayKey, 0, len(f.relays))
	for _, k := range f.relays {
		out = append(out, *k)
	}
	return out, nil
}
func (f *fakeRepo) DeleteRelayKey(id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.relays, id)
	return nil
}
func (f *fakeRepo) GetIdentityLink(userID, relayKeyID int) (*IdentityLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range f.links {
		if l.UserID == userID && l.RelayKeyID == relayKeyID {
			cp := *l
			return &cp, nil
		}
	}
	return nil, nil
}
func (f *fakeRepo) ListIdentityLinksByUser(userID int) ([]IdentityLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]IdentityLink, 0)
	for _, l := range f.links {
		if l.UserID == userID {
			out = append(out, *l)
		}
	}
	return out, nil
}
func (f *fakeRepo) UpsertIdentityLink(link *IdentityLink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, l := range f.links {
		if l.UserID == link.UserID && l.RelayKeyID == link.RelayKeyID {
			cp := *link
			cp.ID = l.ID
			f.links[id] = &cp
			return nil
		}
	}
	f.nextLinkID++
	link.ID = f.nextLinkID
	f.links[link.ID] = link
	return nil
}
func (f *fakeRepo) TouchIdentityLink(id int) error { return nil }
func (f *fakeRepo) GetTrialQuota(userID, relayKeyID int, dayKey string) (*TrialQuota, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%d-%d-%s", userID, relayKeyID, dayKey)
	if q, ok := f.quotas[key]; ok {
		cp := *q
		return &cp, nil
	}
	return nil, nil
}
func (f *fakeRepo) IncrementTrialQuotaUsed(id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, q := range f.quotas {
		if q.ID == id {
			q.UsedTokens++
			return nil
		}
	}
	return errors.New("quota row not found")
}
func (f *fakeRepo) UpsertTrialQuota(quota *TrialQuota) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%d-%d-%s", quota.UserID, quota.RelayKeyID, quota.DayKey)
	if existing, ok := f.quotas[key]; ok {
		existing.UsedTokens = quota.UsedTokens
		existing.DailyLimit = quota.DailyLimit
		quota.ID = existing.ID
		return nil
	}
	f.nextQuotaID++
	quota.ID = f.nextQuotaID
	f.quotas[key] = quota
	return nil
}

type fakeUserStore struct {
	mu      sync.Mutex
	users   map[string]*models.User
	nextID  int
	created []*models.User
}

func (f *fakeUserStore) Create(user *models.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	user.ID = f.nextID
	f.users[user.Username] = user
	f.created = append(f.created, user)
	return nil
}
func (f *fakeUserStore) GetByUsername(username string) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[username]; ok {
		return u, nil
	}
	return nil, nil
}

type fakeRelay struct {
	selfID int
	minted []string
	key    string
	err    error
}

func (f *fakeRelay) FetchTokenSelf(ctx context.Context, baseURL, accessToken string) (TokenSelf, error) {
	if f.err != nil {
		return TokenSelf{}, f.err
	}
	return TokenSelf{ID: f.selfID, Name: "relay-user"}, nil
}
func (f *fakeRelay) MintToken(ctx context.Context, baseURL, relayToken, name string, budget int64) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.minted = append(f.minted, name)
	return f.key, nil
}

func testService(t *testing.T, repo *fakeRepo, users *fakeUserStore, relay RelayClient) Service {
	t.Helper()
	svc, err := NewService(repo, users, relay, Config{
		EncryptionKey: "test-module-encryption-key-32-bytes-long!!",
		JWTSecret:     "jwt",
		JWTExpiry:     time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

// --- tests ---

func TestCipherRoundTrip(t *testing.T) {
	c, err := newCipher("some-module-secret")
	if err != nil {
		t.Fatalf("newCipher: %v", err)
	}
	secret := "sk-test-abcdef123456"
	enc, err := c.Encrypt(secret)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == secret {
		t.Fatal("encrypted value must differ from plaintext")
	}
	dec, err := c.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if dec != secret {
		t.Fatalf("roundtrip mismatch: got %q want %q", dec, secret)
	}
	if _, err := c.Decrypt("not-base64!"); err == nil {
		t.Fatal("expected error for garbage ciphertext")
	}
}

func TestNewServiceRequiresKey(t *testing.T) {
	if _, err := NewService(nil, nil, nil, Config{}); err == nil {
		t.Fatal("expected ErrModuleNotReady for empty encryption key")
	}
}

func TestExchangeSSO(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	relay := &fakeRelay{selfID: 42, key: "sk-minted-1"}

	// register a relay directly (encrypt token with a known cipher)
	cipher, _ := newCipher("test-module-encryption-key-32-bytes-long!!")
	encRelay, _ := cipher.Encrypt("sk-relay-admin")
	repo.CreateRelayKey(&RelayKey{Name: "prod", BaseURL: "https://relay.example.com", RelayTokenEnc: encRelay, DefaultDailyLimit: 1000, CreatedBy: 1})

	svc := testService(t, repo, users, relay)
	res, err := svc.ExchangeSSO(context.Background(), "prod", "sk-user-access", "user@example.com")
	if err != nil {
		t.Fatalf("ExchangeSSO: %v", err)
	}
	if !res.CreatedUser {
		t.Fatal("expected a new user to be created")
	}
	if res.UserID == 0 {
		t.Fatal("expected a valid user id")
	}
	// user's own access token must never be stored
	for _, u := range users.created {
		if u.PasswordHash == "sk-user-access" || u.Email == "sk-user-access" {
			t.Fatal("user's access token leaked into account")
		}
	}

	// subsequent exchange must reuse the same account
	res2, err := svc.ExchangeSSO(context.Background(), "prod", "sk-user-access-2", "")
	if err != nil {
		t.Fatalf("ExchangeSSO second time: %v", err)
	}
	if res2.CreatedUser {
		t.Fatal("second exchange must reuse the existing account")
	}
	if res2.UserID != res.UserID {
		t.Fatalf("user id changed between exchanges: %d != %d", res2.UserID, res.UserID)
	}
}

func TestExchangeSSOUnknownRelay(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	svc := testService(t, repo, users, &fakeRelay{})
	if _, err := svc.ExchangeSSO(context.Background(), "missing", "sk-x", ""); !errors.Is(err, ErrRelayNotFound) {
		t.Fatalf("expected ErrRelayNotFound, got %v", err)
	}
}

func TestResolveCredentialAndQuota(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	relay := &fakeRelay{selfID: 7, key: "sk-minted-9"}

	cipher, _ := newCipher("test-module-encryption-key-32-bytes-long!!")
	encRelay, _ := cipher.Encrypt("sk-relay-admin")
	repo.CreateRelayKey(&RelayKey{Name: "prod", BaseURL: "https://relay.example.com", RelayTokenEnc: encRelay, DefaultDailyLimit: 2, CreatedBy: 1})

	svc := testService(t, repo, users, relay)
	res, err := svc.ExchangeSSO(context.Background(), "prod", "sk-user", "")
	if err != nil {
		t.Fatalf("ExchangeSSO: %v", err)
	}

	// first two resolutions succeed and count against the daily breaker
	ov1, err := svc.ResolveCredential(context.Background(), res.UserID)
	if err != nil {
		t.Fatalf("ResolveCredential #1: %v", err)
	}
	if ov1 == nil || ov1.BaseURL != "https://relay.example.com/v1" || ov1.APIKey != "sk-minted-9" {
		t.Fatalf("unexpected override: %+v", ov1)
	}
	if _, err := svc.ResolveCredential(context.Background(), res.UserID); err != nil {
		t.Fatalf("ResolveCredential #2: %v", err)
	}
	// third resolution must trip the breaker
	if _, err := svc.ResolveCredential(context.Background(), res.UserID); !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("expected ErrQuotaExhausted, got %v", err)
	}

	// a user with no link falls back to model credentials (nil, nil)
	ovNil, err := svc.ResolveCredential(context.Background(), 99999)
	if err != nil || ovNil != nil {
		t.Fatalf("expected (nil, nil) for unlinked user, got (%+v, %v)", ovNil, err)
	}
}

var _ integration.CredentialResolver = (Service)(nil)
