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
	// Basic flags
	dsnDsn = flagDsn(dsnFlagSet)
	// Awareness flags
	dsnDebug  = flagDebug(dsnFlagSet)
	dsnNowarn = flagNowarn(dsnFlagSet)
	// Data format flags
	dsnCsv   = flagCsv(dsnFlagSet)
	dsnJson  = flagJson(dsnFlagSet)
	dsnJsonl = flagJsonl(dsnFlagSet)

	dsnUsage = "[options]"
	dsnDescr = "Outputs information about current database configuration."
)

func dsnCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{Csv: *dsnCsv, Json: *dsnJson, Jsonl: *dsnJsonl})
		stderr = dio.Open(os.Stderr, dio.Config{Csv: *dsnCsv, Json: *dsnJson, Jsonl: *dsnJsonl})
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
		// Get table count
		sm := con.(db.SchemaManager)
		tables, err := sm.GetTables()
		dio.AssertError(stderr, err, *dsnDebug, "Failed to query tables: %v")
		tableCount := 0
		for _, t := range tables {
			if !t.IsSystem {
				tableCount++
			}
		}
		// Write resulting information
		rows = [][]any{
			{"Driver", "sqlite"},
			{"Path", path},
			{"Schema Version", schemaVersion},
			{"Size", fmt.Sprintf("%d (%s)", size, sizeStr)},
			{"Tables", tableCount},
		}
	case *db.Postgres:
		c := con.GetConnection()
		// Get server version
		data, err := con.QueryData("SHOW server_version;")
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get server version: %v")
		version := conv.String(data.Rows[0][0])
		// Get current database name
		data, err = con.QueryData("SELECT current_database();")
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get current database: %v")
		dbName := conv.String(data.Rows[0][0])
		// Get current user
		data, err = con.QueryData("SELECT current_user;")
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get current user: %v")
		user := conv.String(data.Rows[0][0])
		// Get database size
		data, err = con.QueryData(fmt.Sprintf("SELECT pg_database_size('%s');", dbName))
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get database size: %v")
		size := conv.Int(data.Rows[0][0])
		sizeStr := humanize.Bytes(uint64(size))
		// Get table count
		data, err = con.QueryData("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema');")
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get table count: %v")
		tableCount := conv.Int(data.Rows[0][0])
		// Get active connections
		data, err = con.QueryData(fmt.Sprintf("SELECT COUNT(*) FROM pg_stat_activity WHERE datname = '%s';", dbName))
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get active connections: %v")
		activeConns := conv.Int(data.Rows[0][0])
		// Write resulting information
		rows = [][]any{
			{"Driver", "postgres"},
			{"Host", c.DSN.Host},
			{"Database", dbName},
			{"User", user},
			{"Version", version},
			{"Size", fmt.Sprintf("%d (%s)", size, sizeStr)},
			{"Tables", tableCount},
			{"Active Connections", activeConns},
		}
	case *db.Mysql:
		c := con.GetConnection()
		// Get server version
		data, err := con.QueryData("SELECT VERSION();")
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get server version: %v")
		version := conv.String(data.Rows[0][0])
		// Get current database name
		data, err = con.QueryData("SELECT DATABASE();")
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get current database: %v")
		dbName := conv.String(data.Rows[0][0])
		// Get current user
		data, err = con.QueryData("SELECT CURRENT_USER();")
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get current user: %v")
		user := conv.String(data.Rows[0][0])
		// Get database size
		data, err = con.QueryData(fmt.Sprintf("SELECT SUM(data_length + index_length) FROM information_schema.tables WHERE table_schema = '%s';", dbName))
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get database size: %v")
		size := conv.Int(data.Rows[0][0])
		sizeStr := humanize.Bytes(uint64(size))
		// Get table count
		data, err = con.QueryData(fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '%s';", dbName))
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get table count: %v")
		tableCount := conv.Int(data.Rows[0][0])
		// Get active connections
		data, err = con.QueryData("SELECT COUNT(*) FROM information_schema.processlist;")
		dio.AssertError(stderr, err, *dsnDebug, "Failed to get active connections: %v")
		activeConns := conv.Int(data.Rows[0][0])
		// Write resulting information
		rows = [][]any{
			{"Driver", "mysql"},
			{"Host", c.DSN.Host},
			{"Database", dbName},
			{"User", user},
			{"Version", version},
			{"Size", fmt.Sprintf("%d (%s)", size, sizeStr)},
			{"Tables", tableCount},
			{"Active Connections", activeConns},
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
