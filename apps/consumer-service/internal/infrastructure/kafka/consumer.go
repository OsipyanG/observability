package kafka

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"consumer-service/internal/config"
	"consumer-service/internal/domain"
	"consumer-service/internal/infrastructure/metrics"

	"github.com/segmentio/kafka-go"
)

// Consumer реализует интерфейс EventConsumer
type Consumer struct {
	reader  *kafka.Reader
	handler domain.EventHandler
	logger  domain.Logger
	metrics *metrics.ConsumerMetrics
	config  config.KafkaConfig

	// Статистика
	stats      domain.ConsumerStats
	statsMutex sync.RWMutex

	// Управление жизненным циклом
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed int32

	// Каналы для координации
	eventChan  chan *kafka.Message
	resultChan chan *domain.ProcessingResult

	// Настройки retry
	retryBackoff time.Duration
	maxRetries   int
}

// NewConsumer создает новый Kafka consumer
func NewConsumer(
	cfg config.KafkaConfig,
	handler domain.EventHandler,
	logger domain.Logger,
	metrics *metrics.ConsumerMetrics,
) *Consumer {
	// Создаем reader с расширенными настройками
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		MaxWait:        cfg.MaxWait,
		StartOffset:    cfg.StartOffset,
		CommitInterval: cfg.CommitInterval,

		// Настройки производительности
		ReadBatchTimeout: cfg.MaxWaitTime,

		// Настройки retry
		ReadBackoffMin: cfg.RetryBackoff,
		ReadBackoffMax: cfg.RetryBackoff * 10,

		// Логирование ошибок
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Error(fmt.Sprintf("kafka reader error: "+msg, args...))
		}),
	})

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		reader:  reader,
		handler: handler,
		logger:  logger,
		metrics: metrics,
		config:  cfg,
		ctx:     ctx,
		cancel:  cancel,

		eventChan:  make(chan *kafka.Message, 1000),
		resultChan: make(chan *domain.ProcessingResult, 1000),

		retryBackoff: cfg.RetryBackoff,
		maxRetries:   cfg.MaxRetries,

		stats: domain.ConsumerStats{
			LastMessageTime: time.Now(),
		},
	}
}

// Consume начинает потребление сообщений
func (c *Consumer) Consume(ctx context.Context) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return fmt.Errorf("consumer is closed")
	}

	c.logger.Info("Starting Kafka consumer")

	// Запускаем воркеры для обработки
	workerCount := 5 // можно сделать конфигурируемым
	for i := 0; i < workerCount; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	// Запускаем сборщик результатов
	c.wg.Add(1)
	go c.resultCollector()

	// Запускаем обновление метрик
	c.wg.Add(1)
	go c.metricsUpdater()

	// Основной цикл чтения сообщений
	defer func() {
		close(c.eventChan)
		c.wg.Wait()
		close(c.resultChan)
	}()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Consumer context cancelled, stopping")
			return ctx.Err()
		case <-c.ctx.Done():
			c.logger.Info("Consumer stopped")
			return nil
		default:
			if err := c.readMessage(ctx); err != nil {
				c.logger.Error("Failed to read message", "error", err)
				c.updateErrorStats("read_error")

				// Небольшая пауза при ошибке чтения
				select {
				case <-time.After(c.retryBackoff):
				case <-ctx.Done():
					return ctx.Err()
				}
				continue
			}
		}
	}
}

// ConsumeBatch потребляет события батчами
func (c *Consumer) ConsumeBatch(ctx context.Context, batchSize int) ([]*domain.Event, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, fmt.Errorf("consumer is closed")
	}

	var events []*domain.Event
	timeout := time.After(c.config.MaxWait)

	for len(events) < batchSize {
		select {
		case <-ctx.Done():
			return events, ctx.Err()
		case <-timeout:
			// Возвращаем то, что успели собрать
			return events, nil
		default:
			message, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if len(events) > 0 {
					return events, nil
				}
				return nil, fmt.Errorf("failed to read message: %w", err)
			}

			event, err := c.parseMessage(&message)
			if err != nil {
				c.logger.Error("Failed to parse message", "error", err)
				c.updateErrorStats("parse_error")
				continue
			}

			events = append(events, event)
			c.updateConsumedStats(event, &message)
		}
	}

	return events, nil
}

// readMessage читает одно сообщение и отправляет в канал для обработки
func (c *Consumer) readMessage(ctx context.Context) error {
	message, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}

	// Обновляем метрики соединений
	c.metrics.SetKafkaConnections(1) // упрощенно, в реальности нужно отслеживать

	select {
	case c.eventChan <- &message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending message to processing channel")
	}
}

// worker обрабатывает сообщения из канала
func (c *Consumer) worker(workerID int) {
	defer c.wg.Done()

	c.logger.Debug("Starting worker", "worker_id", workerID)
	c.metrics.SetActiveWorkers(workerID + 1)

	defer func() {
		c.metrics.SetActiveWorkers(workerID)
		c.logger.Debug("Worker stopped", "worker_id", workerID)
	}()

	for message := range c.eventChan {
		result := c.processMessage(message)

		select {
		case c.resultChan <- result:
		case <-c.ctx.Done():
			return
		}
	}
}

