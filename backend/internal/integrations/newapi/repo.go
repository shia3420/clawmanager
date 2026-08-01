package newapi

import (
	"errors"
	"time"

	"github.com/upper/db/v4"
)

// Repository owns all newapi_* table access. It only depends on the upper/db
// session, mirroring the core repository pattern.
type Repository interface {
	CreateRelayKey(key *RelayKey) error
	GetRelayKeyByID(id int) (*RelayKey, error)
	GetRelayKeyByName(name string) (*RelayKey, error)
	ListRelayKeys() ([]RelayKey, error)
	DeleteRelayKey(id int) error

	GetIdentityLink(userID, relayKeyID int) (*IdentityLink, error)
	GetIdentityLinkByExternal(relayKeyID int, externalID string) (*IdentityLink, error)
	ListIdentityLinksByUser(userID int) ([]IdentityLink, error)
	UpsertIdentityLink(link *IdentityLink) error
	TouchIdentityLink(id int) error

	GetTrialQuota(userID, relayKeyID int, dayKey string) (*TrialQuota, error)
	IncrementTrialQuotaUsed(id int) error
	UpsertTrialQuota(quota *TrialQuota) error
}

type relayRepository struct {
	sess db.Session
}

// NewRepository creates a new newapi module repository.
func NewRepository(sess db.Session) Repository {
	return &relayRepository{sess: sess}
}

// ensureTimestamps mirrors the core repository helper so the module can set
// insert timestamps without importing the core repository package (keeping the
// dependency direction one-way).
func ensureTimestamps(createdAt, updatedAt *time.Time) {
	now := time.Now().UTC()
	if createdAt != nil && createdAt.IsZero() {
		*createdAt = now
	}
	if updatedAt != nil && updatedAt.IsZero() {
		*updatedAt = now
	}
}

func (r *relayRepository) relayKeys() db.Collection { return r.sess.Collection("newapi_relay_keys") }
func (r *relayRepository) identityLinks() db.Collection {
	return r.sess.Collection("newapi_identity_links")
}
func (r *relayRepository) trialQuotas() db.Collection {
	return r.sess.Collection("newapi_trial_quotas")
}

func (r *relayRepository) CreateRelayKey(key *RelayKey) error {
	ensureTimestamps(&key.CreatedAt, &key.UpdatedAt)
	res, err := r.relayKeys().Insert(key)
	if err != nil {
		return err
	}
	key.ID = int(res.ID().(int64))
	return nil
}

func (r *relayRepository) GetRelayKeyByID(id int) (*RelayKey, error) {
	var key RelayKey
	err := r.relayKeys().Find(db.Cond{"id": id}).One(&key)
	if err != nil {
		if errors.Is(err, db.ErrNoMoreRows) {
			return nil, nil
		}
		return nil, err
	}
	return &key, nil
}

func (r *relayRepository) GetRelayKeyByName(name string) (*RelayKey, error) {
	var key RelayKey
	err := r.relayKeys().Find(db.Cond{"name": name}).One(&key)
	if err != nil {
		if errors.Is(err, db.ErrNoMoreRows) {
			return nil, nil
		}
		return nil, err
	}
	return &key, nil
}

func (r *relayRepository) ListRelayKeys() ([]RelayKey, error) {
	var keys []RelayKey
	err := r.relayKeys().Find().OrderBy("id").All(&keys)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *relayRepository) DeleteRelayKey(id int) error {
	if err := r.relayKeys().Find(db.Cond{"id": id}).Delete(); err != nil {
		return err
	}
	_ = r.identityLinks().Find(db.Cond{"relay_key_id": id}).Delete()
	_ = r.trialQuotas().Find(db.Cond{"relay_key_id": id}).Delete()
	return nil
}

func (r *relayRepository) GetIdentityLink(userID, relayKeyID int) (*IdentityLink, error) {
	var link IdentityLink
	err := r.identityLinks().Find(db.Cond{"user_id": userID, "relay_key_id": relayKeyID}).One(&link)
	if err != nil {
		if errors.Is(err, db.ErrNoMoreRows) {
			return nil, nil
		}
		return nil, err
	}
	return &link, nil
}

func (r *relayRepository) GetIdentityLinkByExternal(relayKeyID int, externalID string) (*IdentityLink, error) {
	var link IdentityLink
	err := r.identityLinks().Find(db.Cond{"relay_key_id": relayKeyID, "external_id": externalID}).One(&link)
	if err != nil {
		if errors.Is(err, db.ErrNoMoreRows) {
			return nil, nil
		}
		return nil, err
	}
	return &link, nil
}

func (r *relayRepository) ListIdentityLinksByUser(userID int) ([]IdentityLink, error) {
	var links []IdentityLink
	err := r.identityLinks().Find(db.Cond{"user_id": userID}).OrderBy("-last_used_at", "id").All(&links)
	if err != nil {
		return nil, err
	}
	return links, nil
}

func (r *relayRepository) UpsertIdentityLink(link *IdentityLink) error {
	existing, err := r.GetIdentityLink(link.UserID, link.RelayKeyID)
	if err != nil {
		return err
	}
	if existing == nil {
		ensureTimestamps(&link.CreatedAt, &link.UpdatedAt)
		res, err := r.identityLinks().Insert(link)
		if err != nil {
			return err
		}
		link.ID = int(res.ID().(int64))
		return nil
	}
	link.ID = existing.ID
	return r.identityLinks().Find(db.Cond{"id": existing.ID}).Update(map[string]interface{}{
		"external_id":      link.ExternalID,
		"upstream_user_id": link.UpstreamUserID,
		"access_token_enc": link.AccessTokenEnc,
	})
}

func (r *relayRepository) TouchIdentityLink(id int) error {
	return r.identityLinks().Find(db.Cond{"id": id}).Update(map[string]interface{}{
		"last_used_at": db.Raw("NOW()"),
	})
}

func (r *relayRepository) GetTrialQuota(userID, relayKeyID int, dayKey string) (*TrialQuota, error) {
	var quota TrialQuota
	err := r.trialQuotas().Find(db.Cond{"user_id": userID, "relay_key_id": relayKeyID, "day_key": dayKey}).One(&quota)
	if err != nil {
		if errors.Is(err, db.ErrNoMoreRows) {
			return nil, nil
		}
		return nil, err
	}
	return &quota, nil
}

func (r *relayRepository) IncrementTrialQuotaUsed(id int) error {
	return r.trialQuotas().Find(db.Cond{"id": id}).Update(map[string]interface{}{
		"used_tokens": db.Raw("used_tokens + 1"),
	})
}

func (r *relayRepository) UpsertTrialQuota(quota *TrialQuota) error {
	existing, err := r.GetTrialQuota(quota.UserID, quota.RelayKeyID, quota.DayKey)
	if err != nil {
		return err
	}
	if existing == nil {
		ensureTimestamps(&quota.CreatedAt, &quota.UpdatedAt)
		res, err := r.trialQuotas().Insert(quota)
		if err != nil {
			return err
		}
		quota.ID = int(res.ID().(int64))
		return nil
	}
	quota.ID = existing.ID
	return r.trialQuotas().Find(db.Cond{"id": existing.ID}).Update(map[string]interface{}{
		"daily_limit": quota.DailyLimit,
		"used_tokens": quota.UsedTokens,
	})
}
