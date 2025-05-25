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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "sample-app",
	})
}
