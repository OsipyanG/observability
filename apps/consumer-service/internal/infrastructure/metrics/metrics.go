package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics содержит все метрики для consumer сервиса
type Metrics struct {
	MessagesProcessed  *prometheus.CounterVec
	ProcessingDuration *prometheus.HistogramVec
	ProcessingErrors   *prometheus.CounterVec
	ActiveWorkers      prometheus.Gauge
}

// NewMetrics создает новый экземпляр метрик
func NewMetrics() *Metrics {
	messagesProcessed := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "consumer_messages_processed_total",
			Help: "Total number of processed messages",
		},
		[]string{"event_type", "status"},
	)

	processingDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "consumer_message_processing_duration_seconds",
			Help:    "Duration of message processing",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"event_type"},
	)

	processingErrors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "consumer_processing_errors_total",
			Help: "Total number of processing errors",
		},
		[]string{"event_type", "error_type"},
	)

	activeWorkers := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "consumer_active_workers",
			Help: "Number of active workers",
		},
	)

	// Регистрируем метрики
	prometheus.MustRegister(messagesProcessed)
	prometheus.MustRegister(processingDuration)
	prometheus.MustRegister(processingErrors)
	prometheus.MustRegister(activeWorkers)

	return &Metrics{
		MessagesProcessed:  messagesProcessed,
		ProcessingDuration: processingDuration,
		ProcessingErrors:   processingErrors,
		ActiveWorkers:      activeWorkers,
	}
}

// Handler возвращает HTTP handler для метрик
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}
