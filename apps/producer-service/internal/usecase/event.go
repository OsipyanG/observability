package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"producer-service/internal/domain"

	"github.com/sirupsen/logrus"
)

// EventService реализует интерфейс domain.EventService
type EventService struct {
	publisher domain.EventPublisher
	logger    domain.Logger
	stats     *EventServiceStats
	mu        sync.RWMutex
}

// EventServiceStats статистика сервиса событий
type EventServiceStats struct {
	TotalEvents   int64            `json:"total_events"`
	EventsByType  map[string]int64 `json:"events_by_type"`
	ErrorCount    int64            `json:"error_count"`
	LastEventTime *time.Time       `json:"last_event_time,omitempty"`
}

// NewEventService создает новый EventService
func NewEventService(publisher domain.EventPublisher, logger *logrus.Logger) *EventService {
	return &EventService{
		publisher: publisher,
		logger:    &logrusAdapter{logger: logger},
		stats: &EventServiceStats{
			EventsByType: make(map[string]int64),
		},
	}
}

// CreateAndPublish создает и публикует событие
func (s *EventService) CreateAndPublish(ctx context.Context, eventType domain.EventType, data string) (*domain.Event, error) {
	start := time.Now()

	// Создаем событие
	event, err := domain.NewEvent(eventType, data)
	if err != nil {
		s.incrementErrorCount()
		s.logger.Error("Failed to create event",
			"event_type", eventType,
			"error", err)
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Публикуем событие
	if err := s.publisher.Publish(ctx, event); err != nil {
		s.incrementErrorCount()
		s.logger.Error("Failed to publish event",
			"event_id", event.ID,
			"event_type", event.Type,
			"error", err)
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}

	// Обновляем статистику
	s.updateStats(event, start)

	s.logger.Info("Event published successfully",
		"event_id", event.ID,
		"event_type", event.Type,
		"duration", time.Since(start))

	return event, nil
}

// CreateAndPublishJSON создает и публикует событие из JSON данных
func (s *EventService) CreateAndPublishJSON(ctx context.Context, eventType domain.EventType, data interface{}) (*domain.Event, error) {
	// Сериализуем данные в JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		s.incrementErrorCount()
		s.logger.Error("Failed to marshal data to JSON",
			"event_type", eventType,
			"error", err)
		return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	return s.CreateAndPublish(ctx, eventType, string(jsonData))
}

// GetEventStats возвращает статистику по событиям
func (s *EventService) GetEventStats(ctx context.Context) (*domain.EventStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var lastEventTime *string
	if s.stats.LastEventTime != nil {
		timeStr := s.stats.LastEventTime.Format(time.RFC3339)
		lastEventTime = &timeStr
	}

	successRate := float64(0)
	if s.stats.TotalEvents > 0 {
		successRate = float64(s.stats.TotalEvents-s.stats.ErrorCount) / float64(s.stats.TotalEvents) * 100
	}

	return &domain.EventStats{
		TotalEvents:   s.stats.TotalEvents,
		EventsByType:  s.stats.EventsByType,
		LastEventTime: lastEventTime,
		ErrorCount:    s.stats.ErrorCount,
		SuccessRate:   successRate,
	}, nil
}

// CreateUserEvent создает событие создания пользователя
func (s *EventService) CreateUserEvent(ctx context.Context, data string) (*domain.Event, error) {
	return s.CreateAndPublish(ctx, domain.UserCreatedEvent, data)
}

// CreateOrderEvent создает событие размещения заказа
func (s *EventService) CreateOrderEvent(ctx context.Context, data string) (*domain.Event, error) {
	return s.CreateAndPublish(ctx, domain.OrderPlacedEvent, data)
}

// CreatePaymentEvent создает событие обработки платежа
func (s *EventService) CreatePaymentEvent(ctx context.Context, data string) (*domain.Event, error) {
	return s.CreateAndPublish(ctx, domain.PaymentProcessedEvent, data)
}

// CreateUserEventJSON создает событие создания пользователя из JSON
func (s *EventService) CreateUserEventJSON(ctx context.Context, data interface{}) (*domain.Event, error) {
	return s.CreateAndPublishJSON(ctx, domain.UserCreatedEvent, data)
}

// CreateOrderEventJSON создает событие размещения заказа из JSON
func (s *EventService) CreateOrderEventJSON(ctx context.Context, data interface{}) (*domain.Event, error) {
	return s.CreateAndPublishJSON(ctx, domain.OrderPlacedEvent, data)
}

// CreatePaymentEventJSON создает событие обработки платежа из JSON
func (s *EventService) CreatePaymentEventJSON(ctx context.Context, data interface{}) (*domain.Event, error) {
	return s.CreateAndPublishJSON(ctx, domain.PaymentProcessedEvent, data)
}

// updateStats обновляет статистику сервиса
func (s *EventService) updateStats(event *domain.Event, startTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.TotalEvents++
	s.stats.EventsByType[string(event.Type)]++
	now := time.Now()
	s.stats.LastEventTime = &now
}

// incrementErrorCount увеличивает счетчик ошибок
func (s *EventService) incrementErrorCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.ErrorCount++
}

// logrusAdapter адаптер для logrus к domain.Logger интерфейсу
type logrusAdapter struct {
	logger *logrus.Logger
}

func (l *logrusAdapter) Debug(msg string, fields ...interface{}) {
	l.logger.WithFields(l.fieldsToLogrus(fields...)).Debug(msg)
}

func (l *logrusAdapter) Info(msg string, fields ...interface{}) {
	l.logger.WithFields(l.fieldsToLogrus(fields...)).Info(msg)
}

func (l *logrusAdapter) Warn(msg string, fields ...interface{}) {
	l.logger.WithFields(l.fieldsToLogrus(fields...)).Warn(msg)
}

func (l *logrusAdapter) Error(msg string, fields ...interface{}) {
	l.logger.WithFields(l.fieldsToLogrus(fields...)).Error(msg)
}

func (l *logrusAdapter) WithField(key string, value interface{}) domain.Logger {
	return &logrusAdapter{logger: l.logger.WithField(key, value).Logger}
}

func (l *logrusAdapter) WithFields(fields map[string]interface{}) domain.Logger {
	return &logrusAdapter{logger: l.logger.WithFields(fields).Logger}
}

func (l *logrusAdapter) fieldsToLogrus(fields ...interface{}) logrus.Fields {
	logrusFields := make(logrus.Fields)
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok && i+1 < len(fields) {
			logrusFields[key] = fields[i+1]
		}
	}
	return logrusFields
}
