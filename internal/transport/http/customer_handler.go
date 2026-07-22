package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/iman/sms-gateway/internal/service"
)

type CustomerHandler struct {
	sms *service.SMSService
}

func NewCustomerHandler(sms *service.SMSService) *CustomerHandler {
	return &CustomerHandler{sms: sms}
}

type createCustomerRequest struct {
	Name string `json:"name"`
}

// CreateCustomer handles POST /api/v1/customers
// The challenge requires no authentication/identity system, so this
// endpoint is intentionally open: it just registers a billable account
// and hands back its ID, which the caller then uses on every subsequent
// request as customer_id.
func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req createCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	customer, err := h.sms.CreateCustomer(r.Context(), service.CreateCustomerInput{Name: req.Name})
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, customer)
}

// GetCustomer handles GET /api/v1/customers/{id}
func (h *CustomerHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	customer, err := h.sms.GetCustomer(r.Context(), id)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

type increaseBalanceRequest struct {
	Amount int64 `json:"amount"`
}

type increaseBalanceResponse struct {
	CustomerID string `json:"customer_id"`
	Balance    int64  `json:"balance"`
}

// IncreaseBalance handles POST /api/v1/customers/{id}/balance
func (h *CustomerHandler) IncreaseBalance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req increaseBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	balance, err := h.sms.IncreaseBalance(r.Context(), id, req.Amount)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, increaseBalanceResponse{CustomerID: id, Balance: balance})
}
