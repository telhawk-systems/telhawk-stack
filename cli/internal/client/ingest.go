package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type IngestClient struct {
	baseURL string
	client  *http.Client
}

func NewIngestClient(baseURL string) *IngestClient {
	return &IngestClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *IngestClient) SendEvent(hecToken string, event map[string]interface{}, source, sourcetype string) error {
	payload := map[string]interface{}{
		"event":      event,
		"source":     source,
		"sourcetype": sourcetype,
		"time":       time.Now().Unix(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/services/collector/event", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Splunk "+hecToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ingest failed with status %d", resp.StatusCode)
	}

	return nil
}
