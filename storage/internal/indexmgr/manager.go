package indexmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/telhawk-systems/telhawk-stack/storage/internal/client"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/config"
)

type IndexManager struct {
	client *client.OpenSearchClient
	config config.IndexManagementConfig
}

func NewIndexManager(client *client.OpenSearchClient, cfg config.IndexManagementConfig) *IndexManager {
	return &IndexManager{
		client: client,
		config: cfg,
	}
}

func (m *IndexManager) Initialize(ctx context.Context) error {
	if err := m.createIndexTemplate(ctx); err != nil {
		return fmt.Errorf("failed to create index template: %w", err)
	}

	if err := m.createISMPolicy(ctx); err != nil {
		return fmt.Errorf("failed to create ISM policy: %w", err)
	}

	if err := m.createInitialIndex(ctx); err != nil {
		return fmt.Errorf("failed to create initial index: %w", err)
	}

	return nil
}

func (m *IndexManager) createIndexTemplate(ctx context.Context) error {
	template := map[string]interface{}{
		"index_patterns": []string{m.config.IndexPrefix + "-*"},
		"template": map[string]interface{}{
			"settings": map[string]interface{}{
				"number_of_shards":   m.config.ShardCount,
				"number_of_replicas": m.config.ReplicaCount,
				"refresh_interval":   m.config.RefreshInterval,
				"codec":              "best_compression",
			},
			"mappings": m.getOCSFMappings(),
		},
		"priority": 100,
	}

	body, err := json.Marshal(template)
	if err != nil {
		return err
	}

	res, err := m.client.Client().Indices.PutIndexTemplate(
		m.config.IndexPrefix+"-template",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create index template: %s - %s", res.Status(), string(bodyBytes))
	}

	return nil
}

func (m *IndexManager) getOCSFMappings() map[string]interface{} {
	return map[string]interface{}{
		"dynamic": true,
		"dynamic_templates": []map[string]interface{}{
			{
				"strings_as_keywords": map[string]interface{}{
					"match_mapping_type": "string",
					"mapping": map[string]interface{}{
						"type": "text",
						"fields": map[string]interface{}{
							"keyword": map[string]interface{}{
								"type":         "keyword",
								"ignore_above": 256,
							},
						},
					},
				},
			},
		},
		"properties": map[string]interface{}{
			"time": map[string]interface{}{
				"type": "date",
			},
			"@timestamp": map[string]interface{}{
				"type": "date",
			},
			"metadata": map[string]interface{}{
				"properties": map[string]interface{}{
					"version": map[string]interface{}{
						"type": "keyword",
					},
					"product": map[string]interface{}{
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type": "keyword",
							},
							"vendor_name": map[string]interface{}{
								"type": "keyword",
							},
							"version": map[string]interface{}{
								"type": "keyword",
							},
						},
					},
					"log_name": map[string]interface{}{
						"type": "keyword",
					},
					"log_provider": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"class_uid": map[string]interface{}{
				"type": "integer",
			},
			"class_name": map[string]interface{}{
				"type": "keyword",
			},
			"category_uid": map[string]interface{}{
				"type": "integer",
			},
			"category_name": map[string]interface{}{
				"type": "keyword",
			},
			"activity_id": map[string]interface{}{
				"type": "integer",
			},
			"activity_name": map[string]interface{}{
				"type": "keyword",
			},
			"type_uid": map[string]interface{}{
				"type": "long",
			},
			"severity": map[string]interface{}{
				"type": "keyword",
			},
			"severity_id": map[string]interface{}{
				"type": "integer",
			},
			"status": map[string]interface{}{
				"type": "keyword",
			},
			"status_id": map[string]interface{}{
				"type": "integer",
			},
			"status_detail": map[string]interface{}{
				"type": "text",
			},
			"message": map[string]interface{}{
				"type": "text",
			},
			// User object (used in auth, process, file events)
			"user": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "keyword",
					},
					"uid": map[string]interface{}{
						"type": "keyword",
					},
					"email": map[string]interface{}{
						"type": "keyword",
					},
					"domain": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			// Actor object (used in process, file events)
			"actor": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user": map[string]interface{}{
						"type": "object",
					},
					"process": map[string]interface{}{
						"type": "object",
					},
				},
			},
			// Process object
			"process": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pid": map[string]interface{}{
						"type": "integer",
					},
					"name": map[string]interface{}{
						"type": "keyword",
					},
					"cmd_line": map[string]interface{}{
						"type": "text",
					},
					"uid": map[string]interface{}{
						"type": "keyword",
					},
					"parent_process": map[string]interface{}{
						"type": "object",
					},
				},
			},
			// File object
			"file": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type": "keyword",
					},
					"name": map[string]interface{}{
						"type": "keyword",
					},
					"type": map[string]interface{}{
						"type": "keyword",
					},
					"size": map[string]interface{}{
						"type": "long",
					},
					"modified_time": map[string]interface{}{
						"type": "date",
						"format": "epoch_second||strict_date_optional_time||yyyy-MM-dd HH:mm:ss",
					},
				},
			},
			// Endpoint objects (for network, auth)
			"src_endpoint": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ip": map[string]interface{}{
						"type": "ip",
					},
					"port": map[string]interface{}{
						"type": "integer",
					},
					"hostname": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"dst_endpoint": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ip": map[string]interface{}{
						"type": "ip",
					},
					"port": map[string]interface{}{
						"type": "integer",
					},
					"hostname": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			// Network connection info
			"connection_info": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"protocol_name": map[string]interface{}{
						"type": "keyword",
					},
					"direction": map[string]interface{}{
						"type": "keyword",
					},
					"boundary": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"traffic": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"bytes": map[string]interface{}{
						"type": "long",
					},
					"packets": map[string]interface{}{
						"type": "long",
					},
				},
			},
			// DNS objects
			"query": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"hostname": map[string]interface{}{
						"type": "keyword",
					},
					"type": map[string]interface{}{
						"type": "keyword",
					},
					"class": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"answers": map[string]interface{}{
				"type": "nested",
			},
			"response_code": map[string]interface{}{
				"type": "keyword",
			},
			// HTTP objects
			"http_request": map[string]interface{}{
				"type": "object",
			},
			"http_response": map[string]interface{}{
				"type": "object",
			},
			// Detection objects
			"finding": map[string]interface{}{
				"type": "object",
			},
			"attacks": map[string]interface{}{
				"type": "nested",
			},
			"resources": map[string]interface{}{
				"type": "nested",
			},
			// Auth protocol
			"auth_protocol": map[string]interface{}{
				"type": "keyword",
			},
			// Properties (source_type, etc)
			"properties": map[string]interface{}{
				"type": "object",
			},
			// Raw data
			"raw": map[string]interface{}{
				"type":    "object",
				"enabled": false,
			},
			"raw_data": map[string]interface{}{
				"type":  "text",
				"index": false,
			},
		},
	}
}

