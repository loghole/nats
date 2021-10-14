package nats

import (
	"bytes"
	"context"
	"fmt"

	nats "github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

const natsComponent = "NATS"

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
	var parentCtx opentracing.SpanContext

	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentCtx = parent.Context()
	}

	span := c.tracer().StartSpan(defaultNameFunc(natsComponent, subject), opentracing.ChildOf(parentCtx))
	defer span.Finish()

	ext.SpanKindProducer.Set(span)
	ext.MessageBusDestination.Set(span, subject)
	ext.Component.Set(span, natsComponent)

	buf := bytes.NewBuffer(nil)

	// We have no better place to record an error than the Span itself.
	if err := c.tracer().Inject(span.Context(), opentracing.Binary, buf); err != nil {
		span.LogFields(log.String("event", "Tracer.Inject() failed"), log.Error(err))
	}

	// Write payload.
	if _, err := buf.Write(data); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	if err := c.natsConn.Publish(subject, buf.Bytes()); err != nil {
		return fmt.Errorf("nats publish: %w", err)
	}

	return nil
}

func (c *NatsClient) Subscribe(subject string, handler Handler) (*nats.Subscription, error) {
	sub, err := c.natsConn.Subscribe(subject, buildHandler(c.tracer(), natsComponent, subject, handler))
	if err != nil {
		return nil, fmt.Errorf("nats subscribe: %w", err)
	}

	c.subs = append(c.subs, sub)

	return sub, nil
}

func (c *NatsClient) RequestWithContext(ctx context.Context, subject string, data []byte) (*Message, error) {
	var parentCtx opentracing.SpanContext

	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentCtx = parent.Context()
	}

	span := c.tracer().StartSpan(defaultNameFunc(natsComponent, subject), opentracing.ChildOf(parentCtx))
	defer span.Finish()

	ext.SpanKindProducer.Set(span)
	ext.MessageBusDestination.Set(span, subject)
	ext.Component.Set(span, natsComponent)

	buf := bytes.NewBuffer(nil)

	// We have no better place to record an error than the Span itself.
	if err := c.tracer().Inject(span.Context(), opentracing.Binary, buf); err != nil {
		span.LogFields(log.String("event", "Tracer.Inject() failed"), log.Error(err))
	}

	// Write payload.
	if _, err := buf.Write(data); err != nil {
		return nil, fmt.Errorf("write payload: %w", err)
	}

	msg, err := c.natsConn.RequestWithContext(ctx, subject, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("nats request with context: %w", err)
	}

	return msg, nil
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

func (c *NatsClient) tracer() opentracing.Tracer {
	if c.options.tracer != nil {
		return c.options.tracer
	}

	return &opentracing.NoopTracer{}
}
