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
// Panics if more than one format flag is set.
func Open(w io.Writer, cfg Config) DataWriter {
	count := 0
	if cfg.Sql {
		count++
	}
	if cfg.Csv {
		count++
	}
	if cfg.Json {
		count++
	}
	if cfg.Jsonl {
		count++
	}
	if count > 1 {
		panic("only one output format flag may be set at a time (-sql, -csv, -json, -jsonl)")
	}
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
