package main

import (
	"errors"
	"flag"
	"io"
	"os"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/databox/pkg/dio"
)

var (
	grepFlagSet = flag.NewFlagSet("grep", flag.ExitOnError)
	// Awareness flags
	grepDebug  = flagDebug(grepFlagSet)
	grepNowarn = flagNowarn(grepFlagSet)
	// Data format flags
	grepDsn   = flagDsn(grepFlagSet)
	grepSql   = flagSql(grepFlagSet)
	grepCsv   = flagCsv(grepFlagSet)
	grepJson  = flagJson(grepFlagSet)
	grepJsonl = flagJsonl(grepFlagSet)

	grepUsage = "[options] <table> <pattern>"
	grepDescr = "Searches for pattern in the database rows and outputs matching ones. Uses all columns for matching by default."
)

func grepCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Sql: *grepSql, Csv: *grepCsv, Json: *grepJson, Jsonl: *grepJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Sql: *grepSql, Csv: *grepCsv, Json: *grepJson, Jsonl: *grepJsonl})
	)
	// Open database connection
	dsn, err := db.GetDsn(*grepDsn)
	dio.AssertError(stderr, err, *grepDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *grepDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	// Get table and pattern arguments
	if grepFlagSet.NArg() < 2 {
		dio.AssertError(stderr, errors.New("table and pattern arguments are required"), *grepDebug)
	}
	table := grepFlagSet.Arg(0)
	pattern := grepFlagSet.Arg(1)

	// Get table columns
	cols, err := con.QueryColumns(table)
	dio.AssertError(stderr, err, *grepDebug, "Failed to get columns for table %s: %v")

	// Build query to search for pattern in all columns
	query := "SELECT * FROM " + table + " WHERE "
	for i, col := range cols {
		if i > 0 {
			query += " OR "
		}
		query += col.Name + " LIKE '%" + pattern + "%'"
	}

	// Execute query and output results
	data, err := con.QueryData(query)
	dio.AssertError(stderr, err, *grepDebug, "Failed to execute query: %v")
	if tw, ok := stdout.(dio.TableWriter); ok {
		tw.SetTable(table)
	}
	stdout.WriteData(data)
}
