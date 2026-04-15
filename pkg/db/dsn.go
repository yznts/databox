package db

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/yznts/zen/v3/logic"
)

var (
	SqliteDriver   = "sqlite"
	PostgresDriver = "postgres"
	MysqlDriver    = "mysql"
)

// GetDsn is a common dsn resolver.
// It tries to resolve provided dsn string to the actual dsn.
// Sometimes it might be empty (expecting env resolving).
// Also, it normalizes schema/driver names (e.g. postgres -> postgresql).
func GetDsn(dsn string) (string, error) {
	// If dsn is empty, try to resolve it from environment variable
	dsn = logic.Or(dsn,
		os.Getenv("DSN"),
		os.Getenv("DATABASE"),
		os.Getenv("DATABASE_URL"),
		os.Getenv("DATABOX"))
	// If it's still empty, return an error
	if dsn == "" {
		return "", errors.New("dsn is empty")
	}
	// Parse dsn
	dsnurl, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	// Normalize driver/schema names
	switch dsnurl.Scheme {
	case "postgres", "postgresql":
		dsnurl.Scheme = PostgresDriver
	case "mysql", "mariadb":
		dsnurl.Scheme = MysqlDriver
	case "sqlite", "sqlite3":
		dsnurl.Scheme = SqliteDriver
	}
	// Return the normalized dsn string
	return dsnurl.String(), nil
}

// GetSqlitePath extracts the file path from a sqlite dsn string.
func GetSqlitePath(dsn string) string {
	_dsnurl, _ := url.Parse(dsn)
	_dsnurl.Scheme = ""
	_dsnurlstr := strings.ReplaceAll(_dsnurl.String(), "//", "")
	return _dsnurlstr
}
