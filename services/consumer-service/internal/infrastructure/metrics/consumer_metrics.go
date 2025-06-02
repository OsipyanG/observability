package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ConsumerMetrics содержит метрики для consumer
type ConsumerMetrics struct {
	consumedEvents     *prometheus.CounterVec
	failedEvents       *prometheus.CounterVec
	processingDuration *prometheus.HistogramVec
	lagGauge           *prometheus.GaugeVec
	commitDuration     prometheus.Histogram
}

// NewConsumerMetrics создает новые метрики для consumer
func NewConsumerMetrics() *ConsumerMetrics {
	return &ConsumerMetrics{
		consumedEvents: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "consumer_events_consumed_total",
				Help: "Total number of events consumed",
			},
			[]string{"event_type"},
		),
		failedEvents: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "consumer_events_failed_total",
				Help: "Total number of failed events",
			},
			[]string{"event_type", "reason"},
		),
		processingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "consumer_processing_duration_seconds",
				Help:    "Duration of event processing",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"event_type"},
		),
		lagGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "consumer_lag",
				Help: "Consumer lag in messages",
			},
			[]string{"topic", "partition"},
		),
		commitDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "consumer_commit_duration_seconds",
				Help:    "Duration of offset commits",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
		),
	}
}

// IncConsumedEvents увеличивает счетчик потребленных событий
func (m *ConsumerMetrics) IncConsumedEvents(eventType string) {
	m.consumedEvents.WithLabelValues(eventType).Inc()
}

// IncFailedEvents увеличивает счетчик неудачных событий
func (m *ConsumerMetrics) IncFailedEvents(eventType string, reason string) {
	m.failedEvents.WithLabelValues(eventType, reason).Inc()
}

// ObserveProcessingDuration записывает время обработки события
func (m *ConsumerMetrics) ObserveProcessingDuration(eventType string, duration time.Duration) {
	m.processingDuration.WithLabelValues(eventType).Observe(duration.Seconds())
}

// ObserveCommitDuration записывает время коммита offset
func (m *ConsumerMetrics) ObserveCommitDuration(duration time.Duration) {
	m.commitDuration.Observe(duration.Seconds())
}
