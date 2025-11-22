// Package messaging provides abstractions for message broker communication.
// It defines interfaces that allow services to publish and subscribe to messages
// without being coupled to a specific broker implementation.
package messaging

import (
	"context"
	"time"
)

// Message represents a message received from or sent to a message broker.
type Message struct {
	// Subject is the topic/channel the message was published to.
	Subject string

	// Data is the raw message payload.
	Data []byte

	// Reply is an optional subject for request/reply patterns.
	// When set, the receiver can publish a response to this subject.
	Reply string

	// Metadata contains optional key-value pairs for message headers.
	Metadata map[string]string

	// Timestamp is when the message was published.
	Timestamp time.Time
}

// MessageHandler processes a received message.
// Return an error to indicate processing failure (may trigger retry depending on implementation).
type MessageHandler func(ctx context.Context, msg *Message) error

// Subscription represents an active subscription to a subject.
type Subscription interface {
	// Unsubscribe stops receiving messages on this subscription.
	Unsubscribe() error

	// Subject returns the subject this subscription is listening to.
	Subject() string

	// IsValid returns true if the subscription is still active.
	IsValid() bool
}

// Publisher publishes messages to subjects.
type Publisher interface {
	// Publish sends a message to the specified subject.
	// The message is fire-and-forget; use Request for request/reply.
	Publish(ctx context.Context, subject string, data []byte) error

	// PublishMsg sends a Message with full control over headers and metadata.
	PublishMsg(ctx context.Context, msg *Message) error

	// Request sends a message and waits for a response (request/reply pattern).
	// The timeout controls how long to wait for a response.
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) (*Message, error)

	// Close releases any resources held by the publisher.
	Close() error
}

// Subscriber subscribes to messages on subjects.
type Subscriber interface {
	// Subscribe creates a subscription to the specified subject.
	// Each subscriber receives all messages (fan-out).
	Subscribe(subject string, handler MessageHandler) (Subscription, error)

	// QueueSubscribe creates a queue subscription.
	// Messages are load-balanced across subscribers in the same queue group.
	// Use this for worker pools where each message should be processed once.
	QueueSubscribe(subject, queue string, handler MessageHandler) (Subscription, error)

	// Close releases any resources and unsubscribes all active subscriptions.
	Close() error
}

// Client combines Publisher and Subscriber interfaces.
// Most services will use Client for full messaging capabilities.
type Client interface {
	Publisher
	Subscriber

	// Drain gracefully closes the connection, allowing in-flight messages to complete.
	Drain() error

	// IsConnected returns true if the client is connected to the broker.
	IsConnected() bool
}

// PublishOption configures message publishing behavior.
type PublishOption func(*publishOptions)

type publishOptions struct {
	headers map[string]string
}

// WithHeader adds a header to the published message.
func WithHeader(key, value string) PublishOption {
	return func(o *publishOptions) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// SubscribeOption configures subscription behavior.
type SubscribeOption func(*subscribeOptions)

type subscribeOptions struct {
	maxInFlight int
	ackWait     time.Duration
}

// WithMaxInFlight sets the maximum number of unacknowledged messages.
func WithMaxInFlight(n int) SubscribeOption {
	return func(o *subscribeOptions) {
		o.maxInFlight = n
	}
}

// WithAckWait sets the time to wait for acknowledgment before redelivery.
func WithAckWait(d time.Duration) SubscribeOption {
	return func(o *subscribeOptions) {
		o.ackWait = d
	}
}
