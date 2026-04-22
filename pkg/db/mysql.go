package db

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/yznts/zen/v3/slice"
)

// Mysql is a mysql database wrapper implementation.
// It implements QueryExecutor, SchemaManager and ProcessManager interfaces.
type Mysql struct {
	*Connection
}

// Internals

func (m *Mysql) systemSchemas() []string {
	return []string{"mysql", "information_schema", "performance_schema", "sys"}
}

// QueryExecutor implementation

func (m *Mysql) GetConnection() *Connection {
	return m.Connection
}

// QueryData overrides Connection.QueryData to use ColumnTypes() for correct
// MySQL type scanning (the MySQL driver doesn't make type assertions on scan).
func (m *Mysql) QueryData(query string, args ...any) (*Data, error) {
	rows, err := m.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	data := &Data{
		Cols: slice.Map(cols, func(c *sql.ColumnType) string { return c.Name() }),
	}
	var scan []any
	for _, col := range cols {
		scan = append(scan, reflect.New(col.ScanType()).Interface())
	}
	for rows.Next() {
		err = rows.Scan(scan...)
		if err != nil {
			return nil, err
		}
		var row []any
		for _, ptr := range scan {
			if v, ok := ptr.(interface{ Value() (driver.Value, error) }); ok {
				val, _ := v.Value()
				row = append(row, val)
				continue
			}
			row = append(row, reflect.ValueOf(ptr).Elem().Interface())
		}
		data.Rows = append(data.Rows, row)
	}
	return data, rows.Err()
}

// QueryDataStream overrides Connection.QueryDataStream using MySQL-aware type scanning.
func (m *Mysql) QueryDataStream(query string, args ...any) (<-chan *Data, <-chan error) {
	dataCh := make(chan *Data)
	errCh := make(chan error, 1)
	go func() {
		defer close(dataCh)
		defer close(errCh)
		rows, err := m.Query(query, args...)
		if err != nil {
			errCh <- err
			return
		}
		defer rows.Close()
		cols, err := rows.ColumnTypes()
		if err != nil {
			errCh <- err
			return
		}
		colNames := slice.Map(cols, func(c *sql.ColumnType) string { return c.Name() })
		var scan []any
		for _, col := range cols {
			scan = append(scan, reflect.New(col.ScanType()).Interface())
		}
		for rows.Next() {
			err = rows.Scan(scan...)
			if err != nil {
				errCh <- err
				return
			}
			var row []any
			for _, ptr := range scan {
				if v, ok := ptr.(interface{ Value() (driver.Value, error) }); ok {
					val, _ := v.Value()
					row = append(row, val)
					continue
				}
				row = append(row, reflect.ValueOf(ptr).Elem().Interface())
			}
			dataCh <- &Data{Cols: colNames, Rows: [][]any{row}}
		}
		if err := rows.Err(); err != nil {
			errCh <- err
		}
	}()
	return dataCh, errCh
}

// SchemaManager implementation

func (m *Mysql) GetTables() ([]Table, error) {
	// Query the database for the tables
	data, err := m.QueryData("SELECT table_name,table_schema FROM information_schema.tables")
	if err != nil {
		return nil, err
	}
	// Convert the data to a slice of Table objects
	tables := slice.Map(data.Rows, func(r []any) Table {
		return Table{
			Name:   r[0].(string),
			Schema: r[1].(string),
		}
	})
	// Mark system tables
	tables = slice.Map(tables, func(t Table) Table {
		if slice.Contains(m.systemSchemas(), t.Schema) {
			t.IsSystem = true
		}
		return t
	})
	// Return
	return tables, nil
}

