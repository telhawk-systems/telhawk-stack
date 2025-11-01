package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/core/internal/validator"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

// Pipeline orchestrates raw event normalization and validation.
type Pipeline struct {
	normalizers *normalizer.Registry
	validators  *validator.Chain
}

// New creates a pipeline instance.
func New(registry *normalizer.Registry, validators *validator.Chain) *Pipeline {
	return &Pipeline{
		normalizers: registry,
		validators:  validators,
	}
}

// Process converts the raw envelope into a validated OCSF event.
func (p *Pipeline) Process(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
	if p == nil {
		return nil, fmt.Errorf("pipeline not configured")
	}

	n := p.normalizers.Find(envelope)
	if n == nil {
		return nil, fmt.Errorf("no normalizer registered for format=%s source_type=%s", envelope.Format, envelope.SourceType)
	}

	event, err := n.Normalize(ctx, envelope)
	if err != nil {
		return nil, fmt.Errorf("normalize: %w", err)
	}

	if err := p.validators.Validate(ctx, event); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return event, nil
}

// MarshalResult serializes the final OCSF event into JSON for transport.
func MarshalResult(event *ocsf.Event) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("nil event")
	}
	return json.Marshal(event)
}
