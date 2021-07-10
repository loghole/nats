package nats

import (
	"fmt"
)

func (c *JetStreamClient) Close() error {
	defer c.natsConn.Close()

	if err := c.natsConn.Drain(); err != nil {
		return fmt.Errorf("nats drain: %w", err)
	}

	return nil
}

func (c *JetStreamClient) Unsubscribe() error {
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

func (c *JetStreamClient) Drain() error {
	errList := make([]error, 0)

	for _, sub := range c.subs {
		if err := sub.Drain(); err != nil {
			errList = append(errList, fmt.Errorf("unsubscribe '%s': %w", sub.Subject, err))
		}
	}

	if len(errList) > 0 {
		return fmt.Errorf("nats %w: %v", ErrDrainFailed, errList)
	}

	return nil
}
