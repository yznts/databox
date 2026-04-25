<h1 align="center">databox</h1>

<p align="center">
  A Swiss Army knife for databases in the command line
</p>

```bash
go install github.com/yznts/databox/cmd/databox@latest
```

**databox** is a single CLI for working with SQLite, PostgreSQL, and MySQL without juggling different clients (`sqlite3`, `psql`, `mysql`, …). Subcommands follow UNIX-style names: one job per command, predictable flags.

Goals:

- **Unified UX** — same flags and output modes where it makes sense.
- **Script-friendly** — JSON, JSONL, CSV, and SQL (`INSERT`) output for automation.

## Installation

### Go install

Requires Go and `$(go env GOPATH)/bin` (or `GOBIN`) on `PATH`:

```bash
go install github.com/yznts/databox/cmd/databox@latest
```

### Build from source

```bash
git clone git@github.com:yznts/databox.git
cd databox
go build -o databox ./cmd/databox/
# install the binary wherever you keep local tools, e.g. /usr/local/bin
```

### Release binaries

See [Releases](https://github.com/yznts/databox/releases). macOS binaries are not signed.

## Quick start

Assumes `databox` is installed and on your `PATH` (see [Installation](#installation) above). Work through these steps in order. Lines starting with `#` are comments; run the other lines (adjust paths and table names to match your database).

**Step 1 — Point databox at a database**

Pick one style: environment variable (good for a whole shell session) or `-dsn` on each invocation.

```bash
# Option A: session default (see “Connection (DSN)” for other env names).
export DSN="sqlite3://./app.db"

# Option B: inline, no env var (use your real URL).
# databox ls -dsn "postgres://user:pass@localhost:5432/mydb"
```

**Step 2 — Sanity check the connection**

```bash
# Prints driver, paths or host, sizes, etc. (fields depend on engine).
databox dsn

# Minimal query; useful when you only want to know the DB answers.
databox sql "select 1 as ok"
```

**Step 3 — Explore schema**

```bash
# No argument: list tables (add -sys to include system tables where applicable).
databox ls

# With a table name: list that table’s column metadata (-col presets: basic, extended, all).
# Replace `orders` with a table you actually have.
databox ls orders
databox ls -col basic orders
```

**Step 4 — Inspect data**

```bash
# Replace `orders` with your table. -n defaults to 10 if omitted.
databox head -n 5 orders

# All rows (may truncate in gloss/json; use -csv / -jsonl for large exports).
databox cat orders

# Filtered row count (optional -where).
databox count -where "status = 'open'" orders
```

**Step 5 — Machine-readable output**

```bash
# Same commands as above; add one format flag per run.
databox ls -json
databox head -n 3 -jsonl orders
```

## Subcommands

| Command | Purpose |
|--------|---------|
| `ls` | List tables, or **column metadata** for one table (`-col` chooses which fields; `-sys`, `-sql` for DDL). |
| `sql` | Run SQL; pass a query as args or pipe / type on stdin if omitted. |
| `cat` | Print all rows of a table (streaming where the format allows). |
| `head` | First *N* rows (`-n`, default 10). |
| `tail` | Last *N* rows (`-n`, default 10); use `-order col` for stable “last” semantics. |
| `grep` | Rows where any column matches a substring pattern (`LIKE`). |
| `count` | Row count for a table; optional `-where` for a filtered count. |
| `cp` | Copy **one table to another name** in the **same** DSN (`-schema` or `-schema-data`). |
| `migrate` | Copy **schema** or **schema + data** from **source DSN** to **destination DSN** (possibly different engines); `-tables` to limit. |
| `ps` | List server processes (where supported). |
| `kill` | Terminate a process (where supported). |
| `dsn` | Resolve DSN and print connection / database summary. |

Run `databox <command> -h` for flags and examples.

### Row-oriented flags (`cat`, `head`, `tail`)

These commands share:

- `-where` — SQL `WHERE` fragment (e.g. `id > 5`).
- `-col` — comma-separated **data** columns to `SELECT` (default all).
- `-order` — sort column; prefix with `-` for descending (e.g. `-id`).

`count` supports `-where` for filtered counts. `ls -col` is separate: it picks **metadata fields** when listing a table’s columns, not row columns. `grep` does not use `-where` / `-col` / `-order` (pattern matching is orthogonal to a SQL `WHERE` fragment).

## Output formats

Default is terminal **gloss** (tabular). Machine-readable modes:

- `-json` — single JSON document.
- `-jsonl` — one JSON object per line (streams on suitable writers).
- `-csv` — CSV.
- `-sql` — `INSERT` statements (where implemented for that command).

For large results, gloss/JSON may **truncate** with a warning unless you use `-nowarn` (see `-h` on each command).

## Connection (DSN)

Pass `-dsn "<url>"` or set an environment variable (checked in order until one is set):

1. `DSN`
2. `DATABASE`
3. `DATABASE_URL`
4. `DATABOX`

The `-dsn` flag wins when non-empty.

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/dbname"
databox ls
```

### DSN shape

Template:

```text
[scheme]://[user]:[password]@[host]:[port]/[database]?[query-params]
```

Examples:

```text
# SQLite — absolute or relative path
sqlite:///absolute/path/to/db.sqlite
sqlite3://relative/path/to/db.sqlite

# PostgreSQL
postgres://user:password@localhost:5432/dbname
postgresql://user:password@localhost:5432/dbname

# MySQL (TLS / client certs not supported yet)
mysql://user:password@localhost:3306/dbname?parseTime=true
```

`postgresql://` and `sqlite3://` are normalized internally to the drivers databox uses.
