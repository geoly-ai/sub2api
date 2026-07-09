package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildAPIKeyRiskRulesKeyMultiIP30mThreshold(t *testing.T) {
	now := time.Date(2026, 7, 8, 2, 30, 0, 0, time.UTC)
	c := baseAPIKeyRiskCandidate(now)
	c.Requests30m = 10
	c.IPs30m = []string{"10.0.1.1", "10.0.1.2", "10.0.1.3", "10.0.1.4", "10.0.1.5"}

	require.Empty(t, buildAPIKeyRiskRules(c, now))

	c.IPs30m = append(c.IPs30m, "10.0.1.6")
	events := buildAPIKeyRiskRules(c, now)

	require.Len(t, events, 1)
	require.Equal(t, APIKeyRiskRuleKeyMultiIP30m, events[0].RuleCode)
	require.Equal(t, APIKeyRiskSeverityHigh, events[0].Severity)
	require.Equal(t, APIKeyRiskEventStatusBlocked, events[0].Status)
	requireEvidenceWindow(t, events[0], now.Add(-30*time.Minute), now)
}

func TestBuildAPIKeyRiskRulesDoesNotEmitUserMultiIP30m(t *testing.T) {
	now := time.Date(2026, 7, 8, 3, 0, 0, 0, time.UTC)
	c := baseAPIKeyRiskCandidate(now)
	c.Requests30m = 10
	c.UserRequests30m = 20
	c.UserIPs30m = []string{"10.0.1.1", "10.0.2.1", "10.0.3.1", "10.0.4.1", "10.0.5.1", "10.0.6.1"}

	require.Empty(t, buildAPIKeyRiskRules(c, now))
}

func TestBuildAPIKeyRiskRulesOffHoursSpikeBlocksOnlyWithMultipleIPs(t *testing.T) {
	now := time.Date(2026, 7, 8, 1, 0, 0, 0, time.UTC)
	c := baseAPIKeyRiskCandidate(now)
	c.OffHoursRequests = 50
	c.OffHoursHourlyAvg = 10
	c.OffHoursIPs = []string{"10.0.1.1"}

	events := buildAPIKeyRiskRules(c, now)
	require.Len(t, events, 1)
	require.Equal(t, APIKeyRiskRuleOffHoursSpike, events[0].RuleCode)
	require.Equal(t, APIKeyRiskEventStatusOpen, events[0].Status)
	require.Equal(t, APIKeyRiskSeverityMedium, events[0].Severity)
	requireEvidenceWindow(t, events[0], now.Add(-8*time.Hour), now)

	c.OffHoursIPs = append(c.OffHoursIPs, "10.0.2.1")
	events = buildAPIKeyRiskRules(c, now)
	require.Len(t, events, 1)
	require.Equal(t, APIKeyRiskEventStatusBlocked, events[0].Status)
	require.Equal(t, APIKeyRiskSeverityHigh, events[0].Severity)
	requireEvidenceWindow(t, events[0], now.Add(-8*time.Hour), now)
}

func TestBuildAPIKeyRiskRulesUAIPChurn60mWarnsWithoutBlocking(t *testing.T) {
	now := time.Date(2026, 7, 8, 4, 0, 0, 0, time.UTC)
	c := baseAPIKeyRiskCandidate(now)
	c.Requests60m = 10
	c.IPs60m = []string{"10.0.1.1", "10.0.2.1", "10.0.3.1"}
	c.UserAgents60m = []string{
		"curl/8.1",
		"python-requests/2.32",
		"Go-http-client/2.0",
		"okhttp/4.12",
	}

	events := buildAPIKeyRiskRules(c, now)

	require.Len(t, events, 1)
	require.Equal(t, APIKeyRiskRuleUAIPChurn60m, events[0].RuleCode)
	require.Equal(t, APIKeyRiskSeverityMedium, events[0].Severity)
	require.Equal(t, APIKeyRiskEventStatusOpen, events[0].Status)
	requireEvidenceWindow(t, events[0], now.Add(-time.Hour), now)
}

func requireEvidenceWindow(t *testing.T, event APIKeyRiskEvent, wantStart, wantEnd time.Time) {
	t.Helper()
	require.Equal(t, wantStart, event.Evidence["window_start"])
	require.Equal(t, wantEnd, event.Evidence["window_end"])
}

func baseAPIKeyRiskCandidate(now time.Time) APIKeyRiskCandidate {
	return APIKeyRiskCandidate{
		UserID:      1,
		UserEmail:   "user@example.com",
		APIKeyID:    2,
		APIKeyName:  "prod-key",
		WindowStart: now.Add(-time.Hour),
		WindowEnd:   now,
	}
}
