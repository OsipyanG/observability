package domain

import (
	"context"
	"time"
)

// EventConsumer интерфейс для потребления событий
type EventConsumer interface {
	// Consume запускает потребление событий
	Consume(ctx context.Context) error

	// ConsumeBatch потребляет события батчами
	ConsumeBatch(ctx context.Context, batchSize int) ([]*Event, error)

	// Close закрывает consumer
	Close() error

	// Stats возвращает статистику consumer
	Stats() ConsumerStats
}

// EventHandler интерфейс для обработки событий
type EventHandler interface {
	// Handle обрабатывает одно событие
	Handle(ctx context.Context, event *Event) (*ProcessingResult, error)

	// HandleBatch обрабатывает батч событий
	HandleBatch(ctx context.Context, events []*Event) ([]*ProcessingResult, error)

	// CanHandle проверяет, может ли handler обработать событие
	CanHandle(eventType EventType) bool

	// GetSupportedTypes возвращает поддерживаемые типы событий
	GetSupportedTypes() []EventType
}

// EventProcessor интерфейс для процессинга событий
type EventProcessor interface {
	// Process обрабатывает событие
	Process(ctx context.Context, event *Event) (*ProcessingResult, error)

	// ProcessBatch обрабатывает батч событий
	ProcessBatch(ctx context.Context, events []*Event) ([]*ProcessingResult, error)

	// Start запускает процессор
	Start(ctx context.Context) error

	// Stop останавливает процессор
	Stop(ctx context.Context) error

	// GetStats возвращает статистику процессора
	GetStats() ProcessorStats
}

// EventRepository интерфейс для сохранения результатов обработки
type EventRepository interface {
	// SaveResult сохраняет результат обработки
	SaveResult(ctx context.Context, result *ProcessingResult) error

	// GetResults получает результаты обработки
	GetResults(ctx context.Context, filter ResultFilter) ([]*ProcessingResult, error)

	// GetStats получает статистику обработки
	GetStats(ctx context.Context) (*ProcessingStats, error)
}

// HealthChecker интерфейс для проверки здоровья компонентов
type HealthChecker interface {
	// Check проверяет здоровье компонента
	Check(ctx context.Context) error

	// GetStatus возвращает статус компонента
	GetStatus() HealthStatus
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
	// IncProcessedEvents увеличивает счетчик обработанных событий
	IncProcessedEvents(eventType string)

	// IncFailedEvents увеличивает счетчик неудачных событий
	IncFailedEvents(eventType string, reason string)

	// ObserveProcessingDuration записывает время обработки
	ObserveProcessingDuration(eventType string, duration time.Duration)

	// IncBatchSize записывает размер батча
	IncBatchSize(size int)

	// SetActiveWorkers устанавливает количество активных воркеров
	SetActiveWorkers(count int)
}

// ConsumerStats статистика consumer
type ConsumerStats struct {
	MessagesConsumed int64     `json:"messages_consumed"`
	BytesConsumed    int64     `json:"bytes_consumed"`
	Errors           int64     `json:"errors"`
	LastMessageTime  time.Time `json:"last_message_time"`
	Lag              int64     `json:"lag"`
}

// ProcessorStats статистика процессора
type ProcessorStats struct {
	EventsProcessed   int64            `json:"events_processed"`
	EventsFailed      int64            `json:"events_failed"`
	ProcessingRate    float64          `json:"processing_rate"`
	AverageLatency    time.Duration    `json:"average_latency"`
	EventsByType      map[string]int64 `json:"events_by_type"`
	ActiveWorkers     int              `json:"active_workers"`
	LastProcessedTime time.Time        `json:"last_processed_time"`
}

// ProcessingStats общая статистика обработки
type ProcessingStats struct {
	TotalEvents      int64            `json:"total_events"`
	SuccessfulEvents int64            `json:"successful_events"`
	FailedEvents     int64            `json:"failed_events"`
	EventsByType     map[string]int64 `json:"events_by_type"`
	SuccessRate      float64          `json:"success_rate"`
	AverageLatency   time.Duration    `json:"average_latency"`
	LastEventTime    *time.Time       `json:"last_event_time,omitempty"`
}

// ResultFilter фильтр для поиска результатов
type ResultFilter struct {
	EventType EventType  `json:"event_type,omitempty"`
	Success   *bool      `json:"success,omitempty"`
	From      *time.Time `json:"from,omitempty"`
	To        *time.Time `json:"to,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// HealthStatus статус здоровья компонента
type HealthStatus struct {
	Healthy   bool                   `json:"healthy"`
	LastCheck time.Time              `json:"last_check"`
	Error     string                 `json:"error,omitempty"`
	Component string                 `json:"component"`
	Details   map[string]interface{} `json:"details,omitempty"`
}
