package color

import (
	"fmt"
	"io"
	"os"
)

// ANSI color codes
const (
	reset = "\033[0m"

	// Foreground colors
	FgBlack   = 30
	FgRed     = 31
	FgGreen   = 32
	FgYellow  = 33
	FgBlue    = 34
	FgMagenta = 35
	FgCyan    = 36
	FgWhite   = 37

	// Attributes
	Bold      = 1
	Dim       = 2
	Underline = 4
)

// Color represents a text color configuration
type Color struct {
	params []int
}

// New creates a new Color with the given attributes
func New(attrs ...int) *Color {
	return &Color{params: attrs}
}

// format returns the ANSI escape sequence for this color
func (c *Color) format() string {
	if len(c.params) == 0 {
		return ""
	}

	// Build escape sequence: \033[<attr1>;<attr2>m
	seq := "\033["
	for i, param := range c.params {
		if i > 0 {
			seq += ";"
		}
		seq += fmt.Sprintf("%d", param)
	}
	seq += "m"
	return seq
}

// Printf prints formatted output with color to stdout
func (c *Color) Printf(format string, a ...interface{}) {
	fmt.Printf(c.format()+format+reset, a...)
}

// Fprintf prints formatted output with color to the given writer
func (c *Color) Fprintf(w io.Writer, format string, a ...interface{}) {
	fmt.Fprintf(w, c.format()+format+reset, a...)
}

// Sprint returns a colored string
func (c *Color) Sprint(a ...interface{}) string {
	return c.format() + fmt.Sprint(a...) + reset
}

// Sprintf returns a formatted colored string
func (c *Color) Sprintf(format string, a ...interface{}) string {
	return c.format() + fmt.Sprintf(format, a...) + reset
}

// Sprintln returns a colored string with newline
func (c *Color) Sprintln(a ...interface{}) string {
	return c.format() + fmt.Sprintln(a...) + reset
}

// NoColor disables color output (useful for testing or when piping)
var NoColor = false

// Output returns the appropriate writer based on NoColor setting
func Output() io.Writer {
	if NoColor {
		return os.Stdout
	}
	return os.Stdout
}
