package nats

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	nats "github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

const jetstreamComponent = "Jetstream"

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
	// nolint:godox // wait next version
	if err != nil && !strings.Contains(err.Error(), "stream not found") { // TODO !errors.Is(err, nats.ErrStreamNotFound) {
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
	var parentCtx opentracing.SpanContext

	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentCtx = parent.Context()
	}

	span := c.tracer().StartSpan(defaultNameFunc(subject), opentracing.ChildOf(parentCtx))
	defer span.Finish()

	ext.SpanKindProducer.Set(span)
	ext.MessageBusDestination.Set(span, subject)
	ext.Component.Set(span, jetstreamComponent)

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
		return nil, fmt.Errorf("jetstream publish: %w", err)
	}

	return ack, nil
}

func (c *JetstreamClient) Subscribe(subject string, handler Handler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	sub, err := c.jetStream.Subscribe(subject, buildHandler(c.tracer(), jetstreamComponent, subject, handler), opts...)
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

func (c *JetstreamClient) tracer() opentracing.Tracer {
	if c.options.tracer != nil {
		return c.options.tracer
	}

	return &opentracing.NoopTracer{}
}

func defaultNameFunc(method string) string {
	return jetstreamComponent + " " + method
}
