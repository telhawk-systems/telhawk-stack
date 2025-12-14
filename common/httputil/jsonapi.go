package httputil

import (
	"net/http"
	"strconv"
)

// JSONAPIResource represents a single JSON:API resource.
// Use this for structured resource responses.
type JSONAPIResource struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	Attributes    interface{}            `json:"attributes"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
	Links         map[string]string      `json:"links,omitempty"`
}

// WriteJSONAPIResource writes a single JSON:API resource response.
//
// Example:
//
//	httputil.WriteJSONAPIResource(w, http.StatusOK, "user", userID, userAttributes)
func WriteJSONAPIResource(w http.ResponseWriter, status int, resourceType, id string, attributes interface{}) {
	response := map[string]interface{}{
		"data": JSONAPIResource{
			Type:       resourceType,
			ID:         id,
			Attributes: attributes,
		},
	}
	WriteJSONAPI(w, status, response)
}

// WriteJSONAPICollection writes a JSON:API collection response with optional pagination metadata.
//
// The items parameter should be a slice of objects that have "id" and "attributes" or can be
// converted to JSONAPIResource format.
//
// Example:
//
//	items := []map[string]interface{}{
//	    {"id": "1", "attributes": user1},
//	    {"id": "2", "attributes": user2},
//	}
//	pagination := &httputil.Pagination{Page: 1, Limit: 50, Total: 100}
//	httputil.WriteJSONAPICollection(w, http.StatusOK, "user", items, pagination)
func WriteJSONAPICollection(w http.ResponseWriter, status int, resourceType string, items []map[string]interface{}, pagination *Pagination) {
	data := make([]JSONAPIResource, len(items))
	for i, item := range items {
		id, _ := item["id"].(string)
		attributes := item["attributes"]
		if attributes == nil {
			// If no explicit attributes field, use the entire item
			attributes = item
		}

		data[i] = JSONAPIResource{
			Type:       resourceType,
			ID:         id,
			Attributes: attributes,
		}
	}

	response := map[string]interface{}{
		"data": data,
	}

	if pagination != nil {
		totalPages := 0
		if pagination.Limit > 0 {
			totalPages = (pagination.Total + pagination.Limit - 1) / pagination.Limit
		}
		response["meta"] = map[string]interface{}{
			"pagination": map[string]interface{}{
				"page":        pagination.Page,
				"limit":       pagination.Limit,
				"total":       pagination.Total,
				"total_pages": totalPages,
			},
		}
	}

	WriteJSONAPI(w, status, response)
}

// WriteJSONAPIErrorResponse writes a JSON:API compliant error response with multiple errors.
//
// Example:
//
//	errors := []httputil.JSONAPIErrorObject{
//	    {Status: 400, Code: "validation_error", Title: "Validation Failed", Detail: "Username is required"},
//	    {Status: 400, Code: "validation_error", Title: "Validation Failed", Detail: "Email is invalid"},
//	}
//	httputil.WriteJSONAPIErrorResponse(w, http.StatusBadRequest, errors)
func WriteJSONAPIErrorResponse(w http.ResponseWriter, status int, errors []JSONAPIErrorObject) {
	response := map[string]interface{}{
		"errors": errors,
	}
	WriteJSONAPI(w, status, response)
}

// JSONAPIErrorObject represents a single JSON:API error.
type JSONAPIErrorObject struct {
	Status int               `json:"status,omitempty"`
	Code   string            `json:"code,omitempty"`
	Title  string            `json:"title,omitempty"`
	Detail string            `json:"detail,omitempty"`
	Source map[string]string `json:"source,omitempty"` // e.g., {"pointer": "/data/attributes/email"}
}

// NewJSONAPIError creates a single JSON:API error object.
// This is a convenience function for creating simple error responses.
//
// Example:
//
//	err := httputil.NewJSONAPIError(http.StatusBadRequest, "invalid_input", "Invalid Input", "Username cannot be empty")
//	httputil.WriteJSONAPIErrorResponse(w, http.StatusBadRequest, []httputil.JSONAPIErrorObject{err})
func NewJSONAPIError(status int, code, title, detail string) JSONAPIErrorObject {
	return JSONAPIErrorObject{
		Status: status,
		Code:   code,
		Title:  title,
		Detail: detail,
	}
}

// WriteJSONAPIValidationError writes a validation error response.
// This is a convenience wrapper for common validation failures.
//
// Example:
//
//	httputil.WriteJSONAPIValidationError(w, "Username is required")
func WriteJSONAPIValidationError(w http.ResponseWriter, detail string) {
	WriteJSONAPIError(w, http.StatusBadRequest, "validation_failed", "Validation Failed", detail)
}

// WriteJSONAPINotFoundError writes a 404 not found error response.
//
// Example:
//
//	httputil.WriteJSONAPINotFoundError(w, "User", userID)
func WriteJSONAPINotFoundError(w http.ResponseWriter, resourceType, id string) {
	WriteJSONAPIError(w, http.StatusNotFound, "not_found", "Resource Not Found",
		"The requested "+resourceType+" with ID '"+id+"' was not found")
}

// WriteJSONAPIUnauthorizedError writes a 401 unauthorized error response.
//
// Example:
//
//	httputil.WriteJSONAPIUnauthorizedError(w, "Invalid or expired token")
func WriteJSONAPIUnauthorizedError(w http.ResponseWriter, detail string) {
	WriteJSONAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", detail)
}

// WriteJSONAPIForbiddenError writes a 403 forbidden error response.
//
// Example:
//
//	httputil.WriteJSONAPIForbiddenError(w, "Admin role required")
func WriteJSONAPIForbiddenError(w http.ResponseWriter, detail string) {
	WriteJSONAPIError(w, http.StatusForbidden, "forbidden", "Forbidden", detail)
}

// WriteJSONAPIInternalError writes a 500 internal server error response.
// Use this sparingly and ensure errors are logged with context.
//
// Example:
//
//	log.Printf("Database error: %v", err)
//	httputil.WriteJSONAPIInternalError(w, "An internal error occurred")
func WriteJSONAPIInternalError(w http.ResponseWriter, detail string) {
	WriteJSONAPIError(w, http.StatusInternalServerError, "internal_error", "Internal Server Error", detail)
}

// BuildJSONAPIResponse is a helper to manually construct complex JSON:API responses.
// Use this when you need relationships, links, or included resources.
//
// Example:
//
//	resource := httputil.JSONAPIResource{
//	    Type: "user",
//	    ID:   "123",
//	    Attributes: userAttrs,
//	    Relationships: map[string]interface{}{
//	        "posts": map[string]interface{}{
//	            "links": map[string]string{
//	                "related": "/users/123/posts",
//	            },
//	        },
//	    },
//	}
//	response := httputil.BuildJSONAPIResponse(resource, nil)
//	httputil.WriteJSONAPI(w, http.StatusOK, response)
func BuildJSONAPIResponse(data interface{}, included []JSONAPIResource) map[string]interface{} {
	response := map[string]interface{}{
		"data": data,
	}
	if len(included) > 0 {
		response["included"] = included
	}
	return response
}

// Helper function to convert status code to string (needed for JSON:API spec)
func statusToString(status int) string {
	return strconv.Itoa(status)
}
