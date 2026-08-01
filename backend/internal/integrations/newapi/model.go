package newapi

import (
	"embed"
	"time"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// Migrations returns the module's embedded migration directory so the core
// migration runner can register it as an extra (additive) migration source.
func Migrations() (embed.FS, string) {
	return embeddedMigrations, "migrations"
}

// RelayKey is an admin-managed New API relay registration. The relay token is
// encrypted at rest with the module's dedicated encryption key.
type RelayKey struct {
	ID                int       `db:"id,primarykey,autoincrement" json:"id"`
	Name              string    `db:"name" json:"name"`
	BaseURL           string    `db:"base_url" json:"base_url"`
	RelayTokenEnc     string    `db:"relay_token_enc" json:"-"`
	DefaultDailyLimit int64     `db:"default_daily_limit" json:"default_daily_limit"`
	CreatedBy         int       `db:"created_by" json:"created_by"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

// TableName returns the table name.
func (RelayKey) TableName() string { return "newapi_relay_keys" }

// IdentityLink maps a clawmanager user to a relay account. The relay access
// token is the shared relay API key, encrypted at rest with the module's
// dedicated encryption key. ExternalID is the stable upstream identity handle
// (e.g. the user's email) used to keep re-login idempotent across sessions.
type IdentityLink struct {
	ID             int       `db:"id,primarykey,autoincrement" json:"id"`
	UserID         int       `db:"user_id" json:"user_id"`
	RelayKeyID     int       `db:"relay_key_id" json:"relay_key_id"`
	ExternalID     string    `db:"external_id" json:"external_id,omitempty"`
	UpstreamUserID string    `db:"upstream_user_id" json:"upstream_user_id,omitempty"`
	AccessTokenEnc string    `db:"access_token_enc" json:"-"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
	LastUsedAt     *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
}

// TableName returns the table name.
func (IdentityLink) TableName() string { return "newapi_identity_links" }

// TrialQuota tracks daily token usage for a relay user so the module can
// enforce its own four-layer circuit breakers without touching core quotas.
type TrialQuota struct {
	ID         int       `db:"id,primarykey,autoincrement" json:"id"`
	UserID     int       `db:"user_id" json:"user_id"`
	RelayKeyID int       `db:"relay_key_id" json:"relay_key_id"`
	DailyLimit int64     `db:"daily_limit" json:"daily_limit"`
	DayKey     string    `db:"day_key" json:"day_key"`
	UsedTokens int64     `db:"used_tokens" json:"used_tokens"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// TableName returns the table name.
func (TrialQuota) TableName() string { return "newapi_trial_quotas" }
