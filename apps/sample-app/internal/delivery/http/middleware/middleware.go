package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"sample-app/internal/domain"
)

// PrometheusMiddleware создает middleware для сбора метрик
func PrometheusMiddleware(metrics domain.MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем ResponseWriter для захвата статус кода
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start).Seconds()

			// Записываем метрики
			metrics.IncHTTPRequests(r.Method, r.URL.Path, fmt.Sprintf("%d", rw.statusCode))
			metrics.ObserveHTTPDuration(r.Method, r.URL.Path, duration)
		})
	}
}

// LoggingMiddleware создает middleware для логирования запросов
func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			log.Printf(
				"%s %s %d %v",
				r.Method,
				r.URL.Path,
				rw.statusCode,
				time.Since(start),
			)
		})
	}
}

// RecoveryMiddleware создает middleware для восстановления после паники
func RecoveryMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Panic recovered: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
