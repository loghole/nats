package nats

import (
	"context"

	nats "github.com/nats-io/nats.go"
)

// Stan message is stan.Msg.
type Message = nats.Msg

// Handler to handle sub message.
type Handler func(ctx context.Context, msg *Message) error

type PubAck = nats.PubAck
