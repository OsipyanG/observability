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
	ObserveBatchFlushDuration(duration time.Duration)
	IncBufferedEvents()
	DecBufferedEvents()
}

// EventBatch представляет batch событий для отправки
type EventBatch struct {
	Events    []*domain.Event
	Timestamp time.Time
	ResultCh  chan error
}

// Producer реализует интерфейс EventPublisher с асинхронным батчингом
type Producer struct {
	writer  *kafka.Writer
	topic   string
	logger  *logrus.Logger
	metrics ProducerMetrics
	config  config.KafkaConfig
	mu      sync.RWMutex
	closed  bool
	wg      sync.WaitGroup

	// Батчинг
	eventChan    chan *domain.Event
	batchChan    chan *EventBatch
	batchSize    int
	flushTimer   *time.Timer
	currentBatch []*domain.Event
	batchMu      sync.Mutex
}

// NewProducer создает новый Kafka producer с асинхронным батчингом
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

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100 // default batch size
	}

	producer := &Producer{
		writer:       writer,
		topic:        cfg.Topic,
		logger:       logger,
		metrics:      metrics,
		config:       cfg,
		eventChan:    make(chan *domain.Event, batchSize*2),
		batchChan:    make(chan *EventBatch, 10),
		batchSize:    batchSize,
		currentBatch: make([]*domain.Event, 0, batchSize),
	}

	logger.WithFields(logrus.Fields{
		"brokers":     cfg.Brokers,
		"topic":       cfg.Topic,
		"batch_size":  cfg.BatchSize,
		"compression": cfg.CompressionType,
		"async_batch": true,
	}).Info("Kafka producer initialized with async batching")

	return producer, nil
}

// Start запускает асинхронные worker'ы для батчинга
func (p *Producer) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.Unlock()

	p.logger.Info("Starting async batch producer")

	// Запускаем batch collector
	p.wg.Add(1)
	go p.batchCollector(ctx)

	// Запускаем batch sender
	p.wg.Add(1)
	go p.batchSender(ctx)

	return nil
}

// batchCollector собирает события в batch'и
func (p *Producer) batchCollector(ctx context.Context) {
	defer p.wg.Done()
	defer close(p.batchChan)

	flushTicker := time.NewTicker(p.config.BatchTimeout)
	defer flushTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Batch collector context cancelled, flushing final batch")
			p.flushCurrentBatch()
			return

		case event, ok := <-p.eventChan:
			if !ok {
				p.logger.Info("Event channel closed, flushing final batch")
				p.flushCurrentBatch()
				return
			}

			p.batchMu.Lock()
			p.currentBatch = append(p.currentBatch, event)
			shouldFlush := len(p.currentBatch) >= p.batchSize
			p.batchMu.Unlock()

			if shouldFlush {
				p.flushCurrentBatch()
			}

		case <-flushTicker.C:
			p.flushCurrentBatch()
		}
	}
}

// flushCurrentBatch отправляет текущий batch в канал для отправки
func (p *Producer) flushCurrentBatch() {
	p.batchMu.Lock()
	if len(p.currentBatch) == 0 {
		p.batchMu.Unlock()
		return
	}

	batch := &EventBatch{
		Events:    make([]*domain.Event, len(p.currentBatch)),
		Timestamp: time.Now(),
		ResultCh:  make(chan error, 1),
	}
	copy(batch.Events, p.currentBatch)
	p.currentBatch = p.currentBatch[:0] // Очищаем batch
	p.batchMu.Unlock()

	select {
	case p.batchChan <- batch:
		p.logger.WithField("batch_size", len(batch.Events)).Debug("Batch queued for sending")
	default:
		p.logger.Warn("Batch channel full, dropping batch")
		batch.ResultCh <- fmt.Errorf("batch channel full")
		close(batch.ResultCh)
	}
}

// batchSender отправляет batch'и в Kafka
func (p *Producer) batchSender(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Batch sender context cancelled")
			return

		case batch, ok := <-p.batchChan:
			if !ok {
				p.logger.Info("Batch channel closed")
				return
			}

			start := time.Now()
			err := p.sendBatch(ctx, batch.Events)
			duration := time.Since(start)

			p.metrics.ObserveBatchFlushDuration(duration)
			p.metrics.IncBatchSize(len(batch.Events))

			if err != nil {
				p.logger.WithFields(logrus.Fields{
					"batch_size": len(batch.Events),
					"error":      err,
					"duration":   duration,
				}).Error("Failed to send batch")
			} else {
				p.logger.WithFields(logrus.Fields{
					"batch_size": len(batch.Events),
					"duration":   duration,
				}).Debug("Batch sent successfully")
			}

			// Отправляем результат
			select {
			case batch.ResultCh <- err:
			default:
			}
			close(batch.ResultCh)
		}
	}
}

// sendBatch отправляет batch событий в Kafka
func (p *Producer) sendBatch(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	// Подготавливаем сообщения
	messages := make([]kafka.Message, 0, len(events))
	for _, event := range events {
		// Валидируем событие
		if err := event.Validate(); err != nil {
			p.metrics.IncFailedEvents(string(event.Type), "validation_error")
			p.logger.WithFields(logrus.Fields{
				"event_id":   event.ID,
				"event_type": event.Type,
				"error":      err,
			}).Error("Event validation failed")
			continue
		}

		// Сериализуем событие
		eventJSON, err := event.ToJSON()
		if err != nil {
			p.metrics.IncFailedEvents(string(event.Type), "serialization_error")
			p.logger.WithFields(logrus.Fields{
				"event_id":   event.ID,
				"event_type": event.Type,
				"error":      err,
			}).Error("Event serialization failed")
			continue
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

	if len(messages) == 0 {
		return fmt.Errorf("no valid messages to send")
	}

	// Публикуем batch с retry логикой
	err := p.publishBatchWithRetry(ctx, messages)
	if err != nil {
		for _, event := range events {
			p.metrics.IncFailedEvents(string(event.Type), "publish_error")
		}
		return err
	}

	// Обновляем метрики успеха
	for _, event := range events {
		p.metrics.IncPublishedEvents(string(event.Type))
	}

	return nil
}

// Publish публикует событие асинхронно через батчинг
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

	// Валидируем событие перед добавлением в batch
	if err := event.Validate(); err != nil {
		p.metrics.IncFailedEvents(string(event.Type), "validation_error")
		return fmt.Errorf("event validation failed: %w", err)
	}

	p.metrics.IncBufferedEvents()
	defer p.metrics.DecBufferedEvents()

	// Отправляем событие в канал для батчинга
	select {
	case p.eventChan <- event:
		p.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
		}).Debug("Event queued for batching")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Канал полный, отправляем синхронно
		p.logger.Warn("Event channel full, sending synchronously")
		return p.publishSync(ctx, event)
	}
}

// publishSync отправляет событие синхронно (fallback)
func (p *Producer) publishSync(ctx context.Context, event *domain.Event) error {
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
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.metrics.IncPublishedEvents(string(event.Type))
	return nil
}

// PublishBatch публикует несколько событий синхронно
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

	return p.sendBatch(ctx, events)
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
	p.logger.Info("Closing Kafka producer")

	// Закрываем канал событий
	close(p.eventChan)

	// Ждем завершения горутин
	p.wg.Wait()

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
