package usecase

import (
	"context"

	"consumer-service/internal/domain"

	"github.com/sirupsen/logrus"
)

// EventProcessor реализует обработку событий
type EventProcessor struct {
	logger *logrus.Logger
}

// NewEventProcessor создает новый обработчик событий
func NewEventProcessor(logger *logrus.Logger) *EventProcessor {
	return &EventProcessor{
		logger: logger,
	}
}

// ProcessEvent обрабатывает событие
func (p *EventProcessor) ProcessEvent(ctx context.Context, event *domain.Event) error {
	p.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"source":     event.Source,
		"timestamp":  event.Timestamp,
	}).Debug("Processing event")

	// Проверяем контекст
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Обрабатываем в зависимости от типа события
	switch event.Type {
	case domain.UserCreatedEvent:
		return p.processUserCreated(ctx, event)
	default:
		return p.processUnknownEvent(ctx, event)
	}
}

// processUserCreated обрабатывает событие создания пользователя
func (p *EventProcessor) processUserCreated(ctx context.Context, event *domain.Event) error {
	p.logger.WithFields(logrus.Fields{
		"user_id":  event.ID,
		"username": event.Data,
		"email":    event.Data,
	}).Debug("User created event processed")

	// Проверяем контекст перед обработкой
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return nil
}

// processUnknownEvent обрабатывает неизвестные события
func (p *EventProcessor) processUnknownEvent(ctx context.Context, event *domain.Event) error {
	p.logger.WithFields(logrus.Fields{
		"event_type": event.Type,
	}).Debug("Unknown event type, skipping processing")

	// Для неизвестных событий просто логируем и считаем обработанными
	return nil
}
