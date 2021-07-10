package nats

import (
	"fmt"
	"strings"

	nats "github.com/nats-io/nats.go"
)

func (c *JetStreamClient) Connect() (err error) {
	c.natsConn, err = nats.Connect(c.config.Addr, c.options.natsOption...)
	if err != nil {
		return fmt.Errorf("connect nats: %w", err)
	}

	c.jetStream, err = c.natsConn.JetStream()
	if err != nil {
		return fmt.Errorf("get jet stream: %w", err)
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
	}

	return nil
}
