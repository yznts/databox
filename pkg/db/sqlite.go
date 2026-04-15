package db

import (
	"errors"
	"fmt"

	"github.com/yznts/zen/v3/slice"
)

// TODO: Include raw sql.
// SELECT sql FROM sqlite_schema WHERE name='...';
//
// For postgres
// SELECT pg_get_tabledef('mytable'::regclass);
// or
// SELECT pg_get_tabledef(oid) FROM pg_class WHERE relname = 'mytable';

type Sqlite struct {
	*Connection
}

func (s *Sqlite) GetConnection() *Connection {
	return s.Connection
}

func (s *Sqlite) systemTables() []string {
	return []string{"sqlite_master", "sqlite_sequence", "sqlite_stat1"}
}

func (s *Sqlite) QueryTables() ([]Table, error) {
	// Query the database for the tables
	data, err := s.QueryData("SELECT name,'' FROM sqlite_master WHERE type='table'")
	if err != nil {
		return nil, err
	}
	// Convert the data to a slice of Table objects
	tables := slice.Map(data.Rows, func(r []any) Table {
		return Table{
			Name: r[0].(string),
		}
	})
	// SQLite doesn't include system tables into sqlite_master,
	// so we have to manually add them.
	tables = append(
		tables,
		slice.Map(s.systemTables(), func(t string) Table {
			return Table{Name: t, IsSystem: true}
		})...,
	)
	// Return
	return tables, nil
}

func (s *Sqlite) QueryColumns(table string) ([]Column, error) {
	// Query the database for the columns.
	// We can't select exact fields because of 'notnull' issue (syntax error near "notnull").
	// So, here is a reference column list:
	// cid, name, type, notnull, dflt_value, pk
	dataCols, err := s.QueryData(fmt.Sprintf("SELECT * FROM PRAGMA_TABLE_INFO('%s')", table))
	if err != nil {
		return nil, err
	}
	// Query the database for the foreign keys information.
	// Same as above, we can't select exact fields because of syntax error.
	// So, here is a reference column list:
	// id, seq, table, from, to, on_update, on_delete, match
	dataFks, err := s.QueryData(fmt.Sprintf("SELECT * FROM PRAGMA_FOREIGN_KEY_LIST('%s')", table))
	if err != nil {
		return nil, err
	}
	// Compose the columns
	columns := slice.Map(dataCols.Rows, func(r []any) Column {
		// Compose base column
		col := Column{
			Name:       r[1].(string),
			Type:       r[2].(string),
			IsPrimary:  r[5].(int64) > 0,
			IsNullable: r[3].(int64) == 0,
			Default:    r[4],
		}
		// Find foreign key information
		for _, fk := range dataFks.Rows {
			if fk[3].(string) == col.Name {
				col.ForeignRef = fmt.Sprintf("%s(%s)", fk[2].(string), fk[4].(string))
				col.ForeignOnUpdate = fk[5].(string)
				col.ForeignOnDelete = fk[6].(string)
				break
			}
		}
		// Return
		return col
	})
	// Return
	return columns, nil
}

func (s *Sqlite) QueryProcesses() ([]Process, error) {
	return nil, errors.New("sqlite doesn't support process list query, use `lsof <file>` instead")
}

func (s *Sqlite) KillProcess(pid int, force bool) error {
	return errors.New("sqlite doesn't support process killing, use `lsof <file>` + `kill` instead")
}
