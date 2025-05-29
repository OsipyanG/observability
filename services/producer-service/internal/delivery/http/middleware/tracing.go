package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware returns OpenTelemetry tracing middleware for Gorilla Mux
func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	return otelmux.Middleware(serviceName)
}

// CustomTracingMiddleware adds custom tracing attributes
func CustomTracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)

		// Add custom attributes to the span
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.scheme", r.URL.Scheme),
			attribute.String("http.host", r.Host),
			attribute.String("http.user_agent", r.UserAgent()),
		)

		// Add request ID if present
		if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
			span.SetAttributes(attribute.String("http.request_id", requestID))
		}

		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware adds a request ID to the context and response headers
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			// Generate a new request ID using trace ID if available
			span := trace.SpanFromContext(r.Context())
			if span.SpanContext().IsValid() {
				requestID = span.SpanContext().TraceID().String()
			}
		}

		if requestID != "" {
			w.Header().Set("X-Request-ID", requestID)
			r.Header.Set("X-Request-ID", requestID)
		}

		next.ServeHTTP(w, r)
	})
}
