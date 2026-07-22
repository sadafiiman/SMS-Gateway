package domain

import (
	"context"
	"time"
)

type CustomerRepository interface {
	Create(ctx context.Context, c *Customer) error
	Get(ctx context.Context, id string) (*Customer, error)
	DebitBalance(ctx context.Context, id string, amount int64) (newBalance int64, err error)
	CreditBalance(ctx context.Context, id string, amount int64) (newBalance int64, err error)
}

type MessageRepository interface {
	Save(ctx context.Context, m *Message) error
	MarkSent(ctx context.Context, id string, operator string, sentAt time.Time) error
	MarkFailed(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*Message, error)
	List(ctx context.Context, filter MessageFilter) ([]*Message, int, error)
}

type MessageFilter struct {
	CustomerID string
	Type       MessageType
	Status     MessageStatus
	Limit      int
	Offset     int
}
