package logging

import "log/slog"

// Common field names for consistent logging across services.
const (
	FieldService  = "service"
	FieldUserID   = "user_id"
	FieldUsername = "username"
	FieldIP       = "ip"
	FieldMethod   = "method"
	FieldPath     = "path"
	FieldStatus   = "status"
	FieldDuration = "duration_ms"
	FieldError    = "error"
	FieldTokenID  = "token_id"
	FieldEventID  = "event_id"
	FieldQuery    = "query"
)

// Service returns a slog attribute for the service name.
func Service(name string) slog.Attr {
	return slog.String(FieldService, name)
}

// UserID returns a slog attribute for the user ID.
func UserID(id string) slog.Attr {
	return slog.String(FieldUserID, id)
}

// Username returns a slog attribute for the username.
func Username(name string) slog.Attr {
	return slog.String(FieldUsername, name)
}

// IP returns a slog attribute for the IP address.
func IP(ip string) slog.Attr {
	return slog.String(FieldIP, ip)
}

// Method returns a slog attribute for the HTTP method.
func Method(method string) slog.Attr {
	return slog.String(FieldMethod, method)
}

// Path returns a slog attribute for the HTTP path.
func Path(path string) slog.Attr {
	return slog.String(FieldPath, path)
}

// Status returns a slog attribute for the HTTP status code.
func Status(code int) slog.Attr {
	return slog.Int(FieldStatus, code)
}

// Duration returns a slog attribute for duration in milliseconds.
func Duration(ms int64) slog.Attr {
	return slog.Int64(FieldDuration, ms)
}

// Error returns a slog attribute for an error.
func Error(err error) slog.Attr {
	return slog.String(FieldError, err.Error())
}

// TokenID returns a slog attribute for a token ID.
func TokenID(id string) slog.Attr {
	return slog.String(FieldTokenID, id)
}

// EventID returns a slog attribute for an event ID.
func EventID(id string) slog.Attr {
	return slog.String(FieldEventID, id)
}

// Query returns a slog attribute for a query string.
func Query(query string) slog.Attr {
	return slog.String(FieldQuery, query)
}
