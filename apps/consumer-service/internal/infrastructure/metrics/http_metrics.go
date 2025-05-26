package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTPMetrics содержит метрики для HTTP запросов
type HTTPMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestSize     *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
	activeRequests  prometheus.Gauge
}

// NewHTTPMetrics создает новые HTTP метрики
func NewHTTPMetrics(namespace, subsystem string) *HTTPMetrics {
	return &HTTPMetrics{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),

		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint", "status_code"},
		),

		requestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),

		responseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint", "status_code"},
		),

		activeRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_active_requests",
				Help:      "Number of active HTTP requests",
			},
		),
	}
}

// IncRequests увеличивает счетчик HTTP запросов
func (m *HTTPMetrics) IncRequests(method, endpoint, statusCode string) {
	m.requestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
}

// ObserveRequestDuration записывает время выполнения HTTP запроса
func (m *HTTPMetrics) ObserveRequestDuration(method, endpoint, statusCode string, duration time.Duration) {
	m.requestDuration.WithLabelValues(method, endpoint, statusCode).Observe(duration.Seconds())
}

// ObserveRequestSize записывает размер HTTP запроса
func (m *HTTPMetrics) ObserveRequestSize(method, endpoint string, size int64) {
	m.requestSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// ObserveResponseSize записывает размер HTTP ответа
func (m *HTTPMetrics) ObserveResponseSize(method, endpoint, statusCode string, size int64) {
	m.responseSize.WithLabelValues(method, endpoint, statusCode).Observe(float64(size))
}

// IncActiveRequests увеличивает счетчик активных запросов
func (m *HTTPMetrics) IncActiveRequests() {
	m.activeRequests.Inc()
}

// DecActiveRequests уменьшает счетчик активных запросов
func (m *HTTPMetrics) DecActiveRequests() {
	m.activeRequests.Dec()
}
