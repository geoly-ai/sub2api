package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupMCPRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()
	handler := NewUserHandler(adminSvc, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/mcp", handler.HandleMCP)
	return router, adminSvc
}

func postMCP(t *testing.T, router *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/mcp", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func decodeMCPToolText(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Result.Content)
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(resp.Result.Content[0].Text), &out))
	return out
}

func TestUserMCPInitializeAndToolsList(t *testing.T) {
	router, _ := setupMCPRouter()

	rec := postMCP(t, router, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"protocolVersion":"2025-06-18"`)
	require.Contains(t, rec.Body.String(), `"name":"sub2api-admin"`)

	rec = postMCP(t, router, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"name":"admin_search_users"`)
	require.Contains(t, rec.Body.String(), `"name":"admin_batch_create_users"`)
	require.Contains(t, rec.Body.String(), `"name":"admin_batch_add_balance"`)
	require.Contains(t, rec.Body.String(), `"name":"admin_batch_disable_users"`)
}

func TestUserMCPRejectsUnknownMethodAndTool(t *testing.T) {
	router, _ := setupMCPRouter()

	rec := postMCP(t, router, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "missing",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":-32601`)

	rec = postMCP(t, router, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "missing_tool",
			"arguments": map[string]any{},
		},
	})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":-32601`)
}

func TestUserMCPSearchUsersPassesFilters(t *testing.T) {
	router, adminSvc := setupMCPRouter()

	rec := postMCP(t, router, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "admin_search_users",
			"arguments": map[string]any{
				"page":             2,
				"page_size":        10,
				"search":           " user ",
				"status":           service.StatusActive,
				"role":             service.RoleUser,
				"api_key_group_id": 42,
				"sort_by":          "email",
				"sort_order":       "asc",
			},
		},
	})
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeMCPToolText(t, rec)
	require.Equal(t, float64(1), data["total"])
	require.Equal(t, 2, adminSvc.lastListUsers.page)
	require.Equal(t, 10, adminSvc.lastListUsers.pageSize)
	require.Equal(t, "user", adminSvc.lastListUsers.filters.Search)
	require.Equal(t, int64(42), adminSvc.lastListUsers.filters.APIKeyGroupID)
	require.Equal(t, "email", adminSvc.lastListUsers.sortBy)
	require.Equal(t, "asc", adminSvc.lastListUsers.sortOrder)
}

func TestUserMCPBatchCreateUsers(t *testing.T) {
	router, adminSvc := setupMCPRouter()

	rec := postMCP(t, router, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "admin_batch_create_users",
			"arguments": map[string]any{
				"users": []map[string]any{
					{"email": "new@example.com", "password": "secret1", "balance": 3},
					{"email": "bad", "password": "short"},
				},
			},
		},
	})
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeMCPToolText(t, rec)
	require.Equal(t, float64(1), data["success_count"])
	require.Equal(t, float64(1), data["failure_count"])
	require.Len(t, adminSvc.createdUsers, 1)
	require.Equal(t, "new@example.com", adminSvc.createdUsers[0].Email)
}

func TestUserMCPBatchAddBalance(t *testing.T) {
	router, adminSvc := setupMCPRouter()

	rec := postMCP(t, router, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "admin_batch_add_balance",
			"arguments": map[string]any{
				"user_ids": []int64{1, 2},
				"amount":   8.5,
				"notes":    "mcp top-up",
			},
		},
	})
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeMCPToolText(t, rec)
	require.Equal(t, float64(2), data["success_count"])
	require.Len(t, adminSvc.balanceUpdates, 2)
	require.Equal(t, "add", adminSvc.balanceUpdates[0].operation)
	require.Equal(t, 8.5, adminSvc.balanceUpdates[0].balance)
	require.Equal(t, "mcp top-up", adminSvc.balanceUpdates[0].notes)
}

func TestUserMCPBatchDisableUsers(t *testing.T) {
	router, adminSvc := setupMCPRouter()
	notes := "mcp disable"

	rec := postMCP(t, router, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "admin_batch_disable_users",
			"arguments": map[string]any{
				"user_ids": []int64{1, 2},
				"notes":    notes,
			},
		},
	})
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeMCPToolText(t, rec)
	require.Equal(t, float64(2), data["success_count"])
	require.Equal(t, []int64{1, 2}, adminSvc.updatedUsers)
	require.Equal(t, service.StatusDisabled, adminSvc.updatedUserInputs[0].Status)
	require.NotNil(t, adminSvc.updatedUserInputs[0].Notes)
	require.Equal(t, notes, *adminSvc.updatedUserInputs[0].Notes)
}
