package db

import "strings"

// MapDefault translates a column default value for the destination database.
// Database-specific expressions (e.g. nextval(), now()) are mapped to
// equivalents or dropped if the destination has no equivalent.
func MapDefault(def any, dstScheme string) any {
	if def == nil {
		return nil
	}
	s, ok := def.(string)
	if !ok {
		return def
	}
	normalized := strings.ToLower(strings.TrimSpace(s))

	// Detect sequence/autoincrement defaults — drop them,
	// they are handled by the type system (AUTOINCREMENT/SERIAL).
	if strings.HasPrefix(normalized, "nextval(") {
		return nil
	}

	// Map common timestamp functions
	switch {
	case normalized == "now()" ||
		normalized == "current_timestamp" ||
		normalized == "('now'::text)::date":
		switch dstScheme {
		case SqliteDriver:
			return "CURRENT_TIMESTAMP"
		case PostgresDriver:
			return "NOW()"
		case MysqlDriver:
			return "CURRENT_TIMESTAMP"
		}
	case normalized == "current_date":
		switch dstScheme {
		case SqliteDriver:
			return "CURRENT_DATE"
		case PostgresDriver:
			return "CURRENT_DATE"
		case MysqlDriver:
			return "CURRENT_DATE"
		}
	case normalized == "true" || normalized == "false":
		switch dstScheme {
		case SqliteDriver:
			if normalized == "true" {
				return "1"
			}
			return "0"
		default:
			return s
		}
	}

	// Drop any remaining function calls the destination won't understand
	// (e.g. postgres casts like '...'::type)
	if strings.Contains(normalized, "::") {
		return nil
	}

	return def
}
