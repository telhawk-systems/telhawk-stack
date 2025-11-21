package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestSuccess(t *testing.T) {
	output := captureStdout(func() {
		Success("Test successful")
	})

	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "Test successful")
}

func TestSuccess_WithFormatting(t *testing.T) {
	output := captureStdout(func() {
		Success("Created %d items in %s", 5, "database")
	})

	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "Created 5 items in database")
}

func TestError(t *testing.T) {
	output := captureStderr(func() {
		Error("Test error")
	})

	assert.Contains(t, output, "✗")
	assert.Contains(t, output, "Test error")
}

func TestError_WithFormatting(t *testing.T) {
	output := captureStderr(func() {
		Error("Failed to connect to %s on port %d", "server", 8080)
	})

	assert.Contains(t, output, "✗")
	assert.Contains(t, output, "Failed to connect to server on port 8080")
}

func TestInfo(t *testing.T) {
	output := captureStdout(func() {
		Info("Information message")
	})

	assert.Contains(t, output, "Information message")
	assert.NotContains(t, output, "✓") // Info doesn't have checkmark
	assert.NotContains(t, output, "✗")
}

func TestInfo_WithFormatting(t *testing.T) {
	output := captureStdout(func() {
		Info("Processing %d of %d files", 5, 10)
	})

	assert.Contains(t, output, "Processing 5 of 10 files")
}

func TestWarn(t *testing.T) {
	output := captureStdout(func() {
		Warn("Warning message")
	})

	assert.Contains(t, output, "⚠")
	assert.Contains(t, output, "Warning message")
}

func TestWarn_WithFormatting(t *testing.T) {
	output := captureStdout(func() {
		Warn("Disk usage is %d%%", 95)
	})

	assert.Contains(t, output, "⚠")
	assert.Contains(t, output, "Disk usage is 95%")
}

func TestJSON_Simple(t *testing.T) {
	data := map[string]interface{}{
		"name":  "test",
		"count": 42,
	}

	output := captureStdout(func() {
		err := JSON(data)
		assert.NoError(t, err)
	})

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test", parsed["name"])
	assert.Equal(t, float64(42), parsed["count"])
}

func TestJSON_Indented(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "alice",
			"id":   123,
		},
	}

	output := captureStdout(func() {
		err := JSON(data)
		assert.NoError(t, err)
	})

	// Check for indentation (2 spaces)
	assert.Contains(t, output, "  \"user\":")
	assert.Contains(t, output, "    \"name\":")
}

func TestJSON_Array(t *testing.T) {
	data := []string{"one", "two", "three"}

	output := captureStdout(func() {
		err := JSON(data)
		assert.NoError(t, err)
	})

	var parsed []string
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, []string{"one", "two", "three"}, parsed)
}

func TestJSON_Struct(t *testing.T) {
	type TestStruct struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
		Count   int    `json:"count"`
	}

	data := TestStruct{
		Name:    "test-item",
		Enabled: true,
		Count:   100,
	}

	output := captureStdout(func() {
		err := JSON(data)
		assert.NoError(t, err)
	})

	var parsed TestStruct
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test-item", parsed.Name)
	assert.True(t, parsed.Enabled)
	assert.Equal(t, 100, parsed.Count)
}

func TestNewTable(t *testing.T) {
	headers := []string{"Name", "Age", "City"}
	table := NewTable(headers)

	assert.NotNil(t, table)
	assert.Equal(t, headers, table.headers)
	assert.Empty(t, table.rows)
}

func TestTable_AddRow(t *testing.T) {
	table := NewTable([]string{"Col1", "Col2"})

	table.AddRow([]string{"val1", "val2"})
	table.AddRow([]string{"val3", "val4"})

	assert.Len(t, table.rows, 2)
	assert.Equal(t, []string{"val1", "val2"}, table.rows[0])
	assert.Equal(t, []string{"val3", "val4"}, table.rows[1])
}

func TestTable_Render_Empty(t *testing.T) {
	table := NewTable([]string{"Name", "Status"})

	output := captureStdout(func() {
		table.Render()
	})

	// Should have header and separator
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "----") // Separator
}

func TestTable_Render_WithRows(t *testing.T) {
	table := NewTable([]string{"Name", "Age", "City"})
	table.AddRow([]string{"Alice", "30", "NYC"})
	table.AddRow([]string{"Bob", "25", "SF"})

	output := captureStdout(func() {
		table.Render()
	})

	// Check headers
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Age")
	assert.Contains(t, output, "City")

	// Check separator
	assert.Contains(t, output, "----")

	// Check data rows
	assert.Contains(t, output, "Alice")
	assert.Contains(t, output, "30")
	assert.Contains(t, output, "NYC")
	assert.Contains(t, output, "Bob")
	assert.Contains(t, output, "25")
	assert.Contains(t, output, "SF")
}

func TestTable_Render_ColumnAlignment(t *testing.T) {
	table := NewTable([]string{"Short", "VeryLongHeader"})
	table.AddRow([]string{"A", "B"})
	table.AddRow([]string{"LongValue", "C"})

	output := captureStdout(func() {
		table.Render()
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.GreaterOrEqual(t, len(lines), 4) // Header, separator, 2 rows

	// Header line should have proper spacing
	headerLine := lines[0]
	assert.Contains(t, headerLine, "Short")
	assert.Contains(t, headerLine, "VeryLongHeader")

	// Separator should match column widths
	separatorLine := lines[1]
	assert.Contains(t, separatorLine, "----")

	// Data rows should align
	row1 := lines[2]
	row2 := lines[3]
	assert.Contains(t, row1, "A")
	assert.Contains(t, row1, "B")
	assert.Contains(t, row2, "LongValue")
	assert.Contains(t, row2, "C")
}

func TestTable_Render_VariableWidthColumns(t *testing.T) {
	table := NewTable([]string{"ID", "Description"})
	table.AddRow([]string{"1", "Short"})
	table.AddRow([]string{"2", "This is a much longer description"})
	table.AddRow([]string{"100", "Medium"})

	output := captureStdout(func() {
		table.Render()
	})

	// Should contain all values
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "Short")
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "This is a much longer description")
	assert.Contains(t, output, "100")
	assert.Contains(t, output, "Medium")
}

func TestTable_Render_ManyColumns(t *testing.T) {
	table := NewTable([]string{"A", "B", "C", "D", "E"})
	table.AddRow([]string{"1", "2", "3", "4", "5"})
	table.AddRow([]string{"a", "b", "c", "d", "e"})

	output := captureStdout(func() {
		table.Render()
	})

	// All columns should be present
	for _, col := range []string{"A", "B", "C", "D", "E"} {
		assert.Contains(t, output, col)
	}

	// All data should be present
	for _, val := range []string{"1", "2", "3", "4", "5", "a", "b", "c", "d", "e"} {
		assert.Contains(t, output, val)
	}
}

func TestTable_Render_SingleColumn(t *testing.T) {
	table := NewTable([]string{"Status"})
	table.AddRow([]string{"OK"})
	table.AddRow([]string{"ERROR"})

	output := captureStdout(func() {
		table.Render()
	})

	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "OK")
	assert.Contains(t, output, "ERROR")
}

func TestTable_Render_EmptyStrings(t *testing.T) {
	table := NewTable([]string{"Name", "Value"})
	table.AddRow([]string{"test", ""})
	table.AddRow([]string{"", "value"})

	output := captureStdout(func() {
		table.Render()
	})

	// Should handle empty strings gracefully
	assert.Contains(t, output, "test")
	assert.Contains(t, output, "value")
}
