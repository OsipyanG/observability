package usecase

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"consumer-service/internal/domain"
	"consumer-service/internal/infrastructure/metrics"
)

// EventProcessor реализует интерфейс EventProcessor
type EventProcessor struct {
	logger  domain.Logger
	metrics *metrics.ConsumerMetrics

	// Статистика
	stats      domain.ProcessorStats
	statsMutex sync.RWMutex

	// Управление жизненным циклом
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running int32

	// Настройки
	maxConcurrency int
	batchSize      int
	flushInterval  time.Duration

	// Каналы для батчинга
	eventChan  chan *domain.Event
	resultChan chan *domain.ProcessingResult

	// Обработчики по типам событий
	handlers map[domain.EventType]EventTypeHandler
}

// EventTypeHandler интерфейс для обработчиков конкретных типов событий
type EventTypeHandler interface {
	Handle(ctx context.Context, event *domain.Event) (*domain.ProcessingResult, error)
	GetEventType() domain.EventType
}

// NewEventProcessor создает новый процессор событий
func NewEventProcessor(
	logger domain.Logger,
	metrics *metrics.ConsumerMetrics,
	maxConcurrency int,
	batchSize int,
	flushInterval time.Duration,
) *EventProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	processor := &EventProcessor{
		logger:         logger,
		metrics:        metrics,
		ctx:            ctx,
		cancel:         cancel,
		maxConcurrency: maxConcurrency,
		batchSize:      batchSize,
		flushInterval:  flushInterval,
		eventChan:      make(chan *domain.Event, batchSize*2),
		resultChan:     make(chan *domain.ProcessingResult, batchSize*2),
		handlers:       make(map[domain.EventType]EventTypeHandler),
		stats: domain.ProcessorStats{
			EventsByType:      make(map[string]int64),
			LastProcessedTime: time.Now(),
		},
	}

	// Регистрируем обработчики
	processor.registerHandlers()

	return processor
}

// registerHandlers регистрирует обработчики для разных типов событий
func (p *EventProcessor) registerHandlers() {
	p.handlers[domain.UserCreatedEvent] = &UserCreatedHandler{logger: p.logger}
	p.handlers[domain.OrderPlacedEvent] = &OrderPlacedHandler{logger: p.logger}
	p.handlers[domain.PaymentProcessedEvent] = &PaymentProcessedHandler{logger: p.logger}
}

// Start запускает процессор
func (p *EventProcessor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return fmt.Errorf("processor is already running")
	}

	p.logger.Info("Starting event processor")

	// Запускаем воркеры
	for i := 0; i < p.maxConcurrency; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	// Запускаем сборщик результатов
	p.wg.Add(1)
	go p.resultCollector()

	// Запускаем обновление статистики
	p.wg.Add(1)
	go p.statsUpdater()

	return nil
}

// Stop останавливает процессор
func (p *EventProcessor) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return nil // уже остановлен
	}

	p.logger.Info("Stopping event processor")

	// Отменяем контекст
	p.cancel()

	// Закрываем канал событий
	close(p.eventChan)

	// Ждем завершения всех воркеров
	p.wg.Wait()

	// Закрываем канал результатов
	close(p.resultChan)

	p.logger.Info("Event processor stopped")
	return nil
}

// Process обрабатывает одно событие
func (p *EventProcessor) Process(ctx context.Context, event *domain.Event) (*domain.ProcessingResult, error) {
	if atomic.LoadInt32(&p.running) == 0 {
		return nil, fmt.Errorf("processor is not running")
	}

	start := time.Now()

	// Проверяем, есть ли обработчик для этого типа события
	handler, exists := p.handlers[event.Type]
	if !exists {
		result := &domain.ProcessingResult{
			EventID:     event.ID,
			EventType:   event.Type,
			Success:     false,
			Error:       fmt.Sprintf("no handler for event type: %s", event.Type),
			ProcessedAt: start,
			Duration:    time.Since(start),
		}
		p.updateFailedStats(event, "no_handler")
		return result, fmt.Errorf("no handler for event type: %s", event.Type)
	}

	// Обрабатываем событие
	result, err := handler.Handle(ctx, event)
	if err != nil {
		result = &domain.ProcessingResult{
			EventID:     event.ID,
			EventType:   event.Type,
			Success:     false,
			Error:       err.Error(),
			ProcessedAt: start,
			Duration:    time.Since(start),
		}
		p.updateFailedStats(event, "handler_error")
		p.metrics.ObserveProcessingDuration(string(event.Type), "failed", result.Duration)
		return result, err
	}

	// Обновляем статистику успеха
	p.updateSuccessStats(event)
	p.metrics.ObserveProcessingDuration(string(event.Type), "success", result.Duration)

	return result, nil
}

