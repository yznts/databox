package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/databox/pkg/dio"
	"github.com/yznts/zen/v3/conv"
)

var (
	tailFlagSet = flag.NewFlagSet("tail", flag.ExitOnError)
	// Awareness flags
	tailDebug  = flagDebug(tailFlagSet)
	tailNowarn = flagNowarn(tailFlagSet)
	// Data format flags
	tailDsn   = flagDsn(tailFlagSet)
	tailSql   = flagSql(tailFlagSet)
	tailCsv   = flagCsv(tailFlagSet)
	tailJson  = flagJson(tailFlagSet)
	tailJsonl = flagJsonl(tailFlagSet)
	// Additional tool flags
	tailN     = tailFlagSet.Int("n", 10, "Number of rows to output")
	tailOrder = flagOrder(tailFlagSet)

	tailUsage = "[options] <table>"
	tailDescr = "Outputs the last N rows of a table. By default, outputs 10 rows."
)

func tailCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Sql: *tailSql, Csv: *tailCsv, Json: *tailJson, Jsonl: *tailJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Sql: *tailSql, Csv: *tailCsv, Json: *tailJson, Jsonl: *tailJsonl})
	)
	// Open database connection
	dsn, err := db.GetDsn(*tailDsn)
	dio.AssertError(stderr, err, *tailDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *tailDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	// Get table argument
	if tailFlagSet.NArg() < 1 {
		dio.AssertError(stderr, errors.New("table argument is required"), *tailDebug)
	}
	table := tailFlagSet.Arg(0)

	// Get total row count first, then compute offset for last N rows.
	// This approach is cross-database compatible (MySQL doesn't support subqueries in LIMIT/OFFSET).
	countData, err := con.QueryData(fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table))
	dio.AssertError(stderr, err, *tailDebug, "Failed to count rows: %v")
	totalRows := 0
	if len(countData.Rows) > 0 {
		totalRows = conv.Int(countData.Rows[0][0])
	}
	offset := totalRows - *tailN
	if offset < 0 {
		offset = 0
	}

	// Execute query with computed offset
	query := fmt.Sprintf(`SELECT * FROM "%s"%s LIMIT %d OFFSET %d`, table, orderClause(*tailOrder), *tailN, offset)
	data, err := con.QueryData(query)
	dio.AssertError(stderr, err, *tailDebug, "Failed to execute query: %v")
	if tw, ok := stdout.(dio.TableWriter); ok {
		tw.SetTable(table)
	}
	stdout.WriteData(data)
}
