package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"consumer-service/internal/config"
	"consumer-service/internal/domain"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// ConsumerMetrics интерфейс для метрик consumer
type ConsumerMetrics interface {
	IncConsumedEvents(eventType string)
	IncFailedEvents(eventType string, reason string)
	ObserveProcessingDuration(eventType string, duration time.Duration)
	ObserveCommitDuration(duration time.Duration)
	ObserveBatchSize(size int)
	UpdateKafkaReaderStats(messages, bytes, rebalances, timeouts, errors int64)
}

// EventProcessor интерфейс для обработки событий
type EventProcessor interface {
	ProcessEvent(ctx context.Context, event *domain.Event) error
}

// Consumer реализует Kafka consumer
type Consumer struct {
	reader    *kafka.Reader
	processor EventProcessor
	logger    *logrus.Logger
	metrics   ConsumerMetrics
	config    config.KafkaConfig
	mu        sync.RWMutex
	closed    bool
	wg        sync.WaitGroup
}

// NewConsumer создает новый Kafka consumer
func NewConsumer(cfg config.KafkaConfig, processor EventProcessor, logger *logrus.Logger, metrics ConsumerMetrics) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers not configured")
	}

	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic not configured")
	}

	if cfg.GroupID == "" {
		return nil, fmt.Errorf("kafka group ID not configured")
	}

	// Определяем начальный offset
	var startOffset int64
	switch cfg.StartOffset {
	case "earliest":
		startOffset = kafka.FirstOffset
	case "latest":
		startOffset = kafka.LastOffset
	default:
		startOffset = kafka.LastOffset
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		MaxWait:        cfg.MaxWait,
		CommitInterval: cfg.CommitInterval,
		StartOffset:    startOffset,
		ErrorLogger:    kafka.LoggerFunc(logger.Errorf),
	})

	consumer := &Consumer{
		reader:    reader,
		processor: processor,
		logger:    logger,
		metrics:   metrics,
		config:    cfg,
	}

	logger.WithFields(logrus.Fields{
		"brokers":  cfg.Brokers,
		"topic":    cfg.Topic,
		"group_id": cfg.GroupID,
	}).Info("Kafka consumer initialized")

	return consumer, nil
}

// Start запускает consumer
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("consumer is closed")
	}
	c.mu.Unlock()

	c.logger.Info("Starting Kafka consumer")

	// Запускаем горутину для сбора статистики
	c.wg.Add(1)
	go c.collectStats(ctx)

	// Основной цикл потребления
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Consumer context cancelled, stopping")
			return ctx.Err()
		default:
			if err := c.consumeMessage(ctx); err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return err
				}
				c.logger.WithError(err).Error("Error consuming message")
				// Небольшая пауза перед повтором
				time.Sleep(c.config.RetryBackoff)
			}
		}
	}
}

// consumeMessage потребляет одно сообщение
func (c *Consumer) consumeMessage(ctx context.Context) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("consumer is closed")
	}
	reader := c.reader
	c.mu.RUnlock()

	// Читаем сообщение с таймаутом
	message, err := reader.FetchMessage(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch message: %w", err)
	}

	start := time.Now()

	// Парсим событие
	event, err := domain.NewEvent(domain.UserCreatedEvent, string(message.Value))
	if err != nil {
		c.metrics.IncFailedEvents("unknown", "parse_error")
		c.logger.WithFields(logrus.Fields{
			"offset":    message.Offset,
			"partition": message.Partition,
			"error":     err,
		}).Error("Failed to parse event")

		// Коммитим сообщение даже если не смогли его распарсить
		if commitErr := c.commitMessage(ctx, message); commitErr != nil {
			c.logger.WithError(commitErr).Error("Failed to commit message after parse error")
		}
		return nil
	}

	// Валидируем событие
	if err := event.Validate(); err != nil {
		c.metrics.IncFailedEvents(string(event.Type), "validation_error")
		c.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"error":      err,
		}).Error("Event validation failed")

		// Коммитим невалидное сообщение
		if commitErr := c.commitMessage(ctx, message); commitErr != nil {
			c.logger.WithError(commitErr).Error("Failed to commit message after validation error")
		}
		return nil
	}

	// Обрабатываем событие с retry логикой
	if err := c.processEventWithRetry(ctx, event); err != nil {
		c.metrics.IncFailedEvents(string(event.Type), "processing_error")
		c.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"error":      err,
		}).Error("Failed to process event")
		return err
	}

	// Коммитим успешно обработанное сообщение
	if err := c.commitMessage(ctx, message); err != nil {
		c.logger.WithError(err).Error("Failed to commit message")
		return err
	}

	// Записываем метрики
	duration := time.Since(start)
	c.metrics.IncConsumedEvents(string(event.Type))
	c.metrics.ObserveProcessingDuration(string(event.Type), duration)

	c.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"duration":   duration,
		"offset":     message.Offset,
		"partition":  message.Partition,
	}).Debug("Event processed successfully")

	return nil
}

// processEventWithRetry обрабатывает событие с retry логикой
func (c *Consumer) processEventWithRetry(ctx context.Context, event *domain.Event) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Экспоненциальная задержка
			backoff := time.Duration(attempt) * c.config.RetryBackoff
			c.logger.WithFields(logrus.Fields{
				"event_id": event.ID,
				"attempt":  attempt,
				"backoff":  backoff,
			}).Warn("Retrying event processing")

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		if err := c.processor.ProcessEvent(ctx, event); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to process event after %d attempts: %w", c.config.MaxRetries, lastErr)
}

// commitMessage коммитит сообщение
func (c *Consumer) commitMessage(ctx context.Context, message kafka.Message) error {
	start := time.Now()
	defer func() {
		c.metrics.ObserveCommitDuration(time.Since(start))
	}()

	c.mu.RLock()
	reader := c.reader
	c.mu.RUnlock()

	if err := reader.CommitMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to commit message: %w", err)
	}

	return nil
}

// collectStats собирает статистику Kafka reader
func (c *Consumer) collectStats(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			if c.closed {
				c.mu.RUnlock()
				return
			}
			stats := c.reader.Stats()
			c.mu.RUnlock()

			c.metrics.UpdateKafkaReaderStats(
				stats.Messages,
				stats.Bytes,
				stats.Rebalances,
				stats.Timeouts,
				stats.Errors,
			)
		}
	}
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.logger.Info("Closing Kafka consumer")

	// Ждем завершения горутин
	c.wg.Wait()

	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("failed to close kafka reader: %w", err)
	}

	c.logger.Info("Kafka consumer closed")
	return nil
}
