package nats

import (
	"context"

	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

func buildHandler(tracer trace.Tracer, component, subject string, handler Handler) nats.MsgHandler {
	return func(msg *nats.Msg) {
		ctx := new(propagation.TraceContext).Extract(context.Background(), propagation.HeaderCarrier(msg.Header))

		ctx, span := tracer.Start(ctx, defaultNameFunc(component, subject))
		defer span.End()

		span.SetAttributes(
			semconv.MessagingSystemKey.String(_natsComponent),
			semconv.MessagingDestinationKey.String(subject),
			semconv.MessagingMessagePayloadSizeBytesKey.Int(len(msg.Data)),
		)

		if err := handler(ctx, msg); err != nil {
			span.SetAttributes(attribute.Bool("error", true))
			span.RecordError(err)

			return
		}
	}
}

func defaultNameFunc(component, method string) string {
	return component + " " + method
}
