package usecase

import (
	"context"
	"fmt"

	"sample-app/internal/domain"
)

// EventUseCase реализует интерфейс EventUseCase
type EventUseCase struct {
	publisher domain.EventPublisher
	metrics   domain.MetricsCollector
}

// NewEventUseCase создает новый EventUseCase
func NewEventUseCase(publisher domain.EventPublisher, metrics domain.MetricsCollector) *EventUseCase {
	return &EventUseCase{
		publisher: publisher,
		metrics:   metrics,
	}
}

// CreateUserEvent создает событие создания пользователя
func (u *EventUseCase) CreateUserEvent(ctx context.Context, data string) (*domain.Event, error) {
	event := domain.NewEvent(domain.UserCreatedEvent, data)

	if err := event.Validate(); err != nil {
		return nil, err
	}

	if err := u.publisher.Publish(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to publish user event: %w", err)
	}

	return event, nil
}

// CreateOrderEvent создает событие размещения заказа
func (u *EventUseCase) CreateOrderEvent(ctx context.Context, data string) (*domain.Event, error) {
	event := domain.NewEvent(domain.OrderPlacedEvent, data)

	if err := event.Validate(); err != nil {
		return nil, err
	}

	if err := u.publisher.Publish(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to publish order event: %w", err)
	}

	return event, nil
}

// CreatePaymentEvent создает событие обработки платежа
func (u *EventUseCase) CreatePaymentEvent(ctx context.Context, data string) (*domain.Event, error) {
	event := domain.NewEvent(domain.PaymentProcessedEvent, data)

	if err := event.Validate(); err != nil {
		return nil, err
	}

	if err := u.publisher.Publish(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to publish payment event: %w", err)
	}

	return event, nil
}
