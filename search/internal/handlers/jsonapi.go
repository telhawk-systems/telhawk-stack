package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/search/internal/models"
)

// jsonAPIResource represents a JSON:API resource object.
type jsonAPIResource struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	Attributes    map[string]interface{} `json:"attributes"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
}

// writeJSONAPIList writes a JSON:API list response for saved searches.
func (h *Handler) writeJSONAPIList(w http.ResponseWriter, status int, searches []models.SavedSearch) {
	items := make([]jsonAPIResource, 0, len(searches))
	for _, s := range searches {
		items = append(items, toJSONAPI(s))
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items})
}

// writeJSONAPIListWithMeta writes a JSON:API list response with metadata and links.
func (h *Handler) writeJSONAPIListWithMeta(w http.ResponseWriter, status int, searches []models.SavedSearch, meta map[string]interface{}, links map[string]interface{}) {
	items := make([]jsonAPIResource, 0, len(searches))
	for _, s := range searches {
		items = append(items, toJSONAPI(s))
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	resp := map[string]interface{}{"data": items}
	if meta != nil {
		resp["meta"] = meta
	}
	if links != nil {
		resp["links"] = links
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeJSONAPIResource writes a single JSON:API resource response.
func (h *Handler) writeJSONAPIResource(w http.ResponseWriter, status int, s *models.SavedSearch) {
	h.writeJSONAPIResourceWithLinks(w, status, s, nil)
}

// writeJSONAPIResourceWithLinks writes a single JSON:API resource with links.
func (h *Handler) writeJSONAPIResourceWithLinks(w http.ResponseWriter, status int, s *models.SavedSearch, links map[string]interface{}) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	resp := map[string]interface{}{"data": toJSONAPI(*s)}
	if links != nil {
		resp["links"] = links
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeJSONAPIResourceGeneric writes a JSON:API single resource with provided type/id/attributes.
func (h *Handler) writeJSONAPIResourceGeneric(w http.ResponseWriter, status int, typ, id string, attrs map[string]interface{}, rels map[string]interface{}) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	res := jsonAPIResource{Type: typ, ID: id, Attributes: attrs, Relationships: rels}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": res})
}

// writeJSONAPIError writes a JSON:API error response.
func (h *Handler) writeJSONAPIError(w http.ResponseWriter, status int, code, title string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"errors": []map[string]string{{
		"status": fmt.Sprintf("%d", status),
		"code":   code,
		"title":  title,
	}}})
}

// writeJSONAPIUnauthorized writes a 401 Unauthorized JSON:API error.
func (h *Handler) writeJSONAPIUnauthorized(w http.ResponseWriter) {
	h.writeJSONAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
}

// writeJSONAPIConflict writes a 409 Conflict JSON:API error.
func (h *Handler) writeJSONAPIConflict(w http.ResponseWriter, code, title string) {
	h.writeJSONAPIError(w, http.StatusConflict, code, title)
}

// toJSONAPI converts a SavedSearch to a JSON:API resource.
func toJSONAPI(s models.SavedSearch) jsonAPIResource {
	attrs := map[string]interface{}{
		"version_id":  s.VersionID,
		"name":        s.Name,
		"query":       s.Query,
		"filters":     s.Filters,
		"is_global":   s.IsGlobal,
		"created_at":  s.CreatedAt,
		"disabled_at": s.DisabledAt,
		"hidden_at":   s.HiddenAt,
	}
	rels := map[string]interface{}{}
	if s.OwnerID != nil {
		rels["owner"] = map[string]interface{}{
			"data": map[string]interface{}{"type": "user", "id": *s.OwnerID},
		}
	}
	if s.CreatedBy != "" {
		rels["created_by"] = map[string]interface{}{
			"data": map[string]interface{}{"type": "user", "id": s.CreatedBy},
		}
	}
	return jsonAPIResource{Type: "saved-search", ID: s.ID, Attributes: attrs, Relationships: rels}
}

// decodeJSONAPI decodes a JSON:API request body into the destination.
func (h *Handler) decodeJSONAPI(body io.ReadCloser, dst interface{}) error {
	defer body.Close()
	var payload struct {
		Data struct{ Attributes json.RawMessage }
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return err
	}
	return json.Unmarshal(payload.Data.Attributes, dst)
}

// decodeJSONAPIResource decodes a JSON:API resource and returns type, id, and attributes.
func (h *Handler) decodeJSONAPIResource(body io.ReadCloser, dst interface{}) (string, string, error) {
	defer body.Close()
	var payload struct {
		Data struct {
			Type       string          `json:"type"`
			ID         string          `json:"id"`
			Attributes json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return "", "", err
	}
	if err := json.Unmarshal(payload.Data.Attributes, dst); err != nil {
		return "", "", err
	}
	return payload.Data.Type, payload.Data.ID, nil
}

// acceptJSONAPI checks if the request accepts JSON:API content type.
func acceptJSONAPI(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" || accept == "*/*" {
		return true
	}
	return containsJSONAPI(accept)
}

// methodNotAllowedJSONAPI writes a 405 Method Not Allowed response in JSON:API format.
func (h *Handler) methodNotAllowedJSONAPI(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", joinMethods(allowed...))
	h.writeJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
}
