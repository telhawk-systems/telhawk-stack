package coreclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
)

// Client communicates with the core normalization service.
type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

// New constructs a new Client.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: 3,
		retryDelay: 100 * time.Millisecond,
	}
}

// NormalizationResult represents the core response.
type NormalizationResult struct {
	Event json.RawMessage `json:"event"`
}

// Normalize sends the event to core for normalization with retry logic.
func (c *Client) Normalize(ctx context.Context, event *models.Event) (*NormalizationResult, error) {
	if c == nil {
		return nil, fmt.Errorf("core client not configured")
	}

	payload := event.Raw
	if len(payload) == 0 {
		buf, err := json.Marshal(event.Event)
		if err != nil {
			return nil, fmt.Errorf("serialize event: %w", err)
		}
		payload = buf
	}

	reqBody := map[string]interface{}{
		"id":          event.ID,
		"source":      event.Source,
		"source_type": event.SourceType,
		"format":      "json",
		"payload":     base64.StdEncoding.EncodeToString(payload),
		"received_at": event.Timestamp.UTC().Format(time.RFC3339Nano),
		"attributes": map[string]string{
			"host":         event.Host,
			"hec_token_id": event.HECTokenID,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := c.retryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := c.attemptNormalize(ctx, bodyBytes)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) attemptNormalize(ctx context.Context, bodyBytes []byte) (*NormalizationResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/normalize", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, &retryableError{err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		// Server errors are retryable
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, &retryableError{err: fmt.Errorf("core response status %d: %s", resp.StatusCode, errBody["message"])}
	}

	if resp.StatusCode == 429 {
		// Rate limit is retryable
		return nil, &retryableError{err: fmt.Errorf("rate limited by core service")}
	}

	if resp.StatusCode != http.StatusOK {
		// Client errors (4xx except 429) are not retryable
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("core response status %d: %s", resp.StatusCode, errBody["message"])
	}

	var result NormalizationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// retryableError marks an error as retryable.
type retryableError struct {
	err error
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) Unwrap() error {
	return e.err
}

func isRetryable(err error) bool {
	retryableError := &retryableError{}
	ok := errors.As(err, &retryableError)
	return ok
}