func (m *Mysql) GetColumns(table string) ([]Column, error) {
	// Query the database for the columns
	dataCols, err := m.QueryData(`
		SELECT
			column_name,
			data_type,
			(CASE WHEN is_nullable = 'YES' THEN true ELSE false END) AS is_nullable,
			column_default
		FROM information_schema.columns
		WHERE table_name = ?`, table)
	if err != nil {
		return nil, err
	}
	// Query the database for constraints
	dataCons, err := m.QueryData(`
		SELECT DISTINCT
		    tc.CONSTRAINT_NAME,
		    tc.CONSTRAINT_TYPE,
		    kcu.TABLE_NAME AS referencing_table,
		    kcu.COLUMN_NAME AS referencing_column,
		    kcu.REFERENCED_TABLE_NAME AS referenced_table,
		    kcu.REFERENCED_COLUMN_NAME AS referenced_column,
		    rc.UPDATE_RULE AS foreign_on_update,
		    rc.DELETE_RULE AS foreign_on_delete
		FROM
		    information_schema.TABLE_CONSTRAINTS AS tc
		    JOIN information_schema.KEY_COLUMN_USAGE AS kcu
		      ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
		      AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA
		    LEFT JOIN information_schema.REFERENTIAL_CONSTRAINTS AS rc
		      ON rc.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
		      AND rc.CONSTRAINT_SCHEMA = tc.TABLE_SCHEMA
		WHERE
		    tc.TABLE_NAME = ?;
		`, table)
	if err != nil {
		return nil, err
	}
	// Compose the columns
	columns := slice.Map(dataCols.Rows, func(r []any) Column {
		// Compose base column
		col := Column{
			Name:       r[0].(string),
			Type:       r[1].(string),
			IsNullable: r[2].(int64) == 1,
			Default:    r[3],
		}
		// Find constraints information
		for _, con := range dataCons.Rows {
			if con[2].(string) == table && con[3].(string) == col.Name {
				if con[1].(string) == "PRIMARY KEY" {
					col.IsPrimary = true
				}
				if con[1].(string) == "FOREIGN KEY" {
					col.ForeignRef = fmt.Sprintf("%s(%s)", con[4].(string), con[5].(string))
					col.ForeignOnUpdate = con[6].(string)
					col.ForeignOnDelete = con[7].(string)
				}
			}
		}
		// Compose constraints
		return col
	})
	// Return
	return columns, nil
}

func (m *Mysql) CreateTable(table string, columns []Column) error {
	var parts []string
	var primaryKeys []string
	var foreignKeys []string

	for _, col := range columns {
		colDef := m.QuoteIdentifier(col.Name) + " " + col.Type
		if !col.IsNullable {
			colDef += " NOT NULL"
		}
		if mapped := MapDefault(col.Default, m.Scheme); mapped != nil {
			colDef += fmt.Sprintf(" DEFAULT %v", mapped)
		}
		parts = append(parts, colDef)

		if col.IsPrimary {
			primaryKeys = append(primaryKeys, m.QuoteIdentifier(col.Name))
		}
		if col.ForeignRef != "" {
			foreignKeys = append(foreignKeys, fmt.Sprintf(
				"FOREIGN KEY (%s) REFERENCES %s ON UPDATE %s ON DELETE %s",
				m.QuoteIdentifier(col.Name), col.ForeignRef, col.ForeignOnUpdate, col.ForeignOnDelete,
			))
		}
	}

	if len(primaryKeys) > 0 {
		parts = append(parts, "PRIMARY KEY ("+strings.Join(primaryKeys, ", ")+")")
	}
	parts = append(parts, foreignKeys...)

	sql := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", m.QuoteIdentifier(table), strings.Join(parts, ",\n  "))
	_, err := m.Exec(sql)
	return err
}

func (m *Mysql) QuoteIdentifier(name string) string {
	return "`" + name + "`"
}

// ProcessManager implementation

func (m *Mysql) GetProcesses() ([]Process, error) {
	// Query the database for the currently running processes
	query := `
		SELECT id, time, user, db, info
		FROM information_schema.processlist
	`
	data, err := m.QueryData(query)
	if err != nil {
		return nil, err
	}

	// Convert the data to a slice of Process objects
	def := func(v any, def any) any {
		if v == nil {
			return def
		}
		return v
	}
	processes := slice.Map(data.Rows, func(r []any) Process {
		return Process{
			Pid:      int(def(r[0], 0).(uint64)),
			Duration: time.Duration(def(r[1], 0).(int32)) * time.Second,
			Username: def(r[2], "").(string),
			Database: def(r[3], "").(string),
			Query:    strings.Join(strings.Fields(def(r[4], "").(string)), " "),
		}
	})

	// Return the list of processes
	return processes, nil
}

func (m *Mysql) KillProcess(pid int, force bool) error {
	if force {
		_, err := m.Exec(fmt.Sprintf("KILL CONNECTION %d", pid))
		return err
	}
	_, err := m.Exec(fmt.Sprintf("KILL QUERY %d", pid))
	return err
}
