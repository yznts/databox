package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// Connection is a wrapper around sql.DB that also stores the DSN and scheme.
// Also, it holds database-agnostic methods.
type Connection struct {
	*sql.DB

	DSN    *url.URL
	Scheme string
}

// QueryData queries the database with the given query and optional args,
// returning the full result set as a Data pointer.
// Supports parameterized queries: use ? for SQLite/MySQL, $N for PostgreSQL.
//
// This is the generic implementation that uses 'any' for type storage and
// leaves type assertion to the underlying driver. MySQL overrides this method
// to use ColumnTypes() for correct type scanning.
func (c *Connection) QueryData(query string, args ...any) (*Data, error) {
	rows, err := c.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	data := &Data{Cols: cols}
	var scan []any
	for range cols {
		// new(any) is the most generic scan target; type assertion is left to the driver.
		// (postgres driver, unlike MySQL, handles this correctly without ColumnTypes.)
		scan = append(scan, new(any))
	}
	for rows.Next() {
		err = rows.Scan(scan...)
		if err != nil {
			return nil, err
		}
		var row []any
		for _, ptr := range scan {
			row = append(row, reflect.ValueOf(ptr).Elem().Interface())
		}
		data.Rows = append(data.Rows, row)
	}
	return data, rows.Err()
}

// QueryDataStream executes a query and streams results row-by-row via channels.
// The caller should range over the first channel, then read the error channel once.
// The error channel is buffered (size 1) and is always closed after the data channel.
// This avoids loading the full result set into memory for large tables.
func (c *Connection) QueryDataStream(query string, args ...any) (<-chan *Data, <-chan error) {
	dataCh := make(chan *Data)
	errCh := make(chan error, 1)
	go func() {
		defer close(dataCh)
		defer close(errCh)
		rows, err := c.Query(query, args...)
		if err != nil {
			errCh <- err
			return
		}
		defer rows.Close()
		cols, err := rows.Columns()
		if err != nil {
			errCh <- err
			return
		}
		var scan []any
		for range cols {
			scan = append(scan, new(any))
		}
		for rows.Next() {
			err = rows.Scan(scan...)
			if err != nil {
				errCh <- err
				return
			}
			var row []any
			for _, ptr := range scan {
				row = append(row, reflect.ValueOf(ptr).Elem().Interface())
			}
			dataCh <- &Data{Cols: cols, Rows: [][]any{row}}
		}
		if err := rows.Err(); err != nil {
			errCh <- err
		}
	}()
	return dataCh, errCh
}

// CreateTable builds and executes a CREATE TABLE statement from the provided columns.
// Column types are used as-is; the caller is responsible for any type mapping
// before invoking this method.
func (c *Connection) CreateTable(table string, columns []Column) error {
	var parts []string
	var primaryKeys []string
	var foreignKeys []string

	for _, col := range columns {
		colDef := `"` + col.Name + `"` + " " + col.Type
		if !col.IsNullable {
			colDef += " NOT NULL"
		}
		if mapped := MapDefault(col.Default, c.Scheme); mapped != nil {
			colDef += fmt.Sprintf(" DEFAULT %v", mapped)
		}
		parts = append(parts, colDef)

		if col.IsPrimary {
			primaryKeys = append(primaryKeys, `"`+col.Name+`"`)
		}
		if col.ForeignRef != "" {
			foreignKeys = append(foreignKeys, fmt.Sprintf(
				"FOREIGN KEY (%s) REFERENCES %s ON UPDATE %s ON DELETE %s",
				`"`+col.Name+`"`, col.ForeignRef, col.ForeignOnUpdate, col.ForeignOnDelete,
			))
		}
	}

	if len(primaryKeys) > 0 {
		parts = append(parts, "PRIMARY KEY ("+strings.Join(primaryKeys, ", ")+")")
	}
	parts = append(parts, foreignKeys...)

	sql := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", `"`+table+`"`, strings.Join(parts, ",\n  "))
	_, err := c.Exec(sql)
	return err
}
