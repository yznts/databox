package dio

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/zen/v3/slice"
)

// Csv is a writer that writes data as a csv.
type Csv struct {
	*csv.Writer

	// flushed determines if the writer has been flushed.
	// If it hasn't, the first table write will write the columns.
	flushed bool
}

// write wraps the csv writer's Write method.
// If an error occurs, it panics.
// It's unexpected behavior in our case,
// so panic is necessary.
func (c *Csv) write(record []string) {
	err := c.Writer.Write(record)
	if err != nil {
		panic(err)
	}
}

// Csv supports multiple writes.
func (c *Csv) MultiWrite() bool {
	return true
}

// WriteError writes an error in a csv writer.
func (c *Csv) WriteError(err error) {
	c.write([]string{"ERROR", err.Error()})
	c.Flush()
}

// WriteData writes data in a csv writer.
func (c *Csv) WriteData(data *db.Data) {
	// If it's the first write (no flushes), write the columns.
	if !c.flushed {
		c.flushed = true
		c.write(data.Cols)
	}
	// Write the rows.
	for _, row := range data.Rows {
		// Convert the row to a string slice.
		rowstr := slice.Map(row, func(v any) string {
			return fmt.Sprintf("%v", v)
		})
		// Write the row.
		c.write(rowstr)
	}
	// Flush the writer.
	c.Flush()
	// If an error occurs, panic.
	if c.Error() != nil {
		panic(c.Error())
	}
}

// NewCsv creates a new Csv writer.
func NewCsv(w io.Writer) *Csv {
	return &Csv{
		Writer: csv.NewWriter(w),
	}
}
