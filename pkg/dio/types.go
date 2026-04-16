package dio

import "github.com/yznts/databox/pkg/db"

// ErrorWriter determines if a writer can write error messages.
type ErrorWriter interface {
	WriteError(err error)
}

// WarningWriter determines if a writer can write warning messages.
type WarningWriter interface {
	WriteWarning(err error)
}

// MultiWriter determines if a writer can write data multiple times.
type MultiWriter interface {
	MultiWrite() bool
}

// DataWriter determines if a writer can write data.
// This interface should be implemented by all writers.
type DataWriter interface {
	WriteData(data *db.Data)
}

// TableSetter determines if a writer supports setting a table name context.
type TableSetter interface {
	SetTable(name string)
}

// TableWriter determines if a writer can output a full table representation
// (e.g. CREATE TABLE DDL, or any other format that captures the table schema).
type TableWriter interface {
	WriteTable(table string, columns []db.Column)
}
