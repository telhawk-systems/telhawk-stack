// Package nats provides a NATS implementation of the messaging interfaces.
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/telhawk-systems/telhawk-stack/common/messaging"
)

// Client implements messaging.Client using NATS.
type Client struct {
	conn *nats.Conn
	mu   sync.RWMutex
	subs []*subscription
}

// Config holds NATS client configuration.
type Config struct {
	// URL is the NATS server URL (e.g., "nats://localhost:4222").
	URL string

	// Name is the client name for connection identification.
	Name string

	// MaxReconnects is the maximum number of reconnection attempts.
	// Use -1 for infinite reconnects.
	MaxReconnects int

	// ReconnectWait is the time to wait between reconnection attempts.
	ReconnectWait time.Duration

	// Timeout is the connection timeout.
	Timeout time.Duration

	// Username for authentication (optional).
	Username string

	// Password for authentication (optional).
	Password string

	// Token for token-based authentication (optional).
	Token string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		URL:           nats.DefaultURL,
		Name:          "telhawk-client",
		MaxReconnects: -1, // Infinite reconnects
		ReconnectWait: 2 * time.Second,
		Timeout:       5 * time.Second,
	}
}

// NewClient creates a new NATS client with the given configuration.
func NewClient(cfg Config) (*Client, error) {
	opts := []nats.Option{
		nats.Name(cfg.Name),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.Timeout(cfg.Timeout),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				// Log disconnect - services should use their own logger
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			fmt.Println("NATS reconnected")
		}),
	}

	if cfg.Username != "" && cfg.Password != "" {
		opts = append(opts, nats.UserInfo(cfg.Username, cfg.Password))
	}
	if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Client{
		conn: conn,
		subs: make([]*subscription, 0),
	}, nil
}

// Publish sends a message to the specified subject.
func (c *Client) Publish(ctx context.Context, subject string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.conn.Publish(subject, data)
}

// PublishJSON marshals data to JSON and publishes to the subject.
// This is a convenience method for publishing JSON-encoded messages.
func (c *Client) PublishJSON(ctx context.Context, subject string, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	return c.Publish(ctx, subject, bytes)
}

// PublishMsg sends a Message with full control over headers and metadata.
func (c *Client) PublishMsg(ctx context.Context, msg *messaging.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Data:    msg.Data,
		Reply:   msg.Reply,
	}

	// Add headers if present
	if len(msg.Metadata) > 0 {
		natsMsg.Header = make(nats.Header)
		for k, v := range msg.Metadata {
			natsMsg.Header.Set(k, v)
		}
	}

	return c.conn.PublishMsg(natsMsg)
}

// Request sends a message and waits for a response.
func (c *Client) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) (*messaging.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	resp, err := c.conn.Request(subject, data, timeout)
	if err != nil {
		return nil, err
	}

	return natsToMessage(resp), nil
}

// Subscribe creates a subscription to the specified subject.
func (c *Client) Subscribe(subject string, handler messaging.MessageHandler) (messaging.Subscription, error) {
	sub, err := c.conn.Subscribe(subject, func(msg *nats.Msg) {
		ctx := context.Background()
		if err := handler(ctx, natsToMessage(msg)); err != nil {
			// Handler error - could add metrics/logging hook here
			fmt.Printf("Handler error for %s: %v\n", subject, err)
		}
	})
	if err != nil {
		return nil, err
	}

	s := &subscription{natsSub: sub}
	c.mu.Lock()
	c.subs = append(c.subs, s)
	c.mu.Unlock()

	return s, nil
}

// QueueSubscribe creates a queue subscription for load-balanced message processing.
func (c *Client) QueueSubscribe(subject, queue string, handler messaging.MessageHandler) (messaging.Subscription, error) {
	sub, err := c.conn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		ctx := context.Background()
		if err := handler(ctx, natsToMessage(msg)); err != nil {
			fmt.Printf("Handler error for %s (queue: %s): %v\n", subject, queue, err)
		}
	})
	if err != nil {
		return nil, err
	}

	s := &subscription{natsSub: sub}
	c.mu.Lock()
	c.subs = append(c.subs, s)
	c.mu.Unlock()

	return s, nil
}

// Close releases all resources.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Unsubscribe all active subscriptions
	for _, sub := range c.subs {
		_ = sub.Unsubscribe()
	}
	c.subs = nil

	c.conn.Close()
	return nil
}

// Drain gracefully closes, allowing in-flight messages to complete.
func (c *Client) Drain() error {
	return c.conn.Drain()
}

// IsConnected returns true if connected to NATS.
func (c *Client) IsConnected() bool {
	return c.conn.IsConnected()
}

// subscription wraps a NATS subscription.
type subscription struct {
	natsSub *nats.Subscription
}

func (s *subscription) Unsubscribe() error {
	return s.natsSub.Unsubscribe()
}

func (s *subscription) Subject() string {
	return s.natsSub.Subject
}

func (s *subscription) IsValid() bool {
	return s.natsSub.IsValid()
}

// natsToMessage converts a NATS message to our Message type.
func natsToMessage(msg *nats.Msg) *messaging.Message {
	m := &messaging.Message{
		Subject:   msg.Subject,
		Data:      msg.Data,
		Reply:     msg.Reply,
		Timestamp: time.Now(), // NATS core doesn't provide timestamp
	}

	if msg.Header != nil {
		m.Metadata = make(map[string]string)
		for k := range msg.Header {
			m.Metadata[k] = msg.Header.Get(k)
		}
	}

	return m
}
