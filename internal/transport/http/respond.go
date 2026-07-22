package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/iman/sms-gateway/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

type errorBody struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

func mapDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCustomerNotFound), errors.Is(err, domain.ErrMessageNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInsufficientBalance):
		writeError(w, http.StatusPaymentRequired, err.Error())
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrInvalidMessageType):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
