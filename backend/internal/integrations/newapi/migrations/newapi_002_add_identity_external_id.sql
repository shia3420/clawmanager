-- newapi_002: identity links gain a stable external handle (e.g. email) so
-- re-login stays idempotent. The table is new (created by newapi_001) so the
-- column add and unique index are safe to apply on fresh and existing installs.

ALTER TABLE newapi_identity_links
  ADD COLUMN external_id VARCHAR(320) DEFAULT '' AFTER relay_key_id;

ALTER TABLE newapi_identity_links
  ADD UNIQUE KEY uk_newapi_identity_relay_external (relay_key_id, external_id);
