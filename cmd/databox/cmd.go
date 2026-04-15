package main

import "flag"

type SubCommand struct {
	fset  *flag.FlagSet
	usage string
	descr string
	run   func()
}

var SubCommands = map[string]SubCommand{
	"dsn":  {dsnFlagSet, dsnUsage, dsnDescr, dsnCmd},
	"sql":  {sqlFlagSet, sqlUsage, sqlDescr, sqlCmd},
	"ls":   {lsFlagSet, lsUsage, lsDescr, lsCmd},
	"grep": {grepFlagSet, grepUsage, grepDescr, grepCmd},
	"cp":   {cpFlagSet, cpUsage, cpDescr, cpCmd},
	"head": {headFlagSet, headUsage, headDescr, headCmd},
	"tail": {tailFlagSet, tailUsage, tailDescr, tailCmd},
	"cat":  {catFlagSet, catUsage, catDescr, catCmd},
	"ps":   {psFlagSet, psUsage, psDescr, psCmd},
	"kill": {killFlagSet, killUsage, killDescr, killCmd},
}
