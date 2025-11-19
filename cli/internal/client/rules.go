package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RulesClient struct {
	baseURL string
	client  *http.Client
}

// DetectionSchema represents a detection rule in JSON:API format
type DetectionSchema struct {
	Type       string                    `json:"type"`
	ID         string                    `json:"id"`
	Attributes DetectionSchemaAttributes `json:"attributes"`
}

type DetectionSchemaAttributes struct {
	VersionID  string                 `json:"version_id"`
	Model      map[string]interface{} `json:"model"`
	View       map[string]interface{} `json:"view"`
	Controller map[string]interface{} `json:"controller"`
	CreatedAt  time.Time              `json:"created_at"`
	DisabledAt *time.Time             `json:"disabled_at,omitempty"`
	HiddenAt   *time.Time             `json:"hidden_at,omitempty"`
}

type JSONAPIResponse struct {
	Data  interface{}            `json:"data"`
	Meta  map[string]interface{} `json:"meta,omitempty"`
	Links map[string]interface{} `json:"links,omitempty"`
}

type JSONAPIError struct {
	Errors []struct {
		Status string `json:"status"`
		Code   string `json:"code"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

func NewRulesClient(baseURL string) *RulesClient {
	return &RulesClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *RulesClient) doRequest(method, path, token string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.client.Do(req)
}

func (c *RulesClient) ListSchemas(token string, page, limit int) ([]DetectionSchema, map[string]interface{}, error) {
	path := fmt.Sprintf("/schemas?page[number]=%d&page[size]=%d", page, limit)

	resp, err := c.doRequest("GET", path, token, nil)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp JSONAPIError
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && len(errResp.Errors) > 0 {
			return nil, nil, fmt.Errorf("%s: %s", errResp.Errors[0].Title, errResp.Errors[0].Detail)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("failed to list schemas: %s", string(bodyBytes))
	}

	var response struct {
		Data  []DetectionSchema      `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Links map[string]interface{} `json:"links"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, nil, err
	}

	return response.Data, response.Meta, nil
}

func (c *RulesClient) GetSchema(token, id string) (*DetectionSchema, error) {
	path := fmt.Sprintf("/schemas/%s", id)

	resp, err := c.doRequest("GET", path, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp JSONAPIError
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && len(errResp.Errors) > 0 {
			return nil, fmt.Errorf("%s: %s", errResp.Errors[0].Title, errResp.Errors[0].Detail)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get schema: %s", string(bodyBytes))
	}

	var response struct {
		Data DetectionSchema `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (c *RulesClient) CreateSchema(token string, model, view, controller map[string]interface{}) (*DetectionSchema, error) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "detection-schema",
			"attributes": map[string]interface{}{
				"model":      model,
				"view":       view,
				"controller": controller,
			},
		},
	}

	resp, err := c.doRequest("POST", "/schemas", token, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp JSONAPIError
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && len(errResp.Errors) > 0 {
			return nil, fmt.Errorf("%s: %s", errResp.Errors[0].Title, errResp.Errors[0].Detail)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create schema: %s", string(bodyBytes))
	}

	var response struct {
		Data DetectionSchema `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (c *RulesClient) DisableSchema(token, id string) error {
	path := fmt.Sprintf("/schemas/%s/disable", id)

	resp, err := c.doRequest("POST", path, token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp JSONAPIError
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && len(errResp.Errors) > 0 {
			return fmt.Errorf("%s: %s", errResp.Errors[0].Title, errResp.Errors[0].Detail)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to disable schema: %s", string(bodyBytes))
	}

	return nil
}

func (c *RulesClient) EnableSchema(token, id string) error {
	path := fmt.Sprintf("/schemas/%s/enable", id)

	resp, err := c.doRequest("POST", path, token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp JSONAPIError
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && len(errResp.Errors) > 0 {
			return fmt.Errorf("%s: %s", errResp.Errors[0].Title, errResp.Errors[0].Detail)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to enable schema: %s", string(bodyBytes))
	}

	return nil
}

func (c *RulesClient) GetVersionHistory(token, id string) ([]DetectionSchema, error) {
	path := fmt.Sprintf("/schemas/%s/versions", id)

	resp, err := c.doRequest("GET", path, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp JSONAPIError
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && len(errResp.Errors) > 0 {
			return nil, fmt.Errorf("%s: %s", errResp.Errors[0].Title, errResp.Errors[0].Detail)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get version history: %s", string(bodyBytes))
	}

	var response struct {
		Data []DetectionSchema `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Data, nil
}
