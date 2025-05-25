package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

// Доменные ошибки
var (
	ErrInvalidEventData = errors.New("event data cannot be empty")
	ErrEventDataTooLong = errors.New("event data is too long")
	ErrInvalidEventType = errors.New("invalid event type")
)

// EventType представляет тип события
type EventType string

const (
	UserCreatedEvent      EventType = "user_created"
	OrderPlacedEvent      EventType = "order_placed"
	PaymentProcessedEvent EventType = "payment_processed"
)

// Event представляет доменное событие
type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Data      string    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// NewEvent создает новое событие
func NewEvent(eventType EventType, data string) *Event {
	return &Event{
		ID:        generateEventID(eventType),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}
}

// Validate проверяет валидность события
func (e *Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("event ID cannot be empty")
	}
	if e.Type == "" {
		return fmt.Errorf("event type cannot be empty")
	}
	if e.Data == "" {
		return fmt.Errorf("event data cannot be empty")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp cannot be zero")
	}
	return nil
}

func generateEventID(eventType EventType) string {
	timestamp := time.Now().UTC().Format("20060102150405")
	randomSuffix := generateRandomString(8)
	return fmt.Sprintf("%s_%s_%s", eventType, timestamp, randomSuffix)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)

	// Используем crypto/rand для криптографически стойкой генерации
	if _, err := rand.Read(b); err != nil {
		// Fallback на time-based если crypto/rand недоступен
		for i := range b {
			b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		}
		return string(b)
	}

	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}
