package dio

import "io"

// Open returns a Writer based on the given flags.
// Provide flags in the following order:
// sql, csv, json, jsonl
func Open(
	w io.WriteCloser,
	flags ...bool, // sql, csv, json, jsonl
) DataWriter {
	for i := 0; i < len(flags); i++ {
		if i > len(flags) {
			break
		}
		if flags[i] {
			switch i {
			case 0:
				return NewSql(w)
			case 1:
				return NewCsv(w)
			case 2:
				return NewJson(w)
			case 3:
				return NewJsonl(w)
			}
		}
	}
	return NewGloss(w)
}
