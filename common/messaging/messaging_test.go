package messaging

import (
	"testing"
	"time"
)

func TestMessage_Fields(t *testing.T) {
	// Test that Message struct can be created with all fields
	now := time.Now()
	msg := Message{
		Subject:   "test.subject",
		Data:      []byte("test data"),
		Reply:     "reply.subject",
		Metadata:  map[string]string{"key": "value"},
		Timestamp: now,
	}

	if msg.Subject != "test.subject" {
		t.Errorf("expected Subject 'test.subject', got %q", msg.Subject)
	}
	if string(msg.Data) != "test data" {
		t.Errorf("expected Data 'test data', got %q", string(msg.Data))
	}
	if msg.Reply != "reply.subject" {
		t.Errorf("expected Reply 'reply.subject', got %q", msg.Reply)
	}
	if msg.Metadata["key"] != "value" {
		t.Errorf("expected Metadata key 'value', got %q", msg.Metadata["key"])
	}
	if !msg.Timestamp.Equal(now) {
		t.Errorf("expected Timestamp %v, got %v", now, msg.Timestamp)
	}
}

func TestMessage_ZeroValue(t *testing.T) {
	// Test that zero value Message is valid
	var msg Message

	if msg.Subject != "" {
		t.Errorf("expected empty Subject, got %q", msg.Subject)
	}
	if msg.Data != nil {
		t.Errorf("expected nil Data, got %v", msg.Data)
	}
	if msg.Reply != "" {
		t.Errorf("expected empty Reply, got %q", msg.Reply)
	}
	if msg.Metadata != nil {
		t.Errorf("expected nil Metadata, got %v", msg.Metadata)
	}
	if !msg.Timestamp.IsZero() {
		t.Errorf("expected zero Timestamp, got %v", msg.Timestamp)
	}
}

func TestWithHeader(t *testing.T) {
	tests := []struct {
		name     string
		headers  []struct{ key, value string }
		expected map[string]string
	}{
		{
			name:     "single header",
			headers:  []struct{ key, value string }{{"X-Custom", "test"}},
			expected: map[string]string{"X-Custom": "test"},
		},
		{
			name: "multiple headers",
			headers: []struct{ key, value string }{
				{"X-First", "first"},
				{"X-Second", "second"},
			},
			expected: map[string]string{"X-First": "first", "X-Second": "second"},
		},
		{
			name: "overwrite header",
			headers: []struct{ key, value string }{
				{"X-Key", "original"},
				{"X-Key", "updated"},
			},
			expected: map[string]string{"X-Key": "updated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &publishOptions{}

			for _, h := range tt.headers {
				opt := WithHeader(h.key, h.value)
				opt(opts)
			}

			for k, v := range tt.expected {
				if opts.headers[k] != v {
					t.Errorf("expected header %q=%q, got %q", k, v, opts.headers[k])
				}
			}
		})
	}
}

func TestWithHeader_InitializesMap(t *testing.T) {
	// Test that WithHeader initializes the map if nil
	opts := &publishOptions{headers: nil}

	opt := WithHeader("key", "value")
	opt(opts)

	if opts.headers == nil {
		t.Error("expected headers map to be initialized")
	}
	if opts.headers["key"] != "value" {
		t.Errorf("expected header value 'value', got %q", opts.headers["key"])
	}
}

func TestWithMaxInFlight(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"positive value", 100, 100},
		{"zero value", 0, 0},
		{"negative value", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &subscribeOptions{}

			opt := WithMaxInFlight(tt.value)
			opt(opts)

			if opts.maxInFlight != tt.expected {
				t.Errorf("expected maxInFlight %d, got %d", tt.expected, opts.maxInFlight)
			}
		})
	}
}

func TestWithAckWait(t *testing.T) {
	tests := []struct {
		name     string
		value    time.Duration
		expected time.Duration
	}{
		{"30 seconds", 30 * time.Second, 30 * time.Second},
		{"1 minute", time.Minute, time.Minute},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &subscribeOptions{}

			opt := WithAckWait(tt.value)
			opt(opts)

			if opts.ackWait != tt.expected {
				t.Errorf("expected ackWait %v, got %v", tt.expected, opts.ackWait)
			}
		})
	}
}

func TestSubscribeOptions_Multiple(t *testing.T) {
	// Test applying multiple subscribe options
	opts := &subscribeOptions{}

	options := []SubscribeOption{
		WithMaxInFlight(50),
		WithAckWait(45 * time.Second),
	}

	for _, opt := range options {
		opt(opts)
	}

	if opts.maxInFlight != 50 {
		t.Errorf("expected maxInFlight 50, got %d", opts.maxInFlight)
	}
	if opts.ackWait != 45*time.Second {
		t.Errorf("expected ackWait 45s, got %v", opts.ackWait)
	}
}