// ProcessBatch обрабатывает батч событий
func (p *EventProcessor) ProcessBatch(ctx context.Context, events []*domain.Event) ([]*domain.ProcessingResult, error) {
	if len(events) == 0 {
		return []*domain.ProcessingResult{}, nil
	}

	start := time.Now()
	results := make([]*domain.ProcessingResult, len(events))

	// Группируем события по типам для оптимизации
	eventsByType := make(map[domain.EventType][]*domain.Event)
	for _, event := range events {
		eventsByType[event.Type] = append(eventsByType[event.Type], event)
	}

	// Обрабатываем каждую группу
	resultIndex := 0
	for eventType, typeEvents := range eventsByType {
		handler, exists := p.handlers[eventType]
		if !exists {
			// Создаем результаты ошибок для всех событий этого типа
			for _, event := range typeEvents {
				results[resultIndex] = &domain.ProcessingResult{
					EventID:     event.ID,
					EventType:   event.Type,
					Success:     false,
					Error:       fmt.Sprintf("no handler for event type: %s", eventType),
					ProcessedAt: time.Now(),
					Duration:    time.Since(start),
				}
				p.updateFailedStats(event, "no_handler")
				resultIndex++
			}
			continue
		}

		// Обрабатываем события этого типа
		for _, event := range typeEvents {
			result, err := handler.Handle(ctx, event)
			if err != nil {
				result = &domain.ProcessingResult{
					EventID:     event.ID,
					EventType:   event.Type,
					Success:     false,
					Error:       err.Error(),
					ProcessedAt: time.Now(),
					Duration:    time.Since(start),
				}
				p.updateFailedStats(event, "handler_error")
			} else {
				p.updateSuccessStats(event)
			}

			results[resultIndex] = result
			resultIndex++
		}
	}

	// Обновляем метрики батча
	p.metrics.ObserveBatchSize("events", len(events))
	p.metrics.ObserveBatchProcessTime("events", time.Since(start))

	return results, nil
}

// worker обрабатывает события из канала
func (p *EventProcessor) worker(workerID int) {
	defer p.wg.Done()

	p.logger.Debug("Starting processor worker", "worker_id", workerID)

	for event := range p.eventChan {
		result, _ := p.Process(p.ctx, event)

		select {
		case p.resultChan <- result:
		case <-p.ctx.Done():
			return
		}
	}

	p.logger.Debug("Processor worker stopped", "worker_id", workerID)
}

// resultCollector собирает результаты обработки
func (p *EventProcessor) resultCollector() {
	defer p.wg.Done()

	for result := range p.resultChan {
		if result.Success {
			p.logger.Debug("Event processed successfully",
				"event_id", result.EventID,
				"event_type", result.EventType,
				"duration", result.Duration)
		} else {
			p.logger.Error("Event processing failed",
				"event_id", result.EventID,
				"event_type", result.EventType,
				"error", result.Error,
				"duration", result.Duration)
		}
	}
}

// statsUpdater периодически обновляет статистику
func (p *EventProcessor) statsUpdater() {
	defer p.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.updateProcessingRate()
		case <-p.ctx.Done():
			return
		}
	}
}

// updateProcessingRate обновляет скорость обработки
func (p *EventProcessor) updateProcessingRate() {
	p.statsMutex.Lock()
	defer p.statsMutex.Unlock()

	now := time.Now()
	duration := now.Sub(p.stats.LastProcessedTime).Seconds()
	if duration > 0 {
		p.stats.ProcessingRate = float64(p.stats.EventsProcessed) / duration
	}

	// Обновляем метрики throughput
	for eventType, count := range p.stats.EventsByType {
		rate := float64(count) / duration
		p.metrics.SetThroughput(eventType, rate)
	}
}

// updateSuccessStats обновляет статистику успешной обработки
func (p *EventProcessor) updateSuccessStats(event *domain.Event) {
	p.statsMutex.Lock()
	defer p.statsMutex.Unlock()

	p.stats.EventsProcessed++
	p.stats.EventsByType[string(event.Type)]++
	p.stats.LastProcessedTime = time.Now()
}

// updateFailedStats обновляет статистику неудачной обработки
func (p *EventProcessor) updateFailedStats(event *domain.Event, reason string) {
	p.statsMutex.Lock()
	defer p.statsMutex.Unlock()

	p.stats.EventsFailed++
	p.metrics.IncEventsFailed(string(event.Type), reason, "", "")
}

