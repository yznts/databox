package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	usage = "<sub-command> [options]"
	descr = "A collection of tools for working with database in command line."
)

func main() {
	// Complete description with sub-commands information.
	descr += "\n\nAvailable sub-commands:"
	for name, cmd := range SubCommands {
		descr += fmt.Sprintf("\n  - %s\t%s", name, cmd.descr)
	}
	// Provide usage information for the main command.
	flag.Usage = Usage(flag.CommandLine, usage, descr)
	// Parse the main command.
	flag.Parse()

	// Extract the sub-command name.
	name := flag.Arg(0)

	// Look up the sub-command.
	cmd, ok := SubCommands[name]
	if !ok {
		flag.Usage()
		os.Exit(1)
	}

	// Prepare sub-command and execute it.
	cmd.fset.Usage = Usage(cmd.fset, cmd.usage, cmd.descr)
	cmd.fset.Parse(os.Args[2:])
	cmd.run()
}
