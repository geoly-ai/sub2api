package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	UserMessageStatusUnread   = "unread"
	UserMessageStatusRead     = "read"
	UserMessageTypeAPIKeyRisk = "api_key_risk"
)

type UserMessage struct {
	ID        int64          `json:"id"`
	UserID    int64          `json:"user_id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Content   string         `json:"content"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type CreateUserMessageInput struct {
	UserID   int64
	Type     string
	Title    string
	Content  string
	Metadata map[string]any
}

type UserMessageRepository interface {
	Create(ctx context.Context, input CreateUserMessageInput) (*UserMessage, error)
	ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams, unreadOnly bool) ([]UserMessage, *pagination.PaginationResult, error)
	MarkRead(ctx context.Context, userID, messageID int64, readAt time.Time) error
}

type UserMessageService struct {
	repo UserMessageRepository
}

func NewUserMessageService(repo UserMessageRepository) *UserMessageService {
	return &UserMessageService{repo: repo}
}

func (s *UserMessageService) Create(ctx context.Context, input CreateUserMessageInput) (*UserMessage, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("user message service unavailable")
	}
	return s.repo.Create(ctx, input)
}

func (s *UserMessageService) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams, unreadOnly bool) ([]UserMessage, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("user message service unavailable")
	}
	return s.repo.ListByUser(ctx, userID, params, unreadOnly)
}

func (s *UserMessageService) MarkRead(ctx context.Context, userID, messageID int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("user message service unavailable")
	}
	return s.repo.MarkRead(ctx, userID, messageID, time.Now())
}
