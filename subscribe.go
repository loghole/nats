package nats

import (
	"bytes"
	"context"
	"fmt"

	nats "github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func (c *JetStreamClient) Subscribe(subject string, handler Handler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	sub, err := c.jetStream.Subscribe(subject, c.buildHandler(subject, handler), opts...)
	if err != nil {
		return nil, fmt.Errorf("jetStream subscribe: %w", err)
	}

	c.subs = append(c.subs, sub)

	return sub, nil
}

func (c *JetStreamClient) buildHandler(subject string, handler Handler) nats.MsgHandler {
	return func(msg *nats.Msg) {
		// Read trace carrier.
		buf := bytes.NewBuffer(msg.Data)

		spanContext, _ := c.tracer().Extract(opentracing.Binary, buf)

		span := c.tracer().StartSpan(defaultNameFunc(subject), opentracing.FollowsFrom(spanContext))
		defer span.Finish()

		ext.SpanKindConsumer.Set(span)
		ext.MessageBusDestination.Set(span, msg.Subject)
		ext.Component.Set(span, component)

		ctx := opentracing.ContextWithSpan(context.Background(), span)

		// Drop span data from msg data.
		msg.Data = buf.Bytes()

		if err := handler(ctx, msg); err != nil {
			ext.Error.Set(span, true)
			ext.LogError(span, err)

			return
		}
	}
}
