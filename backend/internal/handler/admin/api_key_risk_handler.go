package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type APIKeyRiskHandler struct {
	service *service.APIKeyRiskService
}

func NewAPIKeyRiskHandler(svc *service.APIKeyRiskService) *APIKeyRiskHandler {
	return &APIKeyRiskHandler{service: svc}
}

func (h *APIKeyRiskHandler) ListEvents(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := service.APIKeyRiskEventFilter{
		Pagination: pagination.PaginationParams{Page: page, PageSize: pageSize, SortOrder: pagination.SortOrderDesc},
		RuleCode:   c.Query("rule_code"),
		Severity:   c.Query("severity"),
		Status:     c.Query("status"),
		Search:     c.Query("search"),
	}
	if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid user_id")
			return
		}
		filter.UserID = &id
	}
	if v := strings.TrimSpace(c.Query("api_key_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid api_key_id")
			return
		}
		filter.APIKeyID = &id
	}
	if t, ok, err := parseRiskDate(c.Query("from")); err != nil {
		response.BadRequest(c, "Invalid from")
		return
	} else if ok {
		filter.From = &t
	}
	if t, ok, err := parseRiskDate(c.Query("to")); err != nil {
		response.BadRequest(c, "Invalid to")
		return
	} else if ok {
		filter.To = &t
	}
	items, pageResult, err := h.service.ListEvents(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, pageResult.Total, pageResult.Page, pageResult.PageSize)
}

func (h *APIKeyRiskHandler) ResolveEvent(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Admin not found in context")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid event ID")
		return
	}
	if err := h.service.ResolveEvent(c.Request.Context(), id, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func (h *APIKeyRiskHandler) UnblockKey(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid API key ID")
		return
	}
	if err := h.service.UnblockKey(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func parseRiskDate(raw string) (time.Time, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	return t, err == nil, err
}
