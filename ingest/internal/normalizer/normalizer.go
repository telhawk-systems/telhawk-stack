// Package normalizer converts raw event envelopes into OCSF events.
//
// # Field Hoisting Requirements
//
// For query performance, normalizers MUST hoist certain nested array fields
// to the root level. This avoids expensive OpenSearch nested queries.
//
// Required hoisted fields (from attacks[0] if present):
//
//   - attack_tactic: string       - attacks[0].tactic.name
//   - attack_tactic_uid: string   - attacks[0].tactic.uid (e.g., "TA0003")
//   - attack_technique: string    - attacks[0].technique.name
//   - attack_technique_uid: string - attacks[0].technique.uid (e.g., "T1547")
//
// Example hoisting in Normalize():
//
//	if len(event.Attacks) > 0 {
//	    if event.Attacks[0].Tactic != nil {
//	        event.AttackTactic = event.Attacks[0].Tactic.Name
//	        event.AttackTacticUID = event.Attacks[0].Tactic.UID
//	    }
//	    if event.Attacks[0].Technique != nil {
//	        event.AttackTechnique = event.Attacks[0].Technique.Name
//	        event.AttackTechniqueUID = event.Attacks[0].Technique.UID
//	    }
//	}
//
// See also: common/fields/fields.go for the full list of queryable fields.
package normalizer

import (
	"context"

	"github.com/telhawk-systems/telhawk-stack/common/ocsf"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
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
