package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yznts/dsh/pkg/dconf"
	"github.com/yznts/dsh/pkg/ddb"
	"github.com/yznts/dsh/pkg/dio"
)

// Tool flags
var (
	// Default
	fdsn = flag.String("dsn", "", "Database connection (can be set via DSN/DATABASE/DATABASE_URL env)")
	// Formats
	fsql   = flag.Bool("sql", false, "Output in SQL format")
	fcsv   = flag.Bool("csv", false, "Output in CSV format")
	fjsonl = flag.Bool("jsonl", false, "Output in JSON lines format")
	// Options
	fwhere = flag.String("where", "", "WHERE clause")
)

// Tool usage / description
var (
	fusage = "[flags...] table"
	fdescr = "The dcat utility reads table data and writes it to the standard output in desired format. " +
		"Because of chunked data fetching, output options might be limited. " +
		"Utility tries to avoid accumulating data in the memory. " +
		"If dcat output options are not enough and memory usage is not a concern, consider using dsql instead."
)

// Database connection
var db ddb.Database

// Output writers
var (
	stdout dio.Writer
	stderr dio.Writer
)

// Simplify assignments
var err error

func main() {
	// Provide usage
	flag.Usage = dio.Usage(fusage, fdescr)

	// Parse flags
	flag.Parse()

	// Resolve output writer.
	//
	// We are using only multiline writers here
	// because we are going to query the database in chunks
	// and write the result in chunks as well.
	// Otherwise, we will have to store the whole result in memory.
	//
	// The only exception is gloss writer.
	// We will limit the output to 1k rows in that case.
	stdout = dio.Open(os.Stdout, *fsql, *fcsv, false, *fjsonl)
	stderr = dio.Open(os.Stderr, *fsql, *fcsv, false, *fjsonl)

	// Determine if the output format supports multiple writes.
	// Otherwise, we are limiting the output to 1k rows with a warning.
	limited := false
	if !stdout.Multi() {
		limited = true
		// Let's also check if the output format is gloss.
		// Other formats are not supposed to be used without multiple writes.
		if _, gloss := stdout.(*dio.Gloss); !gloss {
			dio.Assert(stderr, errors.New("output format does not support multiple writes"))
		}
	}

	// Resolve dsn and database connection
	dsn, err := dconf.GetDsn(*fdsn)
	dio.Assert(stderr, err)
	db, err = ddb.Open(dsn)
	dio.Assert(stderr, err)
	if db, iscloser := db.(io.Closer); iscloser {
		defer db.Close()
	}

	// Extract table name from arguments
	table := flag.Arg(0)
	if table == "" {
		dio.Assert(stderr, errors.New("missing table name"))
	}

	// Get rows count
	data, err := db.QueryData(fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	dio.Assert(stderr, err)
	count := int(data.Rows[0][0].(int64))

	// If the total rows count is less than 1k, we can get back to non-limited mode.
	if count < 1000 {
		limited = false
	}

	// Make offsets list
	offsets := []int{}
	for offset := 0; offset < count; offset += 1000 {
		offsets = append(offsets, offset)
	}

	// If we are limited, we need only first chunk.
	// Also, we need to warn the user about it.
	if limited {
		offsets = offsets[:1]
		if stdout, warner := stdout.(dio.WarningWriter); warner {
			stdout.WriteWarning("output is limited to 1k rows")
		}
	}

	// If writer is SQL, we're setting appropriate mode and table name
	if stdout, ok := stdout.(*dio.Sql); ok {
		stdout.SetMode("data")
		stdout.SetTable(table)
	}

	// Iterate over chunks and query the database
	for _, offset := range offsets {
		// Compose limit/offset query with WHERE clause
		query := &strings.Builder{}
		query.WriteString(fmt.Sprintf("SELECT * FROM %s ", table))
		if *fwhere != "" {
			query.WriteString(fmt.Sprintf("WHERE %s ", *fwhere))
		}
		query.WriteString(fmt.Sprintf("LIMIT 1000 OFFSET %d", offset))
		// Execute query
		data, err := db.QueryData(query.String())
		dio.Assert(stderr, err)
		// Don't collect the data and just write it to the output,
		// because we don't want to keep it in memory.
		// That's why we are requiring closable writers here.
		stdout.WriteData(data)
	}
}
