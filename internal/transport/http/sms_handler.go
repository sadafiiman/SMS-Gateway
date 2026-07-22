package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/iman/sms-gateway/internal/domain"
	"github.com/iman/sms-gateway/internal/service"
)

type SMSHandler struct {
	sms *service.SMSService
}

func NewSMSHandler(sms *service.SMSService) *SMSHandler {
	return &SMSHandler{sms: sms}
}

type sendMessageRequest struct {
	CustomerID string `json:"customer_id"`
	Sender     string `json:"sender"`
	Receiver   string `json:"receiver"`
	Message    string `json:"message"`
	Type       string `json:"type"` // "normal" (default) | "otp" | "express"
}

func (h *SMSHandler) Send(w http.ResponseWriter, r *http.Request) {
	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.CustomerID == "" {
		writeError(w, http.StatusBadRequest, "customer_id is required")
		return
	}

	msgType := domain.MessageTypeNormal
	if req.Type != "" {
		msgType = domain.MessageType(req.Type)
	}

	msg, err := h.sms.SendMessage(r.Context(), service.SendMessageInput{
		CustomerID: req.CustomerID,
		Sender:     req.Sender,
		Receiver:   req.Receiver,
		Body:       req.Message,
		Type:       msgType,
	})
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, msg)
}

func (h *SMSHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	msg, err := h.sms.GetMessage(r.Context(), id)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, msg)
}

type reportsResponse struct {
	Messages []*domain.Message `json:"messages"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

func (h *SMSHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	customerID := q.Get("customer_id")
	if customerID == "" {
		writeError(w, http.StatusBadRequest, "customer_id query parameter is required")
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	out, err := h.sms.ListReports(r.Context(), service.ListReportsInput{
		CustomerID: customerID,
		Type:       domain.MessageType(q.Get("type")),
		Status:     domain.MessageStatus(q.Get("status")),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		mapDomainError(w, err)
		return
	}
	if limit <= 0 {
		limit = 50
	}
	writeJSON(w, http.StatusOK, reportsResponse{
		Messages: out.Messages,
		Total:    out.Total,
		Limit:    limit,
		Offset:   offset,
	})
}
