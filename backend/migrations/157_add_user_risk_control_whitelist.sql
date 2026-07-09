-- Add per-user risk-control whitelist.
-- Whitelisted users are exempt from API key risk-control auto-block rules.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS risk_control_whitelisted BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_users_risk_control_whitelisted
  ON users(risk_control_whitelisted)
  WHERE deleted_at IS NULL AND risk_control_whitelisted = TRUE;
