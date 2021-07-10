package nats

import (
	"bytes"
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

func (c *JetStreamClient) Publish(ctx context.Context, subject string, data []byte) (*PubAck, error) {
	var parentCtx opentracing.SpanContext

	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentCtx = parent.Context()
	}

	span := c.tracer().StartSpan(defaultNameFunc(subject), opentracing.ChildOf(parentCtx))
	defer span.Finish()

	ext.SpanKindProducer.Set(span)
	ext.MessageBusDestination.Set(span, subject)
	ext.Component.Set(span, component)

	buf := bytes.NewBuffer(nil)

	// We have no better place to record an error than the Span itself.
	if err := c.tracer().Inject(span.Context(), opentracing.Binary, buf); err != nil {
		span.LogFields(log.String("event", "Tracer.Inject() failed"), log.Error(err))
	}

	// Write payload.
	if _, err := buf.Write(data); err != nil {
		return nil, fmt.Errorf("write payload: %w", err)
	}

	ack, err := c.jetStream.Publish(subject, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("jetStream publish: %w", err)
	}

	return ack, nil
}
