package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/cli/pkg/color"
)

var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	infoColor    = color.New(color.FgCyan)
	warnColor    = color.New(color.FgYellow)
)

func Success(format string, a ...interface{}) {
	successColor.Printf("✓ "+format+"\n", a...)
}

func Error(format string, a ...interface{}) {
	errorColor.Fprintf(os.Stderr, "✗ "+format+"\n", a...)
}

func Info(format string, a ...interface{}) {
	infoColor.Printf(format+"\n", a...)
}

func Warn(format string, a ...interface{}) {
	warnColor.Printf("⚠ "+format+"\n", a...)
}

func JSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

type Table struct {
	headers []string
	rows    [][]string
}

func NewTable(headers []string) *Table {
	return &Table{
		headers: headers,
		rows:    [][]string{},
	}
}

func (t *Table) AddRow(row []string) {
	t.rows = append(t.rows, row)
}

func (t *Table) Render() {
	// Calculate column widths
	widths := make([]int, len(t.headers))
	for i, header := range t.headers {
		widths[i] = len(header)
	}

	for _, row := range t.rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	headerColor := color.New(color.FgWhite, color.Bold)
	for i, header := range t.headers {
		headerColor.Printf("%-*s  ", widths[i], header)
	}
	fmt.Println()

	// Print separator
	for i := range t.headers {
		fmt.Print(strings.Repeat("-", widths[i]) + "  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range t.rows {
		for i, cell := range row {
			fmt.Printf("%-*s  ", widths[i], cell)
		}
		fmt.Println()
	}
}
