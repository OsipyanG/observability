package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ConsumerMetrics содержит все метрики для consumer сервиса
type ConsumerMetrics struct {
	// Счетчики событий
	eventsConsumed *prometheus.CounterVec
	eventsFailed   *prometheus.CounterVec

	// Метрики времени обработки
	processingDuration *prometheus.HistogramVec

	// Метрики батчей
	batchSize        *prometheus.HistogramVec
	batchProcessTime *prometheus.HistogramVec

	// Метрики воркеров
	activeWorkers prometheus.Gauge

	// Метрики Kafka
	kafkaLag         *prometheus.GaugeVec
	kafkaOffset      *prometheus.GaugeVec
	kafkaConnections prometheus.Gauge

	// Метрики ошибок
	retryAttempts *prometheus.CounterVec
	deadLetters   *prometheus.CounterVec

	// Метрики производительности
	throughput *prometheus.GaugeVec
}

// NewConsumerMetrics создает новый экземпляр метрик
func NewConsumerMetrics(namespace, subsystem string) *ConsumerMetrics {
	return &ConsumerMetrics{
		eventsConsumed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "events_consumed_total",
				Help:      "Total number of events consumed",
			},
			[]string{"event_type", "topic", "partition"},
		),

		eventsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "events_failed_total",
				Help:      "Total number of failed events",
			},
			[]string{"event_type", "reason", "topic", "partition"},
		),

		processingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "processing_duration_seconds",
				Help:      "Time spent processing events",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
			},
			[]string{"event_type", "status"},
		),

		batchSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "batch_size",
				Help:      "Size of processed batches",
				Buckets:   prometheus.LinearBuckets(1, 10, 20), // 1 to 200
			},
			[]string{"topic"},
		),

		batchProcessTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "batch_process_duration_seconds",
				Help:      "Time spent processing batches",
				Buckets:   prometheus.ExponentialBuckets(0.01, 2, 12), // 10ms to ~40s
			},
			[]string{"topic"},
		),

		activeWorkers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_workers",
				Help:      "Number of active worker goroutines",
			},
		),

		kafkaLag: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "kafka_lag",
				Help:      "Kafka consumer lag",
			},
			[]string{"topic", "partition", "group_id"},
		),

		kafkaOffset: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "kafka_offset",
				Help:      "Current Kafka offset",
			},
			[]string{"topic", "partition", "group_id"},
		),

		kafkaConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "kafka_connections",
				Help:      "Number of active Kafka connections",
			},
		),

		retryAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "retry_attempts_total",
				Help:      "Total number of retry attempts",
			},
			[]string{"event_type", "attempt"},
		),

		deadLetters: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "dead_letters_total",
				Help:      "Total number of events sent to dead letter queue",
			},
			[]string{"event_type", "reason"},
		),

		throughput: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "throughput_events_per_second",
				Help:      "Current throughput in events per second",
			},
			[]string{"event_type"},
		),
	}
}

// IncEventsConsumed увеличивает счетчик потребленных событий
func (m *ConsumerMetrics) IncEventsConsumed(eventType, topic, partition string) {
	m.eventsConsumed.WithLabelValues(eventType, topic, partition).Inc()
}

// IncEventsFailed увеличивает счетчик неудачных событий
func (m *ConsumerMetrics) IncEventsFailed(eventType, reason, topic, partition string) {
	m.eventsFailed.WithLabelValues(eventType, reason, topic, partition).Inc()
}

// ObserveProcessingDuration записывает время обработки события
func (m *ConsumerMetrics) ObserveProcessingDuration(eventType, status string, duration time.Duration) {
	m.processingDuration.WithLabelValues(eventType, status).Observe(duration.Seconds())
}

// ObserveBatchSize записывает размер батча
func (m *ConsumerMetrics) ObserveBatchSize(topic string, size int) {
	m.batchSize.WithLabelValues(topic).Observe(float64(size))
}

// ObserveBatchProcessTime записывает время обработки батча
func (m *ConsumerMetrics) ObserveBatchProcessTime(topic string, duration time.Duration) {
	m.batchProcessTime.WithLabelValues(topic).Observe(duration.Seconds())
}

// SetActiveWorkers устанавливает количество активных воркеров
func (m *ConsumerMetrics) SetActiveWorkers(count int) {
	m.activeWorkers.Set(float64(count))
}

// SetKafkaLag устанавливает lag для Kafka
func (m *ConsumerMetrics) SetKafkaLag(topic, partition, groupID string, lag int64) {
	m.kafkaLag.WithLabelValues(topic, partition, groupID).Set(float64(lag))
}

// SetKafkaOffset устанавливает текущий offset для Kafka
func (m *ConsumerMetrics) SetKafkaOffset(topic, partition, groupID string, offset int64) {
	m.kafkaOffset.WithLabelValues(topic, partition, groupID).Set(float64(offset))
}

// SetKafkaConnections устанавливает количество активных соединений с Kafka
func (m *ConsumerMetrics) SetKafkaConnections(count int) {
	m.kafkaConnections.Set(float64(count))
}

// IncRetryAttempts увеличивает счетчик попыток повтора
func (m *ConsumerMetrics) IncRetryAttempts(eventType, attempt string) {
	m.retryAttempts.WithLabelValues(eventType, attempt).Inc()
}

// IncDeadLetters увеличивает счетчик событий в dead letter queue
func (m *ConsumerMetrics) IncDeadLetters(eventType, reason string) {
	m.deadLetters.WithLabelValues(eventType, reason).Inc()
}

// SetThroughput устанавливает текущую пропускную способность
func (m *ConsumerMetrics) SetThroughput(eventType string, eventsPerSecond float64) {
	m.throughput.WithLabelValues(eventType).Set(eventsPerSecond)
}
