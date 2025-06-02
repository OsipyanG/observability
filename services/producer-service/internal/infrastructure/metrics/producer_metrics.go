package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ProducerMetrics реализует интерфейс ProducerMetrics
type ProducerMetrics struct {
	publishedEvents *prometheus.CounterVec
	failedEvents    *prometheus.CounterVec
	publishDuration *prometheus.HistogramVec
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
