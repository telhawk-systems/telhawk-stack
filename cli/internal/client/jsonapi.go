package client

// jsonAPIResource represents a single JSON:API resource.
type jsonAPIResource struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
}

// jsonAPIResponse represents a JSON:API response with Data as interface{}.
type jsonAPIResponse struct {
	Data   interface{}    `json:"data"`
	Errors []jsonAPIError `json:"errors,omitempty"`
}

// jsonAPIError represents a JSON:API error object.
type jsonAPIError struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// jsonAPIRequest wraps attributes in JSON:API format for requests.
type jsonAPIRequest struct {
	Data jsonAPIRequestData `json:"data"`
}

type jsonAPIRequestData struct {
	Type       string      `json:"type"`
	Attributes interface{} `json:"attributes"`
}
