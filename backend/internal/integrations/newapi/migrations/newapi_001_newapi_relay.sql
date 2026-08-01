-- newapi_001: integration module tables for the New API token relay / SSO bridge.
-- All tables use the newapi_ prefix and live in their own embedded migration
-- directory (registered as an extra migration source), so they never collide
-- with core migrations. Managed keys are encrypted at rest with the module's
-- own encryption key (NEWAPI_MODULE_ENCRYPTION_KEY), never with the JWT secret.

CREATE TABLE IF NOT EXISTS newapi_relay_keys (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  base_url VARCHAR(512) NOT NULL,
  relay_token_enc LONGTEXT NOT NULL,
  default_daily_limit BIGINT NOT NULL DEFAULT 100000,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_newapi_relay_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS newapi_identity_links (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  relay_key_id INT NOT NULL,
  upstream_user_id VARCHAR(128) DEFAULT '',
  access_token_enc LONGTEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP NULL,
  UNIQUE KEY uk_newapi_identity_user_relay (user_id, relay_key_id),
  KEY idx_newapi_identity_relay (relay_key_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS newapi_trial_quotas (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  relay_key_id INT NOT NULL,
  daily_limit BIGINT NOT NULL DEFAULT 0,
  day_key VARCHAR(10) NOT NULL,
  used_tokens BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_newapi_quota_user_relay_day (user_id, relay_key_id, day_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
