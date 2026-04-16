package dio

import (
	"fmt"

	"github.com/yznts/databox/pkg/db"
)

// StreamParameters holds all parameters for the Stream function.
type StreamParameters struct {
	Con    db.Database
	Stdout DataWriter
	Stderr DataWriter
	Debug  bool
	Nowarn bool
	Table  string
	RowCap int // 0 disables the cap
	Query  string
	Args   []any
}

// Stream executes a query and writes results to Stdout, choosing between
// row-by-row streaming (multi-write writers: CSV, JSONL, SQL) and a full load
// (single-write writers: Gloss, JSON).
//
// RowCap limits the number of rows written for single-write writers. When the
// result exceeds RowCap, the output is truncated and a warning is emitted unless
// Nowarn is true. A RowCap of 0 disables the cap entirely.
func Stream(p StreamParameters) {
	// Set table context if supported by the writer
	if ts, ok := p.Stdout.(TableSetter); ok {
		ts.SetTable(p.Table)
	}
	// Execute query with streaming
	rows, errs := p.Con.QueryDataStream(p.Query, p.Args...)
	// If the writer supports multi-write, stream rows as they come in.
	// Otherwise, fall back to capped load.
	if mw, ok := p.Stdout.(MultiWriter); ok && mw.MultiWrite() {
		for data := range rows {
			p.Stdout.WriteData(data)
		}
		if err := <-errs; err != nil {
			AssertError(p.Stderr, err, p.Debug, "Failed to stream query: %v")
		}
	} else {
		// Read capped number of rows to determine if we need to emit a warning about truncation.
		// If we reached the cap, we stop reading further and emit a warning.
		// If Nowarn is true, we ignore the cap as user is aware of potential high memory usage.
		cappedData := &db.Data{Cols: nil, Rows: nil}
		count := 0
		for data := range rows {
			// Initial columns write
			if count == 0 {
				cappedData.Cols = data.Cols
			}
			// If RowCap is set and we are about to exceed it, truncate the data and stop reading further.
			// Otherwise, keep appending data until we exhaust the stream or reach the cap.
			if p.RowCap > 0 && count+len(data.Rows) > p.RowCap && !p.Nowarn {
				cappedData.Rows = append(cappedData.Rows, data.Rows[:p.RowCap-count]...)
				AssertWarning(p.Stderr, fmt.Errorf(
					"output truncated to %d rows; use -nowarn to suppress this behavior, or change output format",
					p.RowCap,
				), p.Debug, p.Nowarn)
				break
			} else {
				cappedData.Rows = append(cappedData.Rows, data.Rows...)
				count += len(data.Rows)
			}
		}
		// Resulting output
		p.Stdout.WriteData(cappedData)
	}
	// Check for errors
	if err := <-errs; err != nil {
		AssertError(p.Stderr, err, p.Debug, "Failed to stream query: %v")
	}
}
