package main

import (
	"errors"
	"flag"
	"io"
	"os"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/databox/pkg/dio"
	"github.com/yznts/zen/v3/slice"
)

var (
	psFlagSet = flag.NewFlagSet("ps", flag.ExitOnError)
	// Basic flags
	psDsn = flagDsn(psFlagSet)
	// Awareness flags
	psDebug  = flagDebug(psFlagSet)
	psNowarn = flagNowarn(psFlagSet)
	// Data format flags
	psSql   = flagSql(psFlagSet)
	psCsv   = flagCsv(psFlagSet)
	psJson  = flagJson(psFlagSet)
	psJsonl = flagJsonl(psFlagSet)

	psUsage = "[options]"
	psDescr = "Outputs list of database processes."
)

func psCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Sql: *psSql, Csv: *psCsv, Json: *psJson, Jsonl: *psJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Sql: *psSql, Csv: *psCsv, Json: *psJson, Jsonl: *psJsonl})
	)
	// Open database connection
	dsn, err := db.GetDsn(*psDsn)
	dio.AssertError(stderr, err, *psDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *psDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	pm, ok := con.(db.ProcessManager)
	if !ok {
		dio.AssertError(stderr, errors.New("database does not support process management"), *psDebug)
		return
	}

	// Query processes
	processes, err := pm.GetProcesses()
	dio.AssertError(stderr, err, *psDebug, "Failed to query processes: %v")

	// Write processes
	stdout.WriteData(&db.Data{
		Cols: []string{"PID", "DURATION", "USERNAME", "DATABASE", "QUERY"},
		Rows: slice.Map(processes, func(p db.Process) []any {
			return []any{p.Pid, p.Duration, p.Username, p.Database, p.Query}
		}),
	})
}
