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
	migrateFlagSet = flag.NewFlagSet("migrate", flag.ExitOnError)
	// Awareness flags
	migrateDebug  = flagDebug(migrateFlagSet)
	migrateNowarn = flagNowarn(migrateFlagSet)
	// Data format flags
	migrateCsv   = flagCsv(migrateFlagSet)
	migrateJson  = flagJson(migrateFlagSet)
	migrateJsonl = flagJsonl(migrateFlagSet)
	// Type flags
	migrateSchema     = migrateFlagSet.Bool("schema", false, "Migrate schema only from source to destination database (src/dst are DSNs)")
	migrateSchemaData = migrateFlagSet.Bool("schema-data", false, "Migrate schema and data from source to destination database (src/dst are DSNs)")
	// Additional tool flags
	migrateTables = migrateFlagSet.String("tables", "", "Comma-separated list of specific tables to migrate (all by default)")

	migrateUsage = "[options] [migrate-type] <src> <dst>"
	migrateDescr = "Migrates schema/data from one database to another. Databases may be different types."
)

func migrateCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Csv: *migrateCsv, Json: *migrateJson, Jsonl: *migrateJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Csv: *migrateCsv, Json: *migrateJson, Jsonl: *migrateJsonl})
	)

	// Validate arguments
	if migrateFlagSet.NArg() < 2 {
		dio.AssertError(stderr, errors.New("source and destination arguments are required"), *migrateDebug)
	}

	switch {
	case *migrateSchema, *migrateSchemaData:
		migrateRunCmd(stdout, stderr)
	default:
		dio.AssertError(stderr, errors.New("a type flag is required: -schema or -schema-data"), *migrateDebug)
	}
}

func migrateRunCmd(stdout, stderr dio.DataWriter) {
	// Open source database connection
	srcDsn, err := db.GetDsn(migrateFlagSet.Arg(0))
	dio.AssertError(stderr, err, *migrateDebug, "Failed to get source dsn: %v")
	srcCon, err := db.Open(srcDsn)
	dio.AssertError(stderr, err, *migrateDebug, "Failed to connect to source database: %v")
	if srcCon, isCloser := srcCon.(io.Closer); isCloser {
		defer srcCon.Close()
	}

	// Open destination database connection
	dstDsn, err := db.GetDsn(migrateFlagSet.Arg(1))
	dio.AssertError(stderr, err, *migrateDebug, "Failed to get destination dsn: %v")
	dstCon, err := db.Open(dstDsn)
	dio.AssertError(stderr, err, *migrateDebug, "Failed to connect to destination database: %v")
	if dstCon, isCloser := dstCon.(io.Closer); isCloser {
		defer dstCon.Close()
	}

	srcSm, ok := srcCon.(db.SchemaManager)
	if !ok {
		dio.AssertError(stderr, errors.New("source database does not support schema operations"), *migrateDebug)
		return
	}
	dstSm, ok := dstCon.(db.SchemaManager)
	if !ok {
		dio.AssertError(stderr, errors.New("destination database does not support schema operations"), *migrateDebug)
		return
	}

	// Warn about possible lossy type conversions when databases differ
	if srcCon.GetConnection().Scheme != dstCon.GetConnection().Scheme {
		dio.AssertWarning(stderr, fmt.Errorf("source and destination databases are different types; column types will be mapped to generic equivalents and may lose precision or specificity"), *migrateDebug, *migrateNowarn)
	}

	// Read source tables (excluding system tables)
	tables, err := srcSm.GetTables()
	dio.AssertError(stderr, err, *migrateDebug, "Failed to query source tables: %v")
	tables = slice.Filter(tables, func(t db.Table) bool {
		return !t.IsSystem
	})

	// Filter tables if specific tables provided
	if *migrateTables != "" {
		tableList := strings.Split(*migrateTables, ",")
		tables = slice.Filter(tables, func(t db.Table) bool {
			return slice.Contains(tableList, t.Name)
		})
	}

	// Drop destination tables before migration
	for _, table := range tables {
		_, err := dstCon.GetConnection().Exec("DROP TABLE IF EXISTS " + dstSm.QuoteIdentifier(table.Name))
		dio.AssertError(stderr, err, *migrateDebug, "Failed to drop destination table "+table.Name+": %v")
	}

	// Track row counts per table for summary
	rowCounts := make(map[string]int)

	// Migrate schema and data for each table
	for _, table := range tables {
		// Query source columns
		columns, err := srcSm.GetColumns(table.Name)
		dio.AssertError(stderr, err, *migrateDebug, "Failed to query columns for "+table.Name+": %v")

		// Map column types to destination scheme when databases differ
		srcScheme := srcCon.GetConnection().Scheme
		dstScheme := dstCon.GetConnection().Scheme
		if srcScheme != dstScheme {
			for i, col := range columns {
				columns[i].Type = db.MapType(col.Type, dstScheme)
			}
		}

		// Build and execute CREATE TABLE on destination
		err = dstSm.CreateTable(table.Name, columns)
		dio.AssertError(stderr, err, *migrateDebug, "Failed to create table "+table.Name+": %v")

		// Copy data if schema-data mode
		if *migrateSchemaData {
			dataCh, errCh := srcCon.QueryDataStream(`SELECT * FROM "` + table.Name + `"`)

			var (
				cols      []string
				colList   string
				batch     [][]any
				batchSize = 100
			)

			flushBatch := func() {
				if len(batch) == 0 {
					return
				}
				var valueSets []string
				var args []any
				for rowIdx, row := range batch {
					ph := make([]string, len(cols))
					for i := range cols {
						if dstCon.GetConnection().Scheme == db.PostgresDriver {
							ph[i] = fmt.Sprintf("$%d", rowIdx*len(cols)+i+1)
						} else {
							ph[i] = "?"
						}
					}
					valueSets = append(valueSets, "("+strings.Join(ph, ", ")+")")
					args = append(args, row...)
				}
				insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
					dstSm.QuoteIdentifier(table.Name),
					colList,
					strings.Join(valueSets, ", "),
				)
				_, err := dstCon.GetConnection().Exec(insertSQL, args...)
				dio.AssertError(stderr, err, *migrateDebug, "Failed to insert data into "+table.Name+": %v")
				rowCounts[table.Name] += len(batch)
				batch = batch[:0]
			}

			for data := range dataCh {
				if cols == nil {
					cols = data.Cols
					quotedCols := make([]string, len(cols))
					for i, c := range cols {
						quotedCols[i] = dstSm.QuoteIdentifier(c)
					}
					colList = strings.Join(quotedCols, ", ")
				}
				batch = append(batch, data.Rows...)
				if len(batch) >= batchSize {
					flushBatch()
				}
			}
			flushBatch()

			if err := <-errCh; err != nil {
				dio.AssertError(stderr, err, *migrateDebug, "Failed to stream data from "+table.Name+": %v")
			}
		}
	}

	// Output migration summary
	stdout.WriteData(&db.Data{
		Cols: []string{"TABLE", "TYPE", "ROWS"},
		Rows: slice.Map(tables, func(t db.Table) []any {
			transferType := "schema"
			rows := "-"
			if *migrateSchemaData {
				transferType = "schema-data"
				rows = fmt.Sprintf("%d", rowCounts[t.Name])
			}
			return []any{t.Name, transferType, rows}
		}),
	})
}
