package db

import "strings"

// GenericType represents a database-agnostic type category.
type GenericType int

const (
	TypeInteger GenericType = iota
	TypeBigInt
	TypeSmallInt
	TypeFloat
	TypeDouble
	TypeDecimal
	TypeBoolean
	TypeChar
	TypeVarchar
	TypeText
	TypeBlob
	TypeDate
	TypeTime
	TypeTimestamp
	TypeDatetime
	TypeJSON
)

// toGeneric maps known source type names (lowercased) to generic types.
var toGeneric = map[string]GenericType{
	// Integer types
	"integer":   TypeInteger,
	"int":       TypeInteger,
	"int4":      TypeInteger,
	"mediumint": TypeInteger,
	"tinyint":   TypeSmallInt,
	"smallint":  TypeSmallInt,
	"int2":      TypeSmallInt,
	"bigint":    TypeBigInt,
	"int8":      TypeBigInt,

	// Float types
	"real":             TypeFloat,
	"float":            TypeFloat,
	"float4":           TypeFloat,
	"double":           TypeDouble,
	"double precision": TypeDouble,
	"float8":           TypeDouble,
	"numeric":          TypeDecimal,
	"decimal":          TypeDecimal,

	// Boolean types
	"boolean": TypeBoolean,
	"bool":    TypeBoolean,

	// String types
	"char":              TypeChar,
	"character":         TypeChar,
	"varchar":           TypeVarchar,
	"character varying": TypeVarchar,
	"text":              TypeText,
	"tinytext":          TypeText,
	"mediumtext":        TypeText,
	"longtext":          TypeText,
	"clob":              TypeText,

	// Binary types
	"blob":       TypeBlob,
	"tinyblob":   TypeBlob,
	"mediumblob": TypeBlob,
	"longblob":   TypeBlob,
	"bytea":      TypeBlob,
	"binary":     TypeBlob,
	"varbinary":  TypeBlob,

	// Date/time types
	"date":                        TypeDate,
	"time":                        TypeTime,
	"time without time zone":      TypeTime,
	"time with time zone":         TypeTime,
	"timestamp":                   TypeTimestamp,
	"timestamp without time zone": TypeTimestamp,
	"timestamp with time zone":    TypeTimestamp,
	"datetime":                    TypeDatetime,

	// JSON types
	"json":  TypeJSON,
	"jsonb": TypeJSON,

	// Database-specific types (mapped to text as safe fallback)
	"user-defined": TypeText,
	"array":        TypeText,
	"tsvector":     TypeText,
}

// fromGeneric maps generic types to concrete type names per database driver.
var fromGeneric = map[string]map[GenericType]string{
	SqliteDriver: {
		TypeInteger:   "INTEGER",
		TypeBigInt:    "INTEGER",
		TypeSmallInt:  "INTEGER",
		TypeFloat:     "REAL",
		TypeDouble:    "REAL",
		TypeDecimal:   "REAL",
		TypeBoolean:   "INTEGER",
		TypeChar:      "TEXT",
		TypeVarchar:   "TEXT",
		TypeText:      "TEXT",
		TypeBlob:      "BLOB",
		TypeDate:      "TEXT",
		TypeTime:      "TEXT",
		TypeTimestamp: "TEXT",
		TypeDatetime:  "TEXT",
		TypeJSON:      "TEXT",
	},
	PostgresDriver: {
		TypeInteger:   "INTEGER",
		TypeBigInt:    "BIGINT",
		TypeSmallInt:  "SMALLINT",
		TypeFloat:     "REAL",
		TypeDouble:    "DOUBLE PRECISION",
		TypeDecimal:   "NUMERIC",
		TypeBoolean:   "BOOLEAN",
		TypeChar:      "CHAR",
		TypeVarchar:   "VARCHAR",
		TypeText:      "TEXT",
		TypeBlob:      "BYTEA",
		TypeDate:      "DATE",
		TypeTime:      "TIME",
		TypeTimestamp: "TIMESTAMP",
		TypeDatetime:  "TIMESTAMP",
		TypeJSON:      "JSONB",
	},
	MysqlDriver: {
		TypeInteger:   "INT",
		TypeBigInt:    "BIGINT",
		TypeSmallInt:  "SMALLINT",
		TypeFloat:     "FLOAT",
		TypeDouble:    "DOUBLE",
		TypeDecimal:   "DECIMAL",
		TypeBoolean:   "TINYINT(1)",
		TypeChar:      "CHAR",
		TypeVarchar:   "VARCHAR(255)",
		TypeText:      "TEXT",
		TypeBlob:      "BLOB",
		TypeDate:      "DATE",
		TypeTime:      "TIME",
		TypeTimestamp: "TIMESTAMP",
		TypeDatetime:  "DATETIME",
		TypeJSON:      "JSON",
	},
}

// MapType translates a column type from one database to another.
// If the type is not recognized, it is returned as-is.
func MapType(srcType, dstScheme string) string {
	normalized := strings.ToLower(strings.TrimSpace(srcType))

	// Strip length/precision specifiers for lookup (e.g. "varchar(100)" -> "varchar")
	base := normalized
	if idx := strings.IndexByte(base, '('); idx != -1 {
		base = strings.TrimSpace(base[:idx])
	}

	generic, ok := toGeneric[base]
	if !ok {
		// Unknown type — pass through as-is
		return srcType
	}

	dstMap, ok := fromGeneric[dstScheme]
	if !ok {
		return srcType
	}

	return dstMap[generic]
}
