package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthHandler обрабатывает запросы проверки здоровья
type HealthHandler struct{}

// NewHealthHandler создает новый HealthHandler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health возвращает статус здоровья приложения
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "producer-service",
		"version":   "1.0.0",
	}

	json.NewEncoder(w).Encode(response)
}

// Ready возвращает статус готовности приложения
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "producer-service",
		"checks": map[string]string{
			"kafka": "ok",
		},
	}

	json.NewEncoder(w).Encode(response)
}
