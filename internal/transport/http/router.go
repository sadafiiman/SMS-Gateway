package httpapi

import (
	"net/http"

	"github.com/iman/sms-gateway/internal/service"
	"github.com/iman/sms-gateway/internal/transport/http/middleware"
)

func NewRouter(sms *service.SMSService) http.Handler {
	mux := http.NewServeMux()

	customerHandler := NewCustomerHandler(sms)
	smsHandler := NewSMSHandler(sms)

	mux.HandleFunc("GET /healthz", healthCheck)

	mux.HandleFunc("POST /api/v1/customers", customerHandler.CreateCustomer)
	mux.HandleFunc("GET /api/v1/customers/{id}", customerHandler.GetCustomer)
	mux.HandleFunc("POST /api/v1/customers/{id}/balance", customerHandler.IncreaseBalance)

	mux.HandleFunc("POST /api/v1/sms", smsHandler.Send)
	mux.HandleFunc("GET /api/v1/sms", smsHandler.ListReports)
	mux.HandleFunc("GET /api/v1/sms/{id}", smsHandler.GetMessage)

	return middleware.Chain(mux, middleware.Recover, middleware.Logging, middleware.JSON)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
