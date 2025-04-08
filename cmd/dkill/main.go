package main

import (
	"flag"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/yznts/dsh/pkg/dconf"
	"github.com/yznts/dsh/pkg/ddb"
	"github.com/yznts/dsh/pkg/dio"
	"go.kyoto.codes/zen/v3/slice"
)

// Tool flags
var (
	// Default
	fdsn = flag.String("dsn", "", "Database connection (can be set via DSN/DATABASE/DATABASE_URL env)")
	// Options
	fforce  = flag.Bool("force", false, "Terminate the process, instead of graceful shutdown")
	fexceed = flag.Bool("exceed", false, "We're killing all processes exceeding a provided duration (Go time.Duration format)")
	fquery  = flag.Bool("query", false, "We're killing all processes for query regex")
	fuser   = flag.Bool("user", false, "We're killing all processes for username")
	fpid    = flag.Bool("pid", false, "We're killing a process by PID (default)")
	fdb     = flag.Bool("db", false, "We're killing all processes for database")
)

// Tool usage / description
var (
	fusage = "[flags...] <pid|duration|query|username|database>"
	fdescr = "The dkill utility kills processes, depending on the flag and argument provided."
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
	stdout = dio.Open(os.Stdout, false, false, false)
	stderr = dio.Open(os.Stderr, false, false, false)

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

	// Find out processes to kill
	kill := []ddb.Process{}
	switch {
	case *fpid:
		pid, err := strconv.Atoi(flag.Arg(0))
		dio.Assert(stderr, err, "provided PID is not a number")
		kill = slice.Filter(processes, func(p ddb.Process) bool {
			return p.Pid == pid
		})
		break
	case *fexceed:
		dur, err := time.ParseDuration(flag.Arg(0))
		dio.Assert(stderr, err, "provided duration is not a valid Go time.Duration")
		kill = slice.Filter(processes, func(p ddb.Process) bool {
			return p.Duration > dur
		})
		break
	case *fquery:
		rgx, err := regexp.Compile(flag.Arg(0))
		dio.Assert(stderr, err, "provided regex is not a valid Go regexp")
		kill = slice.Filter(processes, func(p ddb.Process) bool {
			return rgx.MatchString(p.Query)
		})
		break
	case *fuser:
		kill = slice.Filter(processes, func(p ddb.Process) bool {
			return p.Username == flag.Arg(0)
		})
		break
	case *fdb:
		kill = slice.Filter(processes, func(p ddb.Process) bool {
			return p.Database == flag.Arg(0)
		})
		break
	default:
		pid, err := strconv.Atoi(flag.Arg(0))
		dio.Assert(stderr, err, "provided PID is not a number")
		kill = slice.Filter(processes, func(p ddb.Process) bool {
			return p.Pid == pid
		})
		break
	}

	// Kill the processes
	statuses := map[int]error{}
	for _, p := range kill {
		statuses[p.Pid] = db.KillProcess(p.Pid, *fforce)
	}

	// Report the status
	stdout.WriteData(&ddb.Data{
		Cols: []string{"PID", "STATUS"},
		Rows: slice.Map(kill, func(p ddb.Process) []any {
			status := "Killed"
			if err := statuses[p.Pid]; err != nil {
				status = err.Error()
			}
			return []any{p.Pid, status}
		}),
	})
}
