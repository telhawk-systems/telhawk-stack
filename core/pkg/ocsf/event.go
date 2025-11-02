package ocsf

import "time"

// Event represents an OCSF compliant event record.
type Event struct {
	// Required OCSF base fields
	CategoryUID int       `json:"category_uid"`
	ClassUID    int       `json:"class_uid"`
	ActivityID  int       `json:"activity_id"`
	TypeUID     int       `json:"type_uid"`
	Time        time.Time `json:"time"`
	SeverityID  int       `json:"severity_id"`
	Metadata    Metadata  `json:"metadata"`

	// Human-readable equivalents
	Category string `json:"category"`
	Class    string `json:"class"`
	Activity string `json:"activity"`
	Severity string `json:"severity"`

	// Status fields
	Status   string `json:"status,omitempty"`
	StatusID int    `json:"status_id,omitempty"`

	// Timing fields
	ObservedTime time.Time `json:"observed_time"`

	// Event-specific fields
	Actor       *Actor                 `json:"actor,omitempty"`
	Target      *Target                `json:"target,omitempty"`
	Enrichments map[string]interface{} `json:"enrichments,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`

	// Raw data preservation
	Raw RawDescriptor `json:"raw"`
}

// Metadata describes the event producer and OCSF schema information.
type Metadata struct {
	Product      Product  `json:"product"`
	Version      string   `json:"version"`
	Profiles     []string `json:"profiles,omitempty"`
	LogProvider  string   `json:"log_provider,omitempty"`
	OriginalTime string   `json:"original_time,omitempty"`
}

// Product identifies the product that generated the event.
type Product struct {
	Name      string `json:"name"`
	Vendor    string `json:"vendor_name"`
	Version   string `json:"version,omitempty"`
	UID       string `json:"uid,omitempty"`
	Feature   string `json:"feature,omitempty"`
	Lang      string `json:"lang,omitempty"`
	URLString string `json:"url_string,omitempty"`
}

// SchemaMetadata contains schema identifiers and versions (deprecated, use Metadata instead).
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
	if e.Actor != nil && e.Actor.Identifiers != nil {
		dup.Actor = &Actor{
			Type:        e.Actor.Type,
			Name:        e.Actor.Name,
			Identifiers: make(map[string]interface{}, len(e.Actor.Identifiers)),
		}
		for k, v := range e.Actor.Identifiers {
			dup.Actor.Identifiers[k] = v
		}
	}
	if e.Target != nil && e.Target.Identifiers != nil {
		dup.Target = &Target{
			Type:        e.Target.Type,
			Name:        e.Target.Name,
			Identifiers: make(map[string]interface{}, len(e.Target.Identifiers)),
		}
		for k, v := range e.Target.Identifiers {
			dup.Target.Identifiers[k] = v
		}
	}
	if len(e.Metadata.Profiles) > 0 {
		dup.Metadata.Profiles = make([]string, len(e.Metadata.Profiles))
		copy(dup.Metadata.Profiles, e.Metadata.Profiles)
	}
	return &dup
}
