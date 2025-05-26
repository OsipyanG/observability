package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// Константы для валидации
const (
	MaxEventDataLength = 10000 // 10KB
	MinEventDataLength = 1
	EventIDLength      = 8
)

// Доменные ошибки
var (
	ErrInvalidEventData      = errors.New("event data cannot be empty")
	ErrEventDataTooLong      = errors.New("event data is too long")
	ErrInvalidEventType      = errors.New("invalid event type")
	ErrInvalidEventID        = errors.New("invalid event ID")
	ErrInvalidTimestamp      = errors.New("invalid timestamp")
	ErrEventValidationFailed = errors.New("event validation failed")
	ErrEventProcessingFailed = errors.New("event processing failed")
)

// EventType представляет тип события
type EventType string

const (
	UserCreatedEvent      EventType = "user_created"
	OrderPlacedEvent      EventType = "order_placed"
	PaymentProcessedEvent EventType = "payment_processed"
)

// String возвращает строковое представление типа события
func (et EventType) String() string {
	return string(et)
}

// IsValid проверяет, является ли тип события валидным
func (et EventType) IsValid() bool {
	switch et {
	case UserCreatedEvent, OrderPlacedEvent, PaymentProcessedEvent:
		return true
	default:
		return false
	}
}

// GetAllEventTypes возвращает все доступные типы событий
func GetAllEventTypes() []EventType {
	return []EventType{
		UserCreatedEvent,
		OrderPlacedEvent,
		PaymentProcessedEvent,
	}
}

// Event представляет доменное событие
type Event struct {
	ID        string    `json:"id" validate:"required,min=1"`
	Type      EventType `json:"type" validate:"required"`
	Data      string    `json:"data" validate:"required,min=1,max=10000"`
	Timestamp time.Time `json:"timestamp" validate:"required"`
	Version   string    `json:"version,omitempty"`
	Source    string    `json:"source,omitempty"`
}

// ProcessingResult результат обработки события
type ProcessingResult struct {
	EventID     string        `json:"event_id"`
	EventType   EventType     `json:"event_type"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
	ProcessedAt time.Time     `json:"processed_at"`
	Duration    time.Duration `json:"duration"`
}

// Validate проверяет валидность события
func (e *Event) Validate() error {
	// Структурная валидация
	validate := validator.New()
	if err := validate.Struct(e); err != nil {
		return fmt.Errorf("%w: %v", ErrEventValidationFailed, err)
	}

	// Бизнес-логика валидации
	if !e.Type.IsValid() {
		return fmt.Errorf("%w: %s", ErrInvalidEventType, e.Type)
	}

	if len(e.Data) > MaxEventDataLength {
		return fmt.Errorf("%w: data length %d exceeds maximum %d",
			ErrEventDataTooLong, len(e.Data), MaxEventDataLength)
	}

	if len(e.Data) < MinEventDataLength {
		return fmt.Errorf("%w: data length %d is below minimum %d",
			ErrInvalidEventData, len(e.Data), MinEventDataLength)
	}

	if e.Timestamp.IsZero() {
		return fmt.Errorf("%w: timestamp cannot be zero", ErrInvalidTimestamp)
	}

	return nil
}

// ToJSON сериализует событие в JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON десериализует событие из JSON
func FromJSON(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("invalid event from JSON: %w", err)
	}

	return &event, nil
}

// Clone создает копию события
func (e *Event) Clone() *Event {
	return &Event{
		ID:        e.ID,
		Type:      e.Type,
		Data:      e.Data,
		Timestamp: e.Timestamp,
		Version:   e.Version,
		Source:    e.Source,
	}
}

// GetEventTypeFromString преобразует строку в EventType
func GetEventTypeFromString(s string) (EventType, error) {
	eventType := EventType(strings.ToLower(strings.TrimSpace(s)))
	if !eventType.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidEventType, s)
	}
	return eventType, nil
}

// IsValidEventType проверяет, является ли строка валидным типом события
func IsValidEventType(eventType string) bool {
	et := EventType(strings.ToLower(strings.TrimSpace(eventType)))
	return et.IsValid()
}
