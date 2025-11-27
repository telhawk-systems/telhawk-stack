package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/search/internal/models"
	"github.com/telhawk-systems/telhawk-stack/search/internal/service"
)

// parseShowAll parses the filter[show_all] query parameter.
func parseShowAll(u *url.URL) bool {
	// JSON:API filter[show_all]=true
	if vals, ok := u.Query()["filter[show_all]"]; ok && len(vals) > 0 {
		v := strings.ToLower(vals[0])
		return v == "true" || v == "1"
	}
	return false
}

// parsePage parses page[number] and page[size] query parameters.
func parsePage(u *url.URL) (int, int) {
	q := u.Query()
	pn := 1
	ps := 20
	if v := q.Get("page[number]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pn = n
		}
	}
	if v := q.Get("page[size]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			ps = n
		}
	}
	return pn, ps
}

// parseCursor parses page[cursor] and page[size] for cursor-based pagination.
// Cursor format: "<unix_ns>|<version_id>"
func parseCursor(u *url.URL) (*time.Time, string, int) {
	q := u.Query()
	cur := q.Get("page[cursor]")
	size := 20
	if v := q.Get("page[size]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			size = n
		}
	}
	if cur == "" {
		return nil, "", size
	}
	parts := strings.SplitN(cur, "|", 2)
	if len(parts) != 2 {
		return nil, "", size
	}
	// parse unix ns
	ns, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, "", size
	}
	t := time.Unix(0, ns).UTC()
	return &t, parts[1], size
}

// buildCursor builds a cursor string from a SavedSearch.
func buildCursor(s *models.SavedSearch) string {
	return fmt.Sprintf("%d|%s", s.CreatedAt.UnixNano(), s.VersionID)
}

// buildNextLink builds a next page link URL.
func buildNextLink(path string, cursor interface{}) string {
	return fmt.Sprintf("%s?page[cursor]=%v", path, cursor)
}

// requireUser extracts and validates bearer token via auth service.
// Deprecated: Use requireUserContext for access to ClientID/OrganizationID.
func (h *Handler) requireUser(r *http.Request) (string, bool) {
	uc, ok := h.requireUserContext(r)
	if !ok {
		return "", false
	}
	return uc.UserID, true
}

// requireUserContext extracts full user context including ClientID for data isolation.
func (h *Handler) requireUserContext(r *http.Request) (*service.UserContext, bool) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return nil, false
	}
	token := strings.TrimSpace(authz[len("Bearer "):])
	uc, err := h.svc.ValidateTokenFull(r.Context(), token)
	if err != nil {
		return nil, false
	}
	return uc, true
}

// svcIDFallback generates a simple fallback ID string when an event lacks an ID field.
func (h *Handler) svcIDFallback() string {
	// Use time-based fallback; not stable but avoids violating JSON:API schema.
	return fmt.Sprintf("evt-%d", time.Now().UnixNano())
}

// decodeJSON decodes JSON from request body.
func decodeJSON(body io.ReadCloser, dst interface{}) error {
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	return nil
}

// containsJSONAPI checks if a string contains the JSON:API media type.
func containsJSONAPI(s string) bool {
	return strings.Contains(s, "application/vnd.api+json")
}

// joinMethods joins HTTP method names with ", ".
func joinMethods(methods ...string) string {
	return strings.Join(methods, ", ")
}
