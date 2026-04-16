package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/databox/pkg/dio"
	"github.com/yznts/zen/v3/slice"
)

var (
	cpFlagSet = flag.NewFlagSet("cp", flag.ExitOnError)
	// Awareness flags
	cpDebug  = flagDebug(cpFlagSet)
	cpNowarn = flagNowarn(cpFlagSet)
	// Data format flags
	cpCsv   = flagCsv(cpFlagSet)
	cpJson  = flagJson(cpFlagSet)
	cpJsonl = flagJsonl(cpFlagSet)
	// Additional tool flags
	cpSchema = cpFlagSet.Bool("schema", false, "Copy schema only (no data)")
	cpTables = cpFlagSet.String("tables", "", "Comma-separated list of specific tables to copy (all by default)")

	cpUsage = "[options] <src> <dst>"
	cpDescr = "Copies schema/data from source database to destination database. Database could be different types."
)

func cpCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Csv: *cpCsv, Json: *cpJson, Jsonl: *cpJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Csv: *cpCsv, Json: *cpJson, Jsonl: *cpJsonl})
	)

	// Validate arguments
	if cpFlagSet.NArg() < 2 {
		dio.AssertError(stderr, errors.New("source and destination arguments are required"), *cpDebug)
	}

	// Open source database connection
	srcDsn, err := db.GetDsn(cpFlagSet.Arg(0))
	dio.AssertError(stderr, err, *cpDebug, "Failed to get source dsn: %v")
	srcCon, err := db.Open(srcDsn)
	dio.AssertError(stderr, err, *cpDebug, "Failed to connect to source database: %v")
	if srcCon, isCloser := srcCon.(io.Closer); isCloser {
		defer srcCon.Close()
	}

	// Open destination database connection
	dstDsn, err := db.GetDsn(cpFlagSet.Arg(1))
	dio.AssertError(stderr, err, *cpDebug, "Failed to get destination dsn: %v")
	dstCon, err := db.Open(dstDsn)
	dio.AssertError(stderr, err, *cpDebug, "Failed to connect to destination database: %v")
	if dstCon, isCloser := dstCon.(io.Closer); isCloser {
		defer dstCon.Close()
	}

	// Warn about possible lossy type conversions when databases differ
	if srcCon.GetConnection().Scheme != dstCon.GetConnection().Scheme {
		dio.AssertWarning(stderr, fmt.Errorf("source and destination databases are different types; column types will be mapped to generic equivalents and may lose precision or specificity"), *cpDebug, *cpNowarn)
	}

	// Read source tables (excluding system tables)
	tables, err := srcCon.QueryTables()
	dio.AssertError(stderr, err, *cpDebug, "Failed to query source tables: %v")
	tables = slice.Filter(tables, func(t db.Table) bool {
		return !t.IsSystem
	})

	// Filter tables if specific tables provided
	if *cpTables != "" {
		tableList := strings.Split(*cpTables, ",")
		tables = slice.Filter(tables, func(t db.Table) bool {
			return slice.Contains(tableList, t.Name)
		})
	}

	// Drop destination tables before migration
	for _, table := range tables {
		dstCon.GetConnection().Exec(`DROP TABLE IF EXISTS "` + table.Name + `"`)
	}

	// Track row counts per table for summary
	rowCounts := make(map[string]int)

	// Migrate schema and data for each table
	for _, table := range tables {
		// Query source columns
		columns, err := srcCon.QueryColumns(table.Name)
		dio.AssertError(stderr, err, *cpDebug, "Failed to query columns for "+table.Name+": %v")

		// Build and execute CREATE TABLE on destination
		srcScheme := srcCon.GetConnection().Scheme
		dstScheme := dstCon.GetConnection().Scheme
		createSQL := cpCreateTable(table.Name, columns, srcScheme, dstScheme)
		_, err = dstCon.GetConnection().Exec(createSQL)
		dio.AssertError(stderr, err, *cpDebug, "Failed to create table "+table.Name+": %v")

		// Copy data unless schema-only mode
		if !*cpSchema {
			data, err := srcCon.QueryData(`SELECT * FROM "` + table.Name + `"`)
			dio.AssertError(stderr, err, *cpDebug, "Failed to query data from "+table.Name+": %v")

			rowCounts[table.Name] = len(data.Rows)

			if len(data.Rows) > 0 {
				// Build INSERT statement with appropriate placeholders for destination database
				quotedCols := make([]string, len(data.Cols))
				for i, c := range data.Cols {
					quotedCols[i] = cpQuote(c)
				}
				colList := strings.Join(quotedCols, ", ")

				// Batch insert rows (batch size 100)
				batchSize := 100
				for i := 0; i < len(data.Rows); i += batchSize {
					end := i + batchSize
					if end > len(data.Rows) {
						end = len(data.Rows)
					}
					batch := data.Rows[i:end]

					// Build multi-row VALUES clause
					var valueSets []string
					var args []any
					for rowIdx, row := range batch {
						placeholder := cpPlaceholders(len(data.Cols), dstCon.GetConnection().Scheme, rowIdx*len(data.Cols))
						valueSets = append(valueSets, "("+placeholder+")")
						args = append(args, row...)
					}

					insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
						cpQuote(table.Name),
						colList,
						strings.Join(valueSets, ", "),
					)
					_, err := dstCon.GetConnection().Exec(insertSQL, args...)
					dio.AssertError(stderr, err, *cpDebug, "Failed to insert data into "+table.Name+": %v")
				}
			}
		}
	}

	// Output migration summary
	stdout.WriteData(&db.Data{
		Cols: []string{"TABLE", "TYPE", "ROWS"},
		Rows: slice.Map(tables, func(t db.Table) []any {
			transferType := "schema"
			rows := "-"
			if !*cpSchema {
				transferType = "schema+data"
				rows = fmt.Sprintf("%d", rowCounts[t.Name])
			}
			return []any{t.Name, transferType, rows}
		}),
	})
}

