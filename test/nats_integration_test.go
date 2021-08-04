// +build integration

package nats_test

import (
	"context"
	"testing"
	"time"

	"github.com/loghole/nats"
	"github.com/stretchr/testify/assert"
	"github.com/uber/jaeger-client-go/config"
)

func TestNatsClient(t *testing.T) {
	time.Sleep(time.Second * 2) // timeout to nats cluster started

	t.Parallel()

	configuration := &config.Configuration{
		ServiceName: "nats-test",
		Sampler:     &config.SamplerConfig{Type: "const", Param: 1},
		Reporter:    &config.ReporterConfig{BufferFlushInterval: time.Second},
	}

	tracer, closer, err := configuration.NewTracer()
	if err != nil {
		t.Fatal(err)
	}

	defer closer.Close()

	client, err := nats.New(&nats.Config{
		Addr: "nats://n1:4222, nats://n2:4223, nats://n3:4224",
	}, nats.WithTracer(tracer))
	if err != nil {
		t.Error(err)

		return
	}

	defer client.Close()

	t.Run("PublishSubscribeTest", PublishSubscribeNatsClientTest(client))

	t.Run("PublishUnsubscribeTest", PublishUnsubscribeNatsClientTest(client))
}

func PublishSubscribeNatsClientTest(client *nats.NatsClient) func(t *testing.T) {
	return func(t *testing.T) {
		var (
			received = make([]string, 0, 0)
			expected = []string{"1", "2", "3", "4", "5"}
			topic    = "test"
		)

		_, err := client.Subscribe(topic, func(ctx context.Context, data *nats.Message) error {
			received = append(received, string(data.Data))

			return nil
		})
		if err != nil {
			t.Error(err)

			return
		}

		for _, val := range expected {
			if err := client.Publish(context.TODO(), topic, []byte(val)); err != nil {
				t.Error(err)

				return
			}
		}

		time.Sleep(time.Second)

		assert.Equal(t, expected, received)
	}
}

func PublishUnsubscribeNatsClientTest(client *nats.NatsClient) func(t *testing.T) {
	return func(t *testing.T) {
		client2, err := nats.New(&nats.Config{
			Addr: "nats://n1:4222, nats://n2:4223, nats://n3:4224",
		})
		if err != nil {
			t.Error(err)

			return
		}

		defer client2.Close()

		var (
			received = make([]string, 0, 0)
			expected = []string{"1", "2", "3", "4", "5"}
			topic    = "test"
		)

		_, err = client2.Subscribe(topic, func(ctx context.Context, data *nats.Message) error {
			received = append(received, string(data.Data))

			return nil
		})
		if err != nil {
			t.Error(err)

			return
		}

		_, err = client.Subscribe(topic, func(ctx context.Context, data *nats.Message) error {
			received = append(received, string(data.Data))

			return nil
		})
		if err != nil {
			t.Error(err)

			return
		}

		client2.Unsubscribe()

		for _, val := range expected {
			if err := client.Publish(context.TODO(), topic, []byte(val)); err != nil {
				t.Error(err)

				return
			}
		}

		time.Sleep(time.Second)

		assert.Equal(t, expected, received)
	}
}
