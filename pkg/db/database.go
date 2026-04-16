// This file holds database level interfaces and types.
// Not all interfaces could be implemented by supported databases (like process management),
// so it's tool responsibility to check for interface support and handle accordingly.
package db

import "time"

// QueryExecutor is an interface for databases that executing queries and retrieving results.
// The most common interface and should be implemented by all database wrappers.
// You'll get this type as a return value from Open(),
// but you can also try to assert other interfaces (like SchemaManager) if you need to perform specific operations.
type QueryExecutor interface {
	// Expose connection for raw operations
	GetConnection() *Connection

	// Data queries
	QueryData(query string, args ...any) (*Data, error) // Return a pointer because data amount might be large
	QueryDataStream(query string, args ...any) (<-chan *Data, <-chan error)
}

// SchemaManager is an interface for databases that support schema management operations.
type SchemaManager interface {
	GetTables() ([]Table, error)
	GetColumns(table string) ([]Column, error)
	CreateTable(table string, columns []Column) error
	QuoteIdentifier(name string) string
}

// ProcessManager is an optional interface for databases that support process management.
type ProcessManager interface {
	GetProcesses() ([]Process, error)
	KillProcess(pid int, force bool) error
}

// Data holds query results.
// Columns and rows are stored separately instead of using maps,
// so we can minimize memory usage and output.
type Data struct {
	Cols []string
	Rows [][]any
}

// Table holds table meta information,
// not the actual data.
type Table struct {
	Schema   string
	Name     string
	IsSystem bool // Indicates whether it's a system table
}

// Column holds column meta information.
type Column struct {
	Name       string
	Type       string
	IsPrimary  bool
	IsNullable bool
	Default    any

	// Foreign key information
	ForeignRef      string
	ForeignOnUpdate string
	ForeignOnDelete string
}

type Process struct {
	Pid      int
	Duration time.Duration
	Username string
	Database string
	Query    string
}
