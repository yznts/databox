package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/yznts/databox/pkg/db"
	"github.com/yznts/databox/pkg/dio"
	"github.com/yznts/zen/v3/slice"
)

var (
	killFlagSet = flag.NewFlagSet("kill", flag.ExitOnError)
	// Awareness flags
	killDebug  = flagDebug(killFlagSet)
	killNowarn = flagNowarn(killFlagSet)
	// Data format flags
	killDsn = flagDsn(killFlagSet)
	// Type flags
	killForce  = killFlagSet.Bool("force", false, "Terminate the process, instead of graceful shutdown")
	killExceed = killFlagSet.Bool("exceed", false, "Kill all processes exceeding a provided duration (Go time.Duration format)")
	killQuery  = killFlagSet.Bool("query", false, "Kill all processes matching a query regex")
	killUser   = killFlagSet.Bool("user", false, "Kill all processes for username")
	killPid    = killFlagSet.Bool("pid", false, "Kill a process by PID (default)")
	killDb     = killFlagSet.Bool("db", false, "Kill all processes for database")

	killUsage = "[options] [kill-type] <pid|duration|query|username|database>"
	killDescr = "Kills database processes, depending on the flag and argument provided."
)

func killCmd() {
	// Open stdout/stderr for output
	var (
		stdout = dio.Open(os.Stdout, dio.Config{})
		stderr = dio.Open(os.Stderr, dio.Config{})
	)
	// Open database connection
	dsn, err := db.GetDsn(*killDsn)
	dio.AssertError(stderr, err, *killDebug, "Failed to get dsn: %v")
	con, err := db.Open(dsn)
	dio.AssertError(stderr, err, *killDebug, "Failed to connect to database: %v")
	if con, isCloser := con.(io.Closer); isCloser {
		defer con.Close()
	}

	pm, ok := con.(db.ProcessManager)
	if !ok {
		dio.AssertError(stderr, errors.New("database does not support process management"), *killDebug)
		return
	}

	// Query the database for currently running processes
	processes, err := pm.GetProcesses()
	dio.AssertError(stderr, err, *killDebug, "Failed to query processes: %v")

	// Find out processes to kill
	var kill []db.Process
	switch {
	case *killExceed:
		dur, err := time.ParseDuration(killFlagSet.Arg(0))
		dio.AssertError(stderr, err, *killDebug, "Provided duration is not a valid Go time.Duration: %v")
		kill = slice.Filter(processes, func(p db.Process) bool {
			return p.Duration > dur
		})
	case *killQuery:
		rgx, err := regexp.Compile(killFlagSet.Arg(0))
		dio.AssertError(stderr, err, *killDebug, "Provided regex is not valid: %v")
		kill = slice.Filter(processes, func(p db.Process) bool {
			return rgx.MatchString(p.Query)
		})
	case *killUser:
		kill = slice.Filter(processes, func(p db.Process) bool {
			return p.Username == killFlagSet.Arg(0)
		})
	case *killDb:
		kill = slice.Filter(processes, func(p db.Process) bool {
			return p.Database == killFlagSet.Arg(0)
		})
	case *killPid:
		pid, err := strconv.Atoi(killFlagSet.Arg(0))
		dio.AssertError(stderr, err, *killDebug, "Provided PID is not a number: %v")
		kill = slice.Filter(processes, func(p db.Process) bool {
			return p.Pid == pid
		})
	default:
		dio.AssertError(stderr, err, *killDebug, "No valid kill type flag provided")
	}

	// Kill the processes
	statuses := map[int]error{}
	for _, p := range kill {
		statuses[p.Pid] = pm.KillProcess(p.Pid, *killForce)
	}

	// Report the status
	stdout.WriteData(&db.Data{
		Cols: []string{"PID", "STATUS"},
		Rows: slice.Map(kill, func(p db.Process) []any {
			status := "Killed"
			if err := statuses[p.Pid]; err != nil {
				status = err.Error()
			}
			return []any{p.Pid, status}
		}),
	})
}
