package dio

import (
	"fmt"
	"io"
	"strings"

	"github.com/yznts/databox/pkg/db"
)

// Sql is a writer that writes data as SQL INSERT statements.
type Sql struct {
	w     io.Writer
	table string
}

// SetTable sets the table name for INSERT statements.
func (s *Sql) SetTable(name string) {
	s.table = name
}

// write wraps the io writer's Write method.
// If an error occurs, it panics.
// It's unexpected behavior in our case,
// so panic is necessary.
func (s *Sql) write(data []byte) {
	_, err := s.w.Write(data)
	if err != nil {
		panic(err)
	}
}

// Sql supports multiple writes.
func (s *Sql) MultiWrite() bool {
	return true
}

// WriteError writes an error as a SQL comment.
func (s *Sql) WriteError(err error) {
	s.write([]byte(fmt.Sprintf("-- ERROR: %s\n", err.Error())))
}

// WriteData writes data as SQL INSERT statements.
func (s *Sql) WriteData(data *db.Data) {
	table := s.table
	if table == "" {
		table = "data"
	}

	// Quote column names
	quotedCols := make([]string, len(data.Cols))
	for i, col := range data.Cols {
		quotedCols[i] = `"` + col + `"`
	}
	colList := strings.Join(quotedCols, ", ")

	// Write each row as an INSERT statement
	for _, row := range data.Rows {
		vals := make([]string, len(row))
		for i, v := range row {
			vals[i] = sqlValue(v)
		}
		s.write([]byte(fmt.Sprintf(
			"INSERT INTO \"%s\" (%s) VALUES (%s);\n",
			table, colList, strings.Join(vals, ", "),
		)))
	}
}

// WriteTable writes a CREATE TABLE statement derived from the given column metadata.
// The output format is SQL DDL, but the method name intentionally stays generic
// to leave room for other schema representations in other writer types.
// Primary keys and foreign keys are emitted as table-level constraints.
func (s *Sql) WriteTable(table string, columns []db.Column) {
	var colDefs []string
	var pks []string
	var fks []string

	for _, col := range columns {
		def := `    "` + col.Name + `" ` + col.Type
		if !col.IsNullable {
			def += " NOT NULL"
		} else {
			def += " NULL"
		}
		if col.Default != nil {
			def += " DEFAULT " + sqlValue(col.Default)
		}
		colDefs = append(colDefs, def)

		if col.IsPrimary {
			pks = append(pks, `"`+col.Name+`"`)
		}
		if col.ForeignRef != "" {
			fk := fmt.Sprintf(`    FOREIGN KEY ("%s") REFERENCES %s`, col.Name, col.ForeignRef)
			if col.ForeignOnUpdate != "" {
				fk += " ON UPDATE " + col.ForeignOnUpdate
			}
			if col.ForeignOnDelete != "" {
				fk += " ON DELETE " + col.ForeignOnDelete
			}
			fks = append(fks, fk)
		}
	}

	if len(pks) > 0 {
		colDefs = append(colDefs, "    PRIMARY KEY ("+strings.Join(pks, ", ")+")")
	}
	colDefs = append(colDefs, fks...)

	s.write([]byte(fmt.Sprintf(
		"CREATE TABLE \"%s\" (\n%s\n);\n",
		table, strings.Join(colDefs, ",\n"),
	)))
}

// sqlValue formats a value for use in a SQL INSERT statement.
func sqlValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", val)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case []uint8:
		return fmt.Sprintf("X'%x'", val)
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	default:
		s := fmt.Sprintf("%v", val)
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
}

// NewSql creates a new Sql writer.
func NewSql(w io.Writer) *Sql {
	return &Sql{w: w}
}
