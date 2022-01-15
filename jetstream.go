package nats

import (
	"context"
	"errors"
	"fmt"

	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

const _jetstreamComponent = "Jetstream"

type JetstreamClient struct {
	config    *Config
	options   *options
	natsConn  *nats.Conn
	jetStream nats.JetStreamContext

	subs []*nats.Subscription
}

func NewJetstream(config *Config, opts ...Option) (*JetstreamClient, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	client := &JetstreamClient{
		config:  config,
		options: newOptions(),
	}

	for _, o := range opts {
		o(client.options)
	}

	client.options.streamConfig.Subjects = client.config.Subjects
	client.options.streamConfig.Name = client.config.StreamName

	if err := client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *JetstreamClient) Connect() (err error) {
	c.natsConn, err = nats.Connect(c.config.Addr, c.options.natsOption...)
	if err != nil {
		return fmt.Errorf("connect nats: %w", err)
	}

	c.jetStream, err = c.natsConn.JetStream()
	if err != nil {
		return fmt.Errorf("get jetstream: %w", err)
	}

	stream, err := c.jetStream.StreamInfo(c.options.streamConfig.Name)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return fmt.Errorf("get stream info: %w", err)
	}

	if stream == nil {
		if _, err = c.jetStream.AddStream(c.options.streamConfig); err != nil {
			return fmt.Errorf("add stream: %w", err)
		}

		return nil
	}

	if _, err = c.jetStream.UpdateStream(c.options.streamConfig); err != nil {
		return fmt.Errorf("add stream: %w", err)
	}

	return nil
}

func (c *JetstreamClient) Publish(ctx context.Context, subject string, data []byte) (*PubAck, error) {
	msg := &Message{
		Subject: subject,
		Header:  make(map[string][]string),
		Data:    data,
	}

	ctx, span := c.tracer().Start(ctx, defaultNameFunc(_jetstreamComponent, subject))
	defer span.End()

	new(propagation.TraceContext).Inject(ctx, propagation.HeaderCarrier(msg.Header))

	span.SetAttributes(
		semconv.MessagingSystemKey.String(_natsComponent),
		semconv.MessagingDestinationKey.String(subject),
		semconv.MessagingMessagePayloadSizeBytesKey.Int(len(data)),
	)

	ack, err := c.jetStream.PublishMsg(msg)
	if err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.RecordError(err)

		return nil, fmt.Errorf("nats publish: %w", err)
	}

	return ack, nil
}

func (c *JetstreamClient) Subscribe(subject string, handler Handler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	sub, err := c.jetStream.Subscribe(subject, buildHandler(c.tracer(), _jetstreamComponent, subject, handler), opts...)
	if err != nil {
		return nil, fmt.Errorf("jetstream subscribe: %w", err)
	}

	c.subs = append(c.subs, sub)

	return sub, nil
}

func (c *JetstreamClient) Unsubscribe() error {
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

func (c *JetstreamClient) Drain() error {
	if err := c.natsConn.Drain(); err != nil {
		return fmt.Errorf("nats drain: %w", err)
	}

	return nil
}

func (c *JetstreamClient) Close() {
	c.natsConn.Close()
}

func (c *JetstreamClient) tracer() trace.Tracer {
	if c.options.tracer != nil {
		return c.options.tracer
	}

	return trace.NewNoopTracerProvider().Tracer("")
}
