package main

import (
	"flag"
	"os"

	"github.com/yznts/dsh/pkg/dconf"
	"github.com/yznts/dsh/pkg/ddb"
	"github.com/yznts/dsh/pkg/dio"
)

// Tool flags
var (
	// Default
	fdsn = flag.String("dsn", "", "Database connection (can be set via DSN/DATABASE/DATABASE_URL env)")
	// Options
	frpc = flag.String("rpc", ":25123", "RPC server address to listen on")
)

// Tool usage / description
var (
	fusage = "[flags...]"
	fdescr = "The dconn utility is a simple intermediary between the database and the client. " +
		"Supports the same databases as pkg/ddb allows. " +
		"Client also included in pkg/ddb. " +
		"Replicates the same method set as ddb.Database provides (for compatibility). " +
		"Main purpose is to provide an ability to communicate with database without using additional drivers/libraries. " +
		"It might be useful for cases when driver is not supported, or you don't want to import a driver at all for some reason."
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
	stdout = dio.Open(os.Stdout)
	stderr = dio.Open(os.Stderr)

	// Resolve dsn and database connection
	dsn, err := dconf.GetDsn(*fdsn)
	dio.Assert(stderr, err)
	db, err = ddb.Open(dsn)
	dio.Assert(stderr, err)

	// Start rpc server
	rpcserver(*frpc).Await()
}
