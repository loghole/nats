package nats

import (
	nats "github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
)

type options struct {
	tracer opentracing.Tracer

	streamConfig *StreamConfig

	natsOption []nats.Option
}

func newOptions() *options {
	return &options{
		streamConfig: &StreamConfig{},
	}
}

type Option func(*options)

func WithTracer(tracer opentracing.Tracer) Option {
	return func(o *options) {
		o.tracer = tracer
	}
}

func WithNatsOptions(opts ...NatsOption) Option {
	return func(o *options) {
		o.natsOption = append(o.natsOption, opts...)
	}
}

func WithStreamConfig(cfg *StreamConfig) Option {
	return func(o *options) {
		o.streamConfig = cfg
	}
}
