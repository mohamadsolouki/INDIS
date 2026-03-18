package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"
)

// HandlerFunc is the callback signature for topic message handlers.
// data contains the raw JSON-encoded event payload.
type HandlerFunc func(ctx context.Context, topic string, data []byte) error

// subscription pairs a kafka.Reader with its registered handler.
type subscription struct {
	reader  *kafka.Reader
	handler HandlerFunc
}

// Consumer wraps one or more Kafka readers for consuming INDIS events.
type Consumer struct {
	brokers []string
	groupID string

	mu   sync.Mutex
	subs map[string]*subscription // keyed by topic
}

// NewConsumer creates a Consumer that connects to the given Kafka brokers and
// uses groupID as the consumer-group identifier.
func NewConsumer(brokers []string, groupID string) (*Consumer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("events: at least one broker address is required")
	}
	if groupID == "" {
		return nil, fmt.Errorf("events: groupID must not be empty")
	}
	return &Consumer{
		brokers: brokers,
		groupID: groupID,
		subs:    make(map[string]*subscription),
	}, nil
}

// Subscribe registers handler to be called for each message received on topic.
// Calling Subscribe after Run has been called is not safe.
func (c *Consumer) Subscribe(topic string, handler HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.brokers,
		GroupID: c.groupID,
		Topic:   topic,
	})
	c.subs[topic] = &subscription{reader: r, handler: handler}
}

// Run starts consuming all subscribed topics concurrently. It blocks until ctx
// is cancelled, then drains and closes all readers.
func (c *Consumer) Run(ctx context.Context) error {
	c.mu.Lock()
	subs := make([]*subscription, 0, len(c.subs))
	for _, s := range c.subs {
		subs = append(subs, s)
	}
	c.mu.Unlock()

	if len(subs) == 0 {
		// Nothing to consume; wait for cancellation.
		<-ctx.Done()
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(subs))

	for _, s := range subs {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				msg, err := s.reader.FetchMessage(ctx)
				if err != nil {
					// ctx cancelled or reader closed — stop the goroutine.
					if ctx.Err() != nil {
						return
					}
					errCh <- fmt.Errorf("events: fetch from %q: %w", s.reader.Config().Topic, err)
					return
				}

				if err := s.handler(ctx, msg.Topic, msg.Value); err != nil {
					// Log-worthy but non-fatal: commit the offset anyway to
					// avoid reprocessing a persistently failing message.
					// Callers should implement their own dead-letter strategy.
					_ = err
				}

				if err := s.reader.CommitMessages(ctx, msg); err != nil {
					if ctx.Err() != nil {
						return
					}
					errCh <- fmt.Errorf("events: commit offset for %q: %w", s.reader.Config().Topic, err)
					return
				}
			}
		}()
	}

	// Wait for ctx cancellation, then close all readers.
	<-ctx.Done()
	_ = c.Close()
	wg.Wait()
	close(errCh)

	// Return the first error encountered, if any.
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

// Close closes all underlying Kafka readers.
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error
	for _, s := range c.subs {
		if err := s.reader.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
