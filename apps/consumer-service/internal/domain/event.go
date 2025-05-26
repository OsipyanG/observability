package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	UserCreatedEvent EventType = "user.created"
)

// Event represents an event in the system
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Version   string                 `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// UserEventData represents the data of a user event
type UserEventData struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Action   string `json:"action"`
}

// FromJSON creates an event from JSON
func FromJSON(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}

// ToJSON serializes the event to JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// Validate validates the event
func (e *Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if e.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if e.Source == "" {
		return fmt.Errorf("event source is required")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}
	return nil
}

// GetUserData extracts user data from the event
func (e *Event) GetUserData() (*UserEventData, error) {
	dataBytes, err := json.Marshal(e.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	var userData UserEventData
	if err := json.Unmarshal(dataBytes, &userData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	return &userData, nil
}

// IsUserEvent checks if the event is a user event
func (e *Event) IsUserEvent() bool {
	switch e.Type {
	case UserCreatedEvent:
		return true
	default:
		return false
	}
}
