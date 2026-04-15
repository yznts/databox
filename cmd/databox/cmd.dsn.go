package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/databox/pkg/dio"
	"github.com/yznts/zen/v3/conv"
)

var (
	dsnFlagSet = flag.NewFlagSet("dsn", flag.ExitOnError)
	// Awareness flags
	dsnDebug  = flagDebug(dsnFlagSet)
	dsnNowarn = flagNowarn(dsnFlagSet)
	// Data format flags
	dsnDsn   = flagDsn(dsnFlagSet)
	dsnCsv   = flagCsv(dsnFlagSet)
	dsnJson  = flagJson(dsnFlagSet)
	dsnJsonl = flagJsonl(dsnFlagSet)

	dsnUsage = "[options]"
	dsnDescr = "Outputs information about current database configuration."
)

func dsnCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, false, *dsnCsv, *dsnJson, *dsnJsonl)
		stderr = dio.Open(os.Stderr, false, *dsnCsv, *dsnJson, *dsnJsonl)
	)
	// Open database connection
	dsn, err := db.GetDsn(*dsnDsn)
	dio.AssertError(stderr, err, *dsnDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *dsnDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	// Define resulting cols/rows holders.
	var (
		cols = []string{"ATTR", "VALUE"}
		rows = [][]any{}
	)
	// Swtich behavior based on the database type.
	switch con.(type) {
	case *db.Sqlite:
		// Get database path and size
		path := db.GetSqlitePath(dsn)
		stat, err := os.Stat(path)
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get database file info: %v")
		size := stat.Size()
		sizeStr := humanize.Bytes(uint64(size))
		// Get schema version
		data, err := con.QueryData("PRAGMA schema_version;")
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get schema version: %v")
		schemaVersion := conv.Int(data.Rows[0][0])
		// Write resulting information
		rows = [][]any{
			{"Driver", "sqlite"},
			{"Path", db.GetSqlitePath(dsn)},
			{"Version", schemaVersion},
			{"Size", fmt.Sprintf("%d (%s)", size, sizeStr)},
		}
	default:
		dio.AssertError(stderr, errors.New("unsupported database type for this tool"), *dsnDebug)
	}

	// Output resulting information
	stdout.WriteData(&db.Data{
		Cols: cols,
		Rows: rows,
	})
}
