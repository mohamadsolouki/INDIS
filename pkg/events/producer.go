package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// Producer wraps a Kafka writer for publishing INDIS events.
// Uses encoding/json for serialization.
// Broker addresses are supplied at construction time.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Producer that connects to the given Kafka brokers.
// The writer uses round-robin balancing and allows automatic topic creation.
func NewProducer(brokers []string) (*Producer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("events: at least one broker address is required")
	}
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.RoundRobin{},
		AllowAutoTopicCreation: true,
	}
	return &Producer{writer: w}, nil
}

// Publish serialises event as JSON and writes a single message to topic.
func (p *Producer) Publish(ctx context.Context, topic string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("events: marshal event for topic %q: %w", topic, err)
	}

	msg := kafka.Message{
		Topic: topic,
		Value: payload,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("events: write message to topic %q: %w", topic, err)
	}
	return nil
}

// Close flushes pending writes and closes the underlying Kafka writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
