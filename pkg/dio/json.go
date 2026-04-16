package dio

import (
	"io"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/zen/v3/jsonx"
)

// Json is a writer that writes a single json object.
type Json struct {
	w io.Writer

	closed bool
}

// write wraps the io writer's Write method.
// If an error occurs, it panics.
// It's unexpected behavior in our case,
// so panic is necessary.
func (j *Json) write(data []byte) {
	data = append(data, '\n')
	if _, err := j.w.Write(data); err != nil {
		panic(err)
	}
}

// Json does not support multiple writes,
// because it must output a single JSON object (unlike JSONL).
func (j *Json) MultiWrite() bool {
	return false
}

// WriteError writes an error in a json writer.
func (j *Json) WriteError(err error) {
	errmap := map[string]any{"ERROR": err.Error()}
	j.write(jsonx.Bytes(errmap))
}

// WriteData writes data in a json writer.
func (j *Json) WriteData(data *db.Data) {
	// If the writer is closed, panic, because it's unexpected behavior.
	if j.closed {
		panic("cannot write to closed writer")
	}
	// Write the data as a json object with "COLS" and "ROWS" fields.
	j.write(jsonx.Bytes(map[string]any{
		"COLS": data.Cols,
		"ROWS": data.Rows,
	}))
	// Mark the writer as closed, because we only support single write.
	j.closed = true
}

func NewJson(w io.Writer) *Json {
	return &Json{w: w}
}
