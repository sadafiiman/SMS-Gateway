package service

import (
	"context"
	"strings"
	"time"

	"github.com/iman/sms-gateway/internal/domain"
	"github.com/iman/sms-gateway/internal/idgen"
)

type SMSService struct {
	customers domain.CustomerRepository
	messages  domain.MessageRepository
	prices    domain.PriceList
	dispatch  *Dispatcher
	now       func() time.Time
}

func NewSMSService(customers domain.CustomerRepository, messages domain.MessageRepository, prices domain.PriceList, dispatch *Dispatcher) *SMSService {
	return &SMSService{
		customers: customers,
		messages:  messages,
		prices:    prices,
		dispatch:  dispatch,
		now:       time.Now,
	}
}

type CreateCustomerInput struct {
	Name string
}

func (s *SMSService) CreateCustomer(ctx context.Context, in CreateCustomerInput) (*domain.Customer, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	c := &domain.Customer{
		ID:        idgen.New(),
		Name:      in.Name,
		Balance:   0,
		CreatedAt: s.now(),
	}
	if err := s.customers.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *SMSService) GetCustomer(ctx context.Context, id string) (*domain.Customer, error) {
	return s.customers.Get(ctx, id)
}

// IncreaseBalance tops up a customer's prepaid balance. Amount must be
// a positive number of Rials.
func (s *SMSService) IncreaseBalance(ctx context.Context, customerID string, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, domain.ErrInvalidInput
	}
	return s.customers.CreditBalance(ctx, customerID, amount)
}

type SendMessageInput struct {
	CustomerID string
	Sender     string
	Receiver   string
	Body       string
	Type       domain.MessageType
}

func (s *SMSService) SendMessage(ctx context.Context, in SendMessageInput) (*domain.Message, error) {
	if strings.TrimSpace(in.Receiver) == "" || strings.TrimSpace(in.Body) == "" {
		return nil, domain.ErrInvalidInput
	}
	if in.Type == "" {
		in.Type = domain.MessageTypeNormal
	}
	if !in.Type.Valid() {
		return nil, domain.ErrInvalidMessageType
	}

	price := s.prices.For(in.Type)

	if _, err := s.customers.DebitBalance(ctx, in.CustomerID, price); err != nil {
		return nil, err
	}

	m := &domain.Message{
		ID:         idgen.New(),
		CustomerID: in.CustomerID,
		Sender:     in.Sender,
		Receiver:   in.Receiver,
		Body:       in.Body,
		Type:       in.Type,
		Price:      price,
		Status:     domain.StatusQueued,
		CreatedAt:  s.now(),
	}
	if in.Type == domain.MessageTypeExpress {
		deadline := s.now().Add(expressSLA)
		m.DeliveryDeadline = &deadline
	}

	if err := s.messages.Save(ctx, m); err != nil {
		_, _ = s.customers.CreditBalance(ctx, in.CustomerID, price)
		return nil, err
	}

	s.dispatch.Enqueue(m)
	return m, nil
}

const expressSLA = 5 * time.Second

func (s *SMSService) GetMessage(ctx context.Context, id string) (*domain.Message, error) {
	return s.messages.Get(ctx, id)
}

type ListReportsInput struct {
	CustomerID string
	Type       domain.MessageType
	Status     domain.MessageStatus
	Limit      int
	Offset     int
}

type ListReportsOutput struct {
	Messages []*domain.Message
	Total    int
}

func (s *SMSService) ListReports(ctx context.Context, in ListReportsInput) (*ListReportsOutput, error) {
	msgs, total, err := s.messages.List(ctx, domain.MessageFilter{
		CustomerID: in.CustomerID,
		Type:       in.Type,
		Status:     in.Status,
		Limit:      in.Limit,
		Offset:     in.Offset,
	})
	if err != nil {
		return nil, err
	}
	return &ListReportsOutput{Messages: msgs, Total: total}, nil
}
