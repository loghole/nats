package nats

import (
	nats "github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
)

const component = "NATS JetStream"

type JetStreamClient struct {
	config    *Config
	options   *options
	natsConn  *nats.Conn
	jetStream nats.JetStreamContext

	subs []*nats.Subscription
}

func New(config *Config, opts ...Option) (*JetStreamClient, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	client := &JetStreamClient{
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

func (c *JetStreamClient) tracer() opentracing.Tracer {
	if c.options.tracer != nil {
		return c.options.tracer
	}

	return &opentracing.NoopTracer{}
}

func defaultNameFunc(method string) string {
	return component + " " + method
}
