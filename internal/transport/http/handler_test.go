package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iman/sms-gateway/internal/domain"
	"github.com/iman/sms-gateway/internal/repository/memory"
	"github.com/iman/sms-gateway/internal/service"
)

func newTestRouter() http.Handler {
	customerRepo := memory.NewCustomerRepository()
	messageRepo := memory.NewMessageRepository()
	router := service.NewOperatorRouter(
		[]service.OperatorGateway{service.NewSimulatedOperator("test-standard", 0, 0, 0)},
		[]service.OperatorGateway{service.NewSimulatedOperator("test-express", 0, 0, 0)},
	)
	dispatcher := service.NewDispatcher(router, messageRepo, 2, 1, 100)
	sms := service.NewSMSService(customerRepo, messageRepo, domain.PriceList{Normal: 100, OTP: 150, Express: 300}, dispatcher)
	return NewRouter(sms)
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateCustomerAndSendFlow(t *testing.T) {
	h := newTestRouter()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/customers", map[string]string{"name": "Acme"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var customer domain.Customer
	if err := json.Unmarshal(rec.Body.Bytes(), &customer); err != nil {
		t.Fatalf("decode customer: %v", err)
	}

	// Sending before any top-up must be rejected with 402.
	rec = doJSON(t, h, http.MethodPost, "/api/v1/sms", map[string]string{
		"customer_id": customer.ID, "receiver": "0912", "message": "hi",
	})
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 before top-up, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/customers/"+customer.ID+"/balance", map[string]int64{"amount": 500})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on top-up, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodPost, "/api/v1/sms", map[string]string{
		"customer_id": customer.ID, "receiver": "0912", "message": "hi", "type": "normal",
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodGet, "/api/v1/sms?customer_id="+customer.ID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on reports, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSendMessage_MissingCustomerID(t *testing.T) {
	h := newTestRouter()
	rec := doJSON(t, h, http.MethodPost, "/api/v1/sms", map[string]string{"receiver": "0912", "message": "hi"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetCustomer_NotFound(t *testing.T) {
	h := newTestRouter()
	rec := doJSON(t, h, http.MethodGet, "/api/v1/customers/does-not-exist", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHealthz(t *testing.T) {
	h := newTestRouter()
	rec := doJSON(t, h, http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
