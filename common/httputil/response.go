package httputil

import (
	"encoding/json"
	"log"
	"net/http"
)

// WriteJSON writes a JSON response with the given status code and data.
// It properly checks for encoding errors and logs them.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("ERROR: failed to encode JSON response: %v", err)
	}
}

// WriteJSONAPI writes a JSON:API compliant response.
// It sets the correct content type (application/vnd.api+json) and status code.
func WriteJSONAPI(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("ERROR: failed to encode JSON:API response: %v", err)
	}
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

// WriteJSONAPIError writes a JSON:API compliant error response.
func WriteJSONAPIError(w http.ResponseWriter, status int, code, title, detail string) {
	response := map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"status": status,
				"code":   code,
				"title":  title,
				"detail": detail,
			},
		},
	}
	WriteJSONAPI(w, status, response)
}
