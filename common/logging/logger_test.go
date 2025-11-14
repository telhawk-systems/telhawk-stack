package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/telhawk-systems/telhawk-stack/common/middleware"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		level  slog.Level
		format string
	}{
		{
			name:   "json format with info level",
			level:  slog.LevelInfo,
			format: "json",
		},
		{
			name:   "text format with debug level",
			level:  slog.LevelDebug,
			format: "text",
		},
		{
			name:   "default format (json) with error level",
			level:  slog.LevelError,
			format: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.level, tt.format)
			if logger == nil {
				t.Fatal("expected non-nil logger")
			}
			if logger.Logger == nil {
				t.Fatal("expected non-nil underlying logger")
			}
		})
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if logger.Logger == nil {
		t.Fatal("expected non-nil underlying logger")
	}
}

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := &Logger{Logger: slog.New(handler)}

	tests := []struct {
		name        string
		ctx         context.Context
		expectReqID bool
		reqID       string
	}{
		{
			name:        "context with request ID",
			ctx:         context.WithValue(context.Background(), middleware.RequestIDKey, "test-req-123"),
			expectReqID: true,
			reqID:       "test-req-123",
		},
		{
			name:        "context without request ID",
			ctx:         context.Background(),
			expectReqID: false,
			reqID:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			contextLogger := logger.WithContext(tt.ctx)
			contextLogger.Info("test message")

			if tt.expectReqID {
				if !strings.Contains(buf.String(), "test-req-123") {
					t.Errorf("expected request ID in log output, got: %s", buf.String())
				}
				if !strings.Contains(buf.String(), "request_id") {
					t.Errorf("expected 'request_id' field in log output, got: %s", buf.String())
				}
			}
		})
	}
}

func TestInfoContext(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := &Logger{Logger: slog.New(handler)}

	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "info-test-123")
	logger.InfoContext(ctx, "test info message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test info message") {
		t.Errorf("expected message in output, got: %s", output)
	}
	if !strings.Contains(output, "info-test-123") {
		t.Errorf("expected request ID in output, got: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("expected INFO level in output, got: %s", output)
	}
}

func TestWarnContext(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})
	logger := &Logger{Logger: slog.New(handler)}

	ctx := context.Background()
	logger.WarnContext(ctx, "test warn message")

	output := buf.String()
	if !strings.Contains(output, "test warn message") {
		t.Errorf("expected message in output, got: %s", output)
	}
	if !strings.Contains(output, "WARN") {
		t.Errorf("expected WARN level in output, got: %s", output)
	}
}

func TestErrorContext(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	})
	logger := &Logger{Logger: slog.New(handler)}

	ctx := context.Background()
	logger.ErrorContext(ctx, "test error message", "error", "something went wrong")

	output := buf.String()
	if !strings.Contains(output, "test error message") {
		t.Errorf("expected message in output, got: %s", output)
	}
	if !strings.Contains(output, "ERROR") {
		t.Errorf("expected ERROR level in output, got: %s", output)
	}
}

func TestDebugContext(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := &Logger{Logger: slog.New(handler)}

	ctx := context.Background()
	logger.DebugContext(ctx, "test debug message")

	output := buf.String()
	if !strings.Contains(output, "test debug message") {
		t.Errorf("expected message in output, got: %s", output)
	}
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("expected DEBUG level in output, got: %s", output)
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := &Logger{Logger: slog.New(handler)}

	enrichedLogger := logger.With("service", "test-service", "version", "1.0")
	if enrichedLogger == nil {
		t.Fatal("expected non-nil logger from With()")
	}

	enrichedLogger.Info("test message")
	output := buf.String()

	if !strings.Contains(output, "test-service") {
		t.Errorf("expected service field in output, got: %s", output)
	}
	if !strings.Contains(output, "1.0") {
		t.Errorf("expected version field in output, got: %s", output)
	}
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := &Logger{Logger: slog.New(handler)}

	groupedLogger := logger.WithGroup("metrics")
	if groupedLogger == nil {
		t.Fatal("expected non-nil logger from WithGroup()")
	}

	groupedLogger.Info("test message", "count", 42)
	output := buf.String()

	// Verify the output contains the group
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err == nil {
		if _, ok := logEntry["metrics"]; !ok {
			t.Errorf("expected 'metrics' group in output, got: %s", output)
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{
			name:     "debug level",
			input:    "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "info level",
			input:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "warn level",
			input:    "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "error level",
			input:    "error",
			expected: slog.LevelError,
		},
		{
			name:     "invalid level defaults to info",
			input:    "invalid",
			expected: slog.LevelInfo,
		},
		{
			name:     "empty string defaults to info",
			input:    "",
			expected: slog.LevelInfo,
		},
		{
			name:     "uppercase DEBUG",
			input:    "DEBUG",
			expected: slog.LevelInfo, // Case sensitive, defaults to info
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSetDefault(t *testing.T) {
	// Save original default
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

	logger := New(slog.LevelInfo, "json")
	SetDefault(logger)

	// Verify slog.Default() returns our logger
	if slog.Default() != logger.Logger {
		t.Error("SetDefault did not update slog.Default()")
	}
}
