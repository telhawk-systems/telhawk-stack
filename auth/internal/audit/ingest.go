package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type IngestClient struct {
	ingestURL string
	hecToken  string
	client    *http.Client
}

func NewIngestClient(ingestURL, hecToken string) *IngestClient {
	return &IngestClient{
		ingestURL: ingestURL,
		hecToken:  hecToken,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// OCSFAuthEvent represents an OCSF Authentication event (class 3002)
type OCSFAuthEvent struct {
	ActivityID  int                      `json:"activity_id"`  // 1=Logon, 2=Logoff, 3=AuthTicket, etc.
	CategoryUID int                      `json:"category_uid"` // 3 = Identity & Access Management
	ClassUID    int                      `json:"class_uid"`    // 3002 = Authentication
	Time        int64                    `json:"time"`         // Unix timestamp in milliseconds
	TypeUID     int                      `json:"type_uid"`     // class_uid * 100 + activity_id
	Severity    string                   `json:"severity"`     // Informational, Low, Medium, High, Critical
	SeverityID  int                      `json:"severity_id"`  // 1-6
	Status      string                   `json:"status"`       // Success, Failure
	StatusID    int                      `json:"status_id"`    // 1=Success, 2=Failure
	User        OCSFUser                 `json:"user"`
	SrcEndpoint *OCSFEndpoint            `json:"src_endpoint,omitempty"`
	Metadata    OCSFMetadata             `json:"metadata"`
	Message     string                   `json:"message,omitempty"`
	Observables []OCSFObservable         `json:"observables,omitempty"`
	RawData     string                   `json:"raw_data,omitempty"`
	Enrichments []map[string]interface{} `json:"enrichments,omitempty"`
}

type OCSFUser struct {
	Name   string   `json:"name,omitempty"`
	UID    string   `json:"uid,omitempty"`
	Groups []string `json:"groups,omitempty"`
	Type   string   `json:"type,omitempty"`
	TypeID int      `json:"type_id,omitempty"`
}

type OCSFEndpoint struct {
	IP       string `json:"ip,omitempty"`
	Port     int    `json:"port,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Domain   string `json:"domain,omitempty"`
	UID      string `json:"uid,omitempty"`
}

type OCSFMetadata struct {
	Version    string            `json:"version"`
	Product    OCSFProduct       `json:"product"`
	LogName    string            `json:"log_name,omitempty"`
	LogLevel   string            `json:"log_level,omitempty"`
	Labels     []string          `json:"labels,omitempty"`
	Profiles   []string          `json:"profiles,omitempty"`
	Extensions map[string]string `json:"extensions,omitempty"`
}

type OCSFProduct struct {
	Name    string `json:"name"`
	Vendor  string `json:"vendor_name"`
	Version string `json:"version"`
	UID     string `json:"uid,omitempty"`
	Feature string `json:"feature,omitempty"`
}

type OCSFObservable struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	TypeID     int             `json:"type_id"`
	Value      string          `json:"value"`
	Reputation *OCSFReputation `json:"reputation,omitempty"`
}

type OCSFReputation struct {
	Score     int `json:"score,omitempty"`
	ScoreID   int `json:"score_id,omitempty"`
	BaseScore int `json:"base_score,omitempty"`
}

// HECEvent wraps the OCSF event for HEC ingestion
type HECEvent struct {
	Time       int64       `json:"time,omitempty"`
	Source     string      `json:"source,omitempty"`
	Sourcetype string      `json:"sourcetype,omitempty"`
	Host       string      `json:"host,omitempty"`
	Event      interface{} `json:"event"`
}

// ForwardEvent sends an audit event to the ingest service
func (c *IngestClient) ForwardEvent(actorType, actorID, actorName, action, resourceType, resourceID, ipAddress, userAgent, result, errorMessage string, metadata map[string]interface{}, timestamp time.Time) error {
	// Map audit action to OCSF activity_id
	activityID := mapActionToActivityID(action)
	statusID := 1 // Success
	if result != "success" {
		statusID = 2 // Failure
	}

	// Create OCSF Authentication event
	ocsfEvent := OCSFAuthEvent{
		ActivityID:  activityID,
		CategoryUID: 3,    // Identity & Access Management
		ClassUID:    3002, // Authentication
		Time:        timestamp.UnixMilli(),
		TypeUID:     300200 + activityID,
		Severity:    mapResultToSeverity(result, action),
		SeverityID:  mapResultToSeverityID(result, action),
		Status:      result,
		StatusID:    statusID,
		User: OCSFUser{
			Name:   actorName,
			UID:    actorID,
			Type:   actorType,
			TypeID: mapActorTypeToID(actorType),
		},
		Metadata: OCSFMetadata{
			Version: "1.1.0",
			Product: OCSFProduct{
				Name:    "TelHawk Auth",
				Vendor:  "TelHawk Systems",
				Version: "1.0.0",
			},
			LogName: "auth_audit",
		},
		Message: fmt.Sprintf("%s %s by %s", action, result, actorName),
	}

	// Add source endpoint if IP available
	if ipAddress != "" {
		ocsfEvent.SrcEndpoint = &OCSFEndpoint{
			IP: ipAddress,
		}
		ocsfEvent.Observables = append(ocsfEvent.Observables, OCSFObservable{
			Name:   "source_ip",
			Type:   "IP Address",
			TypeID: 2,
			Value:  ipAddress,
		})
	}

	// Add error message if present
	if errorMessage != "" {
		ocsfEvent.Message = fmt.Sprintf("%s: %s", ocsfEvent.Message, errorMessage)
	}

	// Add custom metadata
	if len(metadata) > 0 {
		rawMeta, _ := json.Marshal(metadata)
		ocsfEvent.RawData = string(rawMeta)
	}

	// Wrap in HEC format
	hecEvent := HECEvent{
		Time:       timestamp.Unix(),
		Source:     "telhawk:auth",
		Sourcetype: "ocsf:authentication",
		Host:       "auth-service",
		Event:      ocsfEvent,
	}

	// Send to ingest
	body, err := json.Marshal(hecEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequest("POST", c.ingestURL+"/services/collector/event", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Splunk "+c.hecToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ingest returned status %d", resp.StatusCode)
	}

	return nil
}

func mapActionToActivityID(action string) int {
	switch action {
	case "login":
		return 1 // Logon
	case "logout":
		return 2 // Logoff
	case "register":
		return 1 // Logon (user creation)
	case "token_revoke":
		return 2 // Logoff
	case "hec_token_create":
		return 3 // Authentication Ticket
	case "hec_token_revoke":
		return 99 // Other
	case "user_update":
		return 99 // Other
	case "user_delete":
		return 99 // Other
	case "password_change":
		return 99 // Other
	default:
		return 99 // Other
	}
}

func mapResultToSeverity(result, action string) string {
	if result == "success" {
		return "Informational"
	}
	// Failed logins are more critical
	if action == "login" {
		return "Medium"
	}
	return "Low"
}

func mapResultToSeverityID(result, action string) int {
	if result == "success" {
		return 1 // Informational
	}
	if action == "login" {
		return 3 // Medium
	}
	return 2 // Low
}

func mapActorTypeToID(actorType string) int {
	switch actorType {
	case "user":
		return 1 // User
	case "service":
		return 3 // System
	case "system":
		return 3 // System
	default:
		return 99 // Other
	}
}