// cpCreateTable builds a CREATE TABLE statement from column definitions,
// translating column types to the destination database.
func cpCreateTable(name string, columns []db.Column, srcScheme, dstScheme string) string {
	var parts []string
	var primaryKeys []string
	var foreignKeys []string

	for _, col := range columns {
		colType := col.Type
		if srcScheme != dstScheme {
			colType = db.MapType(col.Type, dstScheme)
		}
		colDef := cpQuote(col.Name) + " " + colType
		if !col.IsNullable {
			colDef += " NOT NULL"
		}
		if mapped := db.MapDefault(col.Default, dstScheme); mapped != nil {
			colDef += fmt.Sprintf(" DEFAULT %v", mapped)
		}
		parts = append(parts, colDef)

		if col.IsPrimary {
			primaryKeys = append(primaryKeys, cpQuote(col.Name))
		}
		if col.ForeignRef != "" {
			foreignKeys = append(foreignKeys, fmt.Sprintf(
				"FOREIGN KEY (%s) REFERENCES %s ON UPDATE %s ON DELETE %s",
				cpQuote(col.Name), col.ForeignRef, col.ForeignOnUpdate, col.ForeignOnDelete,
			))
		}
	}

	if len(primaryKeys) > 0 {
		parts = append(parts, "PRIMARY KEY ("+strings.Join(primaryKeys, ", ")+")")
	}
	parts = append(parts, foreignKeys...)

	return fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", cpQuote(name), strings.Join(parts, ",\n  "))
}

// cpQuote wraps an identifier in double quotes.
func cpQuote(name string) string {
	return `"` + name + `"`
}

// cpPlaceholders builds a comma-separated placeholder string
// appropriate for the destination database type.
// offset is the starting parameter index (used for batched inserts).
func cpPlaceholders(n int, scheme string, offset int) string {
	placeholders := make([]string, n)
	for i := range n {
		switch scheme {
		case "postgres":
			placeholders[i] = fmt.Sprintf("$%d", offset+i+1)
		default:
			placeholders[i] = "?"
		}
	}
	return strings.Join(placeholders, ", ")
}
