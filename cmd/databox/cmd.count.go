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
	countFlagSet = flag.NewFlagSet("count", flag.ExitOnError)
	// Awareness flags
	countDebug  = flagDebug(countFlagSet)
	countNowarn = flagNowarn(countFlagSet)
	// Data format flags
	countDsn   = flagDsn(countFlagSet)
	countSql   = flagSql(countFlagSet)
	countCsv   = flagCsv(countFlagSet)
	countJson  = flagJson(countFlagSet)
	countJsonl = flagJsonl(countFlagSet)

	countUsage = "[options] <table>"
	countDescr = "Outputs the row count of a table."
)

func countCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Sql: *countSql, Csv: *countCsv, Json: *countJson, Jsonl: *countJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Sql: *countSql, Csv: *countCsv, Json: *countJson, Jsonl: *countJsonl})
	)
	// Open database connection
	dsn, err := db.GetDsn(*countDsn)
	dio.AssertError(stderr, err, *countDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *countDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	// Get table argument
	if countFlagSet.NArg() < 1 {
		dio.AssertError(stderr, errors.New("table argument is required"), *countDebug)
	}
	table := countFlagSet.Arg(0)

	// Execute count query
	query := fmt.Sprintf(`SELECT COUNT(*) AS "COUNT" FROM "%s"`, table)
	data, err := con.QueryData(query)
	dio.AssertError(stderr, err, *countDebug, "Failed to execute query: %v")
	if tw, ok := stdout.(dio.TableSetter); ok {
		tw.SetTable(table)
	}
	stdout.WriteData(data)
}
