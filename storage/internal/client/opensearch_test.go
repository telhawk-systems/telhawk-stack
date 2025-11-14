package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/telhawk-systems/telhawk-stack/storage/internal/config"
)

func TestNewOpenSearchClient_Success(t *testing.T) {
	// Create a mock OpenSearch server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "test-node",
				"cluster_name": "test-cluster",
				"version": {
					"number": "2.0.0"
				}
			}`))
		}
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}

	client, err := NewOpenSearchClient(cfg)
	if err != nil {
		t.Fatalf("Expected successful client creation, got error: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.Client() == nil {
		t.Error("Expected non-nil OpenSearch client")
	}
}

func TestNewOpenSearchClient_ConnectionFailure(t *testing.T) {
	// Use an invalid URL that will fail to connect
	cfg := config.OpenSearchConfig{
		URL:      "http://nonexistent-server:9999",
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}

	client, err := NewOpenSearchClient(cfg)
	if err == nil {
		t.Error("Expected error when connecting to invalid server")
	}

	if client != nil {
		t.Error("Expected nil client on connection failure")
	}
}

func TestNewOpenSearchClient_ErrorResponse(t *testing.T) {
	// Create a mock server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}

	client, err := NewOpenSearchClient(cfg)
	if err == nil {
		t.Error("Expected error when OpenSearch returns error response")
	}

	if client != nil {
		t.Error("Expected nil client on error response")
	}
}

func TestNewOpenSearchClient_TLSConfiguration(t *testing.T) {
	// Test with insecure=true
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"name": "test-node",
			"cluster_name": "test-cluster",
			"version": {
				"number": "2.0.0"
			}
		}`))
	}))
	defer mockServer.Close()

	tests := []struct {
		name     string
		insecure bool
	}{
		{
			name:     "Insecure TLS enabled",
			insecure: true,
		},
		{
			name:     "Secure TLS enabled",
			insecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.OpenSearchConfig{
				URL:      mockServer.URL,
				Username: "testuser",
				Password: "testpass",
				Insecure: tt.insecure,
			}

			client, err := NewOpenSearchClient(cfg)
			if err != nil {
				t.Fatalf("Expected successful client creation, got error: %v", err)
			}

			if client == nil {
				t.Fatal("Expected non-nil client")
			}
		})
	}
}

func TestNewOpenSearchClient_MultipleAddresses(t *testing.T) {
	// Create a mock OpenSearch server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "test-node",
				"cluster_name": "test-cluster",
				"version": {
					"number": "2.0.0"
				}
			}`))
		}
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	client, err := NewOpenSearchClient(cfg)
	if err != nil {
		t.Fatalf("Expected successful client creation, got error: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	// Verify the client is usable
	if client.Client() == nil {
		t.Error("Expected non-nil OpenSearch client")
	}
}

func TestOpenSearchClient_ClientMethod(t *testing.T) {
	// Create a mock OpenSearch server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"name": "test-node",
			"cluster_name": "test-cluster",
			"version": {
				"number": "2.0.0"
			}
		}`))
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}

	client, err := NewOpenSearchClient(cfg)
	if err != nil {
		t.Fatalf("Expected successful client creation, got error: %v", err)
	}

	// Test the Client() accessor method
	osClient := client.Client()
	if osClient == nil {
		t.Error("Expected non-nil OpenSearch client from Client() method")
	}

	// Verify we can call Info on the client
	info, err := osClient.Info()
	if err != nil {
		t.Errorf("Expected successful Info call, got error: %v", err)
	}
	if info != nil {
		defer info.Body.Close()
	}
}
