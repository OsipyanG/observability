package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ProducerMetrics реализует интерфейс ProducerMetrics
type ProducerMetrics struct {
	publishedEvents    *prometheus.CounterVec
	failedEvents       *prometheus.CounterVec
	publishDuration    *prometheus.HistogramVec
	batchSize          prometheus.Histogram
	batchFlushDuration prometheus.Histogram
	bufferedEvents     prometheus.Gauge
	kafkaWriterStats   *prometheus.GaugeVec
}

// NewProducerMetrics создает новые метрики для producer
func NewProducerMetrics() *ProducerMetrics {
	return &ProducerMetrics{
		publishedEvents: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "producer_events_published_total",
				Help: "Total number of events published",
			},
			[]string{"event_type"},
		),
		failedEvents: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "producer_events_failed_total",
				Help: "Total number of failed events",
			},
			[]string{"event_type", "reason"},
		),
		publishDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "producer_publish_duration_seconds",
				Help:    "Duration of event publishing",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"event_type"},
		),
		batchSize: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "producer_batch_size",
				Help:    "Size of event batches",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
		),
		batchFlushDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "producer_batch_flush_duration_seconds",
				Help:    "Duration of batch flush operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
			},
		),
		bufferedEvents: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "producer_buffered_events",
				Help: "Number of events currently buffered for batching",
			},
		),
		kafkaWriterStats: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "producer_kafka_writer_stats",
				Help: "Kafka writer statistics",
			},
			[]string{"metric"},
		),
	}
}

// IncPublishedEvents увеличивает счетчик опубликованных событий
func (m *ProducerMetrics) IncPublishedEvents(eventType string) {
	m.publishedEvents.WithLabelValues(eventType).Inc()
}

// IncFailedEvents увеличивает счетчик неудачных событий
func (m *ProducerMetrics) IncFailedEvents(eventType string, reason string) {
	m.failedEvents.WithLabelValues(eventType, reason).Inc()
}

// ObservePublishDuration записывает время публикации события
func (m *ProducerMetrics) ObservePublishDuration(eventType string, duration time.Duration) {
	m.publishDuration.WithLabelValues(eventType).Observe(duration.Seconds())
}

// IncBatchSize записывает размер batch
func (m *ProducerMetrics) IncBatchSize(size int) {
	m.batchSize.Observe(float64(size))
}

// ObserveBatchFlushDuration записывает время flush batch'а
func (m *ProducerMetrics) ObserveBatchFlushDuration(duration time.Duration) {
	m.batchFlushDuration.Observe(duration.Seconds())
}

// IncBufferedEvents увеличивает счетчик буферизованных событий
func (m *ProducerMetrics) IncBufferedEvents() {
	m.bufferedEvents.Inc()
}

// DecBufferedEvents уменьшает счетчик буферизованных событий
func (m *ProducerMetrics) DecBufferedEvents() {
	m.bufferedEvents.Dec()
}

// UpdateKafkaWriterStats обновляет статистику Kafka writer
func (m *ProducerMetrics) UpdateKafkaWriterStats(writes, messages, bytes, errors int64) {
	m.kafkaWriterStats.WithLabelValues("writes").Set(float64(writes))
	m.kafkaWriterStats.WithLabelValues("messages").Set(float64(messages))
	m.kafkaWriterStats.WithLabelValues("bytes").Set(float64(bytes))
	m.kafkaWriterStats.WithLabelValues("errors").Set(float64(errors))
}
