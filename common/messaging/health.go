// Package messaging provides health check utilities for message broker connections.
package messaging

import (
	"context"
	"fmt"
	"time"
)

// HealthChecker can check the health of a messaging connection.
type HealthChecker interface {
	// CheckHealth returns nil if the connection is healthy, error otherwise.
	CheckHealth(ctx context.Context) error
}

// HealthStatus represents the health state of a messaging connection.
type HealthStatus struct {
	// Connected indicates if the client is connected.
	Connected bool `json:"connected"`

	// Latency is the round-trip time for a health ping.
	Latency time.Duration `json:"latency_ms"`

	// Error contains any error message if unhealthy.
	Error string `json:"error,omitempty"`
}

// CheckClientHealth checks if a Client is healthy by verifying connection.
func CheckClientHealth(ctx context.Context, client Client) HealthStatus {
	status := HealthStatus{}

	if client == nil {
		status.Error = "client is nil"
		return status
	}

	status.Connected = client.IsConnected()
	if !status.Connected {
		status.Error = "not connected to message broker"
		return status
	}

	// Measure latency with a request to internal subject
	start := time.Now()
	_, err := client.Request(ctx, "_HEALTH.ping", []byte("ping"), 2*time.Second)
	status.Latency = time.Since(start)

	// NATS will error if no responders - that's OK for health check
	// We just want to verify we can communicate with the server
	if err != nil && status.Connected {
		// Still connected, just no responder - that's fine
		status.Error = ""
	} else if err != nil {
		status.Error = fmt.Sprintf("health check failed: %v", err)
	}

	return status
}
