package observability

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// Logger wraps logrus with OpenTelemetry integration
type Logger struct {
	*logrus.Logger
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level              string
	Format             string
	Output             string
	CorrelationEnabled bool
}

// NewLogger creates a new structured logger with OpenTelemetry integration
func NewLogger(config LoggerConfig) (*Logger, error) {
	log := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Set output format
	if config.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	// Set output destination
	switch config.Output {
	case "stdout":
		log.SetOutput(os.Stdout)
	case "stderr":
		log.SetOutput(os.Stderr)
	default:
		log.SetOutput(os.Stdout)
	}

	return &Logger{Logger: log}, nil
}

// WithTraceContext adds trace information to log fields
func (l *Logger) WithTraceContext(ctx context.Context) *logrus.Entry {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return l.WithFields(logrus.Fields{
			"trace_id": span.SpanContext().TraceID().String(),
			"span_id":  span.SpanContext().SpanID().String(),
		})
	}
	return l.WithFields(logrus.Fields{})
}

// InfoWithTrace logs info with trace context
func (l *Logger) InfoWithTrace(ctx context.Context, msg string) {
	l.WithTraceContext(ctx).Info(msg)
}

// ErrorWithTrace logs error with trace context
func (l *Logger) ErrorWithTrace(ctx context.Context, err error, msg string) {
	l.WithTraceContext(ctx).WithError(err).Error(msg)
}

// WarnWithTrace logs warning with trace context
func (l *Logger) WarnWithTrace(ctx context.Context, msg string) {
	l.WithTraceContext(ctx).Warn(msg)
}

// DebugWithTrace logs debug with trace context
func (l *Logger) DebugWithTrace(ctx context.Context, msg string) {
	l.WithTraceContext(ctx).Debug(msg)
}

// WithFieldsAndTrace logs with custom fields and trace context
func (l *Logger) WithFieldsAndTrace(ctx context.Context, fields logrus.Fields) *logrus.Entry {
	entry := l.WithTraceContext(ctx)
	return entry.WithFields(fields)
}
