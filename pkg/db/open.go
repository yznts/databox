package db

import (
	"database/sql"
	"errors"
	"net/url"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Open opens a database connection based on the provided DSN.
// Use GetDsn before calling Open to get a valid normalized DSN.
func Open(dsn string) (Database, error) {
	// Validate and parse dsn
	if dsn == "" {
		return nil, errors.New("empty DSN")
	}
	dsnurl, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	// Resolve connection and actual scheme, depending on the provided DSN scheme
	switch dsnurl.Scheme {

	case SqliteDriver:
		// To open a SQLite database, we need to remove the scheme and leading slashes
		_dsnurl, _ := url.Parse(dsn)
		_dsnurl.Scheme = ""
		_dsnurlstr := strings.ReplaceAll(_dsnurl.String(), "//", "")
		// Open sql database connection
		sqldb, err := sql.Open("sqlite", _dsnurlstr)
		if err != nil {
			return nil, err
		}
		// Compose the database object
		return &Sqlite{
			Connection: &Connection{
				DB:     sqldb,
				DSN:    dsnurl,
				Scheme: "sqlite",
			},
		}, nil

	case PostgresDriver:
		// Open sql database connection
		sqldb, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		// Compose the database object
		return &Postgres{
			Connection: &Connection{
				DB:     sqldb,
				DSN:    dsnurl,
				Scheme: "postgres",
			},
		}, nil

	case MysqlDriver:
		// We're using url-formatted DSNs.
		// MySQL is "special" in our case.
		// - We need to remove the driver prefix from the DSN
		// - We need to specify the protocol in non-url way, so
		//   url parsing of the url-like mysql DSN will not work
		//   (example: user:password@tcp(host:port)/dbname)
		//
		// For initial MySQL support we'll try to avoid complex parsing
		// and will stick to the non-standard DSN format.
		// Example: mysql://user:password@host:port/dbname
		//
		// So, we're implicitly setting tcp protocol
		// and removing the scheme from the DSN.

		// First, let's parse the DSN
		_dsnurl, _ := url.Parse(dsn)
		// Remove the scheme
		_dsnurl.Scheme = ""
		// Wrap host and port in tcp() protocol
		_dsnurl.Host = "tcp(" + _dsnurl.Host + ")"
		_dsnurlstr := strings.ReplaceAll(_dsnurl.String(), "//", "")
		// Open sql database connection
		sqldb, err := sql.Open("mysql", _dsnurlstr)
		if err != nil {
			return nil, err
		}
		// Compose the database object
		return &Mysql{
			Connection: &Connection{
				DB:     sqldb,
				DSN:    dsnurl,
				Scheme: "mysql",
			},
		}, nil

	default:
		return nil, errors.New("unsupported database")
	}
}
