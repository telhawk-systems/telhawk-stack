package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type SavedSearch struct {
	ID         string                 `json:"id"`
	VersionID  string                 `json:"version_id"`
	Name       string                 `json:"name"`
	Query      map[string]interface{} `json:"query"`
	IsGlobal   bool                   `json:"is_global"`
	CreatedAt  string                 `json:"created_at"`
	DisabledAt *string                `json:"disabled_at"`
	HiddenAt   *string                `json:"hidden_at"`
}

type jsonAPIResource struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
}

type jsonAPIResponse struct {
	Data interface{} `json:"data"`
}

func (c *QueryClient) SavedSearchCreate(token string, baseURL string, name string, query map[string]interface{}, filters map[string]interface{}, isGlobal bool) (*SavedSearch, error) {
	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "saved-search",
			"attributes": map[string]interface{}{
				"name":      name,
				"query":     query,
				"filters":   filters,
				"is_global": isGlobal,
			},
		},
	}
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", baseURL+"/api/v1/saved-searches", bytes.NewReader(buf))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("create failed: %d", resp.StatusCode)
	}
	var out jsonAPIResponse
	out.Data = &jsonAPIResource{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	res := out.Data.(*jsonAPIResource)
	return savedFromResource(res), nil
}

func (c *QueryClient) SavedSearchList(baseURL string, showAll bool) ([]SavedSearch, error) {
	u, _ := url.Parse(baseURL + "/api/v1/saved-searches")
	if showAll {
		q := u.Query()
		q.Set("filter[show_all]", "true")
		u.RawQuery = q.Encode()
	}
	req, _ := http.NewRequest("GET", u.String(), http.NoBody)
	req.Header.Set("Accept", "application/vnd.api+json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed: %d", resp.StatusCode)
	}
	var out jsonAPIResponse
	out.Data = &[]jsonAPIResource{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	arr := *(out.Data.(*[]jsonAPIResource))
	items := make([]SavedSearch, 0, len(arr))
	for _, r := range arr {
		items = append(items, *savedFromResource(&r))
	}
	return items, nil
}

func (c *QueryClient) SavedSearchGet(baseURL, id string) (*SavedSearch, error) {
	req, _ := http.NewRequest("GET", baseURL+"/api/v1/saved-searches/"+id, http.NoBody)
	req.Header.Set("Accept", "application/vnd.api+json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get failed: %d", resp.StatusCode)
	}
	var out jsonAPIResponse
	out.Data = &jsonAPIResource{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return savedFromResource(out.Data.(*jsonAPIResource)), nil
}

func (c *QueryClient) SavedSearchUpdate(token, baseURL, id string, attrs map[string]interface{}) (*SavedSearch, error) {
	body := map[string]interface{}{
		"data": map[string]interface{}{
			"id":         id,
			"type":       "saved-search",
			"attributes": attrs,
		},
	}
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest("PATCH", baseURL+"/api/v1/saved-searches/"+id, bytes.NewReader(buf))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("update failed: %d", resp.StatusCode)
	}
	var out jsonAPIResponse
	out.Data = &jsonAPIResource{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return savedFromResource(out.Data.(*jsonAPIResource)), nil
}

func (c *QueryClient) SavedSearchAction(token, baseURL, id, action string) (*SavedSearch, error) {
	req, _ := http.NewRequest("POST", baseURL+"/api/v1/saved-searches/"+id+"/"+action, http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.api+json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s failed: %d", action, resp.StatusCode)
	}
	var out jsonAPIResponse
	out.Data = &jsonAPIResource{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return savedFromResource(out.Data.(*jsonAPIResource)), nil
}

func savedFromResource(r *jsonAPIResource) *SavedSearch {
	s := &SavedSearch{ID: r.ID}
	if v, ok := r.Attributes["version_id"].(string); ok {
		s.VersionID = v
	}
	if v, ok := r.Attributes["name"].(string); ok {
		s.Name = v
	}
	if v, ok := r.Attributes["query"].(map[string]interface{}); ok {
		s.Query = v
	}
	if v, ok := r.Attributes["is_global"].(bool); ok {
		s.IsGlobal = v
	}
	if v, ok := r.Attributes["created_at"].(string); ok {
		s.CreatedAt = v
	} else {
		// if server returns timestamp object, best-effort stringify
		if v2, ok := r.Attributes["created_at"].(map[string]interface{}); ok {
			b, _ := json.Marshal(v2)
			s.CreatedAt = string(b)
		}
	}
	if v, ok := r.Attributes["disabled_at"].(string); ok {
		s.DisabledAt = &v
	}
	if v, ok := r.Attributes["hidden_at"].(string); ok {
		s.HiddenAt = &v
	}
	return s
}
