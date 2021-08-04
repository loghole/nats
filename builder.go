package nats

import (
	"bytes"
	"context"

	nats "github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func buildHandler(tracer opentracing.Tracer, component, subject string, handler Handler) nats.MsgHandler {
	return func(msg *nats.Msg) {
		// Read trace carrier.
		buf := bytes.NewBuffer(msg.Data)

		spanContext, _ := tracer.Extract(opentracing.Binary, buf)

		span := tracer.StartSpan(defaultNameFunc(subject), opentracing.FollowsFrom(spanContext))
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
