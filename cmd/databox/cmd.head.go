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
	// Basic flags
	headDsn = flagDsn(headFlagSet)
	// Awareness flags
	headDebug  = flagDebug(headFlagSet)
	headNowarn = flagNowarn(headFlagSet)
	// Data format flags
	headSql   = flagSql(headFlagSet)
	headCsv   = flagCsv(headFlagSet)
	headJson  = flagJson(headFlagSet)
	headJsonl = flagJsonl(headFlagSet)
	// Additional tool flags
	headN     = headFlagSet.Int("n", 10, "Number of rows to output")
	headOrder = flagOrder(headFlagSet)
	headCol   = flagCol(headFlagSet)
	headWhere = flagWhere(headFlagSet)

	headUsage = "[options] <table>"
	headDescr = "Outputs the first N rows of a table. By default, outputs 10 rows."
)

func headCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Sql: *headSql, Csv: *headCsv, Json: *headJson, Jsonl: *headJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Sql: *headSql, Csv: *headCsv, Json: *headJson, Jsonl: *headJsonl})
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
	query := fmt.Sprintf(`SELECT %s FROM "%s"%s%s LIMIT %d`,
		colClause(*headCol), table, whereClause(*headWhere), orderClause(*headOrder), *headN)
	dio.Stream(dio.StreamParameters{
		Con: con, Stdout: stdout, Stderr: stderr,
		Debug: *headDebug, Nowarn: *headNowarn,
		Table: table, Query: query,
	})
}
