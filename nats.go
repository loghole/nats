package nats

import (
	"context"
	"fmt"

	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

const _natsComponent = "NATS"

type NatsClient struct { // nolint:golint,revive // need nats client.
	config  *Config
	options *options

	natsConn *nats.Conn
	subs     []*nats.Subscription
}

func New(config *Config, opts ...Option) (*NatsClient, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	client := &NatsClient{
		config:  config,
		options: newOptions(),
	}

	for _, o := range opts {
		o(client.options)
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *NatsClient) Connect() (err error) {
	c.natsConn, err = nats.Connect(c.config.Addr, c.options.natsOption...)
	if err != nil {
		return fmt.Errorf("connect nats: %w", err)
	}

	return nil
}

func (c *NatsClient) Publish(ctx context.Context, subject string, data []byte) error {
	msg := &Message{
		Subject: subject,
		Header:  make(map[string][]string),
		Data:    data,
	}

	ctx, span := c.tracer().Start(ctx, defaultNameFunc(_natsComponent, subject))
	defer span.End()

	new(propagation.TraceContext).Inject(ctx, propagation.HeaderCarrier(msg.Header))

	span.SetAttributes(
		semconv.MessagingSystemKey.String(_natsComponent),
		semconv.MessagingDestinationKey.String(subject),
		semconv.MessagingMessagePayloadSizeBytesKey.Int(len(data)),
	)

	if err := c.natsConn.PublishMsg(msg); err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.RecordError(err)

		return fmt.Errorf("nats publish: %w", err)
	}

	return nil
}

func (c *NatsClient) Subscribe(subject string, handler Handler) (*nats.Subscription, error) {
	sub, err := c.natsConn.Subscribe(subject, buildHandler(c.tracer(), _natsComponent, subject, handler))
	if err != nil {
		return nil, fmt.Errorf("nats subscribe: %w", err)
	}

	c.subs = append(c.subs, sub)

	return sub, nil
}

func (c *NatsClient) Unsubscribe() error {
	errList := make([]error, 0)

	for _, sub := range c.subs {
		if err := sub.Unsubscribe(); err != nil {
			errList = append(errList, fmt.Errorf("unsubscribe '%s': %w", sub.Subject, err))
		}
	}

	if len(errList) > 0 {
		return fmt.Errorf("nats %w: %v", ErrUnsubscribeFailed, errList)
	}

	return nil
}

func (c *NatsClient) Drain() error {
	if err := c.natsConn.Drain(); err != nil {
		return fmt.Errorf("nats drain: %w", err)
	}

	return nil
}

func (c *NatsClient) Close() {
	c.natsConn.Close()
}

func (c *NatsClient) tracer() trace.Tracer {
	if c.options.tracer != nil {
		return c.options.tracer
	}

	return trace.NewNoopTracerProvider().Tracer("")
}
