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
	// Basic flags
	tailDsn = flagDsn(tailFlagSet)
	// Awareness flags
	tailDebug  = flagDebug(tailFlagSet)
	tailNowarn = flagNowarn(tailFlagSet)
	// Data format flags
	tailSql   = flagSql(tailFlagSet)
	tailCsv   = flagCsv(tailFlagSet)
	tailJson  = flagJson(tailFlagSet)
	tailJsonl = flagJsonl(tailFlagSet)
	// Additional tool flags
	tailN     = tailFlagSet.Int("n", 10, "Number of rows to output")
	tailOrder = flagOrder(tailFlagSet)
	tailCol   = flagCol(tailFlagSet)
	tailWhere = flagWhere(tailFlagSet)

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

	var query string
	if *tailOrder != "" {
		// When an order column is given, use a single subquery:
		// reverse-sort inner query to get the last N rows, then re-sort outer query
		// to restore the original direction. No COUNT(*) round-trip needed.
		inner := fmt.Sprintf(`SELECT * FROM "%s"%s%s LIMIT %d`,
			table, whereClause(*tailWhere), orderClause(flipOrder(*tailOrder)), *tailN)
		query = fmt.Sprintf(`SELECT %s FROM (%s) AS _tail%s`,
			colClause(*tailCol), inner, orderClause(*tailOrder))
	} else {
		// Without an order column, fall back to COUNT(*)+OFFSET so that the
		// positional meaning of "last N rows" is preserved across all databases.
		countData, err := con.QueryData(
			fmt.Sprintf(`SELECT COUNT(*) FROM "%s"%s`, table, whereClause(*tailWhere)))
		dio.AssertError(stderr, err, *tailDebug, "Failed to count rows: %v")
		totalRows := 0
		if len(countData.Rows) > 0 {
			totalRows = conv.Int(countData.Rows[0][0])
		}
		offset := totalRows - *tailN
		if offset < 0 {
			offset = 0
		}
		query = fmt.Sprintf(`SELECT %s FROM "%s"%s LIMIT %d OFFSET %d`,
			colClause(*tailCol), table, whereClause(*tailWhere), *tailN, offset)
	}

	dio.Stream(dio.StreamParameters{
		Con: con, Stdout: stdout, Stderr: stderr,
		Debug: *tailDebug, Nowarn: *tailNowarn,
		Table: table, Query: query,
	})
}
