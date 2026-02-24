package kafka

import (
	"context"
	"errors"

	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/observability"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type Handler interface {
	Handle(ctx context.Context, record []byte)
}

type kgoRecordCarrier struct {
	record *kgo.Record
}

func (c kgoRecordCarrier) Get(key string) string {
	for _, h := range c.record.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c kgoRecordCarrier) Set(key string, value string) {
	// Not needed for consumer
}

func (c kgoRecordCarrier) Keys() []string {
	keys := make([]string, 0, len(c.record.Headers))
	for _, h := range c.record.Headers {
		keys = append(keys, h.Key)
	}
	return keys
}

type Consumer struct {
	client  *kgo.Client
	handler Handler
}

func New(brokers, topics []string, handler Handler) (*Consumer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup("delivery-service-group"),
		kgo.ConsumeTopics(topics...),
		kgo.OnPartitionsRevoked(func(ctx context.Context, _ *kgo.Client, _ map[string][]int32) {
			observability.GetLogger(ctx).Info("kafka partitions revoked")
		}),
		kgo.OnPartitionsAssigned(func(ctx context.Context, _ *kgo.Client, _ map[string][]int32) {
			observability.GetLogger(ctx).Info("kafka partitions assigned")
		}),
	)
	if err != nil {
		return nil, err
	}
	return &Consumer{client: cl, handler: handler}, nil
}

func (c *Consumer) Start(ctx context.Context) {
	go func() {
		log := observability.GetLogger(ctx)
		log.Info("kafka consumer started")
		for {
			select {
			case <-ctx.Done():
				log.Info("kafka consumer loop stopping: context canceled")
				return
			default:
				fetches := c.client.PollFetches(ctx)
				if errs := fetches.Errors(); len(errs) > 0 {
					for _, ferr := range errs {
						if errors.Is(ferr.Err, context.Canceled) {
							return
						}
						log.Error("kafka fetch error", zap.String("topic", ferr.Topic), zap.Int32("partition", ferr.Partition), zap.Error(ferr.Err))
					}
					continue
				}

				fetches.EachRecord(func(r *kgo.Record) {
					// Extract trace context
					ctx := otel.GetTextMapPropagator().Extract(ctx, kgoRecordCarrier{record: r})
					c.handler.Handle(ctx, r.Value)
				})
			}
		}
	}()
}

func (c *Consumer) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
