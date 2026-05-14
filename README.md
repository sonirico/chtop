# chtop

[![ci](https://github.com/sonirico/chtop/actions/workflows/ci.yml/badge.svg)](https://github.com/sonirico/chtop/actions/workflows/ci.yml)

Terminal UI for ClickHouse. Same idea as `htop` but for a ClickHouse
cluster: tables, running queries with kill, merges in flight, replication
status, cluster topology, query log tail, EXPLAIN viewer, live metrics,
errors, materialized view graph, Kafka-engine consumers.

Built because looking at running queries with `clickhouse-client` is fine
for one query and miserable for forty.

## What it does

- **Tables** list with engine, rows, on-disk size, compression ratio,
  active parts. Filter with `/`. Enter to drill in.
- **Table detail** with tabs:
  - Schema: columns, type, codec, keys, per-column size and ratio.
  - Parts: per-active-part rows, size, level and modification time.
  - Mutations: in-flight + recent with progress and failure reasons.
  - Engine: full CREATE TABLE, partition / sorting / primary keys,
    storage policy.
- **Processes** (`system.processes`) with kill confirmation (`k`, `y/N`).
- **Clusters** (`system.clusters`) and **Replicas** (`system.replicas`)
  with queue, merges in queue, absolute delay.
- **Merges** in flight with a coloured progress bar per row.
- **Query log** tail (`system.query_log`) with filter, status (ok/err)
  and `e` to open the EXPLAIN viewer for the highlighted query.
- **EXPLAIN viewer** with PLAN / PIPELINE / SYNTAX / ESTIMATE tabs,
  cached per mode.
- **Live metrics dashboard** combining `system.metrics`,
  `system.asynchronous_metrics` and `system.events`. Event counters show
  a per-second rate against the previous snapshot.
- **Errors** (`system.errors`) with growth highlighting: rows whose
  counter increased since the previous snapshot turn red.
- **Materialized view graph**: each MV rendered as
  `source -> mv -> target  (rows, size)`.
- **Kafka consumers** (`system.kafka_consumers`) with read count, commit
  count, last poll age and stale flag. Falls back to a friendly message
  on servers without the Kafka engine.
- **mTLS** (client cert / key + custom CA).
- **k9s feel**: `:` command bar, `/` filter, `?` help, dynamic column
  widths, ClickHouse-yellow palette with a red cursor.
- **Doesn't poll itself**: every query chtop issues is tagged so its own
  traffic is filtered out of the processes view and query log tail.

## Install

```
go install github.com/sonirico/chtop/cmd/chtop@latest
```

Or from source:

```
git clone https://github.com/sonirico/chtop
cd chtop
just install
```

Pre-built binaries are attached to each GitHub Release
(`linux/darwin` x `amd64/arm64`).

Needs Go 1.26+ to build from source.

## Configuration

CLI flags or env vars:

```
chtop --host ch1.example.com --port 9000 --user readonly --password ...
```

| Flag | Env | Default |
|------|-----|---------|
| `--host` | `CHTOP_HOST` | `localhost` |
| `--port` | `CHTOP_PORT` | `9000` (`9440` with `--tls`) |
| `--user` | `CHTOP_USER` | `default` |
| `--password` | `CHTOP_PASSWORD` | (empty) |
| `--database` | `CHTOP_DATABASE` | `default` |
| `--tls` |  | off |
| `--tls-cert` | `CHTOP_TLS_CERT` | (empty) |
| `--tls-key` | `CHTOP_TLS_KEY` | (empty) |
| `--tls-ca` | `CHTOP_TLS_CA` | (empty) |
| `--cluster` | `CHTOP_CLUSTER` | (empty, shown in header) |

## Keys

Global:

| Key | Action |
|-----|--------|
| `:` | command bar |
| `/` | filter rows |
| `?` | help |
| `r` | refresh |
| `enter` | drill in |
| `esc` | back |
| `q` / `ctrl+c` | quit |

Commands after `:`: `tables` `processes` `clusters` `replicas` `merges`
`querylog` (`ql`) `metrics` (`met`) `errors` (`err`) `matviews` (`mv`)
`kafka` (`k`) `help` `quit`.

Processes view: `k` kills the highlighted query (asks `y/N`).
Query log view: `e` opens the EXPLAIN viewer for the selected query.
Table detail / EXPLAIN: `1`/`2`/`3`/`4` (or `tab`) switch tabs.

## Layout

```
cmd/chtop/    binary
pkg/ch/       ClickHouse client (importable lib)
internal/
  tui/        bubbletea views
```

## Status

Read-only except `KILL QUERY`. No DDL, no inserts, no schema changes yet.

## Roadmap

### Shipped in v0.1.0

- Tables, processes (with kill), clusters, replicas, merges.
- Table detail (schema / parts / mutations / engine).
- Query log tail with filter and `e` -> EXPLAIN viewer.
- EXPLAIN viewer (Plan / Pipeline / Syntax / Estimate).
- Live metrics dashboard with event rates.
- Errors view with growth highlight.
- Materialized view graph.
- Kafka-engine consumer status.
- mTLS auth (client cert / key + custom CA).
- GitHub Actions CI (vet, golangci-lint, race tests).
- goreleaser cross-platform binaries.

### v0.2 and beyond

- Reset consumer-group / Kafka-engine offsets.
- Alter TTL / storage policy from the detail view.
- Alter table configs and partitions.
- Query log filters by user / db / duration.
- EXPLAIN AST / QUERY TREE modes.
- Deeper Kafka stats: per-partition assignment with lag.
- OAUTHBEARER / cloud auth flows.
- Customizable colour themes.
- Pagination / windowing for very long tables.
