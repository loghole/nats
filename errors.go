package nats

import (
	"errors"
)

var (
	ErrDrainFailed       = errors.New("drain failed")
	ErrUnsubscribeFailed = errors.New("unsubscribe failed")
	ErrInvalidConfig     = errors.New("invalid config value")
)
