package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedWriter wraps kafka.Writer with OpenTelemetry instrumentation
type InstrumentedWriter struct {
	writer *kafka.Writer
	tracer trace.Tracer
}

// NewInstrumentedWriter creates a new instrumented Kafka writer
func NewInstrumentedWriter(config kafka.WriterConfig, serviceName string) *InstrumentedWriter {
	writer := &kafka.Writer{
		Topic:        config.Topic,
		Balancer:     config.Balancer,
		MaxAttempts:  config.MaxAttempts,
		BatchSize:    config.BatchSize,
		BatchTimeout: config.BatchTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		Async:        config.Async,
		Logger:       config.Logger,
		ErrorLogger:  config.ErrorLogger,
	}

	return &InstrumentedWriter{
		writer: writer,
		tracer: otel.Tracer(serviceName),
	}
}

// WriteMessages writes messages to Kafka with tracing
func (iw *InstrumentedWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	// Start a new span for the write operation
	ctx, span := iw.tracer.Start(ctx, "kafka.produce",
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", iw.writer.Topic),
			attribute.String("messaging.operation", "publish"),
			attribute.Int("messaging.batch_size", len(msgs)),
		),
	)
	defer span.End()

	// Add message-specific attributes and inject trace context manually
	for i, msg := range msgs {
		// Add trace context to message headers manually
		if msg.Headers == nil {
			msgs[i].Headers = make([]kafka.Header, 0)
		}

		// Inject trace context manually
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			msgs[i].Headers = append(msgs[i].Headers, kafka.Header{
				Key:   "trace-id",
				Value: []byte(spanCtx.TraceID().String()),
			})
			msgs[i].Headers = append(msgs[i].Headers, kafka.Header{
				Key:   "span-id",
				Value: []byte(spanCtx.SpanID().String()),
			})
		}

		span.SetAttributes(
			attribute.String(fmt.Sprintf("messaging.kafka.message.%d.key", i), string(msg.Key)),
			attribute.Int(fmt.Sprintf("messaging.kafka.message.%d.value_size", i), len(msg.Value)),
		)
	}

	// Write messages
	err := iw.writer.WriteMessages(ctx, msgs...)
	if err != nil {
		span.SetAttributes(attribute.String("error", "true"))
		span.RecordError(err)
	}

	return err
}

// Close closes the Kafka writer
func (iw *InstrumentedWriter) Close() error {
	return iw.writer.Close()
}

// Stats returns writer statistics
func (iw *InstrumentedWriter) Stats() kafka.WriterStats {
	return iw.writer.Stats()
}
