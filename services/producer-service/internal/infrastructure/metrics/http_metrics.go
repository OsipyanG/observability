package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTPMetrics реализует интерфейс HTTPMetrics
type HTTPMetrics struct {
	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
}

// NewHTTPMetrics создает новые HTTP метрики
func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		httpRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		httpDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
	}
}

// IncHTTPRequests увеличивает счетчик HTTP запросов
func (m *HTTPMetrics) IncHTTPRequests(method, endpoint, status string) {
	m.httpRequests.WithLabelValues(method, endpoint, status).Inc()
}

// ObserveHTTPDuration записывает время выполнения HTTP запроса
func (m *HTTPMetrics) ObserveHTTPDuration(method, endpoint string, duration float64) {
	m.httpDuration.WithLabelValues(method, endpoint).Observe(duration)
}
