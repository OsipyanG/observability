package kafka

import (
	"context"
	"fmt"
	"strings"
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

// MessageBatch представляет batch сообщений для обработки
type MessageBatch struct {
	Messages []kafka.Message
	Events   []*domain.Event
}

// Consumer реализует Kafka consumer с поддержкой параллельной обработки
type Consumer struct {
	reader      *kafka.Reader
	processor   EventProcessor
	logger      *logrus.Logger
	metrics     ConsumerMetrics
	config      config.KafkaConfig
	mu          sync.RWMutex
	closed      bool
	wg          sync.WaitGroup
	workerCount int
	batchSize   int
	messageChan chan kafka.Message
	commitChan  chan kafka.Message
}

// NewConsumer создает новый Kafka consumer с параллельной обработкой
func NewConsumer(cfg config.KafkaConfig, consumerCfg config.ConsumerConfig, processor EventProcessor, logger *logrus.Logger, metrics ConsumerMetrics) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
	}

	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic is empty")
	}

	if cfg.GroupID == "" {
		return nil, fmt.Errorf("kafka group ID is empty")
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

	// Создаем Kafka reader
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
		reader:      reader,
		processor:   processor,
		logger:      logger,
		metrics:     metrics,
		config:      cfg,
		workerCount: consumerCfg.WorkerCount,
		batchSize:   consumerCfg.BatchSize,
		messageChan: make(chan kafka.Message, consumerCfg.WorkerCount*2),
		commitChan:  make(chan kafka.Message, consumerCfg.BatchSize*2),
	}

	logger.WithFields(logrus.Fields{
		"brokers":      cfg.Brokers,
		"topic":        cfg.Topic,
		"group_id":     cfg.GroupID,
		"worker_count": consumerCfg.WorkerCount,
		"batch_size":   consumerCfg.BatchSize,
	}).Info("Kafka consumer initialized with parallel processing")

	return consumer, nil
}

// Start запускает consumer с параллельной обработкой
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("consumer is closed")
	}
	c.mu.Unlock()

	c.logger.Info("Starting Kafka consumer with parallel processing")

	// Запускаем горутину для сбора статистики
	c.wg.Add(1)
	go c.collectStats(ctx)

	// Запускаем worker'ы для обработки сообщений
	for i := 0; i < c.workerCount; i++ {
		c.wg.Add(1)
		go c.messageWorker(ctx, i)
	}

	// Запускаем batch committer
	c.wg.Add(1)
	go c.batchCommitter(ctx)

	// Основной цикл чтения сообщений
	c.wg.Add(1)
	go c.messageReader(ctx)

	// Ждем завершения всех горутин
	c.wg.Wait()
	return nil
}

// messageReader читает сообщения из Kafka и отправляет их в канал для обработки
func (c *Consumer) messageReader(ctx context.Context) {
	defer c.wg.Done()
	defer close(c.messageChan)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Message reader context cancelled, stopping")
			return
		default:
			c.mu.RLock()
			if c.closed {
				c.mu.RUnlock()
				return
			}
			reader := c.reader
			c.mu.RUnlock()

			// Создаем контекст с таймаутом для чтения сообщения
			readCtx, cancel := context.WithTimeout(ctx, c.config.MaxWait*2)

			// Читаем сообщение с таймаутом
			message, err := reader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return
				}

				// Проверяем, является ли это обычным таймаутом (пустой топик)
				if isTimeoutError(err) {
					// Для пустого топика это нормально, не логируем как ошибку
					c.logger.WithError(err).Debug("No messages available, waiting...")
					time.Sleep(c.config.RetryBackoff)
					continue
				}

				// Логируем только реальные ошибки
				c.logger.WithError(err).Warn("Error reading message from Kafka")
				time.Sleep(c.config.RetryBackoff)
				continue
			}

			// Отправляем сообщение в канал для обработки
			select {
			case c.messageChan <- message:
			case <-ctx.Done():
				return
			}
		}
	}
}

// isTimeoutError проверяет, является ли ошибка таймаутом чтения
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Проверяем различные типы таймаут ошибок Kafka
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "i/o timeout")
}

// messageWorker обрабатывает сообщения из канала
func (c *Consumer) messageWorker(ctx context.Context, workerID int) {
	defer c.wg.Done()

	logger := c.logger.WithField("worker_id", workerID)
	logger.Info("Message worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Message worker context cancelled, stopping")
			return
		case message, ok := <-c.messageChan:
			if !ok {
				logger.Info("Message channel closed, stopping worker")
				return
			}

			if err := c.processMessage(ctx, message); err != nil {
				logger.WithError(err).Error("Failed to process message")
				continue
			}

			// Отправляем сообщение для коммита
			select {
			case c.commitChan <- message:
			case <-ctx.Done():
				return
			}
		}
	}
}

// processMessage обрабатывает одно сообщение
func (c *Consumer) processMessage(ctx context.Context, message kafka.Message) error {
	start := time.Now()

	// Парсим событие из JSON
	event, err := domain.FromJSON(message.Value)
	if err != nil {
		c.metrics.IncFailedEvents("unknown", "parse_error")
		c.logger.WithFields(logrus.Fields{
			"offset":    message.Offset,
			"partition": message.Partition,
			"error":     err,
		}).Error("Failed to parse event")
		return nil // Не возвращаем ошибку, чтобы не блокировать обработку
	}

	// Валидируем событие
	if err := event.Validate(); err != nil {
		c.metrics.IncFailedEvents(string(event.Type), "validation_error")
		c.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"error":      err,
		}).Error("Event validation failed")
		return nil // Не возвращаем ошибку, чтобы не блокировать обработку
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

// batchCommitter коммитит сообщения batch'ами
func (c *Consumer) batchCommitter(ctx context.Context) {
	defer c.wg.Done()
	defer close(c.commitChan)

	ticker := time.NewTicker(time.Second) // Коммитим каждую секунду
	defer ticker.Stop()

	var batch []kafka.Message
	maxBatchSize := c.batchSize

	commitBatch := func() {
		if len(batch) == 0 {
			return
		}

		start := time.Now()
		if err := c.commitMessages(ctx, batch); err != nil {
			c.logger.WithError(err).Error("Failed to commit message batch")
		} else {
			c.metrics.ObserveCommitDuration(time.Since(start))
			c.metrics.ObserveBatchSize(len(batch))
			c.logger.WithField("batch_size", len(batch)).Debug("Committed message batch")
		}
		batch = batch[:0] // Очищаем batch
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Batch committer context cancelled, committing final batch")
			commitBatch()
			return
		case <-ticker.C:
			commitBatch()
		case message, ok := <-c.commitChan:
			if !ok {
				c.logger.Info("Commit channel closed, committing final batch")
				commitBatch()
				return
			}

			batch = append(batch, message)
			if len(batch) >= maxBatchSize {
				commitBatch()
			}
		}
	}
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

// commitMessages коммитит batch сообщений
func (c *Consumer) commitMessages(ctx context.Context, messages []kafka.Message) error {
	c.mu.RLock()
	reader := c.reader
	c.mu.RUnlock()

	if err := reader.CommitMessages(ctx, messages...); err != nil {
		return fmt.Errorf("failed to commit messages: %w", err)
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
