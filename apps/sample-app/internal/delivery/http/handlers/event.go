package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"sample-app/internal/domain"
)

// EventRequest представляет запрос на создание события
type EventRequest struct {
	Data string `json:"data"`
}

// EventResponse представляет ответ при создании события
type EventResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Event   *domain.Event `json:"event"`
}

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Validate проверяет валидность запроса
func (r *EventRequest) Validate() error {
	r.Data = strings.TrimSpace(r.Data)
	if r.Data == "" {
		return domain.ErrInvalidEventData
	}
	if len(r.Data) > 1000 {
		return domain.ErrEventDataTooLong
	}
	return nil
}

// EventHandler обрабатывает HTTP запросы для событий
type EventHandler struct {
	eventUseCase domain.EventUseCase
}

// NewEventHandler создает новый EventHandler
func NewEventHandler(eventUseCase domain.EventUseCase) *EventHandler {
	return &EventHandler{
		eventUseCase: eventUseCase,
	}
}

// CreateUserEvent обрабатывает создание события пользователя
func (h *EventHandler) CreateUserEvent(w http.ResponseWriter, r *http.Request) {
	req, err := h.parseAndValidateRequest(r, "New user has been created")
	if err != nil {
		h.writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	event, err := h.eventUseCase.CreateUserEvent(r.Context(), req.Data)
	if err != nil {
		h.writeErrorResponse(w, "Failed to create user event", http.StatusInternalServerError)
		return
	}

	h.writeSuccessResponse(w, "User created event sent to Kafka", event)
}

// CreateOrderEvent обрабатывает создание события заказа
func (h *EventHandler) CreateOrderEvent(w http.ResponseWriter, r *http.Request) {
	req, err := h.parseAndValidateRequest(r, "New order has been placed")
	if err != nil {
		h.writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	event, err := h.eventUseCase.CreateOrderEvent(r.Context(), req.Data)
	if err != nil {
		h.writeErrorResponse(w, "Failed to create order event", http.StatusInternalServerError)
		return
	}

	h.writeSuccessResponse(w, "Order placed event sent to Kafka", event)
}

// CreatePaymentEvent обрабатывает создание события платежа
func (h *EventHandler) CreatePaymentEvent(w http.ResponseWriter, r *http.Request) {
	req, err := h.parseAndValidateRequest(r, "Payment has been processed")
	if err != nil {
		h.writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	event, err := h.eventUseCase.CreatePaymentEvent(r.Context(), req.Data)
	if err != nil {
		h.writeErrorResponse(w, "Failed to create payment event", http.StatusInternalServerError)
		return
	}

	h.writeSuccessResponse(w, "Payment processed event sent to Kafka", event)
}

// parseAndValidateRequest парсит и валидирует запрос
func (h *EventHandler) parseAndValidateRequest(r *http.Request, defaultData string) (*EventRequest, error) {
	var req EventRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	if req.Data == "" {
		req.Data = defaultData
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	return &req, nil
}

// writeSuccessResponse записывает успешный ответ
func (h *EventHandler) writeSuccessResponse(w http.ResponseWriter, message string, event *domain.Event) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := EventResponse{
		Status:  "success",
		Message: message,
		Event:   event,
	}

	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse записывает ответ с ошибкой
func (h *EventHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}
