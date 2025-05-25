package domain

import "context"

// EventPublisher интерфейс для публикации событий
type EventPublisher interface {
	Publish(ctx context.Context, event *Event) error
	Close() error
}

// MetricsCollector интерфейс для сбора метрик
type MetricsCollector interface {
	IncHTTPRequests(method, endpoint, status string)
	ObserveHTTPDuration(method, endpoint string, duration float64)
}

// EventUseCase интерфейс для use cases событий
type EventUseCase interface {
	CreateUserEvent(ctx context.Context, data string) (*Event, error)
	CreateOrderEvent(ctx context.Context, data string) (*Event, error)
	CreatePaymentEvent(ctx context.Context, data string) (*Event, error)
}
