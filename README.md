# chtop

[![ci](https://github.com/sonirico/chtop/actions/workflows/ci.yml/badge.svg)](https://github.com/sonirico/chtop/actions/workflows/ci.yml)

Terminal UI for ClickHouse. Tables, running queries with kill, replication
status, cluster topology. Same idea as `htop` but for a ClickHouse cluster.

Built because looking at running queries with `clickhouse-client` is fine for
one query and miserable for forty.

## What it does

- Tables list with engine, row count, on-disk size, part count. Filter with
  `/`.
- Running queries view (`system.processes`) with per-query user, elapsed
  time, peak memory, read rows and the query text. Press `k` on a row to
  kill that query (with a y/N confirmation).
- Cluster topology (`system.clusters`): cluster name, shard, replica, host,
  port, errors_count, status.
- Replication status (`system.replicas`): queue size, merges in queue,
  absolute delay, active/total replicas.
- Periodic refresh, k9s-style command bar (`:`), filter (`/`), help (`?`).

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

Needs Go 1.26+.

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
| `--cluster` | `CHTOP_CLUSTER` | (empty, shown in header) |

## Keys

Global:

| Key | Action |
|-----|--------|
| `:` | command bar (`tables`, `processes`, `clusters`, `replicas`, `help`, `quit`) |
| `/` | filter rows |
| `?` | help |
| `r` | refresh |
| `esc` | back |
| `q` / `ctrl+c` | quit |

Processes view:

| Key | Action |
|-----|--------|
| `k` | kill the highlighted query (asks `y/N`) |

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

- Table detail view with parts, mutations, TTL, storage policy.
- Query log tail (`system.query_log`) with filters.
- Merges view (`system.merges`) with per-merge progress.
- EXPLAIN viewer for queries pulled from the log.
- Live metrics dashboard (`system.metrics`, `system.asynchronous_metrics`).
- Errors view (`system.errors`).
- Materialized views graph (source -> target with lag).
- Kafka engine consumer status.
- mTLS, OAuth/cloud authentication.
- GitHub Actions CI, goreleaser for prebuilt binaries.
