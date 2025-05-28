package domain

import (
	"crypto/rand"
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
)

// EventType представляет тип события
type EventType string

const (
	UserCreatedEvent EventType = "user_created"
)

// String возвращает строковое представление типа события
func (et EventType) String() string {
	return string(et)
}

// IsValid проверяет, является ли тип события валидным
func (et EventType) IsValid() bool {
	switch et {
	case UserCreatedEvent:
		return true
	default:
		return false
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

// NewEvent создает новое событие
func NewEvent(eventType EventType, data string) (*Event, error) {
	event := &Event{
		ID:        generateEventID(eventType),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().UTC(),
		Version:   "1.0",
		Source:    "producer-service",
	}

	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
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

	if e.Timestamp.After(time.Now().Add(time.Minute)) {
		return fmt.Errorf("%w: timestamp cannot be in the future", ErrInvalidTimestamp)
	}

	return nil
}

// FromJSON преобразует JSON в Event
func FromJSON(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
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

func generateEventID(eventType EventType) string {
	timestamp := time.Now().UTC().Format("20060102150405")
	randomSuffix := generateRandomString(EventIDLength)
	return fmt.Sprintf("%s_%s_%s", eventType, timestamp, randomSuffix)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
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

// IsValidEventType проверяет, является ли строка валидным типом события
func IsValidEventType(eventType string) bool {
	return EventType(strings.ToLower(eventType)).IsValid()
}
