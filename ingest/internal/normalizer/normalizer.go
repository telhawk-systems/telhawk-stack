package normalizer

import (
	"context"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
	"github.com/telhawk-systems/telhawk-stack/common/ocsf"
)

// Normalizer converts raw event envelopes into OCSF events.
type Normalizer interface {
	Normalize(ctx context.Context, envelope *models.RawEventEnvelope) (*ocsf.Event, error)
	Supports(format string, sourceType string) bool
}

// Registry holds ordered normalizers and finds a match for a given envelope.
type Registry struct {
	items []Normalizer
}

// NewRegistry constructs a registry with provided normalizers.
func NewRegistry(items ...Normalizer) *Registry {
	return &Registry{items: items}
}

// Find returns the first normalizer that supports the envelope.
func (r *Registry) Find(envelope *models.RawEventEnvelope) Normalizer {
	if r == nil {
		return nil
	}
	for _, n := range r.items {
		if n.Supports(envelope.Format, envelope.SourceType) {
			return n
		}
	}
	return nil
}
