package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/telhawk-systems/telhawk-stack/common/messaging/nats"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
)

// JetStreamQueue writes failed events to NATS JetStream for centralized DLQ.
// Safe for use across multiple ingest instances.
type JetStreamQueue struct {
	js      *nats.JetStreamClient
	stream  jetstream.Stream
	written uint64
}

// NewJetStreamQueue creates a DLQ backed by NATS JetStream.
func NewJetStreamQueue(ctx context.Context, js *nats.JetStreamClient) (*JetStreamQueue, error) {
	if js == nil {
		return nil, fmt.Errorf("jetstream client is nil")
	}

	// Create or update the DLQ stream
	stream, err := js.CreateOrUpdateStream(ctx, nats.IngestDLQStream)
	if err != nil {
		return nil, fmt.Errorf("create dlq stream: %w", err)
	}

	log.Printf("DLQ: JetStream stream %s ready", nats.IngestDLQStream.Name)

	return &JetStreamQueue{
		js:     js,
		stream: stream,
	}, nil
}

// Write records a failed event to the JetStream DLQ.
func (q *JetStreamQueue) Write(ctx context.Context, envelope *models.RawEventEnvelope, err error, reason string) error {
	if q == nil {
		return nil
	}

	failed := FailedEvent{
		Timestamp:   time.Now().UTC(),
		Envelope:    envelope,
		Error:       err.Error(),
		Reason:      reason,
		Attempts:    1,
		LastAttempt: time.Now().UTC(),
	}

	data, marshalErr := json.Marshal(failed)
	if marshalErr != nil {
		log.Printf("ERROR: failed to marshal DLQ entry: %v", marshalErr)
		return marshalErr
	}

	// Subject format: ingest.dlq.<reason>
	subject := fmt.Sprintf("ingest.dlq.%s", reason)

	_, pubErr := q.js.PublishSync(ctx, subject, data)
	if pubErr != nil {
		log.Printf("ERROR: failed to publish DLQ entry: %v", pubErr)
		return pubErr
	}

	atomic.AddUint64(&q.written, 1)
	log.Printf("DLQ: published failed event (reason: %s)", reason)

	return nil
}

// Stats returns DLQ metrics from JetStream.
func (q *JetStreamQueue) Stats(ctx context.Context) map[string]interface{} {
	if q == nil {
		return map[string]interface{}{
			"enabled": false,
			"backend": "jetstream",
		}
	}

	info, err := q.stream.Info(ctx)
	if err != nil {
		log.Printf("ERROR: failed to get DLQ stream info: %v", err)
		return map[string]interface{}{
			"enabled":       true,
			"backend":       "jetstream",
			"written_local": atomic.LoadUint64(&q.written),
			"error":         err.Error(),
		}
	}

	return map[string]interface{}{
		"enabled":        true,
		"backend":        "jetstream",
		"written_local":  atomic.LoadUint64(&q.written),
		"total_messages": info.State.Msgs,
		"total_bytes":    info.State.Bytes,
		"first_seq":      info.State.FirstSeq,
		"last_seq":       info.State.LastSeq,
		"consumer_count": info.State.Consumers,
	}
}

// List returns failed events from the JetStream DLQ.
func (q *JetStreamQueue) List(ctx context.Context, limit int) ([]FailedEvent, error) {
	if q == nil {
		return nil, fmt.Errorf("dlq not enabled")
	}

	if limit <= 0 {
		limit = 100
	}

	// Create an ephemeral consumer to read messages
	consumer, err := q.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		FilterSubject: "ingest.dlq.>",
		AckPolicy:     jetstream.AckNonePolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		MaxDeliver:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("create list consumer: %w", err)
	}

	var events []FailedEvent
	msgs, err := consumer.Fetch(limit, jetstream.FetchMaxWait(2*time.Second))
	if err != nil {
		return nil, fmt.Errorf("fetch messages: %w", err)
	}

	for msg := range msgs.Messages() {
		var failed FailedEvent
		if err := json.Unmarshal(msg.Data(), &failed); err != nil {
			log.Printf("ERROR: failed to parse DLQ message: %v", err)
			continue
		}
		events = append(events, failed)
	}

	if msgs.Error() != nil {
		log.Printf("WARN: fetch completed with error: %v", msgs.Error())
	}

	return events, nil
}

// Purge removes all events from the DLQ stream.
func (q *JetStreamQueue) Purge(ctx context.Context) error {
	if q == nil {
		return fmt.Errorf("dlq not enabled")
	}

	err := q.stream.Purge(ctx)
	if err != nil {
		return fmt.Errorf("purge dlq stream: %w", err)
	}

	log.Printf("DLQ: purged all messages from stream")
	return nil
}
