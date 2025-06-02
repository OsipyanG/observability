package domain

import "context"

// EventPublisher интерфейс для публикации событий
type EventPublisher interface {
	// Publish публикует событие
	Publish(ctx context.Context, event *Event) error

	// Close закрывает publisher
	Close() error
}

// EventService интерфейс для бизнес-логики работы с событиями
type EventService interface {
	// CreateAndPublish создает и публикует событие
	CreateAndPublish(ctx context.Context, eventType EventType, data string) (*Event, error)

	// GetEventStats получает статистику по событиям
	GetEventStats(ctx context.Context) (*EventStats, error)

	// CreateUserEvent создает событие создания пользователя
	CreateUserEvent(ctx context.Context, data string) (*Event, error)
}

// EventStats статистика по событиям
type EventStats struct {
	TotalEvents   int64            `json:"total_events"`
	EventsByType  map[string]int64 `json:"events_by_type"`
	LastEventTime *string          `json:"last_event_time,omitempty"`
	ErrorCount    int64            `json:"error_count"`
	SuccessRate   float64          `json:"success_rate"`
}

// HealthChecker интерфейс для проверки здоровья сервиса
type HealthChecker interface {
	// Check проверяет здоровье компонента
	Check(ctx context.Context) error
}

// Logger интерфейс для логирования
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// MetricsCollector интерфейс для сбора метрик
type MetricsCollector interface {
	IncHTTPRequests(method, endpoint, status string)
	ObserveHTTPDuration(method, endpoint string, duration float64)
}
