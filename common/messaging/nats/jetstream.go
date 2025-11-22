// Package nats provides JetStream support for durable, persistent messaging.
package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/telhawk-systems/telhawk-stack/common/messaging"
)

// JetStreamClient extends Client with JetStream persistence capabilities.
type JetStreamClient struct {
	*Client
	js jetstream.JetStream
}

// StreamConfig defines a JetStream stream configuration.
type StreamConfig struct {
	// Name is the stream name.
	Name string

	// Subjects are the subjects this stream captures.
	Subjects []string

	// MaxAge is the maximum age of messages in the stream.
	MaxAge time.Duration

	// MaxBytes is the maximum total size of the stream.
	MaxBytes int64

	// MaxMsgs is the maximum number of messages in the stream.
	MaxMsgs int64

	// Retention policy (LimitsPolicy, InterestPolicy, WorkQueuePolicy).
	Retention jetstream.RetentionPolicy

	// Storage type (FileStorage, MemoryStorage).
	Storage jetstream.StorageType
}

// ConsumerConfig defines a JetStream consumer configuration.
type ConsumerConfig struct {
	// Name is the durable consumer name.
	Name string

	// FilterSubject filters which messages this consumer receives.
	FilterSubject string

	// AckWait is time to wait for acknowledgment before redelivery.
	AckWait time.Duration

	// MaxDeliver is maximum delivery attempts before giving up.
	MaxDeliver int

	// MaxAckPending is maximum unacknowledged messages.
	MaxAckPending int
}

// DefaultStreamConfig returns sensible defaults for a stream.
func DefaultStreamConfig(name string, subjects []string) StreamConfig {
	return StreamConfig{
		Name:      name,
		Subjects:  subjects,
		MaxAge:    24 * time.Hour,     // Keep messages for 24 hours
		MaxBytes:  1024 * 1024 * 1024, // 1GB
		MaxMsgs:   1000000,
		Retention: jetstream.WorkQueuePolicy, // Each message delivered once
		Storage:   jetstream.FileStorage,
	}
}

// DefaultConsumerConfig returns sensible defaults for a consumer.
func DefaultConsumerConfig(name, filterSubject string) ConsumerConfig {
	return ConsumerConfig{
		Name:          name,
		FilterSubject: filterSubject,
		AckWait:       30 * time.Second,
		MaxDeliver:    3,
		MaxAckPending: 100,
	}
}

// NewJetStreamClient creates a JetStream-enabled client.
func NewJetStreamClient(cfg Config) (*JetStreamClient, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(client.conn)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &JetStreamClient{
		Client: client,
		js:     js,
	}, nil
}

// CreateOrUpdateStream creates or updates a stream.
func (c *JetStreamClient) CreateOrUpdateStream(ctx context.Context, cfg StreamConfig) (jetstream.Stream, error) {
	streamCfg := jetstream.StreamConfig{
		Name:      cfg.Name,
		Subjects:  cfg.Subjects,
		MaxAge:    cfg.MaxAge,
		MaxBytes:  cfg.MaxBytes,
		MaxMsgs:   cfg.MaxMsgs,
		Retention: cfg.Retention,
		Storage:   cfg.Storage,
	}

	stream, err := c.js.CreateOrUpdateStream(ctx, streamCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create/update stream %s: %w", cfg.Name, err)
	}

	return stream, nil
}

// CreateOrUpdateConsumer creates or updates a durable consumer.
func (c *JetStreamClient) CreateOrUpdateConsumer(ctx context.Context, streamName string, cfg ConsumerConfig) (jetstream.Consumer, error) {
	consumerCfg := jetstream.ConsumerConfig{
		Name:          cfg.Name,
		Durable:       cfg.Name,
		FilterSubject: cfg.FilterSubject,
		AckWait:       cfg.AckWait,
		MaxDeliver:    cfg.MaxDeliver,
		MaxAckPending: cfg.MaxAckPending,
		AckPolicy:     jetstream.AckExplicitPolicy,
	}

	stream, err := c.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create/update consumer %s: %w", cfg.Name, err)
	}

	return consumer, nil
}

// PublishAsync publishes a message to JetStream with acknowledgment.
func (c *JetStreamClient) PublishAsync(ctx context.Context, subject string, data []byte) (jetstream.PubAckFuture, error) {
	return c.js.PublishAsync(subject, data)
}

// PublishSync publishes a message and waits for acknowledgment.
func (c *JetStreamClient) PublishSync(ctx context.Context, subject string, data []byte) (*jetstream.PubAck, error) {
	return c.js.Publish(ctx, subject, data)
}

// ConsumeMessages starts consuming messages from a consumer with the given handler.
// Returns a context.CancelFunc to stop consuming.
func (c *JetStreamClient) ConsumeMessages(ctx context.Context, streamName, consumerName string, handler messaging.MessageHandler) (func(), error) {
	stream, err := c.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	consumer, err := stream.Consumer(ctx, consumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer %s: %w", consumerName, err)
	}

	consumeCtx, cancel := context.WithCancel(ctx)

	cons, err := consumer.Consume(func(msg jetstream.Msg) {
		m := &messaging.Message{
			Subject:   msg.Subject(),
			Data:      msg.Data(),
			Timestamp: time.Now(),
		}

		if headers := msg.Headers(); headers != nil {
			m.Metadata = make(map[string]string)
			for k := range headers {
				m.Metadata[k] = headers.Get(k)
			}
		}

		if err := handler(consumeCtx, m); err != nil {
			// NAK with delay for retry
			_ = msg.NakWithDelay(5 * time.Second)
			return
		}

		// Acknowledge successful processing
		_ = msg.Ack()
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	// Return a stop function that cancels context and stops consumer
	return func() {
		cancel()
		cons.Stop()
	}, nil
}

// Predefined stream configurations for TelHawk.
var (
	// SearchJobsStream captures search and correlation job requests.
	SearchJobsStream = StreamConfig{
		Name:      "SEARCH_JOBS",
		Subjects:  []string{"search.jobs.>"},
		MaxAge:    1 * time.Hour,
		MaxBytes:  100 * 1024 * 1024, // 100MB
		MaxMsgs:   10000,
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
	}

	// SearchResultsStream captures search results.
	SearchResultsStream = StreamConfig{
		Name:      "SEARCH_RESULTS",
		Subjects:  []string{"search.results.>"},
		MaxAge:    1 * time.Hour,
		MaxBytes:  500 * 1024 * 1024, // 500MB (results can be large)
		MaxMsgs:   10000,
		Retention: jetstream.InterestPolicy, // Delete when all consumers ack
		Storage:   jetstream.FileStorage,
	}

	// RespondEventsStream captures alert and case lifecycle events.
	RespondEventsStream = StreamConfig{
		Name:      "RESPOND_EVENTS",
		Subjects:  []string{"respond.>"},
		MaxAge:    24 * time.Hour,
		MaxBytes:  100 * 1024 * 1024, // 100MB
		MaxMsgs:   100000,
		Retention: jetstream.InterestPolicy,
		Storage:   jetstream.FileStorage,
	}
)
