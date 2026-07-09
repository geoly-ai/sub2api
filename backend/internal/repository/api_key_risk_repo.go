package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type apiKeyRiskRepository struct {
	db *sql.DB
}

func NewAPIKeyRiskRepository(db *sql.DB) service.APIKeyRiskRepository {
	return &apiKeyRiskRepository{db: db}
}

func (r *apiKeyRiskRepository) TryAcquireScanLock(ctx context.Context, lockID int64) (bool, error) {
	var locked bool
	err := r.db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&locked)
	return locked, err
}

func (r *apiKeyRiskRepository) ReleaseScanLock(ctx context.Context, lockID int64) error {
	_, err := r.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lockID)
	return err
}

func (r *apiKeyRiskRepository) ListCandidates(ctx context.Context, now time.Time) ([]service.APIKeyRiskCandidate, error) {
	windowStart := now.Add(-60 * time.Minute)
	window30 := now.Add(-30 * time.Minute)
	query := `
WITH key_stats AS (
	SELECT
		ul.user_id,
		u.email AS user_email,
		ul.api_key_id,
		ak.name AS api_key_name,
		ak.key AS api_key,
		COUNT(*) FILTER (WHERE ul.created_at >= $2) AS requests_30m,
		COUNT(*) AS requests_60m,
		COALESCE(array_remove(array_agg(DISTINCT ul.ip_address) FILTER (WHERE ul.created_at >= $2), NULL), ARRAY[]::text[]) AS ips_30m,
		COALESCE(array_remove(array_agg(DISTINCT ul.ip_address), NULL), ARRAY[]::text[]) AS ips_60m,
		COALESCE(array_remove(array_agg(DISTINCT ul.user_agent), NULL), ARRAY[]::text[]) AS user_agents_60m
	FROM usage_logs ul
	JOIN api_keys ak ON ak.id = ul.api_key_id
	JOIN users u ON u.id = ul.user_id
	WHERE ul.created_at >= $1
	  AND ul.created_at < $3
	  AND ak.deleted_at IS NULL
	  AND ak.status = 'active'
	  AND u.deleted_at IS NULL
	  AND COALESCE(u.risk_control_whitelisted, FALSE) = FALSE
	GROUP BY ul.user_id, u.email, ul.api_key_id, ak.name, ak.key
),
off_hours AS (
	SELECT
		ul.api_key_id,
		COUNT(*) AS off_hours_requests,
		COALESCE(array_remove(array_agg(DISTINCT ul.ip_address), NULL), ARRAY[]::text[]) AS off_hours_ips
	FROM usage_logs ul
	WHERE ul.created_at >= ($3 - INTERVAL '8 hours')
	  AND ul.created_at < $3
	  AND EXTRACT(HOUR FROM ul.created_at AT TIME ZONE 'Asia/Shanghai') BETWEEN 0 AND 7
	GROUP BY ul.api_key_id
),
historical AS (
	SELECT api_key_id, COALESCE(AVG(hourly_count), 0) AS hourly_avg
	FROM (
		SELECT ul.api_key_id, date_trunc('hour', ul.created_at AT TIME ZONE 'Asia/Shanghai') AS bucket, COUNT(*) AS hourly_count
		FROM usage_logs ul
		WHERE ul.created_at >= ($3 - INTERVAL '7 days')
		  AND ul.created_at < ($3 - INTERVAL '8 hours')
		  AND EXTRACT(HOUR FROM ul.created_at AT TIME ZONE 'Asia/Shanghai') BETWEEN 0 AND 7
		GROUP BY ul.api_key_id, bucket
	) h
	GROUP BY api_key_id
)
SELECT
	ks.user_id, ks.user_email, ks.api_key_id, ks.api_key_name, ks.api_key,
	ks.requests_30m, ks.requests_60m, ks.ips_30m, ks.ips_60m, ks.user_agents_60m,
	COALESCE(oh.off_hours_requests, 0), COALESCE(oh.off_hours_ips, ARRAY[]::text[]),
	COALESCE(h.hourly_avg, 0)
FROM key_stats ks
LEFT JOIN off_hours oh ON oh.api_key_id = ks.api_key_id
LEFT JOIN historical h ON h.api_key_id = ks.api_key_id
WHERE ks.requests_30m >= 10 OR ks.requests_60m >= 10 OR COALESCE(oh.off_hours_requests, 0) >= 50
`
	rows, err := r.db.QueryContext(ctx, query, windowStart, window30, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.APIKeyRiskCandidate{}
	for rows.Next() {
		var c service.APIKeyRiskCandidate
		var ips30, ips60, uas60, offIPs pq.StringArray
		if err := rows.Scan(
			&c.UserID, &c.UserEmail, &c.APIKeyID, &c.APIKeyName, &c.APIKey,
			&c.Requests30m, &c.Requests60m, &ips30, &ips60, &uas60,
			&c.OffHoursRequests, &offIPs,
			&c.OffHoursHourlyAvg,
		); err != nil {
			return nil, err
		}
		c.IPs30m = []string(ips30)
		c.IPs60m = []string(ips60)
		c.UserAgents60m = []string(uas60)
		c.OffHoursIPs = []string(offIPs)
		c.WindowStart = windowStart
		c.WindowEnd = now
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *apiKeyRiskRepository) CreateEvent(ctx context.Context, event *service.APIKeyRiskEvent) (bool, error) {
	evidence, err := json.Marshal(event.Evidence)
	if err != nil {
		return false, err
	}
	query := `
		INSERT INTO api_key_risk_events (user_id, api_key_id, rule_code, severity, score, status, evidence, time_bucket, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, NOW(), NOW())
		ON CONFLICT (api_key_id, rule_code, time_bucket) DO NOTHING
		RETURNING id, created_at, updated_at
	`
	err = r.db.QueryRowContext(ctx, query, event.UserID, event.APIKeyID, event.RuleCode, event.Severity, event.Score, event.Status, string(evidence), event.TimeBucket).
		Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *apiKeyRiskRepository) MarkEventBlocked(ctx context.Context, eventID int64, blockedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE api_key_risk_events
		SET status = $1, blocked_at = $2, updated_at = NOW()
		WHERE id = $3
	`, service.APIKeyRiskEventStatusBlocked, blockedAt, eventID)
	return err
}

func (r *apiKeyRiskRepository) ResolveEvent(ctx context.Context, eventID int64, adminID int64, resolvedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE api_key_risk_events
		SET status = $1, resolved_at = $2, resolved_by = $3, updated_at = NOW()
		WHERE id = $4
	`, service.APIKeyRiskEventStatusResolved, resolvedAt, adminID, eventID)
	return err
}

func (r *apiKeyRiskRepository) ListEvents(ctx context.Context, filter service.APIKeyRiskEventFilter) ([]service.APIKeyRiskEvent, *pagination.PaginationResult, error) {
	params := filter.Pagination
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 200 {
		params.PageSize = 200
	}
	where, args := buildAPIKeyRiskWhere(filter)
	countQuery := `SELECT COUNT(*) FROM api_key_risk_events e JOIN users u ON u.id = e.user_id JOIN api_keys ak ON ak.id = e.api_key_id ` + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, err
	}
	limitPos := len(args) + 1
	offsetPos := len(args) + 2
	query := fmt.Sprintf(`
		SELECT e.id, e.user_id, u.email, e.api_key_id, ak.name, e.rule_code, e.severity, e.score, e.status,
		       e.evidence, e.time_bucket, e.blocked_at, e.resolved_at, e.resolved_by, e.created_at, e.updated_at
		FROM api_key_risk_events e
		JOIN users u ON u.id = e.user_id
		JOIN api_keys ak ON ak.id = e.api_key_id
		%s
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $%d OFFSET $%d
	`, where, limitPos, offsetPos)
	args = append(args, params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []service.APIKeyRiskEvent{}
	for rows.Next() {
		var item service.APIKeyRiskEvent
		var evidenceBytes []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserEmail, &item.APIKeyID, &item.APIKeyName, &item.RuleCode, &item.Severity, &item.Score, &item.Status, &evidenceBytes, &item.TimeBucket, &item.BlockedAt, &item.ResolvedAt, &item.ResolvedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, nil, err
		}
		if len(evidenceBytes) > 0 {
			_ = json.Unmarshal(evidenceBytes, &item.Evidence)
		}
		if item.Evidence == nil {
			item.Evidence = service.APIKeyRiskEvidence{}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	pages := 0
	if params.PageSize > 0 {
		pages = int((total + int64(params.PageSize) - 1) / int64(params.PageSize))
	}
	return items, &pagination.PaginationResult{Total: total, Page: params.Page, PageSize: params.PageSize, Pages: pages}, nil
}

func buildAPIKeyRiskWhere(filter service.APIKeyRiskEventFilter) (string, []any) {
	parts := []string{"WHERE 1=1"}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		parts = append(parts, fmt.Sprintf(clause, len(args)))
	}
	if filter.UserID != nil && *filter.UserID > 0 {
		add("e.user_id = $%d", *filter.UserID)
	}
	if filter.APIKeyID != nil && *filter.APIKeyID > 0 {
		add("e.api_key_id = $%d", *filter.APIKeyID)
	}
	if strings.TrimSpace(filter.RuleCode) != "" {
		add("e.rule_code = $%d", strings.TrimSpace(filter.RuleCode))
	}
	if strings.TrimSpace(filter.Severity) != "" {
		add("e.severity = $%d", strings.TrimSpace(filter.Severity))
	}
	if strings.TrimSpace(filter.Status) != "" {
		add("e.status = $%d", strings.TrimSpace(filter.Status))
	}
	if filter.From != nil {
		add("e.created_at >= $%d", *filter.From)
	}
	if filter.To != nil {
		add("e.created_at <= $%d", *filter.To)
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		parts = append(parts, fmt.Sprintf("(u.email ILIKE $%d OR ak.name ILIKE $%d OR e.rule_code ILIKE $%d)", len(args), len(args), len(args)))
	}
	return strings.Join(parts, " "), args
}
