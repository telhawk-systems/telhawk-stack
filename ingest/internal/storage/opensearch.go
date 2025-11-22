package storage

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchutil"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/storageclient"
)

// Config holds OpenSearch connection and index management configuration
type Config struct {
	URL             string
	Username        string
	Password        string
	TLSSkipVerify   bool
	IndexPrefix     string
	ShardCount      int
	ReplicaCount    int
	RefreshInterval string
	RetentionDays   int
	RolloverSizeGB  int
	RolloverAge     time.Duration
}

// DefaultConfig returns sensible defaults for OpenSearch configuration
func DefaultConfig() Config {
	return Config{
		URL:             "https://localhost:9200",
		Username:        "admin",
		Password:        "admin",
		TLSSkipVerify:   true,
		IndexPrefix:     "telhawk-events",
		ShardCount:      1,
		ReplicaCount:    0,
		RefreshInterval: "5s",
		RetentionDays:   30,
		RolloverSizeGB:  50,
		RolloverAge:     24 * time.Hour,
	}
}

// Client is a direct OpenSearch client that bypasses the storage service
type Client struct {
	osClient    *opensearch.Client
	config      Config
	initialized bool
}

// NewClient creates a new direct OpenSearch client
func NewClient(cfg Config) (*Client, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.TLSSkipVerify,
			},
		},
	}

	osCfg := opensearch.Config{
		Addresses: []string{cfg.URL},
		Username:  cfg.Username,
		Password:  cfg.Password,
		Transport: httpClient.Transport,
	}

	client, err := opensearch.NewClient(osCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create opensearch client: %w", err)
	}

	return &Client{
		osClient: client,
		config:   cfg,
	}, nil
}

// Initialize sets up index templates, ISM policies, and initial indices
func (c *Client) Initialize(ctx context.Context) error {
	if c.initialized {
		return nil
	}

	// Verify connection
	info, err := c.osClient.Info()
	if err != nil {
		return fmt.Errorf("failed to connect to opensearch: %w", err)
	}
	defer info.Body.Close()

	if info.IsError() {
		return fmt.Errorf("opensearch returned error: %s", info.Status())
	}

	log.Println("Connected to OpenSearch successfully")

	if err := c.createIndexTemplate(ctx); err != nil {
		return fmt.Errorf("failed to create index template: %w", err)
	}

	if err := c.createISMPolicy(ctx); err != nil {
		return fmt.Errorf("failed to create ISM policy: %w", err)
	}

	if err := c.createInitialIndex(ctx); err != nil {
		return fmt.Errorf("failed to create initial index: %w", err)
	}

	c.initialized = true
	log.Printf("OpenSearch initialized with index prefix: %s", c.config.IndexPrefix)
	return nil
}

// Ingest implements the storageclient interface for direct OpenSearch indexing
func (c *Client) Ingest(ctx context.Context, events []map[string]interface{}) (*storageclient.IngestResponse, error) {
	if c.osClient == nil {
		return nil, fmt.Errorf("opensearch client not initialized")
	}

	resp := &storageclient.IngestResponse{}
	indexName := c.GetWriteAlias()

	bi, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client: c.osClient,
		Index:  indexName,
	})
	if err != nil {
		resp.Failed = len(events)
		resp.Errors = []string{fmt.Sprintf("Failed to create bulk indexer: %v", err)}
		return resp, nil
	}

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("Failed to marshal event: %v", err))
			continue
		}

		// DEBUG: Log JSON being sent to OpenSearch (controlled by DEBUG_DUMP_JSON env var)
		if os.Getenv("DEBUG_DUMP_JSON") == "true" {
			log.Printf("=== DEBUG: JSON being indexed to OpenSearch ===\n%s\n=== END DEBUG ===", string(data))
		}

		err = bi.Add(ctx, opensearchutil.BulkIndexerItem{
			Action: "index",
			Body:   bytes.NewReader(data),
			OnSuccess: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem) {
				resp.Indexed++
			},
			OnFailure: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem, err error) {
				resp.Failed++
				if err != nil {
					resp.Errors = append(resp.Errors, err.Error())
				} else {
					resp.Errors = append(resp.Errors, fmt.Sprintf("%s: %s", res.Error.Type, res.Error.Reason))
				}
			},
		})

		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("Failed to add to bulk indexer: %v", err))
		}
	}

	if err := bi.Close(ctx); err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("Bulk indexer close error: %v", err))
	}

	return resp, nil
}

