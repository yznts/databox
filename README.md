<h1 align="center">databox</h1>

<p align="center">
  A Swiss Army knife for databases in the command line
</p>

```bash
go install github.com/yznts/databox/cmd/databox@latest
```

Main goal of the project is to provide a tool
to work with databases in a unified way,
avoiding differences in UX between clients like `psql`, `sqlite3`, `mysql`, etc.

It tries to stick with the UNIX-like naming and approach,
where each sub-command does one thing and does it well.
List database tables, or table columns? Just use `databox ls`.
Get table contents? Use `databox cat`.
Or, if you need just to execute an SQL query, `databox sql` is here for you.
Want to get the output in JSON, JSONL, CSV, or even SQL INSERT statements?
No problem, just specify an according flag, like `-json` or `-csv`.

Available sub-commands:
- `ls`   - lists database tables or table columns
- `sql`  - executes SQL queries
- `cat`  - outputs all rows of a table
- `head` - outputs the first N rows of a table
- `tail` - outputs the last N rows of a table
- `grep` - searches for pattern in database rows
- `cp`   - copies schema/data between databases
- `ps`   - lists database processes (if supported by the database)
- `kill` - kills database processes (if supported by the database)
- `dsn`  - resolves and outputs the current DSN

May be used with:
- `sqlite`
- `postgresql`
- `mysql` (no certificates support yet)

And supports these output formats:
- `gloss` (default beautified terminal output)
- `json`
- `jsonl`
- `csv`
- `sql` (INSERT statements)

## Installation

You have multiple ways to install/use this tool:
- Install in Go-way
- Build by yourself
- Download binaries

### Install in Go-way

This is the easiest way to install,
but you need to have Go installed on your machine,
including `GOBIN` in your `PATH`.

```bash
go install github.com/yznts/databox/cmd/databox@latest
```

### Build by yourself

This way still requires Go to be installed on your machine,
but it's up to you to decide where to put the binary.

```bash
git clone git@github.com:yznts/databox.git
cd databox
go build -o databox ./cmd/databox/
# Feel free to move the binary to the desired location, e.g. /usr/local/bin.
```

### Download binaries

Also you have an option to download the latest binaries from the
[Releases](https://github.com/yznts/databox/releases) page.
Please note, that darwin(macos) binaries are not signed!

## Usage

Each sub-command has its own help message, which you can get by
running it with `-h` flag. From there you can understand the sub-command purpose,
how to use it, and what flags are available.

```bash
databox ls -h
databox sql -h
```

To avoid providing database connection details each time you run a sub-command,
you can use environment variables.

```bash
$ export DSN="postgres://user:password@localhost:5432/dbname"
$ databox ls # No need to provide -dsn here
```

DSN can also be provided as different environment variables: `DATABASE_URL`, `DB_URL`, `DATABASE_DSN`, `DB_DSN`.  
It could be also provided as a flag as well: `... -dsn "postgres://user:password@localhost:5432/dbname"`

DSN composition might be a bit challenging.
Here is a general template for it:

```
[protocol]://[username]:[password]@[host]:[port]/[database]?[params]
```

Some examples of DSNs for different databases:

```
# SQLite
# We can use both absolute and relative paths.
sqlite:///abs/path/to/db.sqlite
sqlite3://rel/path/to/db.sqlite

# Postgres
# Postgres DSN is quite straightforward.
postgres://user:password@localhost:5432/dbname
postgresql://user:password@localhost:5432/dbname

# MySQL
# Please note, that our MySQL integration doesn't support certificates yet.
# Also, DSN is a bit different from the standard one.
# It doesn't have a protocol part, which wraps the host+port part.
mysql://user:password@localhost:3306/dbname?parseTime=true
```
