package httputil

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// SourceType represents how a request was made (web, CLI, API, etc.)
type SourceType int

const (
	SourceTypeUnknown SourceType = 0
	SourceTypeWeb     SourceType = 1
	SourceTypeCLI     SourceType = 2
	SourceTypeAPI     SourceType = 3
	SourceTypeSystem  SourceType = 4
)

// String returns a human-readable representation of the source type.
func (s SourceType) String() string {
	switch s {
	case SourceTypeWeb:
		return "web"
	case SourceTypeCLI:
		return "cli"
	case SourceTypeAPI:
		return "api"
	case SourceTypeSystem:
		return "system"
	default:
		return "unknown"
	}
}

// RequestContext holds audit context information about the HTTP request.
// Used to populate audit fields (IP address, source type) in database records.
type RequestContext struct {
	IP         net.IP     // Client IP address
	SourceType SourceType // How the request was made
	UserAgent  string     // User-Agent header for additional context
}

// requestContextKey is the context key for RequestContext.
type requestContextKey struct{}

// NewRequestContext creates a RequestContext from an HTTP request.
// Source type is determined from headers:
//   - X-TelHawk-Source: "web", "cli", "api" (explicit)
//   - User-Agent containing "thawk" or "TelHawk CLI" -> CLI
//   - Default: Web (most requests come from the UI)
func NewRequestContext(r *http.Request) *RequestContext {
	ipStr := GetClientIP(r)
	// Strip port if present (from RemoteAddr format "ip:port")
	if host, _, err := net.SplitHostPort(ipStr); err == nil {
		ipStr = host
	}

	ctx := &RequestContext{
		IP:        net.ParseIP(ipStr),
		UserAgent: r.Header.Get("User-Agent"),
	}

	// Determine source type
	if source := r.Header.Get("X-TelHawk-Source"); source != "" {
		switch strings.ToLower(source) {
		case "web":
			ctx.SourceType = SourceTypeWeb
		case "cli":
			ctx.SourceType = SourceTypeCLI
		case "api":
			ctx.SourceType = SourceTypeAPI
		case "system":
			ctx.SourceType = SourceTypeSystem
		default:
			ctx.SourceType = SourceTypeUnknown
		}
	} else if ua := ctx.UserAgent; ua != "" {
		// Infer from User-Agent
		uaLower := strings.ToLower(ua)
		if strings.Contains(uaLower, "thawk") || strings.Contains(uaLower, "telhawk-cli") {
			ctx.SourceType = SourceTypeCLI
		} else {
			ctx.SourceType = SourceTypeWeb // Default for browser-like requests
		}
	}

	return ctx
}

// WithRequestContext adds RequestContext to the context.
func WithRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
	return context.WithValue(ctx, requestContextKey{}, reqCtx)
}

// GetRequestContext retrieves RequestContext from the context.
// Returns nil if not present.
func GetRequestContext(ctx context.Context) *RequestContext {
	if rc, ok := ctx.Value(requestContextKey{}).(*RequestContext); ok {
		return rc
	}
	return nil
}

// IPString returns the IP address as a string, or empty string if nil.
func (rc *RequestContext) IPString() string {
	if rc == nil || rc.IP == nil {
		return ""
	}
	return rc.IP.String()
}

// GetClientIP extracts the real client IP address from request headers.
// It handles proxy scenarios by checking headers in this order:
//  1. X-Forwarded-For (extracts first/client IP from comma-separated list)
//  2. X-Real-IP (single IP from reverse proxy)
//  3. RemoteAddr (direct connection)
//
// Example X-Forwarded-For: "203.0.113.195, 70.41.3.18, 150.172.238.178"
// Returns: "203.0.113.195" (the original client)
func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// We want the first (client) IP
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// ParseIntParam parses an integer query parameter with a default value.
// Returns defaultVal if the parameter is empty or invalid.
//
// Example:
//
//	page := httputil.ParseIntParam(r.URL.Query().Get("page"), 1)
//	limit := httputil.ParseIntParam(r.URL.Query().Get("limit"), 50)
func ParseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultVal
}

// Pagination represents common pagination parameters for API responses.
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total,omitempty"`
}

// ParsePagination extracts pagination parameters from query string.
// It enforces sensible defaults and maximum limits to prevent abuse.
//
// Parameters:
//   - r: HTTP request with query parameters
//   - defaultLimit: Default page size if not specified
//   - maxLimit: Maximum allowed page size (prevents excessive queries)
//
// Example:
//
//	pagination := httputil.ParsePagination(r, 50, 1000)
//	// Use pagination.Page and pagination.Limit for database queries
func ParsePagination(r *http.Request, defaultLimit, maxLimit int) Pagination {
	page := ParseIntParam(r.URL.Query().Get("page"), 1)
	limit := ParseIntParam(r.URL.Query().Get("limit"), defaultLimit)

	// Enforce maximum limit to prevent abuse
	if limit > maxLimit {
		limit = maxLimit
	}

	// Ensure minimum page is 1
	if page < 1 {
		page = 1
	}

	return Pagination{
		Page:  page,
		Limit: limit,
	}
}

// Offset calculates the database offset for pagination.
// Returns (page-1) * limit for use in SQL OFFSET clauses.
//
// Example:
//
//	pagination := httputil.ParsePagination(r, 50, 1000)
//	offset := pagination.Offset()
//	// SELECT * FROM users LIMIT pagination.Limit OFFSET offset
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}