// processMessage обрабатывает одно сообщение
func (c *Consumer) processMessage(message *kafka.Message) *domain.ProcessingResult {
	start := time.Now()

	result := &domain.ProcessingResult{
		ProcessedAt: start,
	}

	// Парсим событие
	event, err := c.parseMessage(message)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("parse error: %v", err)
		result.Duration = time.Since(start)
		c.updateErrorStats("parse_error")
		return result
	}

	result.EventID = event.ID
	result.EventType = event.Type

	// Обрабатываем с retry логикой
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		processResult, err := c.handler.Handle(c.ctx, event)
		if err == nil && processResult != nil && processResult.Success {
			result.Success = true
			result.Duration = time.Since(start)
			c.updateSuccessStats(event, message)
			c.metrics.ObserveProcessingDuration(string(event.Type), "success", result.Duration)
			return result
		}

		// Логируем попытку retry
		if attempt < c.maxRetries {
			c.logger.Warn("Retrying event processing",
				"event_id", event.ID,
				"attempt", attempt+1,
				"error", err)
			c.metrics.IncRetryAttempts(string(event.Type), strconv.Itoa(attempt+1))

			// Exponential backoff
			backoff := c.retryBackoff * time.Duration(1<<attempt)
			time.Sleep(backoff)
		} else {
			// Исчерпали все попытки
			result.Success = false
			result.Error = fmt.Sprintf("max retries exceeded: %v", err)
			result.Duration = time.Since(start)
			c.updateErrorStats("max_retries_exceeded")
			c.metrics.IncDeadLetters(string(event.Type), "max_retries")
			c.metrics.ObserveProcessingDuration(string(event.Type), "failed", result.Duration)
		}
	}

	return result
}

// parseMessage парсит Kafka сообщение в доменное событие
func (c *Consumer) parseMessage(message *kafka.Message) (*domain.Event, error) {
	event, err := domain.FromJSON(message.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event from JSON: %w", err)
	}

	return event, nil
}

// resultCollector собирает результаты обработки
func (c *Consumer) resultCollector() {
	defer c.wg.Done()

	for result := range c.resultChan {
		if result.Success {
			c.logger.Debug("Event processed successfully",
				"event_id", result.EventID,
				"event_type", result.EventType,
				"duration", result.Duration)
		} else {
			c.logger.Error("Event processing failed",
				"event_id", result.EventID,
				"event_type", result.EventType,
				"error", result.Error,
				"duration", result.Duration)
		}
	}
}

// metricsUpdater периодически обновляет метрики
func (c *Consumer) metricsUpdater() {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.updateKafkaMetrics()
		case <-c.ctx.Done():
			return
		}
	}
}

// updateKafkaMetrics обновляет метрики Kafka
func (c *Consumer) updateKafkaMetrics() {
	stats := c.reader.Stats()

	// Обновляем основные метрики
	c.metrics.SetKafkaOffset(stats.Topic, stats.Partition, c.config.GroupID, stats.Offset)
	c.metrics.SetKafkaLag(stats.Topic, stats.Partition, c.config.GroupID, stats.Lag)
}

// updateConsumedStats обновляет статистику потребленных сообщений
func (c *Consumer) updateConsumedStats(event *domain.Event, message *kafka.Message) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	c.stats.MessagesConsumed++
	c.stats.BytesConsumed += int64(len(message.Value))
	c.stats.LastMessageTime = time.Now()

	partitionStr := strconv.Itoa(message.Partition)
	c.metrics.IncEventsConsumed(string(event.Type), message.Topic, partitionStr)
}

// updateSuccessStats обновляет статистику успешной обработки
func (c *Consumer) updateSuccessStats(event *domain.Event, message *kafka.Message) {
	c.updateConsumedStats(event, message)
}

// updateErrorStats обновляет статистику ошибок
func (c *Consumer) updateErrorStats(reason string) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	c.stats.Errors++
}

// Stats возвращает статистику consumer
func (c *Consumer) Stats() domain.ConsumerStats {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()

	// Создаем копию для безопасности
	return domain.ConsumerStats{
		MessagesConsumed: c.stats.MessagesConsumed,
		BytesConsumed:    c.stats.BytesConsumed,
		Errors:           c.stats.Errors,
		LastMessageTime:  c.stats.LastMessageTime,
		Lag:              c.stats.Lag, // можно получить из Kafka метрик
	}
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil // уже закрыт
	}

	c.logger.Info("Closing Kafka consumer")

	// Отменяем контекст
	c.cancel()

	// Ждем завершения всех горутин
	c.wg.Wait()

	// Закрываем reader
	if err := c.reader.Close(); err != nil {
		c.logger.Error("Failed to close Kafka reader", "error", err)
		return fmt.Errorf("failed to close kafka reader: %w", err)
	}

	c.logger.Info("Kafka consumer closed successfully")
	return nil
}
