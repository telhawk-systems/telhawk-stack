package color

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	c := New(FgRed, Bold)
	assert.NotNil(t, c)
	assert.Equal(t, []int{FgRed, Bold}, c.params)
}

func TestNew_NoParams(t *testing.T) {
	c := New()
	assert.NotNil(t, c)
	assert.Empty(t, c.params)
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		params   []int
		expected string
	}{
		{
			name:     "single color",
			params:   []int{FgRed},
			expected: "\033[31m",
		},
		{
			name:     "color with bold",
			params:   []int{FgGreen, Bold},
			expected: "\033[32;1m",
		},
		{
			name:     "multiple attributes",
			params:   []int{FgYellow, Bold, Underline},
			expected: "\033[33;1;4m",
		},
		{
			name:     "no params",
			params:   []int{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.params...)
			result := c.format()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSprint(t *testing.T) {
	c := New(FgRed)
	result := c.Sprint("Hello", " ", "World")

	assert.Contains(t, result, "Hello World")
	assert.Contains(t, result, "\033[31m") // Red color code
	assert.Contains(t, result, reset)      // Reset code
}

func TestSprintf(t *testing.T) {
	c := New(FgGreen, Bold)
	result := c.Sprintf("User %s has %d points", "Alice", 100)

	assert.Contains(t, result, "User Alice has 100 points")
	assert.Contains(t, result, "\033[32;1m") // Green + Bold
	assert.Contains(t, result, reset)
}

func TestSprintln(t *testing.T) {
	c := New(FgCyan)
	result := c.Sprintln("Test message")

	assert.Contains(t, result, "Test message")
	assert.Contains(t, result, "\n")
	assert.Contains(t, result, "\033[36m") // Cyan
	assert.Contains(t, result, reset)
}

func TestPrintf(t *testing.T) {
	c := New(FgYellow)

	// Can't easily test Printf since it writes to stdout
	// but we can test that it doesn't panic
	assert.NotPanics(t, func() {
		c.Printf("Test %s", "message")
	})
}

func TestFprintf(t *testing.T) {
	var buf bytes.Buffer
	c := New(FgMagenta, Bold)

	c.Fprintf(&buf, "Formatted %s: %d", "value", 42)

	output := buf.String()
	assert.Contains(t, output, "Formatted value: 42")
	assert.Contains(t, output, "\033[35;1m") // Magenta + Bold
	assert.Contains(t, output, reset)
}

func TestColorCodes(t *testing.T) {
	tests := []struct {
		name  string
		code  int
		color string
	}{
		{"FgBlack", FgBlack, "\033[30m"},
		{"FgRed", FgRed, "\033[31m"},
		{"FgGreen", FgGreen, "\033[32m"},
		{"FgYellow", FgYellow, "\033[33m"},
		{"FgBlue", FgBlue, "\033[34m"},
		{"FgMagenta", FgMagenta, "\033[35m"},
		{"FgCyan", FgCyan, "\033[36m"},
		{"FgWhite", FgWhite, "\033[37m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.code)
			assert.Equal(t, tt.color, c.format())
		})
	}
}

func TestAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attr     int
		expected string
	}{
		{"Bold", Bold, "\033[1m"},
		{"Dim", Dim, "\033[2m"},
		{"Underline", Underline, "\033[4m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.attr)
			assert.Equal(t, tt.expected, c.format())
		})
	}
}

func TestCombinedAttributes(t *testing.T) {
	c := New(FgRed, Bold, Underline)
	result := c.format()

	assert.Equal(t, "\033[31;1;4m", result)
}

func TestSprint_NoColor(t *testing.T) {
	c := New()
	result := c.Sprint("Plain text")

	// With no params, format() returns empty string, so only reset is added
	assert.Contains(t, result, "Plain text")
	assert.Contains(t, result, reset)
}

func TestFprintf_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	red := New(FgRed)
	green := New(FgGreen)

	red.Fprintf(&buf, "Error: ")
	green.Fprintf(&buf, "Success")

	output := buf.String()
	assert.Contains(t, output, "Error: ")
	assert.Contains(t, output, "Success")
	assert.Contains(t, output, "\033[31m") // Red
	assert.Contains(t, output, "\033[32m") // Green
}

func TestOutput(t *testing.T) {
	// Test default behavior
	NoColor = false
	writer := Output()
	assert.NotNil(t, writer)

	// Test with NoColor enabled
	NoColor = true
	writer = Output()
	assert.NotNil(t, writer)

	// Reset
	NoColor = false
}

func TestReset(t *testing.T) {
	assert.Equal(t, "\033[0m", reset)
}

func TestFormat_EmptyParams(t *testing.T) {
	c := &Color{params: []int{}}
	result := c.format()
	assert.Empty(t, result)
}

func TestSprintf_EmptyFormat(t *testing.T) {
	c := New(FgBlue)
	result := c.Sprintf("")

	assert.Equal(t, "\033[34m"+reset, result)
}

func TestSprint_EmptyArgs(t *testing.T) {
	c := New(FgCyan)
	result := c.Sprint()

	assert.Contains(t, result, "\033[36m")
	assert.Contains(t, result, reset)
}
