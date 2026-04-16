package main

import (
	"flag"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
)

func Usage(fset *flag.FlagSet, usage, descr string) func() {
	return func() {
		// Let's limit the description width to 90 characters
		// to make it more readable.
		descr = lipgloss.NewStyle().Width(90).Render(descr)
		// Provide usage
		fmt.Fprintf(fset.Output(), "Usage: %s %s\n\n", os.Args[0], usage)
		// Provide description
		fmt.Fprintf(fset.Output(), "%s \n\n", descr)
		// Provide flags,
		// this handled by flag package.
		fset.PrintDefaults()
	}
}

// Awareness flags

func flagDebug(fset *flag.FlagSet) *bool {
	return fset.Bool("debug", false, "Enable debug mode with panic on error")
}

func flagNowarn(fset *flag.FlagSet) *bool {
	return fset.Bool("nowarn", false, "Ignore warnings (even in debug mode)")
}

// Data format flags

func flagDsn(fset *flag.FlagSet) *string {
	return fset.String("dsn", "", "dsn to connect to the database (can be set via DSN/DATABASE/DATABASE_URL/DATABOX env)")
}

func flagSql(fset *flag.FlagSet) *bool {
	return fset.Bool("sql", false, "Output as SQL statements")
}

func flagCsv(fset *flag.FlagSet) *bool {
	return fset.Bool("csv", false, "Output in CSV format")
}

func flagJson(fset *flag.FlagSet) *bool {
	return fset.Bool("json", false, "Output in JSON format")
}

func flagJsonl(fset *flag.FlagSet) *bool {
	return fset.Bool("jsonl", false, "Output in JSON Lines format")
}

// flagOrder adds an -order flag to a flag set.
func flagOrder(fset *flag.FlagSet) *string {
	return fset.String("order", "", "Column name to order by (prefix with - for descending, e.g. -id)")
}

// orderClause builds an ORDER BY clause from the order flag value.
// Returns an empty string if the flag is empty.
func orderClause(order string) string {
	if order == "" {
		return ""
	}
	if order[0] == '-' {
		return fmt.Sprintf(` ORDER BY "%s" DESC`, order[1:])
	}
	return fmt.Sprintf(` ORDER BY "%s" ASC`, order)
}
