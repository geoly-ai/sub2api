ALTER TABLE api_keys
	ADD COLUMN IF NOT EXISTS risk_blocked_reason TEXT NULL,
	ADD COLUMN IF NOT EXISTS risk_blocked_at TIMESTAMPTZ NULL;

CREATE TABLE IF NOT EXISTS api_key_risk_events (
	id BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL REFERENCES users(id),
	api_key_id BIGINT NOT NULL REFERENCES api_keys(id),
	rule_code VARCHAR(64) NOT NULL,
	severity VARCHAR(20) NOT NULL,
	score INTEGER NOT NULL DEFAULT 0,
	status VARCHAR(20) NOT NULL DEFAULT 'open',
	evidence JSONB NULL DEFAULT '{}'::jsonb,
	time_bucket TIMESTAMPTZ NOT NULL,
	blocked_at TIMESTAMPTZ NULL,
	resolved_at TIMESTAMPTZ NULL,
	resolved_by BIGINT NULL REFERENCES users(id),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_key_risk_events_dedup
	ON api_key_risk_events(api_key_id, rule_code, time_bucket);
CREATE INDEX IF NOT EXISTS idx_api_key_risk_events_user_id ON api_key_risk_events(user_id);
CREATE INDEX IF NOT EXISTS idx_api_key_risk_events_api_key_id ON api_key_risk_events(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_risk_events_rule_code ON api_key_risk_events(rule_code);
CREATE INDEX IF NOT EXISTS idx_api_key_risk_events_status ON api_key_risk_events(status);
CREATE INDEX IF NOT EXISTS idx_api_key_risk_events_created_at ON api_key_risk_events(created_at);

CREATE TABLE IF NOT EXISTS user_messages (
	id BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL REFERENCES users(id),
	type VARCHAR(50) NOT NULL,
	title VARCHAR(200) NOT NULL,
	content TEXT NOT NULL,
	status VARCHAR(20) NOT NULL DEFAULT 'unread',
	metadata JSONB NULL DEFAULT '{}'::jsonb,
	read_at TIMESTAMPTZ NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_messages_user_id ON user_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_user_messages_status ON user_messages(status);
CREATE INDEX IF NOT EXISTS idx_user_messages_created_at ON user_messages(created_at);
CREATE INDEX IF NOT EXISTS idx_user_messages_user_status_created ON user_messages(user_id, status, created_at);
