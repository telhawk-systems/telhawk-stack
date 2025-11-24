// Package fields defines the valid queryable field paths for OCSF events.
// These fields are derived from the OpenSearch mappings and represent
// the fields that can be used in detection rules and queries.
package fields

import (
	"strings"
)

// ValidFields contains all queryable field paths supported by the OpenSearch mapping.
// Fields use jq-style dot notation (e.g., ".class_uid", ".actor.user.name").
// This list is derived from storage/internal/indexmgr/manager.go getOCSFMappings().
var ValidFields = map[string]FieldInfo{
	// Timestamps
	".time":       {Type: "date", Description: "Event timestamp"},
	".@timestamp": {Type: "date", Description: "Event timestamp (alternative)"},

	// Event classification
	".class_uid":     {Type: "integer", Description: "OCSF event class identifier"},
	".class_name":    {Type: "keyword", Description: "OCSF event class name"},
	".category_uid":  {Type: "integer", Description: "OCSF category identifier"},
	".category_name": {Type: "keyword", Description: "OCSF category name"},
	".activity_id":   {Type: "integer", Description: "Activity identifier"},
	".activity_name": {Type: "keyword", Description: "Activity name"},
	".type_uid":      {Type: "long", Description: "Full type identifier (class_uid * 100 + activity_id)"},

	// Severity and status
	".severity":      {Type: "keyword", Description: "Severity level name"},
	".severity_id":   {Type: "integer", Description: "Severity level identifier"},
	".status":        {Type: "keyword", Description: "Status name"},
	".status_id":     {Type: "integer", Description: "Status identifier (1=Success, 2=Failure)"},
	".status_detail": {Type: "text", Description: "Detailed status message"},
	".message":       {Type: "text", Description: "Event message"},

	// Metadata object
	".metadata.version":             {Type: "keyword", Description: "OCSF schema version"},
	".metadata.product.name":        {Type: "keyword", Description: "Product name"},
	".metadata.product.vendor_name": {Type: "keyword", Description: "Vendor name"},
	".metadata.product.version":     {Type: "keyword", Description: "Product version"},
	".metadata.log_name":            {Type: "keyword", Description: "Log source name"},
	".metadata.log_provider":        {Type: "keyword", Description: "Log provider"},

	// User object (target user in auth events)
	".user.name":   {Type: "keyword", Description: "Target user name"},
	".user.uid":    {Type: "keyword", Description: "Target user identifier"},
	".user.email":  {Type: "keyword", Description: "Target user email"},
	".user.domain": {Type: "keyword", Description: "Target user domain"},

	// Actor object (who performed the action)
	".actor.user":    {Type: "object", Description: "Actor user object", AllowNested: true},
	".actor.process": {Type: "object", Description: "Actor process object", AllowNested: true},

	// Process object
	".process.pid":            {Type: "integer", Description: "Process ID"},
	".process.name":           {Type: "keyword", Description: "Process name"},
	".process.cmd_line":       {Type: "text", Description: "Command line"},
	".process.uid":            {Type: "keyword", Description: "Process unique identifier"},
	".process.parent_process": {Type: "object", Description: "Parent process object", AllowNested: true},
	".process.file":           {Type: "object", Description: "Process executable file object", AllowNested: true},
	".process.file.path":      {Type: "keyword", Description: "Path to process executable"},
	".process.file.name":      {Type: "keyword", Description: "Process executable filename"},

	// File object
	".file.path":          {Type: "keyword", Description: "File path"},
	".file.name":          {Type: "keyword", Description: "File name"},
	".file.type":          {Type: "keyword", Description: "File type"},
	".file.size":          {Type: "long", Description: "File size in bytes"},
	".file.modified_time": {Type: "date", Description: "Last modified time"},

	// Source endpoint
	".src_endpoint.ip":       {Type: "ip", Description: "Source IP address"},
	".src_endpoint.port":     {Type: "integer", Description: "Source port"},
	".src_endpoint.hostname": {Type: "keyword", Description: "Source hostname"},

	// Destination endpoint
	".dst_endpoint.ip":       {Type: "ip", Description: "Destination IP address"},
	".dst_endpoint.port":     {Type: "integer", Description: "Destination port"},
	".dst_endpoint.hostname": {Type: "keyword", Description: "Destination hostname"},

	// Network connection info
	".connection_info.protocol_name": {Type: "keyword", Description: "Protocol name (TCP, UDP, etc)"},
	".connection_info.direction":     {Type: "keyword", Description: "Connection direction"},
	".connection_info.boundary":      {Type: "keyword", Description: "Network boundary"},

	// Traffic
	".traffic.bytes":   {Type: "long", Description: "Bytes transferred"},
	".traffic.packets": {Type: "long", Description: "Packets transferred"},

	// DNS objects
	".query.hostname": {Type: "keyword", Description: "DNS query hostname"},
	".query.type":     {Type: "keyword", Description: "DNS query type"},
	".query.class":    {Type: "keyword", Description: "DNS query class"},
	".answers":        {Type: "nested", Description: "DNS answers array"},
	".response_code":  {Type: "keyword", Description: "DNS response code"},

	// HTTP objects
	".http_request":  {Type: "object", Description: "HTTP request object", AllowNested: true},
	".http_response": {Type: "object", Description: "HTTP response object", AllowNested: true},

	// Detection objects
	".finding":   {Type: "object", Description: "Finding/detection object", AllowNested: true},
	".attacks":   {Type: "nested", Description: "MITRE ATT&CK mappings", AllowNested: true},
	".resources": {Type: "nested", Description: "Affected resources", AllowNested: true},

	// Risk score
	".risk_score": {Type: "integer", Description: "Risk score (0-100)"},

	// Hoisted MITRE ATT&CK fields (from first element of attacks array)
	// These are hoisted during normalization for query performance.
	// The full attacks[] nested array is still available for detailed analysis.
	".attack_tactic":        {Type: "keyword", Description: "MITRE ATT&CK tactic name (hoisted from attacks[0])"},
	".attack_tactic_uid":    {Type: "keyword", Description: "MITRE ATT&CK tactic UID, e.g. TA0003 (hoisted from attacks[0])"},
	".attack_technique":     {Type: "keyword", Description: "MITRE ATT&CK technique name (hoisted from attacks[0])"},
	".attack_technique_uid": {Type: "keyword", Description: "MITRE ATT&CK technique UID, e.g. T1547 (hoisted from attacks[0])"},

	// Auth protocol
	".auth_protocol": {Type: "keyword", Description: "Authentication protocol"},

	// Properties
	".properties": {Type: "object", Description: "Custom properties object", AllowNested: true},

	// Device object (commonly used but not explicitly mapped - allow via dynamic mapping)
	".device.hostname": {Type: "keyword", Description: "Device hostname"},
	".device.ip":       {Type: "ip", Description: "Device IP address"},
	".device.type":     {Type: "keyword", Description: "Device type"},
	".device.type_id":  {Type: "integer", Description: "Device type ID"},
	".device.os.name":  {Type: "keyword", Description: "Operating system name"},
	".device.os.type":  {Type: "keyword", Description: "Operating system type"},
}

