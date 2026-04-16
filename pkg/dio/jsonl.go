package dio

import (
	"io"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/zen/v3/jsonx"
)

// Jsonl is a writer that writes json lines.
type Jsonl struct {
	w io.Writer
}

// write wraps the io writer's Write method.
// If an error occurs, it panics.
// It's unexpected behavior in our case,
// so panic is necessary.
func (j *Jsonl) write(data []byte) {
	// Append newline
	data = append(data, '\n')
	// Write and panic on error
	_, err := j.w.Write(data)
	if err != nil {
		panic(err)
	}
}

// MultiWrite returns true if the writer supports multiple writes.
// Jsonl supports multiple writes.
func (j *Jsonl) MultiWrite() bool {
	return true
}

func (j *Jsonl) WriteError(err error) {
	errmap := map[string]any{"error": err.Error()}
	j.write(jsonx.Bytes(errmap))
}

func (j *Jsonl) WriteData(data *db.Data) {
	for _, row := range data.Rows {
		obj := map[string]any{}
		for i, col := range data.Cols {
			obj[col] = row[i]
		}
		j.write(jsonx.Bytes(obj))
	}
}

func NewJsonl(w io.Writer) *Jsonl {
	return &Jsonl{w: w}
}
