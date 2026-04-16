package dio

import "io"

// Config holds the output format configuration for a writer.
type Config struct {
	Sql   bool
	Csv   bool
	Json  bool
	Jsonl bool
}

// Open returns a Writer based on the given config.
// If no format flag is set, defaults to Gloss (styled terminal output).
func Open(w io.Writer, cfg Config) DataWriter {
	switch {
	case cfg.Sql:
		return NewSql(w)
	case cfg.Csv:
		return NewCsv(w)
	case cfg.Json:
		return NewJson(w)
	case cfg.Jsonl:
		return NewJsonl(w)
	default:
		return NewGloss(w)
	}
}
