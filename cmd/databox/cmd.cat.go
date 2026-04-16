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
	catFlagSet = flag.NewFlagSet("cat", flag.ExitOnError)
	// Awareness flags
	catDebug  = flagDebug(catFlagSet)
	catNowarn = flagNowarn(catFlagSet)
	// Data format flags
	catDsn   = flagDsn(catFlagSet)
	catSql   = flagSql(catFlagSet)
	catCsv   = flagCsv(catFlagSet)
	catJson  = flagJson(catFlagSet)
	catJsonl = flagJsonl(catFlagSet)

	// Additional tool flags
	catOrder = flagOrder(catFlagSet)
	catCol   = flagCol(catFlagSet)
	catWhere = flagWhere(catFlagSet)

	catUsage = "[options] <table>"
	catDescr = "Outputs all rows of a table."
)

func catCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Sql: *catSql, Csv: *catCsv, Json: *catJson, Jsonl: *catJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Sql: *catSql, Csv: *catCsv, Json: *catJson, Jsonl: *catJsonl})
	)
	// Open database connection
	dsn, err := db.GetDsn(*catDsn)
	dio.AssertError(stderr, err, *catDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *catDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	// Get table argument
	if catFlagSet.NArg() < 1 {
		dio.AssertError(stderr, errors.New("table argument is required"), *catDebug)
	}
	table := catFlagSet.Arg(0)

	// Execute query and output results
	query := fmt.Sprintf(`SELECT %s FROM "%s"%s%s`,
		colClause(*catCol), table, whereClause(*catWhere), orderClause(*catOrder))
	dio.Stream(dio.StreamParameters{
		Con: con, Stdout: stdout, Stderr: stderr,
		Debug: *catDebug, Nowarn: *catNowarn,
		Table: table, RowCap: 1000, Query: query,
	})
}
