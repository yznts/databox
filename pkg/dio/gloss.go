package dio

import (
	"fmt"
	"io"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/zen/v3/slice"
)

// Gloss is a writer that writes a formatted output,
// like a table or styled error/warn messages.
// Uses lipgloss for styling.
type Gloss struct {
	w      io.Writer
	closed bool
}

// write wraps the io writer's Write method.
// If an error occurs, it panics.
// It's unexpected behavior in our case,
// so panic is necessary.
func (g *Gloss) write(data []byte) {
	_, err := g.w.Write(data)
	if err != nil {
		panic(err)
	}
}

// Gloss does not support multiple writes,
// because it outputs in a formatted way that cannot be appended to (i.e. closed table).
func (g *Gloss) MultiWrite() bool {
	return false
}

// WriteError writes an error message in a styled format.
func (g *Gloss) WriteError(err error) {
	msg := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f66f81")).
		Bold(true).
		Render(fmt.Sprintf("error occured: %s", err.Error()))
	g.write([]byte(msg + "\n"))
	// No need to close writer, because it's just an error message.
	// We can write more data after that.
}

// WriteWarning writes a warning message in a styled format.
func (g *Gloss) WriteWarning(err error) {
	_msg := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f6ef6f")).
		Bold(true).
		Render(fmt.Sprintf("warning: %s", err.Error()))
	g.w.Write([]byte(_msg + "\n"))
	// No need to close writer, because it's just a warning message.
	// We can write more data after that.
}

// WriteData writes data in a formatted table.
// After writing the table, we forbid any more writes, because the table is closed.
func (g *Gloss) WriteData(data *db.Data) {
	// If the writer is closed, panic, because it's unexpected behavior.
	if g.closed {
		panic("cannot write to closed writer")
	}
	// Transform rows to string
	rowsstr := slice.Map(data.Rows, func(v []any) []string {
		return slice.Map(v, func(v any) string {
			// If value is []uint8, don't print it, just mark as not supported.
			// Probably this type is a blob or something that driver can't convert.
			if _, ok := v.([]uint8); ok {
				return "<n/s>"
			}
			return fmt.Sprintf("%v", v)
		})
	})
	// Create table
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true).Padding(0, 2)
			} else {
				return lipgloss.NewStyle().MaxHeight(5).MaxWidth(80).Padding(0, 2)
			}
		}).
		Headers(data.Cols...).
		Rows(rowsstr...)
	// Write table
	g.write([]byte(t.String() + "\n"))
	// Close writer, because we doesn't support multiple writes (yet).
	g.closed = true
}

// Create a new gloss writer.
func NewGloss(w io.Writer) *Gloss {
	return &Gloss{w: w}
}
