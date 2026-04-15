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
	headFlagSet = flag.NewFlagSet("head", flag.ExitOnError)
	// Awareness flags
	headDebug  = flagDebug(headFlagSet)
	headNowarn = flagNowarn(headFlagSet)
	// Data format flags
	headDsn   = flagDsn(headFlagSet)
	headSql   = flagSql(headFlagSet)
	headCsv   = flagCsv(headFlagSet)
	headJson  = flagJson(headFlagSet)
	headJsonl = flagJsonl(headFlagSet)
	// Additional tool flags
	headN = headFlagSet.Int("n", 10, "Number of rows to output")

	headUsage = "[options] <table>"
	headDescr = "Outputs the first N rows of a table. By default, outputs 10 rows."
)

func headCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, *headSql, *headCsv, *headJson, *headJsonl)
		stderr = dio.Open(os.Stderr, *headSql, *headCsv, *headJson, *headJsonl)
	)
	// Open database connection
	dsn, err := db.GetDsn(*headDsn)
	dio.AssertError(stderr, err, *headDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *headDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	// Get table argument
	if headFlagSet.NArg() < 1 {
		dio.AssertError(stderr, errors.New("table argument is required"), *headDebug)
	}
	table := headFlagSet.Arg(0)

	// Execute query and output results
	query := fmt.Sprintf(`SELECT * FROM "%s" LIMIT %d`, table, *headN)
	data, err := con.QueryData(query)
	dio.AssertError(stderr, err, *headDebug, "Failed to execute query: %v")
	if tw, ok := stdout.(dio.TableWriter); ok {
		tw.SetTable(table)
	}
	stdout.WriteData(data)
}
