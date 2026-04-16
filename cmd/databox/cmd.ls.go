package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"strings"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/databox/pkg/dio"
	"github.com/yznts/zen/v3/conv"
	"github.com/yznts/zen/v3/slice"
)

var (
	lsFlagSet = flag.NewFlagSet("ls", flag.ExitOnError)
	// Awareness flags
	lsDebug  = flagDebug(lsFlagSet)
	lsNowarn = flagNowarn(lsFlagSet)
	// Data format flags
	lsDsn   = flagDsn(lsFlagSet)
	lsSql   = flagSql(lsFlagSet)
	lsCsv   = flagCsv(lsFlagSet)
	lsJson  = flagJson(lsFlagSet)
	lsJsonl = flagJsonl(lsFlagSet)
	// Additional tool flags
	lsSys = lsFlagSet.Bool("sys", false, "Include system tables")
	lsCol = lsFlagSet.String("col", "extended", "Comma-separated list of column attributes to display (available: COLUMN_NAME/COLUMN_TYPE/IS_PK/IS_NL/DEFAULT/FK/FK_DEL/FK_UPD, aliases for sets: basic/extended/all)")

	lsUsage = "[options] <sql-query>"
	lsDescr = "Lists tables/columns in the database."
)

func lsCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Sql: *lsSql, Csv: *lsCsv, Json: *lsJson, Jsonl: *lsJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Sql: *lsSql, Csv: *lsCsv, Json: *lsJson, Jsonl: *lsJsonl})
	)
	// Open database connection
	dsn, err := db.GetDsn(*lsDsn)
	dio.AssertError(stderr, err, *lsDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *lsDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	// If -sql is set, output CREATE TABLE DDL via the TableWriter interface.
	// With a table argument: DDL for that table only.
	// Without a table argument: DDL for all (non-system) tables.
	if *lsSql {
		tw, ok := stdout.(dio.TableWriter)
		if !ok {
			dio.AssertError(stderr, errors.New("-sql output not supported by current writer"), *lsDebug)
		}
		tableNames := []string{}
		if lsFlagSet.NArg() >= 1 {
			tableNames = []string{lsFlagSet.Arg(0)}
		} else {
			allTables, err := con.QueryTables()
			dio.AssertError(stderr, err, *lsDebug, "Failed to list tables: %v")
			if !*lsSys {
				allTables = slice.Filter(allTables, func(t db.Table) bool { return !t.IsSystem })
			}
			for _, t := range allTables {
				tableNames = append(tableNames, t.Name)
			}
		}
		for _, name := range tableNames {
			columns, err := con.QueryColumns(name)
			dio.AssertError(stderr, err, *lsDebug, "Failed to list columns: %v")
			tw.WriteTable(name, columns)
		}
		return
	}

	// If no arguments, list tables.
	// Otherwise, list columns for provided table name.
	if lsFlagSet.NArg() == 0 {
		// Get tables
		tables, err := con.QueryTables()
		dio.AssertError(stderr, err, *lsDebug, "Failed to list tables: %v")
		// Filter system tables
		if !*lsSys {
			tables = slice.Filter(tables, func(t db.Table) bool {
				return !t.IsSystem
			})
		}
		// If no schema, print 'N/A'
		if slice.All(tables, func(t db.Table) bool { return t.Schema == "" }) {
			tables = slice.Map(tables, func(t db.Table) db.Table {
				t.Schema = "N/A"
				return t
			})
		}
		// Write tables
		stdout.WriteData(&db.Data{
			Cols: []string{"TABLE_SCHEMA", "TABLE_NAME", "IS_SYSTEM"},
			Rows: slice.Map(tables, func(t db.Table) []any {
				return []any{t.Schema, t.Name, t.IsSystem}
			}),
		})
	} else {
		// If argument provided, it's a table name.
		// We are going to list columns for that table.

		// Get table columns
		columns, err := con.QueryColumns(lsFlagSet.Arg(0))
		dio.AssertError(stderr, err, *lsDebug, "Failed to list columns: %v")

		// Extend aliases
		switch *lsCol {
		case "basic":
			*lsCol = "COLUMN_NAME,COLUMN_TYPE"
		case "extended":
			*lsCol = "COLUMN_NAME,COLUMN_TYPE,IS_PK,IS_NL,DEFAULT,FK"
		case "all":
			*lsCol = "COLUMN_NAME,COLUMN_TYPE,IS_PK,IS_NL,DEFAULT,FK,FK_DEL,FK_UPD"
		}

		// Validate -col flag and filter expected columns based on it.
		colAttrs := strings.Split(*lsCol, ",")
		validAttrs := []string{"COLUMN_NAME", "COLUMN_TYPE", "IS_PK", "IS_NL", "DEFAULT", "FK", "FK_DEL", "FK_UPD"}
		for _, attr := range colAttrs {
			if !slice.Contains(validAttrs, attr) {
				dio.AssertWarning(stderr, errors.New("invalid column attribute: "+attr), *lsDebug, *lsNowarn)
				colAttrs = slice.Filter(colAttrs, func(a string) bool {
					return a != attr
				})
			}
		}

		// Map column attributes to output columns
		colMap := map[string]func(c db.Column) any{
			"COLUMN_NAME": func(c db.Column) any { return c.Name },
			"COLUMN_TYPE": func(c db.Column) any { return c.Type },
			"IS_PK":       func(c db.Column) any { return c.IsPrimary },
			"IS_NL":       func(c db.Column) any { return c.IsNullable },
			"DEFAULT":     func(c db.Column) any { return conv.String(c.Default) },
			"FK":          func(c db.Column) any { return c.ForeignRef },
			"FK_DEL":      func(c db.Column) any { return c.ForeignOnDelete },
			"FK_UPD":      func(c db.Column) any { return c.ForeignOnUpdate },
		}

		// Write columns
		stdout.WriteData(&db.Data{
			Cols: colAttrs,
			Rows: slice.Map(columns, func(c db.Column) []any {
				row := make([]any, len(colAttrs))
				for i, attr := range colAttrs {
					row[i] = colMap[attr](c)
				}
				return row
			}),
		})
	}
}