// WildcardPrefixes are field path prefixes that allow arbitrary nested fields.
// These correspond to objects that use dynamic mapping in OpenSearch.
var WildcardPrefixes = []string{
	".metadata.",
	".actor.user.",
	".actor.process.",
	".actor.session.",
	".process.parent_process.",
	".process.file.",
	".http_request.",
	".http_response.",
	".finding.",
	".properties.",
	".device.",
	".device.os.",
	".raw.",
}

// FieldInfo contains metadata about a field
type FieldInfo struct {
	Type        string // OpenSearch field type
	Description string // Human-readable description
	AllowNested bool   // Whether nested paths under this field are allowed
}

// IsValidField checks if a field path is valid for querying.
// It returns true if the field is explicitly defined or matches a wildcard prefix.
func IsValidField(fieldPath string) bool {
	// Normalize: ensure leading dot
	if !strings.HasPrefix(fieldPath, ".") {
		fieldPath = "." + fieldPath
	}

	// Check explicit field
	if _, exists := ValidFields[fieldPath]; exists {
		return true
	}

	// Check wildcard prefixes
	for _, prefix := range WildcardPrefixes {
		if strings.HasPrefix(fieldPath, prefix) {
			return true
		}
	}

	return false
}

// ValidateFields checks multiple field paths and returns all invalid fields.
// Field paths can be with or without the leading dot.
func ValidateFields(fieldPaths []string) []string {
	var invalid []string
	for _, path := range fieldPaths {
		if !IsValidField(path) {
			invalid = append(invalid, path)
		}
	}
	return invalid
}

// GetFieldInfo returns information about a field, or nil if the field is not explicitly defined.
func GetFieldInfo(fieldPath string) *FieldInfo {
	if !strings.HasPrefix(fieldPath, ".") {
		fieldPath = "." + fieldPath
	}
	if info, exists := ValidFields[fieldPath]; exists {
		return &info
	}
	return nil
}

// ListFields returns all explicitly defined field paths (sorted for consistency).
func ListFields() []string {
	fields := make([]string, 0, len(ValidFields))
	for field := range ValidFields {
		fields = append(fields, field)
	}
	return fields
}
