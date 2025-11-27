package nats

import (
	"context"

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
	return p.client.PublishJSON(ctx, messaging.SubjectRespondAlertsCreated, event)
}

// PublishAlertUpdated publishes an alert updated event.
func (p *Publisher) PublishAlertUpdated(ctx context.Context, event *AlertUpdatedEvent) error {
	return p.client.PublishJSON(ctx, messaging.SubjectRespondAlertsUpdated, event)
}

// PublishCaseCreated publishes a case created event.
func (p *Publisher) PublishCaseCreated(ctx context.Context, event *CaseCreatedEvent) error {
	return p.client.PublishJSON(ctx, messaging.SubjectRespondCasesCreated, event)
}

// PublishCaseUpdated publishes a case updated event.
func (p *Publisher) PublishCaseUpdated(ctx context.Context, event *CaseUpdatedEvent) error {
	return p.client.PublishJSON(ctx, messaging.SubjectRespondCasesUpdated, event)
}

// PublishCaseAssigned publishes a case assigned event.
func (p *Publisher) PublishCaseAssigned(ctx context.Context, event *CaseAssignedEvent) error {
	return p.client.PublishJSON(ctx, messaging.SubjectRespondCasesAssigned, event)
}

// RequestCorrelation publishes a correlation job request to the search service.
func (p *Publisher) RequestCorrelation(ctx context.Context, req *CorrelationJobRequest) error {
	return p.client.PublishJSON(ctx, messaging.SubjectSearchJobsCorrelate, req)
}
