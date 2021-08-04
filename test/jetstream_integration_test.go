// +build integration

package nats_test

import (
	"context"
	"testing"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/uber/jaeger-client-go/config"

	"github.com/loghole/nats"
)

func TestJetstreamClient(t *testing.T) {
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

	client, err := nats.NewJetstream(&nats.Config{
		Addr:       "nats://n1:4222, nats://n2:4223, nats://n3:4224",
		StreamName: "TEST",
		Subjects:   []string{"TEST.test"},
	}, nats.WithTracer(tracer))
	if err != nil {
		t.Error(err)

		return
	}

	defer client.Close()

	t.Run("PublishSubscribeTest", PublishSubscribeJetstreamClientTest(client))
}

func PublishSubscribeJetstreamClientTest(client *nats.JetstreamClient) func(t *testing.T) {
	return func(t *testing.T) {
		var (
			received = make([]string, 0, 0)
			expected = []string{"1", "2", "3", "4", "5"}
			topic    = "TEST.test"
		)

		_, err := client.Subscribe(topic, func(ctx context.Context, data *nats.Message) error {
			received = append(received, string(data.Data))

			return nil
		}, gonats.DeliverNew())
		if err != nil {
			t.Error(err)

			return
		}

		for _, val := range expected {
			_, err = client.Publish(context.TODO(), topic, []byte(val))
			if err != nil {
				t.Error(err)

				return
			}
		}

		time.Sleep(time.Second)

		assert.Equal(t, expected, received)

		if err := client.Drain(); err != nil {
			t.Error(err)
		}
	}
}
