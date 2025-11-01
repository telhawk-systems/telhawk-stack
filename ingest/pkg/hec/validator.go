package hec

import (
	"strings"
)

// Token extracts HEC token from Authorization header
// Supports formats:
// - "Splunk <token>"
// - "Bearer <token>"
func ExtractToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return ""
	}

	// Accept both "Splunk" and "Bearer" prefixes
	scheme := strings.ToLower(parts[0])
	if scheme == "splunk" || scheme == "bearer" {
		return parts[1]
	}

	return ""
}

// ValidateEventFormat checks if event payload is valid
func ValidateEventFormat(event interface{}) error {
	if event == nil {
		return ErrInvalidEvent
	}
	return nil
}

var (
	ErrInvalidEvent = &HECError{Code: 6, Text: "Invalid data format"}
	ErrNoData       = &HECError{Code: 5, Text: "No data"}
	ErrUnauthorized = &HECError{Code: 4, Text: "Invalid authorization"}
	ErrServerBusy   = &HECError{Code: 9, Text: "Server is busy"}
)

type HECError struct {
	Code int
	Text string
}

func (e *HECError) Error() string {
	return e.Text
}
