package nats

import (
	"context"

	nats "github.com/nats-io/nats.go"
)

// Message is nats.Msg.
type Message = nats.Msg

// Handler to handle sub message.
type Handler func(ctx context.Context, msg *Message) error

type PubAck = nats.PubAck

type StreamConfig = nats.StreamConfig

type NatsOption = nats.Option // nolint:golint,revive // need nats option.
