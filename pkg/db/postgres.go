package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/yznts/zen/v3/slice"
)

// Postgres is a postgres database wrapper implementation.
// It implements QueryExecutor, SchemaManager and ProcessManager interfaces.
type Postgres struct {
	*Connection
}

// Internals

func (p *Postgres) systemSchemas() []string {
	return []string{"pg_catalog", "information_schema"}
}

// QueryExecutor implementation

func (p *Postgres) GetConnection() *Connection {
	return p.Connection
}

// SchemaManager implementation

func (p *Postgres) GetTables() ([]Table, error) {
	// Query the database for the tables
	data, err := p.QueryData("SELECT table_name,table_schema FROM information_schema.tables")
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
		if slice.Contains(p.systemSchemas(), t.Schema) {
			t.IsSystem = true
		}
		return t
	})
	// Return
	return tables, nil
}

func (p *Postgres) GetColumns(table string) ([]Column, error) {
	// Query the database for the columns
	dataCols, err := p.QueryData(`
		SELECT
			column_name,
			data_type,
			(CASE WHEN is_nullable = 'YES' THEN true ELSE false END) AS is_nullable,
			column_default
		FROM information_schema.columns
		WHERE table_name = $1`, table)
	if err != nil {
		return nil, err
	}
	// Query the database for constraints
	dataCons, err := p.QueryData(`
		SELECT DISTINCT
		    tc.constraint_name,
		    tc.constraint_type,
		    tc.table_name AS referencing_table,
		    kcu.column_name AS referencing_column,
		    ccu.table_name AS referenced_table,
		    ccu.column_name AS referenced_column,
			fk.update_rule AS foreign_on_update,
		    fk.delete_rule AS foreign_on_delete
		FROM
		    information_schema.table_constraints AS tc
		    JOIN information_schema.key_column_usage AS kcu
		      ON tc.constraint_name = kcu.constraint_name
		      AND tc.table_schema = kcu.table_schema
		    JOIN information_schema.constraint_column_usage AS ccu
		      ON ccu.constraint_name = tc.constraint_name
		      AND ccu.table_schema = tc.table_schema
			LEFT JOIN information_schema.referential_constraints AS fk
			  ON fk.constraint_name = tc.constraint_name
		WHERE
		     tc.table_name = $1;
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
			IsNullable: r[2].(bool),
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

func (p *Postgres) QuoteIdentifier(name string) string {
	return `"` + name + `"`
}

// ProcessManager implementation

func (p *Postgres) GetProcesses() ([]Process, error) {
	// Query the database for the currently running processes
	query := `
		SELECT
			pid,
			date_part('epoch', now() - pg_stat_activity.query_start) AS duration,
			usename,
			datname,
			query
		FROM
			pg_stat_activity
	`
	data, err := p.QueryData(query)
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
			Pid:      int(def(r[0], 0).(int64)),
			Duration: time.Duration(def(r[1], 0.0).(float64)) * time.Second,
			Username: def(r[2], "").(string),
			Database: def(r[3], "").(string),
			Query:    strings.Join(strings.Fields(def(r[4], "").(string)), " "),
		}
	})

	// Return the list of processes
	return processes, nil
}

func (p *Postgres) KillProcess(pid int, force bool) error {
	if !force {
		_, err := p.Exec(fmt.Sprintf("SELECT pg_cancel_backend(%d)", pid))
		return err
	} else {
		_, err := p.Exec(fmt.Sprintf("SELECT pg_terminate_backend(%d)", pid))
		return err
	}
}
