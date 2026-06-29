package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/mail"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	mcpProtocolVersion = "2025-06-18"
	mcpBatchLimit      = 500
)

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type mcpToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type mcpToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpToolResult struct {
	Content []mcpToolContent `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

// HandleMCP exposes a small admin-only MCP server for user management.
// POST /api/v1/admin/mcp
func (h *UserHandler) HandleMCP(c *gin.Context) {
	var req mcpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, newMCPError(nil, -32600, "Invalid JSON-RPC request"))
		return
	}
	if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		c.JSON(http.StatusOK, newMCPError(req.ID, -32600, "Invalid JSON-RPC request"))
		return
	}

	switch req.Method {
	case "initialize":
		c.JSON(http.StatusOK, mcpResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: gin.H{
				"protocolVersion": mcpProtocolVersion,
				"capabilities": gin.H{
					"tools": gin.H{"listChanged": false},
				},
				"serverInfo": gin.H{
					"name":    "sub2api-admin",
					"version": "1.0.0",
				},
			},
		})
	case "notifications/initialized":
		c.Status(http.StatusNoContent)
	case "tools/list":
		c.JSON(http.StatusOK, mcpResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  gin.H{"tools": adminMCPTools()},
		})
	case "tools/call":
		result, rpcErr := h.callMCPTool(c, req.Params)
		if rpcErr != nil {
			c.JSON(http.StatusOK, newMCPError(req.ID, rpcErr.code, rpcErr.message))
			return
		}
		c.JSON(http.StatusOK, mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
	default:
		c.JSON(http.StatusOK, newMCPError(req.ID, -32601, "Method not found"))
	}
}

type mcpRPCError struct {
	code    int
	message string
}

func newMCPError(id json.RawMessage, code int, message string) mcpResponse {
	return mcpResponse{JSONRPC: "2.0", ID: id, Error: &mcpError{Code: code, Message: message}}
}

func (h *UserHandler) callMCPTool(c *gin.Context, rawParams json.RawMessage) (*mcpToolResult, *mcpRPCError) {
	var params mcpToolCallParams
	if len(rawParams) == 0 {
		return nil, &mcpRPCError{code: -32602, message: "Missing tool call params"}
	}
	if err := json.Unmarshal(rawParams, &params); err != nil || strings.TrimSpace(params.Name) == "" {
		return nil, &mcpRPCError{code: -32602, message: "Invalid tool call params"}
	}
	if len(params.Arguments) == 0 || string(params.Arguments) == "null" {
		params.Arguments = []byte(`{}`)
	}

	ctx := c.Request.Context()
	execute := func(execCtx context.Context) (any, error) {
		switch params.Name {
		case "admin_search_users":
			return h.mcpSearchUsers(execCtx, params.Arguments)
		case "admin_batch_create_users":
			return h.mcpBatchCreateUsers(execCtx, params.Arguments)
		case "admin_batch_add_balance":
			return h.mcpBatchAddBalance(execCtx, params.Arguments)
		case "admin_batch_disable_users":
			return h.mcpBatchDisableUsers(execCtx, params.Arguments)
		default:
			return nil, errMCPToolNotFound
		}
	}

	var data any
	var err error
	switch params.Name {
	case "admin_search_users":
		data, err = execute(ctx)
	case "admin_batch_create_users", "admin_batch_add_balance", "admin_batch_disable_users":
		if strings.TrimSpace(c.GetHeader("Idempotency-Key")) != "" {
			result, execErr := executeAdminIdempotent(c, "admin.mcp."+params.Name, mcpIdempotencyPayload{
				Tool:      params.Name,
				Arguments: json.RawMessage(append([]byte(nil), params.Arguments...)),
			}, service.DefaultWriteIdempotencyTTL(), execute)
			if execErr != nil {
				err = execErr
			} else if result != nil {
				data = result.Data
				if result.Replayed {
					c.Header("X-Idempotency-Replayed", "true")
				}
			}
		} else {
			data, err = execute(ctx)
		}
	default:
		return nil, &mcpRPCError{code: -32601, message: "Tool not found"}
	}
	if err != nil {
		return mcpJSONToolResult(gin.H{"error": err.Error()}, true), nil
	}
	return mcpJSONToolResult(data, false), nil
}

var errMCPToolNotFound = errors.New("tool not found")

type mcpIdempotencyPayload struct {
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
}

func mcpJSONToolResult(data any, isError bool) *mcpToolResult {
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		payload = []byte(fmt.Sprintf(`{"error":%q}`, err.Error()))
		isError = true
	}
	return &mcpToolResult{
		Content: []mcpToolContent{{Type: "text", Text: string(payload)}},
		IsError: isError,
	}
}

type mcpSearchUsersArgs struct {
	Page                 int    `json:"page"`
	PageSize             int    `json:"page_size"`
	Search               string `json:"search"`
	Status               string `json:"status"`
	Role                 string `json:"role"`
	GroupName            string `json:"group_name"`
	APIKeyGroupID        int64  `json:"api_key_group_id"`
	IncludeSubscriptions *bool  `json:"include_subscriptions"`
	SortBy               string `json:"sort_by"`
	SortOrder            string `json:"sort_order"`
}

func (h *UserHandler) mcpSearchUsers(ctx context.Context, raw json.RawMessage) (any, error) {
	var args mcpSearchUsersArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Page <= 0 {
		args.Page = 1
	}
	if args.PageSize <= 0 {
		args.PageSize = 20
	}
	if args.PageSize > 100 {
		args.PageSize = 100
	}
	sortBy := strings.TrimSpace(args.SortBy)
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := strings.TrimSpace(args.SortOrder)
	if sortOrder == "" {
		sortOrder = "desc"
	}
	search := strings.TrimSpace(args.Search)
	if runes := []rune(search); len(runes) > 100 {
		search = string(runes[:100])
	}

	users, total, err := h.adminService.ListUsers(ctx, args.Page, args.PageSize, service.UserListFilters{
		Status:               strings.TrimSpace(args.Status),
		Role:                 strings.TrimSpace(args.Role),
		Search:               search,
		GroupName:            strings.TrimSpace(args.GroupName),
		APIKeyGroupID:        args.APIKeyGroupID,
		IncludeSubscriptions: args.IncludeSubscriptions,
	}, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.AdminUser, 0, len(users))
	for i := range users {
		out = append(out, dto.UserFromServiceAdmin(&users[i]))
	}
	pages := int(math.Ceil(float64(total) / float64(args.PageSize)))
	if pages < 1 {
		pages = 1
	}
	return gin.H{
		"items":     out,
		"total":     total,
		"page":      args.Page,
		"page_size": args.PageSize,
		"pages":     pages,
	}, nil
}

type mcpBatchCreateUsersArgs struct {
	Users []mcpCreateUserInput `json:"users"`
}

type mcpCreateUserInput struct {
	Email         string   `json:"email"`
	Password      string   `json:"password"`
	Username      string   `json:"username"`
	Notes         string   `json:"notes"`
	Balance       *float64 `json:"balance"`
	Concurrency   int      `json:"concurrency"`
	RPMLimit      int      `json:"rpm_limit"`
	AllowedGroups []int64  `json:"allowed_groups"`
}

type mcpBatchItemResult struct {
	Index   int            `json:"index"`
	UserID  int64          `json:"user_id,omitempty"`
	Email   string         `json:"email,omitempty"`
	Success bool           `json:"success"`
	Error   string         `json:"error,omitempty"`
	User    *dto.AdminUser `json:"user,omitempty"`
}

func (h *UserHandler) mcpBatchCreateUsers(ctx context.Context, raw json.RawMessage) (any, error) {
	var args mcpBatchCreateUsersArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if len(args.Users) == 0 {
		return nil, errors.New("users is required")
	}
	if len(args.Users) > mcpBatchLimit {
		return nil, fmt.Errorf("users cannot exceed %d", mcpBatchLimit)
	}
	results := make([]mcpBatchItemResult, 0, len(args.Users))
	successCount := 0
	for i, input := range args.Users {
		item := mcpBatchItemResult{Index: i, Email: strings.TrimSpace(input.Email)}
		if err := validateMCPCreateUser(input); err != nil {
			item.Error = err.Error()
			results = append(results, item)
			continue
		}
		user, err := h.adminService.CreateUser(ctx, &service.CreateUserInput{
			Email:         item.Email,
			Password:      input.Password,
			Username:      strings.TrimSpace(input.Username),
			Notes:         input.Notes,
			Balance:       input.Balance,
			Concurrency:   input.Concurrency,
			RPMLimit:      input.RPMLimit,
			AllowedGroups: input.AllowedGroups,
		})
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Success = true
			item.UserID = user.ID
			item.User = dto.UserFromServiceAdmin(user)
			successCount++
		}
		results = append(results, item)
	}
	return gin.H{"success_count": successCount, "failure_count": len(results) - successCount, "results": results}, nil
}

func validateMCPCreateUser(input mcpCreateUserInput) error {
	email := strings.TrimSpace(input.Email)
	if email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("email is invalid")
	}
	if len(input.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if input.Balance != nil && (*input.Balance < 0 || math.IsNaN(*input.Balance) || math.IsInf(*input.Balance, 0)) {
		return errors.New("balance must be a finite number >= 0")
	}
	return nil
}

type mcpBatchAddBalanceArgs struct {
	UserIDs []int64 `json:"user_ids"`
	Amount  float64 `json:"amount"`
	Notes   string  `json:"notes"`
}

func (h *UserHandler) mcpBatchAddBalance(ctx context.Context, raw json.RawMessage) (any, error) {
	var args mcpBatchAddBalanceArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if len(args.UserIDs) == 0 {
		return nil, errors.New("user_ids is required")
	}
	if len(args.UserIDs) > mcpBatchLimit {
		return nil, fmt.Errorf("user_ids cannot exceed %d", mcpBatchLimit)
	}
	if args.Amount <= 0 || math.IsNaN(args.Amount) || math.IsInf(args.Amount, 0) {
		return nil, errors.New("amount must be a finite number > 0")
	}
	results := make([]mcpBatchItemResult, 0, len(args.UserIDs))
	successCount := 0
	for i, userID := range args.UserIDs {
		item := mcpBatchItemResult{Index: i, UserID: userID}
		if userID <= 0 {
			item.Error = "user_id must be > 0"
			results = append(results, item)
			continue
		}
		user, err := h.adminService.UpdateUserBalance(ctx, userID, args.Amount, "add", args.Notes)
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Success = true
			item.Email = user.Email
			item.User = dto.UserFromServiceAdmin(user)
			successCount++
		}
		results = append(results, item)
	}
	return gin.H{"success_count": successCount, "failure_count": len(results) - successCount, "results": results}, nil
}

type mcpBatchDisableUsersArgs struct {
	UserIDs []int64 `json:"user_ids"`
	Notes   *string `json:"notes"`
}

func (h *UserHandler) mcpBatchDisableUsers(ctx context.Context, raw json.RawMessage) (any, error) {
	var args mcpBatchDisableUsersArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if len(args.UserIDs) == 0 {
		return nil, errors.New("user_ids is required")
	}
	if len(args.UserIDs) > mcpBatchLimit {
		return nil, fmt.Errorf("user_ids cannot exceed %d", mcpBatchLimit)
	}
	results := make([]mcpBatchItemResult, 0, len(args.UserIDs))
	successCount := 0
	for i, userID := range args.UserIDs {
		item := mcpBatchItemResult{Index: i, UserID: userID}
		if userID <= 0 {
			item.Error = "user_id must be > 0"
			results = append(results, item)
			continue
		}
		user, err := h.adminService.UpdateUser(ctx, userID, &service.UpdateUserInput{
			Status: service.StatusDisabled,
			Notes:  args.Notes,
		})
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Success = true
			item.Email = user.Email
			item.User = dto.UserFromServiceAdmin(user)
			successCount++
		}
		results = append(results, item)
	}
	return gin.H{"success_count": successCount, "failure_count": len(results) - successCount, "results": results}, nil
}

func adminMCPTools() []mcpTool {
	tools := []mcpTool{
		{
			Name:        "admin_batch_add_balance",
			Description: "批量为用户余额加值。仅管理员可用。",
			InputSchema: objectSchema(map[string]any{
				"user_ids": arraySchema(map[string]any{"type": "integer", "minimum": 1}, "目标用户 ID，最多 500 个。"),
				"amount":   map[string]any{"type": "number", "exclusiveMinimum": 0, "description": "每个用户增加的余额。"},
				"notes":    map[string]any{"type": "string", "description": "写入余额调整审计记录的备注。"},
			}, []string{"user_ids", "amount"}),
		},
		{
			Name:        "admin_batch_create_users",
			Description: "批量创建普通用户。仅管理员可用。",
			InputSchema: objectSchema(map[string]any{
				"users": arraySchema(objectSchema(map[string]any{
					"email":          map[string]any{"type": "string", "format": "email"},
					"password":       map[string]any{"type": "string", "minLength": 6},
					"username":       map[string]any{"type": "string"},
					"notes":          map[string]any{"type": "string"},
					"balance":        map[string]any{"type": "number", "minimum": 0},
					"concurrency":    map[string]any{"type": "integer", "minimum": 0},
					"rpm_limit":      map[string]any{"type": "integer", "minimum": 0},
					"allowed_groups": arraySchema(map[string]any{"type": "integer", "minimum": 1}, "允许绑定的专属分组 ID。"),
				}, []string{"email", "password"}), "待创建用户，最多 500 个。"),
			}, []string{"users"}),
		},
		{
			Name:        "admin_batch_disable_users",
			Description: "批量禁用用户。管理员账号会被后端保护而拒绝禁用。",
			InputSchema: objectSchema(map[string]any{
				"user_ids": arraySchema(map[string]any{"type": "integer", "minimum": 1}, "目标用户 ID，最多 500 个。"),
				"notes":    map[string]any{"type": "string", "description": "可选管理员备注。"},
			}, []string{"user_ids"}),
		},
		{
			Name:        "admin_search_users",
			Description: "查询用户列表，支持分页、搜索、状态、角色和分组过滤。",
			InputSchema: objectSchema(map[string]any{
				"page":                  map[string]any{"type": "integer", "minimum": 1, "default": 1},
				"page_size":             map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 20},
				"search":                map[string]any{"type": "string"},
				"status":                map[string]any{"type": "string", "enum": []string{"", service.StatusActive, service.StatusDisabled}},
				"role":                  map[string]any{"type": "string", "enum": []string{"", service.RoleUser, service.RoleAdmin}},
				"group_name":            map[string]any{"type": "string"},
				"api_key_group_id":      map[string]any{"type": "integer", "minimum": 0},
				"include_subscriptions": map[string]any{"type": "boolean"},
				"sort_by":               map[string]any{"type": "string", "default": "created_at"},
				"sort_order":            map[string]any{"type": "string", "enum": []string{"asc", "desc", "ASC", "DESC"}, "default": "desc"},
			}, nil),
		},
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
	return tools
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func arraySchema(items map[string]any, description string) map[string]any {
	return map[string]any{
		"type":        "array",
		"items":       items,
		"maxItems":    mcpBatchLimit,
		"description": description,
	}
}
