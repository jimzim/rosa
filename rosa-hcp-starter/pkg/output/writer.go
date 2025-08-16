// Package output provides formatted output for the CLI
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// Format represents the output format
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// Writer handles formatted output
type Writer struct {
	out    io.Writer
	format Format
	styles Styles
}

// Styles contains lipgloss styles for output
type Styles struct {
	Title       lipgloss.Style
	Success     lipgloss.Style
	Error       lipgloss.Style
	Warning     lipgloss.Style
	Info        lipgloss.Style
	Subtle      lipgloss.Style
	Bold        lipgloss.Style
	TableHeader lipgloss.Style
	TableRow    lipgloss.Style
	TableBorder lipgloss.Style
}

// DefaultStyles returns the default styles
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),
		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")),
		Subtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		Bold: lipgloss.NewStyle().
			Bold(true),
		TableHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("241")),
		TableRow: lipgloss.NewStyle().
			Padding(0, 1),
		TableBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241")),
	}
}

// NewWriter creates a new writer with the specified format
func NewWriter(format Format) *Writer {
	return &Writer{
		out:    os.Stdout,
		format: format,
		styles: DefaultStyles(),
	}
}

// SetOutput sets the output writer
func (w *Writer) SetOutput(out io.Writer) {
	w.out = out
}

// Print outputs a value in the configured format
func (w *Writer) Print(v interface{}) error {
	switch w.format {
	case FormatJSON:
		return w.printJSON(v)
	case FormatYAML:
		return w.printYAML(v)
	default:
		return w.printText(v)
	}
}

func (w *Writer) printJSON(v interface{}) error {
	encoder := json.NewEncoder(w.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func (w *Writer) printYAML(v interface{}) error {
	encoder := yaml.NewEncoder(w.out)
	defer encoder.Close()
	return encoder.Encode(v)
}

func (w *Writer) printText(v interface{}) error {
	switch val := v.(type) {
	case fmt.Stringer:
		fmt.Fprintln(w.out, val.String())
	case string:
		fmt.Fprintln(w.out, val)
	default:
		fmt.Fprintf(w.out, "%+v\n", val)
	}
	return nil
}

// Success prints a success message
func (w *Writer) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(w.out, w.styles.Success.Render("✓ "+msg))
}

// Error prints an error message
func (w *Writer) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(w.out, w.styles.Error.Render("✗ "+msg))
}

// Warning prints a warning message
func (w *Writer) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(w.out, w.styles.Warning.Render("⚠ "+msg))
}

// Info prints an info message
func (w *Writer) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(w.out, w.styles.Info.Render("ℹ "+msg))
}

// Title prints a title
func (w *Writer) Title(title string) {
	fmt.Fprintln(w.out, w.styles.Title.Render(title))
}

// Table renders a table with headers and rows
func (w *Writer) Table(headers []string, rows [][]string) {
	if len(headers) == 0 || len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Build header
	var headerCells []string
	for i, h := range headers {
		cell := lipgloss.NewStyle().
			Width(widths[i]+2).
			Padding(0, 1).
			Bold(true).
			Render(h)
		headerCells = append(headerCells, cell)
	}
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, headerCells...)

	// Build rows
	var tableRows []string
	tableRows = append(tableRows, headerRow)

	// Add separator
	var separatorCells []string
	for _, width := range widths {
		sep := strings.Repeat("─", width+2)
		separatorCells = append(separatorCells, sep)
	}
	separator := lipgloss.JoinHorizontal(lipgloss.Top, separatorCells...)
	tableRows = append(tableRows, w.styles.Subtle.Render(separator))

	// Add data rows
	for _, row := range rows {
		var cells []string
		for i, cell := range row {
			if i < len(widths) {
				styledCell := lipgloss.NewStyle().
					Width(widths[i]+2).
					Padding(0, 1).
					Render(cell)
				cells = append(cells, styledCell)
			}
		}
		dataRow := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
		tableRows = append(tableRows, dataRow)
	}

	// Join all rows and apply border
	table := lipgloss.JoinVertical(lipgloss.Left, tableRows...)
	borderedTable := w.styles.TableBorder.Render(table)

	fmt.Fprintln(w.out, borderedTable)
}

// List renders a list of items
func (w *Writer) List(items []string) {
	for _, item := range items {
		fmt.Fprintf(w.out, "  • %s\n", item)
	}
}

// KeyValue renders key-value pairs
func (w *Writer) KeyValue(pairs map[string]string) {
	maxKeyLen := 0
	for k := range pairs {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	for k, v := range pairs {
		key := w.styles.Bold.Render(fmt.Sprintf("%-*s", maxKeyLen, k))
		fmt.Fprintf(w.out, "%s : %s\n", key, v)
	}
}

// Progress represents a progress indicator
type Progress struct {
	spinner spinner.Model
	title   string
	program *tea.Program
}

// NewProgress creates a new progress indicator
func NewProgress(title string) *Progress {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &Progress{
		spinner: s,
		title:   title,
	}
}

// progressModel is the model for the progress indicator
type progressModel struct {
	spinner spinner.Model
	title   string
	done    bool
}

func (m progressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m progressModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), m.title)
}

// Start starts the progress indicator
func (p *Progress) Start() {
	model := progressModel{
		spinner: p.spinner,
		title:   p.title,
	}
	p.program = tea.NewProgram(model)
	go func() {
		if _, err := p.program.Run(); err != nil {
			fmt.Printf("Error running progress: %v\n", err)
		}
	}()
}

// Stop stops the progress indicator
func (p *Progress) Stop() {
	if p.program != nil {
		p.program.Quit()
		// Give it a moment to clean up
		time.Sleep(100 * time.Millisecond)
	}
}

// Success stops the progress indicator and shows a success message
func (p *Progress) Success(message string) {
	p.Stop()
	w := NewWriter(FormatText)
	w.Success(message)
}

// Error stops the progress indicator and shows an error message
func (p *Progress) Error(message string) {
	p.Stop()
	w := NewWriter(FormatText)
	w.Error(message)
}

// Global convenience functions

var defaultWriter = NewWriter(FormatText)

// Print prints a value using the default writer
func Print(v interface{}, format string) error {
	w := NewWriter(Format(format))
	return w.Print(v)
}

// Success prints a success message using the default writer
func Success(format string, args ...interface{}) {
	defaultWriter.Success(format, args...)
}

// Error prints an error message using the default writer
func Error(format string, args ...interface{}) {
	defaultWriter.Error(format, args...)
}

// Warning prints a warning message using the default writer
func Warning(format string, args ...interface{}) {
	defaultWriter.Warning(format, args...)
}

// Info prints an info message using the default writer
func Info(format string, args ...interface{}) {
	defaultWriter.Info(format, args...)
}
