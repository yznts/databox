package main

import "flag"

type SubCommand struct {
	fset  *flag.FlagSet
	usage string
	descr string
	run   func()
}

var SubCommands = map[string]SubCommand{
	"dsn":   {dsnFlagSet, dsnUsage, dsnDescr, dsnCmd},
	"ls":    {lsFlagSet, lsUsage, lsDescr, lsCmd},
	"sql":   {sqlFlagSet, sqlUsage, sqlDescr, sqlCmd},
	"cat":   {catFlagSet, catUsage, catDescr, catCmd},
	"grep":  {grepFlagSet, grepUsage, grepDescr, grepCmd},
	"head":  {headFlagSet, headUsage, headDescr, headCmd},
	"tail":  {tailFlagSet, tailUsage, tailDescr, tailCmd},
	"count": {countFlagSet, countUsage, countDescr, countCmd},
	"cp":    {cpFlagSet, cpUsage, cpDescr, cpCmd},
	"ps":    {psFlagSet, psUsage, psDescr, psCmd},
	"kill":  {killFlagSet, killUsage, killDescr, killCmd},
}
