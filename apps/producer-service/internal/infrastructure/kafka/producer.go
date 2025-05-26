package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"producer-service/internal/config"
	"producer-service/internal/domain"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// ProducerMetrics интерфейс для метрик producer
type ProducerMetrics interface {
	IncPublishedEvents(eventType string)
	IncFailedEvents(eventType string, reason string)
	ObservePublishDuration(eventType string, duration time.Duration)
	IncBatchSize(size int)
}

// Producer реализует интерфейс EventPublisher
type Producer struct {
	writer  *kafka.Writer
	topic   string
	logger  *logrus.Logger
	metrics ProducerMetrics
	config  config.KafkaConfig
	mu      sync.RWMutex
	closed  bool
}

// NewProducer создает новый Kafka producer с улучшенной конфигурацией
func NewProducer(cfg config.KafkaConfig, logger *logrus.Logger, metrics ProducerMetrics) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers not configured")
	}

	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic not configured")
	}

	// Настраиваем компрессию
	var compression kafka.Compression
	switch cfg.CompressionType {
	case "gzip":
		compression = kafka.Gzip
	case "snappy":
		compression = kafka.Snappy
	case "lz4":
		compression = kafka.Lz4
	case "zstd":
		compression = kafka.Zstd
	default:
		compression = 0 // no compression
	}

	// Настраиваем balancer
	balancer := &kafka.LeastBytes{}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     balancer,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),
		Compression:  compression,
		ErrorLogger:  kafka.LoggerFunc(logger.Errorf),
	}

	producer := &Producer{
		writer:  writer,
		topic:   cfg.Topic,
		logger:  logger,
		metrics: metrics,
		config:  cfg,
	}

	logger.WithFields(logrus.Fields{
		"brokers":     cfg.Brokers,
		"topic":       cfg.Topic,
		"batch_size":  cfg.BatchSize,
		"compression": cfg.CompressionType,
	}).Info("Kafka producer initialized")

	return producer, nil
}

// Publish публикует одно событие в Kafka
func (p *Producer) Publish(ctx context.Context, event *domain.Event) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.metrics.ObservePublishDuration(string(event.Type), duration)
	}()

	// Валидируем событие
	if err := event.Validate(); err != nil {
		p.metrics.IncFailedEvents(string(event.Type), "validation_error")
		return fmt.Errorf("event validation failed: %w", err)
	}

	// Сериализуем событие
	eventJSON, err := event.ToJSON()
	if err != nil {
		p.metrics.IncFailedEvents(string(event.Type), "serialization_error")
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Создаем сообщение Kafka
	message := kafka.Message{
		Key:   []byte(event.ID),
		Value: eventJSON,
		Time:  event.Timestamp,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte(event.Type)},
			{Key: "event-id", Value: []byte(event.ID)},
			{Key: "event-version", Value: []byte(event.Version)},
			{Key: "event-source", Value: []byte(event.Source)},
		},
	}

	// Публикуем с retry логикой
	err = p.publishWithRetry(ctx, message)
	if err != nil {
		p.metrics.IncFailedEvents(string(event.Type), "publish_error")
		p.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"error":      err,
		}).Error("Failed to publish event")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.metrics.IncPublishedEvents(string(event.Type))
	p.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"topic":      p.topic,
	}).Debug("Event published successfully")

	return nil
}

// PublishBatch публикует несколько событий в Kafka
func (p *Producer) PublishBatch(ctx context.Context, events []*domain.Event) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	if len(events) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.metrics.IncBatchSize(len(events))
		// Записываем среднее время для batch
		avgDuration := duration / time.Duration(len(events))
		for _, event := range events {
			p.metrics.ObservePublishDuration(string(event.Type), avgDuration)
		}
	}()

	// Подготавливаем сообщения
	messages := make([]kafka.Message, 0, len(events))
	for _, event := range events {
		// Валидируем событие
		if err := event.Validate(); err != nil {
			p.metrics.IncFailedEvents(string(event.Type), "validation_error")
			return fmt.Errorf("event validation failed for event %s: %w", event.ID, err)
		}

		// Сериализуем событие
		eventJSON, err := event.ToJSON()
		if err != nil {
			p.metrics.IncFailedEvents(string(event.Type), "serialization_error")
			return fmt.Errorf("failed to marshal event %s: %w", event.ID, err)
		}

		message := kafka.Message{
			Key:   []byte(event.ID),
			Value: eventJSON,
			Time:  event.Timestamp,
			Headers: []kafka.Header{
				{Key: "event-type", Value: []byte(event.Type)},
				{Key: "event-id", Value: []byte(event.ID)},
				{Key: "event-version", Value: []byte(event.Version)},
				{Key: "event-source", Value: []byte(event.Source)},
			},
		}
		messages = append(messages, message)
	}

	// Публикуем batch с retry логикой
	err := p.publishBatchWithRetry(ctx, messages)
	if err != nil {
		for _, event := range events {
			p.metrics.IncFailedEvents(string(event.Type), "publish_error")
		}
		p.logger.WithFields(logrus.Fields{
			"batch_size": len(events),
			"error":      err,
		}).Error("Failed to publish event batch")
		return fmt.Errorf("failed to publish event batch: %w", err)
	}

	// Обновляем метрики успеха
	for _, event := range events {
		p.metrics.IncPublishedEvents(string(event.Type))
	}

	p.logger.WithFields(logrus.Fields{
		"batch_size": len(events),
		"topic":      p.topic,
	}).Debug("Event batch published successfully")

	return nil
}

// publishWithRetry публикует сообщение с retry логикой
func (p *Producer) publishWithRetry(ctx context.Context, message kafka.Message) error {
	var lastErr error

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * p.config.RetryBackoff
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := p.writer.WriteMessages(ctx, message)
		if err == nil {
			return nil
		}

		lastErr = err
		p.logger.WithFields(logrus.Fields{
			"attempt":     attempt + 1,
			"max_retries": p.config.MaxRetries,
			"error":       err,
		}).Warn("Failed to publish message, retrying")
	}

	return fmt.Errorf("failed to publish after %d attempts: %w", p.config.MaxRetries+1, lastErr)
}

// publishBatchWithRetry публикует batch сообщений с retry логикой
func (p *Producer) publishBatchWithRetry(ctx context.Context, messages []kafka.Message) error {
	var lastErr error

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * p.config.RetryBackoff
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := p.writer.WriteMessages(ctx, messages...)
		if err == nil {
			return nil
		}

		lastErr = err
		p.logger.WithFields(logrus.Fields{
			"attempt":     attempt + 1,
			"max_retries": p.config.MaxRetries,
			"batch_size":  len(messages),
			"error":       err,
		}).Warn("Failed to publish batch, retrying")
	}

	return fmt.Errorf("failed to publish batch after %d attempts: %w", p.config.MaxRetries+1, lastErr)
}

// Close закрывает Kafka producer
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	err := p.writer.Close()
	if err != nil {
		p.logger.WithError(err).Error("Failed to close Kafka writer")
		return fmt.Errorf("failed to close kafka writer: %w", err)
	}

	p.logger.Info("Kafka producer closed")
	return nil
}

// Stats возвращает статистику producer
func (p *Producer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}