// GetWriteAlias returns the write alias for the index
func (c *Client) GetWriteAlias() string {
	return c.config.IndexPrefix + "-write"
}

// GetCurrentWriteIndex returns the current write index name
func (c *Client) GetCurrentWriteIndex() string {
	timestamp := time.Now().Format("2006.01.02")
	return fmt.Sprintf("%s-%s-000001", c.config.IndexPrefix, timestamp)
}

func (c *Client) createIndexTemplate(ctx context.Context) error {
	template := map[string]interface{}{
		"index_patterns": []string{c.config.IndexPrefix + "-*"},
		"template": map[string]interface{}{
			"settings": map[string]interface{}{
				"number_of_shards":   c.config.ShardCount,
				"number_of_replicas": c.config.ReplicaCount,
				"refresh_interval":   c.config.RefreshInterval,
				"codec":              "best_compression",
			},
			"mappings": c.getOCSFMappings(),
		},
		"priority": 100,
	}

	body, err := json.Marshal(template)
	if err != nil {
		return err
	}

	res, err := c.osClient.Indices.PutIndexTemplate(
		c.config.IndexPrefix+"-template",
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

	log.Println("Index template created/updated successfully")
	return nil
}

func (c *Client) getOCSFMappings() map[string]interface{} {
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
						"type":   "date",
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

func (c *Client) createISMPolicy(ctx context.Context) error {
	policy := map[string]interface{}{
		"policy": map[string]interface{}{
			"description":   "TelHawk events index lifecycle policy",
			"default_state": "hot",
			"states": []map[string]interface{}{
				{
					"name": "hot",
					"actions": []map[string]interface{}{
						{
							"rollover": map[string]interface{}{
								"min_size":      fmt.Sprintf("%dGB", c.config.RolloverSizeGB),
								"min_index_age": formatDurationForOpenSearch(c.config.RolloverAge),
							},
						},
					},
					"transitions": []map[string]interface{}{
						{
							"state_name": "delete",
							"conditions": map[string]interface{}{
								"min_index_age": fmt.Sprintf("%dd", c.config.RetentionDays),
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

	policyName := c.config.IndexPrefix + "-policy"

	// Check if policy exists
	req := c.osClient.Transport.Perform
	checkReq, err := http.NewRequestWithContext(ctx, "GET", "/_plugins/_ism/policies/"+policyName, http.NoBody)
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

	log.Println("ISM policy created/updated successfully")
	return nil
}

func (c *Client) createInitialIndex(ctx context.Context) error {
	indexName := c.GetCurrentWriteIndex()
	writeAlias := c.config.IndexPrefix + "-write"

	exists, err := c.osClient.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}
	defer exists.Body.Close()

	if exists.StatusCode == 200 {
		log.Printf("Index %s already exists", indexName)
		return nil
	}

	// Create the index first without the write alias
	res, err := c.osClient.Indices.Create(indexName)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create initial index: %s - %s", res.Status(), string(bodyBytes))
	}

	// Update alias atomically - remove is_write_index from all indices, then add to new one
	aliasActions := map[string]interface{}{
		"actions": []map[string]interface{}{
			{
				"remove": map[string]interface{}{
					"index": c.config.IndexPrefix + "-*",
					"alias": writeAlias,
				},
			},
			{
				"add": map[string]interface{}{
					"index":          indexName,
					"alias":          writeAlias,
					"is_write_index": true,
				},
			},
		},
	}

	body, err := json.Marshal(aliasActions)
	if err != nil {
		return err
	}

	aliasReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"/_aliases",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	aliasReq.Header.Set("Content-Type", "application/json")

	aliasRes, err := c.osClient.Transport.Perform(aliasReq)
	if err != nil {
		return err
	}
	defer aliasRes.Body.Close()

	if aliasRes.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(aliasRes.Body)
		return fmt.Errorf("failed to update write alias: %d - %s", aliasRes.StatusCode, string(bodyBytes))
	}

	log.Printf("Initial index %s created with write alias %s", indexName, writeAlias)
	return nil
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
