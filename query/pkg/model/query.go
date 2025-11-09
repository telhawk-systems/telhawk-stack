package model

import "time"

// Query represents the canonical JSON query structure for TelHawk.
// All query interfaces (UI filter chips, text syntax, raw JSON) translate to this format.
type Query struct {
	Select       []string      `json:"select,omitempty"`
	Filter       *FilterExpr   `json:"filter,omitempty"`
	TimeRange    *TimeRangeDef `json:"timeRange,omitempty"`
	Aggregations []Aggregation `json:"aggregations,omitempty"`
	Sort         []SortSpec    `json:"sort,omitempty"`
	Limit        int           `json:"limit,omitempty"`
	Offset       int           `json:"offset,omitempty"`
	Cursor       string        `json:"cursor,omitempty"`
}

// FilterExpr represents a filter expression which can be either a simple condition
// or a compound condition (AND, OR, NOT).
type FilterExpr struct {
	// Simple condition fields
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`

	// Compound condition fields
	Type       string       `json:"type,omitempty"` // "and", "or", "not"
	Conditions []FilterExpr `json:"conditions,omitempty"`
	Condition  *FilterExpr  `json:"condition,omitempty"` // for NOT
}

// TimeRangeDef defines the time bounds for a query.
// Supports both absolute timestamps and relative time ranges.
type TimeRangeDef struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
	Last  string     `json:"last,omitempty"` // e.g., "15m", "1h", "24h", "7d"
}

// Aggregation defines a statistical aggregation to perform on query results.
type Aggregation struct {
	Type         string        `json:"type"`                   // "terms", "date_histogram", "avg", "sum", "min", "max", "stats", "cardinality"
	Field        string        `json:"field,omitempty"`        // OCSF field path (e.g., ".actor.user.name")
	Name         string        `json:"name"`                   // Identifier for this aggregation in results
	Size         int           `json:"size,omitempty"`         // For terms aggregations
	Interval     string        `json:"interval,omitempty"`     // For date_histogram (e.g., "1h")
	Aggregations []Aggregation `json:"aggregations,omitempty"` // Nested aggregations
}

// SortSpec defines a field to sort by and the sort direction.
type SortSpec struct {
	Field string `json:"field"`           // OCSF field path (e.g., ".time")
	Order string `json:"order,omitempty"` // "asc" or "desc"
}

// Supported filter operators
const (
	OpEq         = "eq"         // Equals
	OpNe         = "ne"         // Not equals
	OpGt         = "gt"         // Greater than
	OpGte        = "gte"        // Greater than or equal
	OpLt         = "lt"         // Less than
	OpLte        = "lte"        // Less than or equal
	OpIn         = "in"         // In array
	OpContains   = "contains"   // String contains
	OpStartsWith = "startsWith" // String starts with
	OpEndsWith   = "endsWith"   // String ends with
	OpRegex      = "regex"      // Regular expression
	OpExists     = "exists"     // Field exists
	OpCIDR       = "cidr"       // IP in CIDR range
)

// Supported compound filter types
const (
	FilterTypeAnd = "and"
	FilterTypeOr  = "or"
	FilterTypeNot = "not"
)

// Supported aggregation types
const (
	AggTypeTerms         = "terms"
	AggTypeDateHistogram = "date_histogram"
	AggTypeAvg           = "avg"
	AggTypeSum           = "sum"
	AggTypeMin           = "min"
	AggTypeMax           = "max"
	AggTypeStats         = "stats"
	AggTypeCardinality   = "cardinality"
)

// IsSimpleCondition returns true if this filter is a simple field condition (not compound).
func (f *FilterExpr) IsSimpleCondition() bool {
	return f.Type == "" && f.Field != ""
}

// IsCompoundCondition returns true if this filter is a compound condition (AND/OR/NOT).
func (f *FilterExpr) IsCompoundCondition() bool {
	return f.Type != ""
}
