package main

import (
	"flag"
	"io"
	"os"

	"github.com/yznts/dsh/pkg/dconf"
	"github.com/yznts/dsh/pkg/ddb"
	"github.com/yznts/dsh/pkg/dio"
	"go.kyoto.codes/zen/v3/slice"
)

// Tool flags
var (
	// Default
	fdsn = flag.String("dsn", "", "Database connection (can be set via DSN/DATABASE/DATABASE_URL env)")
	// Formats
	fcsv   = flag.Bool("csv", false, "Output in CSV format")
	fjson  = flag.Bool("json", false, "Output in JSON format")
	fjsonl = flag.Bool("jsonl", false, "Output in JSON lines format")
)

// Tool usage / description
var (
	fusage = "[flags...]"
	fdescr = "The dps utility outputs list of database processes."
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

	// Resolve output writer
	stdout = dio.Open(os.Stdout, *fcsv, *fjson, *fjsonl)
	stderr = dio.Open(os.Stderr, *fcsv, *fjson, *fjsonl)

	// Resolve dsn and database connection
	dsn, err := dconf.GetDsn(*fdsn)
	dio.Assert(stderr, err)
	db, err = ddb.Open(dsn)
	dio.Assert(stderr, err)
	if db, iscloser := db.(io.Closer); iscloser {
		defer db.Close()
	}

	// Query the database for the currently running processes
	processes, err := db.QueryProcesses()
	dio.Assert(stderr, err)

	// Write processes
	stdout.WriteData(&ddb.Data{
		Cols: []string{"PID", "DURATION", "USERNAME", "DATABASE", "QUERY"},
		Rows: slice.Map(processes, func(p ddb.Process) []any {
			return []any{p.Pid, p.Duration, p.Username, p.Database, p.Query}
		}),
	})
}
