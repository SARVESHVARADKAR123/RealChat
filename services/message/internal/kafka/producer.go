package kafka

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
)

type Producer struct {
	p     *kafka.Producer
	topic string
}

func NewProducer(brokers, topic string) (*Producer, error) {

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":                     brokers,
		"acks":                                  "all",
		"enable.idempotence":                    true,
		"retries":                               1000000,
		"max.in.flight.requests.per.connection": 5,
	})
	if err != nil {
		return nil, err
	}

	return &Producer{
		p:     p,
		topic: topic,
	}, nil
}

type kafkaHeaderCarrier struct {
	headers *[]kafka.Header
}

func (c kafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c kafkaHeaderCarrier) Set(key string, value string) {
	*c.headers = append(*c.headers, kafka.Header{
		Key:   key,
		Value: []byte(value),
	})
}

func (c kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.headers))
	for _, h := range *c.headers {
		keys = append(keys, h.Key)
	}
	return keys
}

func (p *Producer) Publish(
	ctx context.Context,
	key string,
	value []byte,
) error {

	headers := []kafka.Header{}
	otel.GetTextMapPropagator().Inject(ctx, kafkaHeaderCarrier{headers: &headers})

	deliveryChan := make(chan kafka.Event, 1)

	err := p.p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &p.topic,
			Partition: kafka.PartitionAny,
		},
		Key:     []byte(key),
		Value:   value,
		Headers: headers,
	}, deliveryChan)

	if err != nil {
		return err
	}

	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		return m.TopicPartition.Error
	}

	return nil
}

// Flush waits up to timeoutMs for all in-flight messages to be delivered.
func (p *Producer) Flush(timeoutMs int) {
	p.p.Flush(timeoutMs)
}
