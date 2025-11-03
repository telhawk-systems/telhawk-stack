package client

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/config"
)

type OpenSearchClient struct {
	client *opensearch.Client
}

func NewOpenSearchClient(cfg config.OpenSearchConfig) (*OpenSearchClient, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.Insecure,
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

	info, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to ping opensearch: %w", err)
	}
	defer info.Body.Close()

	if info.IsError() {
		return nil, fmt.Errorf("opensearch returned error: %s", info.Status())
	}

	return &OpenSearchClient{client: client}, nil
}

func (c *OpenSearchClient) Client() *opensearch.Client {
	return c.client
}
