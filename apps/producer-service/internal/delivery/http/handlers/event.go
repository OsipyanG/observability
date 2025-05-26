package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"producer-service/internal/domain"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// EventRequest представляет запрос на создание события
type EventRequest struct {
	Data     string                 `json:"data" validate:"required,min=1,max=10000"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EventResponse представляет ответ при создании события
type EventResponse struct {
	Status    string        `json:"status"`
	Message   string        `json:"message"`
	Event     *domain.Event `json:"event"`
	Timestamp time.Time     `json:"timestamp"`
}

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Code      string    `json:"code,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// StatsResponse представляет ответ со статистикой
type StatsResponse struct {
	Status string             `json:"status"`
	Data   *domain.EventStats `json:"data"`
}

// Validate проверяет валидность запроса
func (r *EventRequest) Validate() error {
	validate := validator.New()
	if err := validate.Struct(r); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	r.Data = strings.TrimSpace(r.Data)
	if r.Data == "" {
		return domain.ErrInvalidEventData
	}

	return nil
}

// EventHandler обрабатывает HTTP запросы для событий
type EventHandler struct {
	eventService domain.EventService
	logger       *logrus.Logger
	metrics      HTTPMetrics
}

// HTTPMetrics интерфейс для HTTP метрик
type HTTPMetrics interface {
	IncHTTPRequests(method, endpoint, status string)
	ObserveHTTPDuration(method, endpoint string, duration float64)
}

// NewEventHandler создает новый EventHandler
func NewEventHandler(eventService domain.EventService, logger *logrus.Logger, metrics HTTPMetrics) *EventHandler {
	return &EventHandler{
		eventService: eventService,
		logger:       logger,
		metrics:      metrics,
	}
}

// CreateUserEvent обрабатывает создание события пользователя
func (h *EventHandler) CreateUserEvent(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	endpoint := "/events/user"

	defer func() {
		duration := time.Since(start).Seconds()
		h.metrics.ObserveHTTPDuration(r.Method, endpoint, duration)
	}()

	req, err := h.parseAndValidateRequest(r)
	if err != nil {
		h.metrics.IncHTTPRequests(r.Method, endpoint, "400")
		h.writeErrorResponse(w, err.Error(), http.StatusBadRequest, "VALIDATION_ERROR")
		return
	}

	// Если данные не переданы, используем дефолтные
	if req.Data == "" {
		req.Data = `{"message": "New user has been created"}`
	}

	event, err := h.eventService.CreateUserEvent(r.Context(), req.Data)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
			"error":    err,
			"data":     req.Data,
		}).Error("Failed to create user event")

		h.metrics.IncHTTPRequests(r.Method, endpoint, "500")
		h.writeErrorResponse(w, "Failed to create user event", http.StatusInternalServerError, "INTERNAL_ERROR")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"endpoint": endpoint,
		"event_id": event.ID,
		"duration": time.Since(start),
	}).Info("User event created successfully")

	h.metrics.IncHTTPRequests(r.Method, endpoint, "200")
	h.writeSuccessResponse(w, "User created event sent to Kafka", event)
}

// CreateOrderEvent обрабатывает создание события заказа
func (h *EventHandler) CreateOrderEvent(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	endpoint := "/events/order"

	defer func() {
		duration := time.Since(start).Seconds()
		h.metrics.ObserveHTTPDuration(r.Method, endpoint, duration)
	}()

	req, err := h.parseAndValidateRequest(r)
	if err != nil {
		h.metrics.IncHTTPRequests(r.Method, endpoint, "400")
		h.writeErrorResponse(w, err.Error(), http.StatusBadRequest, "VALIDATION_ERROR")
		return
	}

	if req.Data == "" {
		req.Data = `{"message": "New order has been placed"}`
	}

	event, err := h.eventService.CreateOrderEvent(r.Context(), req.Data)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
			"error":    err,
			"data":     req.Data,
		}).Error("Failed to create order event")

		h.metrics.IncHTTPRequests(r.Method, endpoint, "500")
		h.writeErrorResponse(w, "Failed to create order event", http.StatusInternalServerError, "INTERNAL_ERROR")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"endpoint": endpoint,
		"event_id": event.ID,
		"duration": time.Since(start),
	}).Info("Order event created successfully")

	h.metrics.IncHTTPRequests(r.Method, endpoint, "200")
	h.writeSuccessResponse(w, "Order placed event sent to Kafka", event)
}

// CreatePaymentEvent обрабатывает создание события платежа
func (h *EventHandler) CreatePaymentEvent(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	endpoint := "/events/payment"

	defer func() {
		duration := time.Since(start).Seconds()
		h.metrics.ObserveHTTPDuration(r.Method, endpoint, duration)
	}()

	req, err := h.parseAndValidateRequest(r)
	if err != nil {
		h.metrics.IncHTTPRequests(r.Method, endpoint, "400")
		h.writeErrorResponse(w, err.Error(), http.StatusBadRequest, "VALIDATION_ERROR")
		return
	}

	if req.Data == "" {
		req.Data = `{"message": "Payment has been processed"}`
	}

	event, err := h.eventService.CreatePaymentEvent(r.Context(), req.Data)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
			"error":    err,
			"data":     req.Data,
		}).Error("Failed to create payment event")

		h.metrics.IncHTTPRequests(r.Method, endpoint, "500")
		h.writeErrorResponse(w, "Failed to create payment event", http.StatusInternalServerError, "INTERNAL_ERROR")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"endpoint": endpoint,
		"event_id": event.ID,
		"duration": time.Since(start),
	}).Info("Payment event created successfully")

	h.metrics.IncHTTPRequests(r.Method, endpoint, "200")
	h.writeSuccessResponse(w, "Payment processed event sent to Kafka", event)
}

// GetEventStats возвращает статистику событий
func (h *EventHandler) GetEventStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	endpoint := "/events/stats"

	defer func() {
		duration := time.Since(start).Seconds()
		h.metrics.ObserveHTTPDuration(r.Method, endpoint, duration)
	}()

	stats, err := h.eventService.GetEventStats(r.Context())
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
			"error":    err,
		}).Error("Failed to get event stats")

		h.metrics.IncHTTPRequests(r.Method, endpoint, "500")
		h.writeErrorResponse(w, "Failed to get event stats", http.StatusInternalServerError, "INTERNAL_ERROR")
		return
	}

	h.metrics.IncHTTPRequests(r.Method, endpoint, "200")
	h.writeStatsResponse(w, stats)
}

// parseAndValidateRequest парсит и валидирует запрос
func (h *EventHandler) parseAndValidateRequest(r *http.Request) (*EventRequest, error) {
	var req EventRequest

	if r.Body == nil {
		return &req, nil
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
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
		Status:    "success",
		Message:   message,
		Event:     event,
		Timestamp: time.Now().UTC(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.WithError(err).Error("Failed to encode success response")
	}
}

// writeStatsResponse записывает ответ со статистикой
func (h *EventHandler) writeStatsResponse(w http.ResponseWriter, stats *domain.EventStats) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := StatsResponse{
		Status: "success",
		Data:   stats,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.WithError(err).Error("Failed to encode stats response")
	}
}

// writeErrorResponse записывает ответ с ошибкой
func (h *EventHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:     http.StatusText(statusCode),
		Message:   message,
		Code:      code,
		Timestamp: time.Now().UTC(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.WithError(err).Error("Failed to encode error response")
	}
}
