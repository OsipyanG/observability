package middleware

import (
	"fmt"
	"net/http"
	"producer-service/internal/domain"
	"runtime/debug"
	"time"

	"github.com/sirupsen/logrus"
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
func LoggingMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			logger.WithFields(logrus.Fields{
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     rw.statusCode,
				"duration":   duration,
				"user_agent": r.UserAgent(),
				"remote_ip":  getClientIP(r),
			}).Info("HTTP request processed")
		})
	}
}

// RecoveryMiddleware создает middleware для восстановления после паники
func RecoveryMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithFields(logrus.Fields{
						"error":      err,
						"method":     r.Method,
						"path":       r.URL.Path,
						"stack":      string(debug.Stack()),
						"user_agent": r.UserAgent(),
						"remote_ip":  getClientIP(r),
					}).Error("Panic recovered")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"Internal Server Error","message":"An unexpected error occurred"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware создает middleware для обработки CORS
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityMiddleware добавляет заголовки безопасности
func SecurityMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP получает IP адрес клиента
func getClientIP(r *http.Request) string {
	// Проверяем заголовки прокси
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
