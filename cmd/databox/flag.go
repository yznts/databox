package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
	return fset.String("dsn", "", "database URL; if empty, uses DSN, DATABASE, DATABASE_URL, or DATABOX env (first non-empty)")
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

// flagCol adds a -col flag to a flag set for column selection.
func flagCol(fset *flag.FlagSet) *string {
	return fset.String("col", "", "Comma-separated column names to select (default: all)")
}

// flagWhere adds a -where flag to a flag set for row filtering.
func flagWhere(fset *flag.FlagSet) *string {
	return fset.String("where", "", "SQL WHERE expression to filter rows (e.g. 'id > 5')")
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

// colClause builds a SELECT column list from a comma-separated column string.
// Returns * if empty.
func colClause(col string) string {
	if col == "" {
		return "*"
	}
	parts := strings.Split(col, ",")
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = `"` + strings.TrimSpace(p) + `"`
	}
	return strings.Join(quoted, ", ")
}

// whereClause builds a WHERE clause from a raw filter expression.
// Returns an empty string if the expression is empty.
func whereClause(where string) string {
	if where == "" {
		return ""
	}
	return " WHERE " + where
}

// flipOrder reverses the sort direction in an order flag value.
// Used by tail to reverse the sort for the inner subquery.
func flipOrder(order string) string {
	if order == "" {
		return ""
	}
	if order[0] == '-' {
		return order[1:]
	}
	return "-" + order
}
