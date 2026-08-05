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
func (f *fakeRepo) GetIdentityLinkByExternal(relayKeyID int, externalID string) (*IdentityLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range f.links {
		if l.RelayKeyID == relayKeyID && l.ExternalID == externalID {
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
func (f *fakeRepo) ListIdentityLinks() ([]IdentityLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]IdentityLink, 0, len(f.links))
	for _, l := range f.links {
		out = append(out, *l)
	}
	return out, nil
}
func (f *fakeRepo) DeleteIdentityLink(id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.links[id]; !ok {
		return ErrIdentityLinkNotFound
	}
	delete(f.links, id)
	return nil
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
func (f *fakeUserStore) GetByID(id int) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

type fakeRelay struct {
	valid    bool
	err      error
	self     *UpstreamUser
	mintID   int
	fullKey  string
	mintName string
}

func (f *fakeRelay) ValidateCredential(ctx context.Context, baseURL, apiKey string) error {
	if f.err != nil {
		return f.err
	}
	if !f.valid {
		return ErrUpstreamRejected
	}
	return nil
}

func (f *fakeRelay) FetchSelf(ctx context.Context, baseURL, dashboardToken string) (*UpstreamUser, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.self == nil {
		return nil, ErrUpstreamRejected
	}
	return f.self, nil
}

func (f *fakeRelay) MintToken(ctx context.Context, baseURL, dashboardToken, name, group string) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.mintName = name
	return f.mintID, nil
}

func (f *fakeRelay) FetchTokenKey(ctx context.Context, baseURL, dashboardToken string, tokenID int) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.fullKey, nil
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

// registerRelay inserts a relay directly with a known cipher, mirroring what
// CreateRelayKey does for the admin-facing path.
func registerRelay(t *testing.T, repo *fakeRepo, name, baseURL string) {
	t.Helper()
	cipher, err := newCipher("test-module-encryption-key-32-bytes-long!!")
	if err != nil {
		t.Fatalf("newCipher: %v", err)
	}
	encRelay, err := cipher.Encrypt("sk-relay-admin")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if err := repo.CreateRelayKey(&RelayKey{Name: name, BaseURL: baseURL, RelayTokenEnc: encRelay, DefaultDailyLimit: 1000, CreatedBy: 1}); err != nil {
		t.Fatalf("CreateRelayKey: %v", err)
	}
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

func TestExchangeSSOMintsPersonalKey(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	registerRelay(t, repo, "prod", "https://relay.example.com")

	relay := &fakeRelay{
		self:    &UpstreamUser{ID: 7, Username: "alice", Email: "alice@example.com", Group: "vip"},
		mintID:  42,
		fullKey: "rawkey1234567890",
	}
	svc := testService(t, repo, users, relay)

	res, err := svc.ExchangeSSO(context.Background(), "prod", "some-dashboard-jwt")
	if err != nil {
		t.Fatalf("ExchangeSSO: %v", err)
	}
	if !res.CreatedUser {
		t.Fatal("expected a new user to be created")
	}
	if res.UserID == 0 {
		t.Fatal("expected a valid user id")
	}
	if relay.mintName != "clawmanager-u1" {
		t.Fatalf("unexpected minted token name: %s", relay.mintName)
	}

	// the minted personal key must be stored encrypted (never plaintext)
	links, _ := repo.ListIdentityLinksByUser(res.UserID)
	if len(links) != 1 {
		t.Fatalf("expected exactly one identity link, got %d", len(links))
	}
	cipher, _ := newCipher("test-module-encryption-key-32-bytes-long!!")
	dec, err := cipher.Decrypt(links[0].AccessTokenEnc)
	if err != nil {
		t.Fatalf("Decrypt stored personal key: %v", err)
	}
	if dec != "sk-rawkey1234567890" {
		t.Fatalf("stored personal key mismatch: got %q", dec)
	}
	if links[0].ExternalID != "alice@example.com" {
		t.Fatalf("external id mismatch: %q", links[0].ExternalID)
	}
	if links[0].UpstreamUserID != "7" {
		t.Fatalf("upstream user id mismatch: %q", links[0].UpstreamUserID)
	}
	// the shared relay key must never leak into the provisioned account
	for _, u := range users.created {
		if u.PasswordHash == "sk-relay-admin" || u.Email == "sk-relay-admin" {
			t.Fatal("shared relay key leaked into account")
		}
	}

	// subsequent exchange with the same upstream identity must reuse the account
	res2, err := svc.ExchangeSSO(context.Background(), "prod", "some-dashboard-jwt")
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

func TestExchangeSSORequiresToken(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	svc := testService(t, repo, users, &fakeRelay{})
	if _, err := svc.ExchangeSSO(context.Background(), "prod", ""); !errors.Is(err, ErrRelayInvalid) {
		t.Fatalf("expected ErrRelayInvalid for empty dashboard token, got %v", err)
	}
}

func TestExchangeSSOUnknownRelay(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	svc := testService(t, repo, users, &fakeRelay{})
	if _, err := svc.ExchangeSSO(context.Background(), "missing", "some-dashboard-jwt"); !errors.Is(err, ErrRelayNotFound) {
		t.Fatalf("expected ErrRelayNotFound, got %v", err)
	}
}

func TestExchangeSSORejectsBadToken(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	registerRelay(t, repo, "prod", "https://relay.example.com")
	svc := testService(t, repo, users, &fakeRelay{})
	if _, err := svc.ExchangeSSO(context.Background(), "prod", "bad-jwt"); !errors.Is(err, ErrUpstreamRejected) {
		t.Fatalf("expected ErrUpstreamRejected for invalid dashboard token, got %v", err)
	}
}

func TestCreateRelayKeyValidatesCredential(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	relay := &fakeRelay{valid: false}
	svc := testService(t, repo, users, relay)

	if err := svc.CreateRelayKey(context.Background(), "bad", "https://relay.example.com", "sk-invalid", 1000, 1); !errors.Is(err, ErrUpstreamRejected) {
		t.Fatalf("expected ErrUpstreamRejected for invalid credential, got %v", err)
	}
	if k, _ := repo.GetRelayKeyByName("bad"); k != nil {
		t.Fatal("invalid relay must not be registered")
	}

	relay.valid = true
	if err := svc.CreateRelayKey(context.Background(), "good", "https://relay.example.com", "sk-good", 1000, 1); err != nil {
		t.Fatalf("valid relay registration failed: %v", err)
	}
}

func TestResolveCredentialUsesPersonalKey(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	registerRelay(t, repo, "prod", "https://relay.example.com")

	relay := &fakeRelay{
		self:    &UpstreamUser{ID: 7, Username: "alice", Email: "alice@example.com", Group: "default"},
		mintID:  42,
		fullKey: "rawkey1234567890",
	}
	svc := testService(t, repo, users, relay)
	res, err := svc.ExchangeSSO(context.Background(), "prod", "some-dashboard-jwt")
	if err != nil {
		t.Fatalf("ExchangeSSO: %v", err)
	}

	// the gateway must receive the user's own key, not the shared relay key
	ov, err := svc.ResolveCredential(context.Background(), res.UserID)
	if err != nil {
		t.Fatalf("ResolveCredential: %v", err)
	}
	if ov == nil {
		t.Fatal("expected a credential override")
	}
	if ov.BaseURL != "https://relay.example.com/v1" {
		t.Fatalf("unexpected base url: %s", ov.BaseURL)
	}
	if ov.APIKey != "sk-rawkey1234567890" {
		t.Fatalf("expected the user's personal key, got %q", ov.APIKey)
	}

	// repeated resolution is always allowed (no shared-key breaker)
	for i := 0; i < 3; i++ {
		if _, err := svc.ResolveCredential(context.Background(), res.UserID); err != nil {
			t.Fatalf("ResolveCredential #%d: %v", i, err)
		}
	}

	// a user with no link falls back to model credentials (nil, nil)
	ovNil, err := svc.ResolveCredential(context.Background(), 99999)
	if err != nil || ovNil != nil {
		t.Fatalf("expected (nil, nil) for unlinked user, got (%+v, %v)", ovNil, err)
	}
}

func TestResolveCredentialSkipsBrokenLink(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	registerRelay(t, repo, "prod", "https://relay.example.com")

	relay := &fakeRelay{
		self:    &UpstreamUser{ID: 7, Username: "alice", Email: "alice@example.com", Group: "default"},
		mintID:  42,
		fullKey: "rawkey1234567890",
	}
	svc := testService(t, repo, users, relay)
	res, err := svc.ExchangeSSO(context.Background(), "prod", "some-dashboard-jwt")
	if err != nil {
		t.Fatalf("ExchangeSSO: %v", err)
	}

	// corrupt the stored personal key so decryption fails
	links, _ := repo.ListIdentityLinksByUser(res.UserID)
	links[0].AccessTokenEnc = "not-a-ciphertext"
	_ = repo.UpsertIdentityLink(&links[0])

	ov, err := svc.ResolveCredential(context.Background(), res.UserID)
	if err != nil {
		t.Fatalf("ResolveCredential with broken link: %v", err)
	}
	if ov != nil {
		t.Fatalf("expected fallback (nil) for undecryptable link, got %+v", ov)
	}
}

var _ integration.CredentialResolver = (Service)(nil)

func TestListIdentityLinksJoinsUserAndUsage(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	registerRelay(t, repo, "prod", "https://relay.example.com")

	relay := &fakeRelay{
		self:    &UpstreamUser{ID: 7, Username: "alice", Email: "alice@example.com", Group: "vip"},
		mintID:  42,
		fullKey: "rawkey1234567890",
	}
	svc := testService(t, repo, users, relay)
	res, err := svc.ExchangeSSO(context.Background(), "prod", "some-dashboard-jwt")
	if err != nil {
		t.Fatalf("ExchangeSSO: %v", err)
	}

	key, _ := repo.GetRelayKeyByName("prod")
	day := time.Now().Format("2006-01-02")
	_ = repo.UpsertTrialQuota(&TrialQuota{UserID: res.UserID, RelayKeyID: key.ID, DayKey: day, UsedTokens: 123})

	views, err := svc.ListIdentityLinks()
	if err != nil {
		t.Fatalf("ListIdentityLinks: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected one view, got %d", len(views))
	}
	v := views[0]
	if v.Username == "" {
		t.Fatal("expected the linked user's username to be joined")
	}
	if v.RelayName != "prod" || v.RelayBaseURL != "https://relay.example.com" {
		t.Fatalf("unexpected relay info: %+v", v)
	}
	if v.TokenName != "clawmanager-u1" {
		t.Fatalf("unexpected token name: %s", v.TokenName)
	}
	if !v.HasCredential {
		t.Fatal("expected credential presence to be reported")
	}
	if v.TodayUsed != 123 {
		t.Fatalf("expected today usage 123, got %d", v.TodayUsed)
	}
	if v.TodayLimit == 0 {
		t.Fatal("expected the relay daily limit to be joined")
	}
}

func TestUnlinkIdentityLink(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUserStore{users: map[string]*models.User{}}
	registerRelay(t, repo, "prod", "https://relay.example.com")

	relay := &fakeRelay{
		self:    &UpstreamUser{ID: 7, Username: "alice", Email: "alice@example.com", Group: "vip"},
		mintID:  42,
		fullKey: "rawkey1234567890",
	}
	svc := testService(t, repo, users, relay)
	res, err := svc.ExchangeSSO(context.Background(), "prod", "some-dashboard-jwt")
	if err != nil {
		t.Fatalf("ExchangeSSO: %v", err)
	}

	links, _ := repo.ListIdentityLinksByUser(res.UserID)
	if len(links) != 1 {
		t.Fatalf("expected one link before unlink, got %d", len(links))
	}
	linkID := links[0].ID

	// resolving still works while linked
	if ov, err := svc.ResolveCredential(context.Background(), res.UserID); err != nil || ov == nil {
		t.Fatalf("expected credential before unlink, got (%+v, %v)", ov, err)
	}

	if err := svc.UnlinkIdentityLink(linkID); err != nil {
		t.Fatalf("UnlinkIdentityLink: %v", err)
	}
	after, _ := repo.ListIdentityLinksByUser(res.UserID)
	if len(after) != 0 {
		t.Fatalf("expected link to be removed, got %d links", len(after))
	}

	// unlinking an unknown id must be a sentinel error
	if err := svc.UnlinkIdentityLink(linkID); !errors.Is(err, ErrIdentityLinkNotFound) {
		t.Fatalf("expected ErrIdentityLinkNotFound, got %v", err)
	}

	// the user falls back to model credentials after unlink
	ov, err := svc.ResolveCredential(context.Background(), res.UserID)
	if err != nil || ov != nil {
		t.Fatalf("expected (nil, nil) after unlink, got (%+v, %v)", ov, err)
	}
}
