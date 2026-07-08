package service

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	APIKeyRiskRuleKeyMultiIP30m  = "key_multi_ip_30m"
	APIKeyRiskRuleUserMultiIP30m = "user_multi_ip_30m"
	APIKeyRiskRuleOffHoursSpike  = "off_hours_spike"
	APIKeyRiskRuleUAIPChurn60m   = "ua_ip_churn_60m"

	APIKeyRiskSeverityMedium = "medium"
	APIKeyRiskSeverityHigh   = "high"

	APIKeyRiskEventStatusOpen     = "open"
	APIKeyRiskEventStatusBlocked  = "blocked"
	APIKeyRiskEventStatusResolved = "resolved"
)

const apiKeyRiskScanLockID int64 = 584920241127001

type APIKeyRiskEvidence map[string]any

type APIKeyRiskEvent struct {
	ID         int64              `json:"id"`
	UserID     int64              `json:"user_id"`
	UserEmail  string             `json:"user_email"`
	APIKeyID   int64              `json:"api_key_id"`
	APIKeyName string             `json:"api_key_name"`
	RuleCode   string             `json:"rule_code"`
	Severity   string             `json:"severity"`
	Score      int                `json:"score"`
	Status     string             `json:"status"`
	Evidence   APIKeyRiskEvidence `json:"evidence"`
	TimeBucket time.Time          `json:"time_bucket"`
	BlockedAt  *time.Time         `json:"blocked_at,omitempty"`
	ResolvedAt *time.Time         `json:"resolved_at,omitempty"`
	ResolvedBy *int64             `json:"resolved_by,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

type APIKeyRiskCandidate struct {
	UserID            int64
	UserEmail         string
	APIKeyID          int64
	APIKeyName        string
	APIKey            string
	Requests30m       int
	Requests60m       int
	IPs30m            []string
	IPs60m            []string
	UserAgents60m     []string
	UserRequests30m   int
	UserIPs30m        []string
	OffHoursRequests  int
	OffHoursIPs       []string
	OffHoursHourlyAvg float64
	WindowStart       time.Time
	WindowEnd         time.Time
}

type APIKeyRiskEventFilter struct {
	Pagination pagination.PaginationParams
	UserID     *int64
	APIKeyID   *int64
	RuleCode   string
	Severity   string
	Status     string
	Search     string
	From       *time.Time
	To         *time.Time
}

type APIKeyRiskRepository interface {
	TryAcquireScanLock(ctx context.Context, lockID int64) (bool, error)
	ReleaseScanLock(ctx context.Context, lockID int64) error
	ListCandidates(ctx context.Context, now time.Time) ([]APIKeyRiskCandidate, error)
	CreateEvent(ctx context.Context, event *APIKeyRiskEvent) (bool, error)
	MarkEventBlocked(ctx context.Context, eventID int64, blockedAt time.Time) error
	ListEvents(ctx context.Context, filter APIKeyRiskEventFilter) ([]APIKeyRiskEvent, *pagination.PaginationResult, error)
	ResolveEvent(ctx context.Context, eventID int64, adminID int64, resolvedAt time.Time) error
}

type APIKeyRiskAPIKeyRepository interface {
	BlockForRisk(ctx context.Context, id int64, reason string, blockedAt time.Time) (string, int64, error)
	UnblockRisk(ctx context.Context, id int64) (string, int64, error)
}

type APIKeyRiskService struct {
	repo            APIKeyRiskRepository
	apiKeyRepo      APIKeyRiskAPIKeyRepository
	messageService  *UserMessageService
	authInvalidator APIKeyAuthCacheInvalidator
	interval        time.Duration
	stopCh          chan struct{}
	doneCh          chan struct{}
	startOnce       sync.Once
	stopOnce        sync.Once
}

func NewAPIKeyRiskService(repo APIKeyRiskRepository, apiKeyRepo APIKeyRiskAPIKeyRepository, messageService *UserMessageService, authInvalidator APIKeyAuthCacheInvalidator) *APIKeyRiskService {
	return &APIKeyRiskService{
		repo:            repo,
		apiKeyRepo:      apiKeyRepo,
		messageService:  messageService,
		authInvalidator: authInvalidator,
		interval:        time.Minute,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}
}

func ProvideAPIKeyRiskService(repo APIKeyRiskRepository, apiKeyRepo APIKeyRiskAPIKeyRepository, messageService *UserMessageService, authInvalidator APIKeyAuthCacheInvalidator) *APIKeyRiskService {
	svc := NewAPIKeyRiskService(repo, apiKeyRepo, messageService, authInvalidator)
	svc.Start()
	return svc
}

func (s *APIKeyRiskService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		go s.loop()
	})
}

func (s *APIKeyRiskService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		<-s.doneCh
	})
}

func (s *APIKeyRiskService) loop() {
	defer close(s.doneCh)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	_ = s.Scan(context.Background(), time.Now())
	for {
		select {
		case <-ticker.C:
			_ = s.Scan(context.Background(), time.Now())
		case <-s.stopCh:
			return
		}
	}
}

func (s *APIKeyRiskService) Scan(ctx context.Context, now time.Time) error {
	if s == nil || s.repo == nil || s.apiKeyRepo == nil {
		return nil
	}
	locked, err := s.repo.TryAcquireScanLock(ctx, apiKeyRiskScanLockID)
	if err != nil || !locked {
		return err
	}
	defer func() { _ = s.repo.ReleaseScanLock(context.Background(), apiKeyRiskScanLockID) }()

	candidates, err := s.repo.ListCandidates(ctx, now)
	if err != nil {
		return err
	}
	for i := range candidates {
		if err := s.evaluateCandidate(ctx, candidates[i], now); err != nil {
			return err
		}
	}
	return nil
}

func (s *APIKeyRiskService) evaluateCandidate(ctx context.Context, c APIKeyRiskCandidate, now time.Time) error {
	rules := buildAPIKeyRiskRules(c, now)
	for _, event := range rules {
		created, err := s.repo.CreateEvent(ctx, &event)
		if err != nil {
			return err
		}
		if !created || event.Status != APIKeyRiskEventStatusBlocked {
			continue
		}
		reason := riskBlockedReason(event.RuleCode)
		key, userID, err := s.apiKeyRepo.BlockForRisk(ctx, event.APIKeyID, reason, now)
		if err != nil {
			return err
		}
		if s.authInvalidator != nil {
			s.authInvalidator.InvalidateAuthCacheByKey(ctx, key)
		}
		if err := s.repo.MarkEventBlocked(ctx, event.ID, now); err != nil {
			return err
		}
		if s.messageService != nil {
			_, _ = s.messageService.Create(ctx, CreateUserMessageInput{
				UserID:  userID,
				Type:    UserMessageTypeAPIKeyRisk,
				Title:   "API Key 已因异常调用被封禁",
				Content: fmt.Sprintf("你的 API Key「%s」因疑似泄露或异常调用已被系统封禁。请删除或更新该 key，并检查近期调用来源。封禁原因：%s", event.APIKeyName, reason),
				Metadata: map[string]any{
					"api_key_id": event.APIKeyID,
					"event_id":   event.ID,
					"rule_code":  event.RuleCode,
				},
			})
		}
	}
	return nil
}

func (s *APIKeyRiskService) ListEvents(ctx context.Context, filter APIKeyRiskEventFilter) ([]APIKeyRiskEvent, *pagination.PaginationResult, error) {
	return s.repo.ListEvents(ctx, filter)
}

func (s *APIKeyRiskService) ResolveEvent(ctx context.Context, eventID int64, adminID int64) error {
	return s.repo.ResolveEvent(ctx, eventID, adminID, time.Now())
}

func (s *APIKeyRiskService) UnblockKey(ctx context.Context, apiKeyID int64) error {
	key, _, err := s.apiKeyRepo.UnblockRisk(ctx, apiKeyID)
	if err != nil {
		return err
	}
	if s.authInvalidator != nil {
		s.authInvalidator.InvalidateAuthCacheByKey(ctx, key)
	}
	return nil
}

func buildAPIKeyRiskRules(c APIKeyRiskCandidate, now time.Time) []APIKeyRiskEvent {
	bucket := now.Truncate(time.Minute)
	out := make([]APIKeyRiskEvent, 0, 4)
	ip30 := uniqueNonEmpty(c.IPs30m)
	ip60 := uniqueNonEmpty(c.IPs60m)
	userIP30 := uniqueNonEmpty(c.UserIPs30m)
	uaFamilies := userAgentFamilies(c.UserAgents60m)
	prefix30 := ipPrefixes(ip30)
	prefix60 := ipPrefixes(ip60)
	userPrefix30 := ipPrefixes(userIP30)

	base := func(rule, severity string, score int, status string, evidence APIKeyRiskEvidence) APIKeyRiskEvent {
		return APIKeyRiskEvent{
			UserID:     c.UserID,
			UserEmail:  c.UserEmail,
			APIKeyID:   c.APIKeyID,
			APIKeyName: c.APIKeyName,
			RuleCode:   rule,
			Severity:   severity,
			Score:      score,
			Status:     status,
			Evidence:   evidence,
			TimeBucket: bucket,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}

	if c.Requests30m >= 10 && (len(ip30) >= 4 || len(prefix30) >= 3) {
		out = append(out, base(APIKeyRiskRuleKeyMultiIP30m, APIKeyRiskSeverityHigh, 90, APIKeyRiskEventStatusBlocked, evidence(c, ip30, prefix30, uaFamilies, map[string]any{
			"requests": c.Requests30m, "ip_count": len(ip30), "ip_prefix_count": len(prefix30),
			"threshold": "30m requests >= 10 and (ip_count >= 4 or ip_prefix_count >= 3)",
		})))
	}
	if c.UserRequests30m >= 10 && (len(userIP30) >= 6 || len(userPrefix30) >= 4) && c.Requests30m*2 >= c.UserRequests30m {
		out = append(out, base(APIKeyRiskRuleUserMultiIP30m, APIKeyRiskSeverityHigh, 90, APIKeyRiskEventStatusBlocked, evidence(c, userIP30, userPrefix30, uaFamilies, map[string]any{
			"user_requests": c.UserRequests30m, "key_requests": c.Requests30m, "user_ip_count": len(userIP30), "user_ip_prefix_count": len(userPrefix30),
			"threshold": "user 30m ip_count >= 6 or ip_prefix_count >= 4, concentrated on this key",
		})))
	}
	if c.OffHoursRequests >= 50 {
		avg := c.OffHoursHourlyAvg
		if avg < 1 {
			avg = 1
		}
		status := APIKeyRiskEventStatusOpen
		severity := APIKeyRiskSeverityMedium
		score := 65
		if float64(c.OffHoursRequests) >= avg*3 && len(uniqueNonEmpty(c.OffHoursIPs)) >= 2 {
			status = APIKeyRiskEventStatusBlocked
			severity = APIKeyRiskSeverityHigh
			score = 85
		}
		out = append(out, base(APIKeyRiskRuleOffHoursSpike, severity, score, status, evidence(c, uniqueNonEmpty(c.OffHoursIPs), ipPrefixes(c.OffHoursIPs), uaFamilies, map[string]any{
			"off_hours_requests": c.OffHoursRequests, "off_hours_ip_count": len(uniqueNonEmpty(c.OffHoursIPs)), "historical_hourly_avg": c.OffHoursHourlyAvg,
			"threshold": "00:00-08:00 requests >= 50 and >= 3x historical average; auto-block only with >= 2 IPs",
		})))
	}
	if c.Requests60m >= 10 && len(uaFamilies) >= 4 && len(prefix60) >= 3 {
		out = append(out, base(APIKeyRiskRuleUAIPChurn60m, APIKeyRiskSeverityMedium, 60, APIKeyRiskEventStatusOpen, evidence(c, ip60, prefix60, uaFamilies, map[string]any{
			"requests": c.Requests60m, "ua_family_count": len(uaFamilies), "ip_prefix_count": len(prefix60),
			"threshold": "60m ua_family_count >= 4 and ip_prefix_count >= 3",
		})))
	}
	return out
}

func evidence(c APIKeyRiskCandidate, ips, prefixes, uaFamilies []string, extra map[string]any) APIKeyRiskEvidence {
	ev := APIKeyRiskEvidence{
		"window_start":       c.WindowStart,
		"window_end":         c.WindowEnd,
		"sample_ips":         sampleStrings(ips, 8),
		"sample_ip_prefixes": sampleStrings(prefixes, 8),
		"sample_user_agents": sampleStrings(uaFamilies, 8),
	}
	for k, v := range extra {
		ev[k] = v
	}
	return ev
}

func riskBlockedReason(rule string) string {
	switch rule {
	case APIKeyRiskRuleKeyMultiIP30m:
		return "API key 在 30 分钟内出现多个异常 IP 来源，疑似泄露"
	case APIKeyRiskRuleUserMultiIP30m:
		return "用户调用来源 IP 在短时间内异常变化，疑似 API key 泄露"
	case APIKeyRiskRuleOffHoursSpike:
		return "API key 在 0-8 点出现异常高频调用，疑似泄露"
	default:
		return "API key 因疑似泄露或异常调用已被封禁，请在控制台更新 key"
	}
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func ipPrefixes(ips []string) []string {
	seen := map[string]struct{}{}
	for _, raw := range ips {
		ip := net.ParseIP(strings.TrimSpace(raw))
		if ip == nil {
			continue
		}
		prefix := ""
		if v4 := ip.To4(); v4 != nil {
			prefix = fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2])
		} else {
			parts := strings.Split(ip.String(), ":")
			if len(parts) >= 4 {
				prefix = strings.Join(parts[:4], ":") + "::/64"
			}
		}
		if prefix != "" {
			seen[prefix] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func userAgentFamilies(values []string) []string {
	seen := map[string]struct{}{}
	for _, ua := range values {
		ua = strings.ToLower(strings.TrimSpace(ua))
		if ua == "" {
			continue
		}
		family := strings.Fields(strings.NewReplacer("(", " ", ")", " ", "/", " ").Replace(ua))
		if len(family) == 0 {
			continue
		}
		seen[family[0]] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for ua := range seen {
		out = append(out, ua)
	}
	sort.Strings(out)
	return out
}

func sampleStrings(values []string, limit int) []string {
	values = uniqueNonEmpty(values)
	if len(values) > limit {
		return values[:limit]
	}
	return values
}
