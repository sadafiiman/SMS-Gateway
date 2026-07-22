package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/iman/sms-gateway/internal/domain"
	"github.com/iman/sms-gateway/internal/repository/memory"
)

func newTestService(t *testing.T) (*SMSService, *domain.Customer) {
	t.Helper()
	customerRepo := memory.NewCustomerRepository()
	messageRepo := memory.NewMessageRepository()

	router := NewOperatorRouter(
		[]OperatorGateway{NewSimulatedOperator("test-standard", 0, 0, 0)},
		[]OperatorGateway{NewSimulatedOperator("test-express", 0, 0, 0)},
	)
	dispatcher := NewDispatcher(router, messageRepo, 4, 2, 1000)
	prices := domain.PriceList{Normal: 100, OTP: 150, Express: 300}
	svc := NewSMSService(customerRepo, messageRepo, prices, dispatcher)

	ctx := context.Background()
	c, err := svc.CreateCustomer(ctx, CreateCustomerInput{Name: "Test Co"})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	return svc, c
}

func TestSendMessage_DebitsExactPrice(t *testing.T) {
	svc, c := newTestService(t)
	ctx := context.Background()

	if _, err := svc.IncreaseBalance(ctx, c.ID, 1000); err != nil {
		t.Fatalf("increase balance: %v", err)
	}

	msg, err := svc.SendMessage(ctx, SendMessageInput{
		CustomerID: c.ID,
		Receiver:   "0912",
		Body:       "hi",
		Type:       domain.MessageTypeOTP,
	})
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if msg.Price != 150 {
		t.Fatalf("expected OTP price 150, got %d", msg.Price)
	}

	updated, err := svc.GetCustomer(ctx, c.ID)
	if err != nil {
		t.Fatalf("get customer: %v", err)
	}
	if updated.Balance != 850 {
		t.Fatalf("expected balance 850, got %d", updated.Balance)
	}
}

func TestSendMessage_RejectsWhenBalanceInsufficient(t *testing.T) {
	svc, c := newTestService(t)
	ctx := context.Background()

	_, err := svc.SendMessage(ctx, SendMessageInput{
		CustomerID: c.ID,
		Receiver:   "0912",
		Body:       "hi",
		Type:       domain.MessageTypeNormal,
	})
	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestSendMessage_RejectsInvalidType(t *testing.T) {
	svc, c := newTestService(t)
	ctx := context.Background()
	_, _ = svc.IncreaseBalance(ctx, c.ID, 1000)

	_, err := svc.SendMessage(ctx, SendMessageInput{
		CustomerID: c.ID,
		Receiver:   "0912",
		Body:       "hi",
		Type:       "carrier-pigeon",
	})
	if !errors.Is(err, domain.ErrInvalidMessageType) {
		t.Fatalf("expected ErrInvalidMessageType, got %v", err)
	}
}

func TestSendMessage_RejectsMissingReceiver(t *testing.T) {
	svc, c := newTestService(t)
	ctx := context.Background()
	_, _ = svc.IncreaseBalance(ctx, c.ID, 1000)

	_, err := svc.SendMessage(ctx, SendMessageInput{
		CustomerID: c.ID,
		Body:       "hi",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSendMessage_ConcurrentSendsNeverOversell(t *testing.T) {
	svc, c := newTestService(t)
	ctx := context.Background()

	const balance = int64(5_000)
	const price = int64(100) // normal
	const attempts = 200

	if _, err := svc.IncreaseBalance(ctx, c.ID, balance); err != nil {
		t.Fatalf("increase balance: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var succeeded int

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.SendMessage(ctx, SendMessageInput{
				CustomerID: c.ID,
				Receiver:   "0912",
				Body:       "load test",
				Type:       domain.MessageTypeNormal,
			})
			if err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			} else if !errors.Is(err, domain.ErrInsufficientBalance) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	final, err := svc.GetCustomer(ctx, c.ID)
	if err != nil {
		t.Fatalf("get customer: %v", err)
	}
	if final.Balance < 0 {
		t.Fatalf("balance went negative: %d", final.Balance)
	}

	expectedSuccesses := int(balance / price)
	if succeeded != expectedSuccesses {
		t.Fatalf("expected %d successful sends, got %d", expectedSuccesses, succeeded)
	}

	time.Sleep(200 * time.Millisecond)
	report, err := svc.ListReports(ctx, ListReportsInput{CustomerID: c.ID, Limit: 200})
	if err != nil {
		t.Fatalf("list reports: %v", err)
	}
	if report.Total != expectedSuccesses {
		t.Fatalf("expected %d recorded messages, got %d", expectedSuccesses, report.Total)
	}
	for _, m := range report.Messages {
		if m.Status != domain.StatusSent {
			t.Fatalf("message %s expected status sent, got %s", m.ID, m.Status)
		}
	}
}
