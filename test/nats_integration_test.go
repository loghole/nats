//go:build integration
// +build integration

package nats_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	"github.com/loghole/nats"
)

func TestNatsClient(t *testing.T) {
	time.Sleep(time.Second * 5) // timeout to nats cluster started

	t.Parallel()

	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithAgentEndpoint(jaeger.WithAgentHost("jaeger")))
	if err != nil {
		panic(err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSpanProcessor(tracesdk.NewBatchSpanProcessor(exp)),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("nats-test"),
		)),
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	defer tp.Shutdown(context.Background())

	client, err := nats.New(&nats.Config{
		Addr: "nats://n1:4222, nats://n2:4223, nats://n3:4224",
	}, nats.WithTracer(tp.Tracer("default")))
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
