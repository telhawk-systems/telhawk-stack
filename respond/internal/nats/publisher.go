package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/telhawk-systems/telhawk-stack/common/messaging"
	natsclient "github.com/telhawk-systems/telhawk-stack/common/messaging/nats"
)

// Publisher publishes events to NATS subjects for the respond service.
type Publisher struct {
	client *natsclient.Client
}

// NewPublisher creates a new NATS publisher.
func NewPublisher(client *natsclient.Client) *Publisher {
	return &Publisher{client: client}
}

// PublishAlertCreated publishes an alert created event.
func (p *Publisher) PublishAlertCreated(ctx context.Context, event *AlertCreatedEvent) error {
	return p.publish(ctx, messaging.SubjectRespondAlertsCreated, event)
}

// PublishAlertUpdated publishes an alert updated event.
func (p *Publisher) PublishAlertUpdated(ctx context.Context, event *AlertUpdatedEvent) error {
	return p.publish(ctx, messaging.SubjectRespondAlertsUpdated, event)
}

// PublishCaseCreated publishes a case created event.
func (p *Publisher) PublishCaseCreated(ctx context.Context, event *CaseCreatedEvent) error {
	return p.publish(ctx, messaging.SubjectRespondCasesCreated, event)
}

// PublishCaseUpdated publishes a case updated event.
func (p *Publisher) PublishCaseUpdated(ctx context.Context, event *CaseUpdatedEvent) error {
	return p.publish(ctx, messaging.SubjectRespondCasesUpdated, event)
}

// PublishCaseAssigned publishes a case assigned event.
func (p *Publisher) PublishCaseAssigned(ctx context.Context, event *CaseAssignedEvent) error {
	return p.publish(ctx, messaging.SubjectRespondCasesAssigned, event)
}

// RequestCorrelation publishes a correlation job request to the search service.
func (p *Publisher) RequestCorrelation(ctx context.Context, req *CorrelationJobRequest) error {
	return p.publish(ctx, messaging.SubjectSearchJobsCorrelate, req)
}

// publish marshals data to JSON and publishes to the specified subject.
func (p *Publisher) publish(ctx context.Context, subject string, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	return p.client.Publish(ctx, subject, bytes)
}
