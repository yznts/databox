package main

import (
	"flag"
	"io"
	"os"
	"strings"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/databox/pkg/dio"
)

var (
	sqlFlagSet = flag.NewFlagSet("sql", flag.ExitOnError)
	// Basic flags
	sqlDsn = flagDsn(sqlFlagSet)
	// Awareness flags
	sqlDebug  = flagDebug(sqlFlagSet)
	sqlNowarn = flagNowarn(sqlFlagSet)
	// Data format flags
	sqlOutSql = flagSql(sqlFlagSet)
	sqlCsv    = flagCsv(sqlFlagSet)
	sqlJson   = flagJson(sqlFlagSet)
	sqlJsonl  = flagJsonl(sqlFlagSet)

	sqlUsage = "[options] <sql-query>"
	sqlDescr = "Executes the provided SQL query and outputs the result in the specified format. Query can be provided as an argument or read from STDIN if no argument is given."
)

func sqlCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Sql: *sqlOutSql, Csv: *sqlCsv, Json: *sqlJson, Jsonl: *sqlJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Sql: *sqlOutSql, Csv: *sqlCsv, Json: *sqlJson, Jsonl: *sqlJsonl})
	)
	// Open database connection
	dsn, err := db.GetDsn(*sqlDsn)
	dio.AssertError(stderr, err, *sqlDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *sqlDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	// Extract sql query from arguments
	query := strings.Join(sqlFlagSet.Args(), " ")
	// If no query provided, read from STDIN
	if query == "" {
		querybts, err := io.ReadAll(os.Stdin)
		dio.AssertError(stderr, err, *sqlDebug, "Failed to read query from STDIN: %v")
		query = string(querybts)
	}

	// Execute the query
	data, err := con.QueryData(query)
	dio.AssertError(stderr, err, *sqlDebug, "Failed to execute query: %v")

	// Write the result
	stdout.WriteData(data)
}