func (m *IndexManager) createISMPolicy(ctx context.Context) error {
	policy := map[string]interface{}{
		"policy": map[string]interface{}{
			"description": "TelHawk events index lifecycle policy",
			"default_state": "hot",
			"states": []map[string]interface{}{
				{
					"name": "hot",
					"actions": []map[string]interface{}{
						{
							"rollover": map[string]interface{}{
								"min_size":        fmt.Sprintf("%dGB", m.config.RolloverSizeGB),
								"min_index_age":   formatDurationForOpenSearch(m.config.RolloverAge),
							},
						},
					},
					"transitions": []map[string]interface{}{
						{
							"state_name": "delete",
							"conditions": map[string]interface{}{
								"min_index_age": fmt.Sprintf("%dd", m.config.RetentionDays),
							},
						},
					},
				},
				{
					"name": "delete",
					"actions": []map[string]interface{}{
						{
							"delete": map[string]interface{}{},
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	policyName := m.config.IndexPrefix + "-policy"
	
	// Check if policy exists
	req := m.client.Client().Transport.Perform
	checkReq, err := http.NewRequestWithContext(
		ctx,
		"GET",
		"/_plugins/_ism/policies/"+policyName,
		nil,
	)
	if err != nil {
		return err
	}
	
	checkRes, err := req(checkReq)
	if err != nil {
		return err
	}
	checkRes.Body.Close()
	
	// If policy exists (200), update it. Otherwise create it (404)
	method := "PUT"
	url := "/_plugins/_ism/policies/" + policyName
	if checkRes.StatusCode == 200 {
		// Policy exists, update with sequence number
		url += "?if_seq_no=1&if_primary_term=1"
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		method,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := req(httpReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Accept 200 (updated), 201 (created), or 409 (already exists with same content)
	if res.StatusCode >= 400 && res.StatusCode != 409 {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create ISM policy: %d - %s", res.StatusCode, string(bodyBytes))
	}

	return nil
}

func (m *IndexManager) createInitialIndex(ctx context.Context) error {
	indexName := m.GetCurrentWriteIndex()

	exists, err := m.client.Client().Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}
	defer exists.Body.Close()

	if exists.StatusCode == 200 {
		return nil
	}

	settings := map[string]interface{}{
		"aliases": map[string]interface{}{
			m.config.IndexPrefix + "-write": map[string]interface{}{
				"is_write_index": true,
			},
		},
	}

	body, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	res, err := m.client.Client().Indices.Create(
		indexName,
		m.client.Client().Indices.Create.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create initial index: %s - %s", res.Status(), string(bodyBytes))
	}

	return nil
}

func (m *IndexManager) GetCurrentWriteIndex() string {
	timestamp := time.Now().Format("2006.01.02")
	return fmt.Sprintf("%s-%s-000001", m.config.IndexPrefix, timestamp)
}

func (m *IndexManager) GetWriteAlias() string {
	return m.config.IndexPrefix + "-write"
}

func (m *IndexManager) ResolveIndexPattern(classUID int) string {
	return m.config.IndexPrefix + "-*"
}

// formatDurationForOpenSearch converts Go duration to OpenSearch-compatible format
// OpenSearch expects simple durations like "24h", "7d", not Go's "24h0m0s" format
func formatDurationForOpenSearch(d time.Duration) string {
	hours := int(d.Hours())
	if hours%24 == 0 {
		days := hours / 24
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dh", hours)
}
