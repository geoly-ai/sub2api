package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLoginAbuseProtectionClassification(t *testing.T) {
	h := &AuthHandler{cfg: &config.Config{Security: config.SecurityConfig{
		LoginAbuseProtection: config.LoginAbuseProtectionConfig{
			Enabled:                 true,
			LowFrictionEmailDomains: []string{"rijoy.ai", "cyberklick.com"},
			StandardEmailDomains:    []string{"gmail.com", "qilyear.com"},
		},
	}}}

	newContext := func(headers map[string]string) *gin.Context {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		req := httptest.NewRequest(http.MethodPost, "https://app.example.test/api/v1/auth/login", nil)
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		ctx.Request = req
		return ctx
	}

	require.False(t, h.shouldApplyStrictLoginAbuseProtection(newContext(nil).Request, "user@rijoy.ai"))
	require.False(t, h.shouldApplyStrictLoginAbuseProtection(newContext(nil).Request, "user@cyberklick.com"))
	require.False(t, h.shouldApplyStrictLoginAbuseProtection(newContext(nil).Request, "user@gmail.com"))
	require.False(t, h.shouldApplyStrictLoginAbuseProtection(newContext(nil).Request, "user@qilyear.com"))
	require.True(t, h.shouldApplyStrictLoginAbuseProtection(newContext(nil).Request, "user@example.net"))
	require.False(t, h.shouldApplyStrictLoginAbuseProtection(newContext(map[string]string{
		"Origin":         "https://app.example.test",
		"Sec-Fetch-Site": "same-origin",
		"Sec-CH-UA":      `"Chromium";v="1"`,
	}).Request, "user@example.net"))
}

func TestLoginAbuseProtectionReturnsRetryResponseForSuspiciousDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "https://app.example.test/api/v1/auth/login", nil)
	h := &AuthHandler{cfg: &config.Config{Security: config.SecurityConfig{
		LoginAbuseProtection: config.LoginAbuseProtectionConfig{Enabled: true},
	}}}

	require.True(t, h.applyLoginAbuseProtection(ctx, "user@example.net"))
	require.True(t, ctx.IsAborted())
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.NotEmpty(t, recorder.Header().Get("Retry-After"))
}
