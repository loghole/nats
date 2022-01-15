//go:build integration
// +build integration

package nats_test

import (
	"context"
	"testing"
	"time"

	gonats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	"github.com/loghole/nats"
)

func TestJetstreamClient(t *testing.T) {
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
			semconv.ServiceNameKey.String("jetstream-test"),
		)),
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	defer tp.Shutdown(context.Background())

	client, err := nats.NewJetstream(&nats.Config{
		Addr:       "nats://n1:4222, nats://n2:4223, nats://n3:4224",
		StreamName: "TEST",
		Subjects:   []string{"TEST.test"},
	}, nats.WithTracer(tp.Tracer("default")))
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
