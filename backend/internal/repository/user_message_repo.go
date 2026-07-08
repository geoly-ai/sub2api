package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/usermessage"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type userMessageRepository struct {
	client *dbent.Client
}

func NewUserMessageRepository(client *dbent.Client) service.UserMessageRepository {
	return &userMessageRepository{client: client}
}

func (r *userMessageRepository) Create(ctx context.Context, input service.CreateUserMessageInput) (*service.UserMessage, error) {
	metadata := input.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	m, err := r.client.UserMessage.Create().
		SetUserID(input.UserID).
		SetType(input.Type).
		SetTitle(input.Title).
		SetContent(input.Content).
		SetStatus(service.UserMessageStatusUnread).
		SetMetadata(metadata).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return userMessageEntityToService(m), nil
}

func (r *userMessageRepository) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams, unreadOnly bool) ([]service.UserMessage, *pagination.PaginationResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	q := r.client.UserMessage.Query().Where(usermessage.UserIDEQ(userID))
	if unreadOnly {
		q = q.Where(usermessage.StatusEQ(service.UserMessageStatusUnread))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	items, err := q.Order(dbent.Desc(usermessage.FieldCreatedAt), dbent.Desc(usermessage.FieldID)).
		Limit(params.PageSize).
		Offset((params.Page - 1) * params.PageSize).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]service.UserMessage, 0, len(items))
	for _, item := range items {
		out = append(out, *userMessageEntityToService(item))
	}
	pages := 0
	if params.PageSize > 0 {
		pages = (total + params.PageSize - 1) / params.PageSize
	}
	return out, &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize, Pages: pages}, nil
}

func (r *userMessageRepository) MarkRead(ctx context.Context, userID, messageID int64, readAt time.Time) error {
	_, err := r.client.UserMessage.Update().
		Where(usermessage.IDEQ(messageID), usermessage.UserIDEQ(userID)).
		SetStatus(service.UserMessageStatusRead).
		SetReadAt(readAt).
		Save(ctx)
	return err
}

func userMessageEntityToService(m *dbent.UserMessage) *service.UserMessage {
	if m == nil {
		return nil
	}
	return &service.UserMessage{
		ID:        m.ID,
		UserID:    m.UserID,
		Type:      m.Type,
		Title:     m.Title,
		Content:   m.Content,
		Status:    m.Status,
		Metadata:  m.Metadata,
		ReadAt:    m.ReadAt,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
