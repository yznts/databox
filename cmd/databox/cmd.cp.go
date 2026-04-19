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
)

var (
	cpFlagSet = flag.NewFlagSet("cp", flag.ExitOnError)
	// Connection flags
	cpDsn = flagDsn(cpFlagSet)
	// Awareness flags
	cpDebug = flagDebug(cpFlagSet)
	// Data format flags
	cpCsv   = flagCsv(cpFlagSet)
	cpJson  = flagJson(cpFlagSet)
	cpJsonl = flagJsonl(cpFlagSet)
	// Type flags
	cpSchema     = cpFlagSet.Bool("schema", false, "Copy table schema only (src/dst are table names, requires -dsn)")
	cpSchemaData = cpFlagSet.Bool("schema-data", false, "Copy table schema and data (src/dst are table names, requires -dsn)")

	cpUsage = "[options] [copy-type] <src> <dst>"
	cpDescr = "Copies a table within the same database. Use -schema for schema-only or -schema-data to include data."
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

	switch {
	case *cpSchema, *cpSchemaData:
		cpLocalCmd(stdout, stderr)
	default:
		dio.AssertError(stderr, errors.New("a type flag is required: -schema or -schema-data"), *cpDebug)
	}
}

func cpLocalCmd(stdout, stderr dio.DataWriter) {
	dsn, err := db.GetDsn(*cpDsn)
	dio.AssertError(stderr, err, *cpDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *cpDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	sm, ok := con.(db.SchemaManager)
	if !ok {
		dio.AssertError(stderr, errors.New("database does not support schema operations"), *cpDebug)
		return
	}

	srcTable := cpFlagSet.Arg(0)
	dstTable := cpFlagSet.Arg(1)

	// Query source columns
	columns, err := sm.GetColumns(srcTable)
	dio.AssertError(stderr, err, *cpDebug, "Failed to query columns for "+srcTable+": %v")

	// Drop destination table before copy
	con.GetConnection().Exec("DROP TABLE IF EXISTS " + sm.QuoteIdentifier(dstTable))

	// Build and execute CREATE TABLE on destination
	err = sm.CreateTable(dstTable, columns)
	dio.AssertError(stderr, err, *cpDebug, "Failed to create table "+dstTable+": %v")

	rowCount := 0

	// Copy data if schema-data mode
	if *cpSchemaData {
		dataCh, errCh := con.QueryDataStream(`SELECT * FROM "` + srcTable + `"`)

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
					if con.GetConnection().Scheme == db.PostgresDriver {
						ph[i] = fmt.Sprintf("$%d", rowIdx*len(cols)+i+1)
					} else {
						ph[i] = "?"
					}
				}
				valueSets = append(valueSets, "("+strings.Join(ph, ", ")+")")
				args = append(args, row...)
			}
			insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
				sm.QuoteIdentifier(dstTable),
				colList,
				strings.Join(valueSets, ", "),
			)
			_, err := con.GetConnection().Exec(insertSQL, args...)
			dio.AssertError(stderr, err, *cpDebug, "Failed to insert data into "+dstTable+": %v")
			rowCount += len(batch)
			batch = batch[:0]
		}

		for data := range dataCh {
			if cols == nil {
				cols = data.Cols
				quotedCols := make([]string, len(cols))
				for i, c := range cols {
					quotedCols[i] = sm.QuoteIdentifier(c)
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
			dio.AssertError(stderr, err, *cpDebug, "Failed to stream data from "+srcTable+": %v")
		}
	}

	transferType := "schema"
	rows := "-"
	if *cpSchemaData {
		transferType = "schema-data"
		rows = fmt.Sprintf("%d", rowCount)
	}
	stdout.WriteData(&db.Data{
		Cols: []string{"SRC", "DST", "TYPE", "ROWS"},
		Rows: [][]any{{srcTable, dstTable, transferType, rows}},
	})
}
