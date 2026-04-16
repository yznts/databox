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

	// Build parameterized query to search for pattern in all columns.
	// Using $N placeholders for PostgreSQL, ? for MySQL/SQLite.
	query, args := buildGrepQuery(table, cols, con.GetConnection().Scheme, pattern)

	// Execute query and output results
	dio.Stream(dio.StreamParameters{
		Con: con, Stdout: stdout, Stderr: stderr,
		Debug: *grepDebug, Nowarn: *grepNowarn,
		Table: table, RowCap: 1000, Query: query, Args: args,
	})
}

// buildGrepQuery builds a parameterized LIKE query across all columns of a table.
// It uses $N placeholders for PostgreSQL and ? for MySQL/SQLite.
// For PostgreSQL, columns are cast to text so LIKE works on non-text types (e.g. integers).
func buildGrepQuery(table string, cols []db.Column, scheme, pattern string) (string, []any) {
	like := "%" + pattern + "%"
	parts := make([]string, len(cols))
	args := make([]any, len(cols))
	for i, col := range cols {
		var expr string
		if scheme == "postgres" {
			expr = fmt.Sprintf(`"%s"::text LIKE $%d`, col.Name, i+1)
		} else {
			expr = `"` + col.Name + `" LIKE ?`
		}
		parts[i] = expr
		args[i] = like
	}
	query := fmt.Sprintf(`SELECT * FROM "%s" WHERE %s`, table, strings.Join(parts, " OR "))
	return query, args
}
