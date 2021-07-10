package nats

import (
	"fmt"
)

type Config struct {
	Addr       string // "nats://localhost:4222, nats://localhost:4223, nats://localhost:4224"
	StreamName string
	Subjects   []string
}

func (c *Config) validate() error {
	if c.Addr == "" {
		return fmt.Errorf("%w: empty addr", ErrInvalidConfig)
	}

	return nil
}
