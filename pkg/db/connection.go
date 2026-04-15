package db

import (
	"database/sql"
	"net/url"
	"reflect"
)

// Connection is a wrapper around sql.DB that also stores the DSN and scheme.
// Also, it holds database-agnostic methods.
type Connection struct {
	*sql.DB

	DSN    *url.URL
	Scheme string
}

// QueryData is a database-agnostic method that queries the database
// with the given query and returns the result as a Data struct pointer.
//
// The Data struct contains the columns and rows of the result.
// Method is returning a pointer to avoid copying the Data struct,
// which might be large.
//
// This exact implementation is the most generic one.
// It utilizes 'any' type to store the values of the result
// and leaves all type assertion to the underlying driver.
// For some databases, like MySQL, we might need to override this method.
func (c *Connection) QueryData(query string) (*Data, error) {
	// Execute the query.
	rows, err := c.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// Get columns information.
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	// Initialize the Data struct.
	// It holds both the columns and rows of the result.
	data := &Data{
		Cols: cols,
	}
	// Define scan target row.
	// This is a slice of pointers,
	// so we need to copy values on each iteration.
	var scan []any
	for range cols {
		// We're using new(any) here as the most generic solution,
		// so we're leaving the type assertion to the driver.
		// In some cases (like MySQL) we will need to override QueryData method
		// to handle type assertion correctly.
		//
		// We're not using col.ScanType() here because it's not always correct.
		// For example, postgres driver doesn't report nullable types correctly (sql.NullString).
		scan = append(scan, new(any))
	}
	for rows.Next() {
		// Scan the row into prepared pointers
		err = rows.Scan(scan...)
		if err != nil {
			return nil, err
		}
		// Copy exact values from the pointers to the Data struct
		var row []any
		for _, ptr := range scan {
			// Get value from the pointer and append it to the row
			row = append(row, reflect.ValueOf(ptr).Elem().Interface())
		}
		// Append the row to the Data holder
		data.Rows = append(data.Rows, row)
	}
	return data, nil
}
