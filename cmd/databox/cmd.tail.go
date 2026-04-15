package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/databox/pkg/dio"
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
	tailN = tailFlagSet.Int("n", 10, "Number of rows to output")

	tailUsage = "[options] <table>"
	tailDescr = "Outputs the last N rows of a table. By default, outputs 10 rows."
)

func tailCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, *tailSql, *tailCsv, *tailJson, *tailJsonl)
		stderr = dio.Open(os.Stderr, *tailSql, *tailCsv, *tailJson, *tailJsonl)
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

	// Execute query with subquery to reverse row order, outputting last N rows
	query := fmt.Sprintf(
		`SELECT * FROM (SELECT * FROM "%s" LIMIT %d OFFSET (SELECT MAX(0, COUNT(*) - %d) FROM "%s")) sub`,
		table, *tailN, *tailN, table,
	)
	data, err := con.QueryData(query)
	dio.AssertError(stderr, err, *tailDebug, "Failed to execute query: %v")
	if tw, ok := stdout.(dio.TableWriter); ok {
		tw.SetTable(table)
	}
	stdout.WriteData(data)
}
