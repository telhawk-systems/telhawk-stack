package client

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/telhawk-systems/telhawk-stack/common/config"
)

type OpenSearchClient struct {
	client *opensearch.Client
	index  string
}

func NewOpenSearchClient() (*OpenSearchClient, error) {
	cfg := config.GetConfig()
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.OpenSearch.Insecure,
			},
		},
	}

	osCfg := opensearch.Config{
		Addresses: []string{cfg.OpenSearch.URL},
		Username:  cfg.OpenSearch.Username,
		Password:  cfg.OpenSearch.Password,
		Transport: httpClient.Transport,
	}

	client, err := opensearch.NewClient(osCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create opensearch client: %w", err)
	}

	info, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to ping opensearch: %w", err)
	}
	defer info.Body.Close()

	if info.IsError() {
		return nil, fmt.Errorf("opensearch returned error: %s", info.Status())
	}

	return &OpenSearchClient{
		client: client,
		index:  cfg.OpenSearch.Index,
	}, nil
}

func (c *OpenSearchClient) Client() *opensearch.Client {
	return c.client
}

func (c *OpenSearchClient) Index() string {
	return c.index
}
