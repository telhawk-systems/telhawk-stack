package ocsf

import "time"

// Event represents an OCSF compliant event record.
type Event struct {
	Class        string                 `json:"class"`
	Category     string                 `json:"category"`
	Activity     string                 `json:"activity"`
	Schema       SchemaMetadata         `json:"schema"`
	Severity     string                 `json:"severity"`
	ObservedTime time.Time              `json:"observed_time"`
	Actor        Actor                  `json:"actor"`
	Target       Target                 `json:"target"`
	Enrichments  map[string]interface{} `json:"enrichments,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Raw          RawDescriptor          `json:"raw"`
}

// SchemaMetadata contains schema identifiers and versions.
type SchemaMetadata struct {
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

// RawDescriptor keeps original payload metadata for traceability.
type RawDescriptor struct {
	Format string      `json:"format"`
	Data   interface{} `json:"data"`
}

// Actor describes the initiating party of the event.
type Actor struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name,omitempty"`
	Identifiers map[string]interface{} `json:"identifiers,omitempty"`
}

// Target describes the affected resource.
type Target struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name,omitempty"`
	Identifiers map[string]interface{} `json:"identifiers,omitempty"`
}

// Clone returns a deep copy of the event for safe mutation.
func (e *Event) Clone() *Event {
	dup := *e
	if e.Enrichments != nil {
		dup.Enrichments = make(map[string]interface{}, len(e.Enrichments))
		for k, v := range e.Enrichments {
			dup.Enrichments[k] = v
		}
	}
	if e.Properties != nil {
		dup.Properties = make(map[string]interface{}, len(e.Properties))
		for k, v := range e.Properties {
			dup.Properties[k] = v
		}
	}
	if e.Actor.Identifiers != nil {
		dup.Actor.Identifiers = make(map[string]interface{}, len(e.Actor.Identifiers))
		for k, v := range e.Actor.Identifiers {
			dup.Actor.Identifiers[k] = v
		}
	}
	if e.Target.Identifiers != nil {
		dup.Target.Identifiers = make(map[string]interface{}, len(e.Target.Identifiers))
		for k, v := range e.Target.Identifiers {
			dup.Target.Identifiers[k] = v
		}
	}
	return &dup
}
