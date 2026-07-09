package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRiskRepositoryListCandidatesExcludesWhitelistedUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAPIKeyRiskRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta("COALESCE(u.risk_control_whitelisted, FALSE) = FALSE")).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "user_email", "api_key_id", "api_key_name", "api_key",
			"requests_30m", "requests_60m", "ips_30m", "ips_60m", "user_agents_60m",
			"user_requests_30m", "user_ips_30m", "off_hours_requests", "off_hours_ips", "hourly_avg",
		}))

	candidates, err := repo.ListCandidates(context.Background(), time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Empty(t, candidates)
	require.NoError(t, mock.ExpectationsWereMet())
}
