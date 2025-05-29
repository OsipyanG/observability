package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedReader wraps kafka.Reader with OpenTelemetry instrumentation
type InstrumentedReader struct {
	reader *kafka.Reader
	tracer trace.Tracer
}

// NewInstrumentedReader creates a new instrumented Kafka reader
func NewInstrumentedReader(config kafka.ReaderConfig, serviceName string) *InstrumentedReader {
	reader := kafka.NewReader(config)

	return &InstrumentedReader{
		reader: reader,
		tracer: otel.Tracer(serviceName),
	}
}

// ReadMessage reads a single message with tracing
func (ir *InstrumentedReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	// Start a new span for the read operation
	ctx, span := ir.tracer.Start(ctx, "kafka.consume",
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.source", ir.reader.Config().Topic),
			attribute.String("messaging.operation", "receive"),
		),
	)
	defer span.End()

	// Read message
	msg, err := ir.reader.ReadMessage(ctx)
	if err != nil {
		span.SetAttributes(attribute.String("error", "true"))
		span.RecordError(err)
		return msg, err
	}

	// Extract trace context from message headers and add as child span
	parentCtx := extractTraceContext(msg.Headers)
	if parentCtx != nil {
		// Create child span with parent context
		_, childSpan := ir.tracer.Start(*parentCtx, "kafka.message.process",
			trace.WithAttributes(
				attribute.String("messaging.kafka.partition", fmt.Sprintf("%d", msg.Partition)),
				attribute.String("messaging.kafka.offset", fmt.Sprintf("%d", msg.Offset)),
				attribute.String("messaging.kafka.key", string(msg.Key)),
				attribute.Int("messaging.kafka.message_size", len(msg.Value)),
			),
		)
		defer childSpan.End()
	}

	// Add message attributes to main span
	span.SetAttributes(
		attribute.String("messaging.kafka.partition", fmt.Sprintf("%d", msg.Partition)),
		attribute.String("messaging.kafka.offset", fmt.Sprintf("%d", msg.Offset)),
		attribute.String("messaging.kafka.key", string(msg.Key)),
		attribute.Int("messaging.kafka.message_size", len(msg.Value)),
	)

	return msg, nil
}

// extractTraceContext extracts trace context from Kafka message headers
func extractTraceContext(headers []kafka.Header) *context.Context {
	var traceID, spanID string

	for _, header := range headers {
		switch header.Key {
		case "trace-id":
			traceID = string(header.Value)
		case "span-id":
			spanID = string(header.Value)
		}
	}

	if traceID != "" && spanID != "" {
		// For simplicity, just return context.Background()
		// In production, you'd properly reconstruct the trace context
		ctx := context.Background()
		return &ctx
	}

	return nil
}

// CommitMessages commits messages with tracing
func (ir *InstrumentedReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	ctx, span := ir.tracer.Start(ctx, "kafka.commit",
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.Int("messaging.batch_size", len(msgs)),
		),
	)
	defer span.End()

	err := ir.reader.CommitMessages(ctx, msgs...)
	if err != nil {
		span.SetAttributes(attribute.String("error", "true"))
		span.RecordError(err)
	}

	return err
}

// Close closes the Kafka reader
func (ir *InstrumentedReader) Close() error {
	return ir.reader.Close()
}

// Stats returns reader statistics
func (ir *InstrumentedReader) Stats() kafka.ReaderStats {
	return ir.reader.Stats()
}
