package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusCollector реализует интерфейс MetricsCollector
type PrometheusCollector struct {
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	kafkaMessagesTotal  *prometheus.CounterVec
}

// NewPrometheusCollector создает новый Prometheus collector
func NewPrometheusCollector() *PrometheusCollector {
	httpRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	kafkaMessagesTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_total",
			Help: "Total number of messages sent to Kafka",
		},
		[]string{"topic", "status"},
	)

	// Регистрируем метрики
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(kafkaMessagesTotal)

	return &PrometheusCollector{
		httpRequestsTotal:   httpRequestsTotal,
		httpRequestDuration: httpRequestDuration,
		kafkaMessagesTotal:  kafkaMessagesTotal,
	}
}

// IncHTTPRequests увеличивает счетчик HTTP запросов
func (p *PrometheusCollector) IncHTTPRequests(method, endpoint, status string) {
	p.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// ObserveHTTPDuration записывает длительность HTTP запроса
func (p *PrometheusCollector) ObserveHTTPDuration(method, endpoint string, duration float64) {
	p.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// IncKafkaMessages увеличивает счетчик Kafka сообщений
func (p *PrometheusCollector) IncKafkaMessages(topic, status string) {
	p.kafkaMessagesTotal.WithLabelValues(topic, status).Inc()
}

// Handler возвращает HTTP handler для метрик
func (p *PrometheusCollector) Handler() http.Handler {
	return promhttp.Handler()
}
