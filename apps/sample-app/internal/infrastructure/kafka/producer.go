package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"sample-app/internal/config"
	"sample-app/internal/domain"

	"github.com/segmentio/kafka-go"
)

// Producer реализует интерфейс EventPublisher
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer создает новый Kafka producer
func NewProducer(cfg config.KafkaConfig) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers not configured")
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{
		writer: writer,
		topic:  cfg.Topic,
	}, nil
}

// Publish публикует событие в Kafka
func (p *Producer) Publish(ctx context.Context, event *domain.Event) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message := kafka.Message{
		Key:   []byte(event.ID),
		Value: eventJSON,
		Time:  event.Timestamp,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte(event.Type)},
		},
	}

	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	return nil
}

// Close закрывает Kafka writer
func (p *Producer) Close() error {
	return p.writer.Close()
}
