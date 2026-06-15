# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- Build all packages: `go build ./...`
- Run all tests: `go test ./...`
- Run tests for one package: `go test ./internal/engine`
- Run one test by name: `go test ./internal/engine -run TestName`
- Run the crawler once: `go run ./cmd/crawler -config config.yaml -once`
- Run the crawler as a cron daemon: `go run ./cmd/crawler -config config.yaml`
- Create a local config from the example before running: `cp config.example.yaml config.yaml`

There is no Makefile in the current repository; use the Go toolchain directly.

## Architecture

This is a Go 1.22 information aggregation/crawler project. The current implementation is centered on a CLI crawler in `cmd/crawler` that loads YAML config, opens storage, registers fetchers, and either runs once or schedules repeated runs with `robfig/cron`.

High-level runtime flow:

1. `cmd/crawler/main.go` loads `config.yaml` through `internal/config`.
2. It opens a SQLite-backed `internal/db.Store`, which creates the `articles` and `queries` tables automatically if needed.
3. It creates a `storage.FileStore` for raw article HTML under the configured content root.
4. It creates a `fetcher.Registry`, currently registering only the Zhihu fetcher explicitly.
5. `internal/engine.Engine` iterates enabled fetcher entries and their query keywords, calls `Fetcher.Fetch`, saves raw HTML when present, and upserts metadata into the database.

Key package responsibilities:

- `internal/config`: YAML config structs, defaults, and validation. Defaults include SQLite at `data/aggregator.db`, content storage at `data/articles`, and cron `0 2 * * *`.
- `internal/fetcher`: Source-agnostic article shape, `Fetcher` interface, and registry. New sources should implement `Name`, `Configure`, and `Fetch`.
- `internal/fetcher/sources/zhihu`: Zhihu fetcher stub and config parsing. `Fetch` currently returns `not implemented yet`.
- `internal/engine`: Orchestration layer. It configures enabled fetchers, runs each configured query, handles duplicate counts, updates query status, and maps fetched articles into DB records.
- `internal/db`: SQLite store and schema. Deduplication is based on the `(url, title)` unique key and checked in `UpsertArticle`.
- `internal/storage`: File-backed storage for raw HTML, saved under `{content_root}/{source}/{YYYY-MM-DD}/{unixnano}.html`.
- `pkg/models`: Public/shared article and query data models. The current internal DB and fetcher code use their own structs rather than these models.

## Configuration

Use `config.example.yaml` as the template. The top-level sections are:

- `database`: `driver` and `dsn`; only `sqlite` is implemented in code.
- `storage`: `content_root` for saved article HTML.
- `schedule`: cron expression used when not running with `-once`.
- `fetchers`: list of source entries with `name`, `enabled`, `queries`, and source-specific `config`.

## Design Notes

`docs/design/README.md` describes the intended aggregator design and future scope. Some items in that design are already implemented (config loading, registry, SQLite schema, engine loop, file storage), while others are still incomplete, most notably the real Zhihu fetcher implementation and unit tests.
