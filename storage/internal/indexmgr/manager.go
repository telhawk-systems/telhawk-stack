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
		"properties": map[string]interface{}{
			"time": map[string]interface{}{
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
			"message": map[string]interface{}{
				"type": "text",
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
								"min_index_age":   m.config.RolloverAge.String(),
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

	req := m.client.Client().Transport.Perform
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"PUT",
		"/_plugins/_ism/policies/"+m.config.IndexPrefix+"-policy",
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

	if res.StatusCode >= 400 && res.StatusCode != 404 {
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