// GetStats возвращает статистику процессора
func (p *EventProcessor) GetStats() domain.ProcessorStats {
	p.statsMutex.RLock()
	defer p.statsMutex.RUnlock()

	// Создаем копию для безопасности
	eventsByType := make(map[string]int64)
	for k, v := range p.stats.EventsByType {
		eventsByType[k] = v
	}

	return domain.ProcessorStats{
		EventsProcessed:   p.stats.EventsProcessed,
		EventsFailed:      p.stats.EventsFailed,
		ProcessingRate:    p.stats.ProcessingRate,
		AverageLatency:    p.stats.AverageLatency,
		EventsByType:      eventsByType,
		ActiveWorkers:     p.stats.ActiveWorkers,
		LastProcessedTime: p.stats.LastProcessedTime,
	}
}

// Handle обрабатывает событие (для совместимости с интерфейсом EventHandler)
func (p *EventProcessor) Handle(ctx context.Context, event *domain.Event) (*domain.ProcessingResult, error) {
	return p.Process(ctx, event)
}

// HandleBatch обрабатывает батч событий (для совместимости с интерфейсом EventHandler)
func (p *EventProcessor) HandleBatch(ctx context.Context, events []*domain.Event) ([]*domain.ProcessingResult, error) {
	return p.ProcessBatch(ctx, events)
}

// CanHandle проверяет, может ли процессор обработать событие
func (p *EventProcessor) CanHandle(eventType domain.EventType) bool {
	_, exists := p.handlers[eventType]
	return exists
}

// GetSupportedTypes возвращает поддерживаемые типы событий
func (p *EventProcessor) GetSupportedTypes() []domain.EventType {
	types := make([]domain.EventType, 0, len(p.handlers))
	for eventType := range p.handlers {
		types = append(types, eventType)
	}
	return types
}

// Конкретные обработчики событий

// UserCreatedHandler обработчик события создания пользователя
type UserCreatedHandler struct {
	logger domain.Logger
}

func (h *UserCreatedHandler) Handle(ctx context.Context, event *domain.Event) (*domain.ProcessingResult, error) {
	start := time.Now()

	h.logger.Info("Processing user created event",
		"event_id", event.ID,
		"event_type", event.Type,
		"timestamp", event.Timestamp)

	// Симуляция бизнес-логики:
	// - Отправка welcome email
	// - Создание профиля пользователя
	// - Инициализация настроек по умолчанию
	// - Отправка уведомлений в другие сервисы

	// Симуляция времени обработки
	time.Sleep(10 * time.Millisecond)

	return &domain.ProcessingResult{
		EventID:     event.ID,
		EventType:   event.Type,
		Success:     true,
		ProcessedAt: start,
		Duration:    time.Since(start),
	}, nil
}

func (h *UserCreatedHandler) GetEventType() domain.EventType {
	return domain.UserCreatedEvent
}

// OrderPlacedHandler обработчик события размещения заказа
type OrderPlacedHandler struct {
	logger domain.Logger
}

func (h *OrderPlacedHandler) Handle(ctx context.Context, event *domain.Event) (*domain.ProcessingResult, error) {
	start := time.Now()

	h.logger.Info("Processing order placed event",
		"event_id", event.ID,
		"event_type", event.Type,
		"timestamp", event.Timestamp)

	// Симуляция бизнес-логики:
	// - Резервирование товаров на складе
	// - Расчет стоимости доставки
	// - Отправка уведомления продавцу
	// - Создание задач для сборки заказа
	// - Обновление аналитики продаж

	// Симуляция времени обработки
	time.Sleep(15 * time.Millisecond)

	return &domain.ProcessingResult{
		EventID:     event.ID,
		EventType:   event.Type,
		Success:     true,
		ProcessedAt: start,
		Duration:    time.Since(start),
	}, nil
}

func (h *OrderPlacedHandler) GetEventType() domain.EventType {
	return domain.OrderPlacedEvent
}

// PaymentProcessedHandler обработчик события обработки платежа
type PaymentProcessedHandler struct {
	logger domain.Logger
}

func (h *PaymentProcessedHandler) Handle(ctx context.Context, event *domain.Event) (*domain.ProcessingResult, error) {
	start := time.Now()

	h.logger.Info("Processing payment processed event",
		"event_id", event.ID,
		"event_type", event.Type,
		"timestamp", event.Timestamp)

	// Симуляция бизнес-логики:
	// - Подтверждение заказа
	// - Отправка чека клиенту
	// - Обновление статуса заказа
	// - Запуск процесса доставки
	// - Обновление финансовой отчетности

	// Симуляция времени обработки
	time.Sleep(20 * time.Millisecond)

	return &domain.ProcessingResult{
		EventID:     event.ID,
		EventType:   event.Type,
		Success:     true,
		ProcessedAt: start,
		Duration:    time.Since(start),
	}, nil
}

func (h *PaymentProcessedHandler) GetEventType() domain.EventType {
	return domain.PaymentProcessedEvent
}
