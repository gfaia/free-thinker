# Information Aggregator System Design

## 1. Overview

A daily information aggregation system that fetches content from configurable data source platforms (starting with Zhihu) based on user-defined queries, stores articles with deduplication.

## 2. Architecture

### 2.1 High-Level Flow

```
┌─────────────────────────────────────────────────────────┐
│                    Scheduler (Cron)                    │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                   Fetcher Registry                      │
│   (Reads config.yaml, registers enabled sources)       │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                 Fetcher Engine (Loop)                   │
│  For each (source, query) pair:                         │
│    - Call source.Fetch(keyword)                         │
│    - Deduplicate via (url, title)                       │
│    - Persist metadata + content                         │
└─────────────────────────────────────────────────────────┘
```

### 2.2 Fetcher Interface Abstraction

All data sources implement a shared `Fetcher` interface so the engine is source-agnostic:

```go
// internal/fetcher/fetcher.go
package fetcher

// Article represents a normalized item fetched from a data source.
type Article struct {
    URL          string
    Title        string
    Author       string
    PublishedAt  time.Time
    Summary      string
    RawHTML      string
    Source       string   // e.g. "zhihu"
    QueryKeyword string   // the keyword that produced this hit
}

// Fetcher is the interface every data source must implement.
type Fetcher interface {
    // Name returns the unique source identifier (e.g. "zhihu").
    Name() string

    // Fetch searches the source using the given keyword and returns articles.
    Fetch(ctx context.Context, keyword string) ([]Article, error)

    // Configure applies the source-specific config block from config.yaml.
    Configure(rawConfig map[string]interface{}) error
}

// Registry tracks all registered fetchers by name.
type Registry struct {
    sources map[string]Fetcher
}

func (r *Registry) Register(f Fetcher)          { r.sources[f.Name()] = f }
func (r *Registry) Get(name string) (Fetcher, bool) { f, ok := r.sources[name]; return f, ok }
func (r *Registry) Names() []string              { /* ... */ }
```

### 2.3 Plug-in Registration Flow

```
1. main() loads config.yaml
2. For each enabled source in config:
     a. Look up constructor in fetcher.Registry
     b. Call fetcher.Configure(source.config)
     c. Activate it in the running engine
3. Engine dispatches queries to only the active fetchers
```

New data sources are added by:
- Creating a new package under `internal/fetcher/{source_name}/`
- Implementing the `Fetcher` interface
- Calling `registry.Register()` from an `init()` function (or via explicit wiring)
- Adding an entry to `config.yaml` under `sources:`

## 3. Database Schema

### Table: articles

| Column        | Type         | Constraints                | Description              |
|---------------|--------------|----------------------------|--------------------------|
| id            | BIGSERIAL    | PRIMARY KEY                | Auto-increment ID        |
| url           | VARCHAR(512) | NOT NULL                   | Article URL              |
| title         | VARCHAR(1024)| NOT NULL                   | Article title            |
| author        | VARCHAR(256) |                            | Author name              |
| published_at  | TIMESTAMP    |                            | Publication date         |
| source        | VARCHAR(64)  | NOT NULL                   | Data source platform     |
| query_keyword | VARCHAR(256) |                            | Keyword that triggered   |
| content_path  | VARCHAR(512) |                            | Local content file path  |
| summary       | VARCHAR(4096)|                            | Short summary/snippet    |
| created_at    | TIMESTAMP    | DEFAULT NOW()              | Record creation time     |

**Unique Constraint**: `(url, title)` - Deduplication key

**Index**: `(source, query_keyword)` for retrieval grouping

### Table: queries

| Column     | Type         | Constraints         | Description              |
|------------|--------------|---------------------|--------------------------|
| id         | BIGSERIAL    | PRIMARY KEY         | Auto-increment ID        |
| keyword    | VARCHAR(256) | NOT NULL            | Search keyword           |
| platform   | VARCHAR(64)  | NOT NULL            | Target platform          |
| last_run   | TIMESTAMP    |                     | Last execution time      |
| status     | VARCHAR(32)  |                     | pending/running/completed/failed |

**Unique Constraint**: `(keyword, platform)`

## 4. Storage Strategy

- **Metadata**: Stored in relational database (SQLite for dev, PostgreSQL for production). Schema defined in `scripts/init_db.sql`.
- **Article Content**: Stored as local files at `data/articles/{source}/{YYYY-MM-DD}/{id}.html`
- **Content Path**: Stored in the `content_path` column, linking metadata to actual file content

## 5. Deduplication Logic

1. Before inserting, compute a hash / lookup key: `(url, title)`
2. Query the database: if the key exists, drop the item
3. Use the database unique constraint as the final guard (defensive)
4. Return `is_duplicate` flag so the caller can log stats

## 6. Configuration (config.yaml)

```yaml
# config.yaml
database:
  driver: sqlite            # sqlite | postgres
  dsn:    data/aggregator.db

storage:
  content_root: data/articles

schedule:
  cron: "0 2 * * *"         # daily at 2 AM local

fetchers:
  - name: zhihu
    enabled: true
    queries:
      - golang
      - distributed systems
    config:
      max_pages: 3
      rate_limit_ms: 800
      # user_agent: "..."
      # cookie:   "..."   # optional, injected only if present

  # - name: weibo
  #   enabled: false
  #   queries:
  #     - golang
  #   config: { ... }
```

## 7. Project Structure

```
free-thinker/
├── cmd/
│   └── crawler/
│       └── main.go                # Entry: loads config, wires fetchers, runs once / cron
├── internal/
│   ├── config/
│   │   └── config.go              # config.yaml loader
│   ├── db/
│   │   ├── db.go                  # DB connection / queries
│   │   └── migrations/
│   │       └── 001_init.sql       # Table schema
│   ├── fetcher/
│   │   ├── fetcher.go             # Fetcher interface + Article type + Registry
│   │   └── sources/
│   │       └── zhihu/
│   │           └── fetcher.go     # Zhihu concrete Fetcher
│   ├── storage/
│   │   └── file.go                # Writes article content to disk
│   └── engine/
│       └── engine.go              # Orchestrator: loop fetcher -> dedup -> persist
├── pkg/
│   └── models/
│       └── article.go             # Shared domain model
├── config.example.yaml            # Template config
├── data/
│   └── articles/                  # Article content storage (gitignored)
├── docs/
│   └── design/
│       └── README.md              # This document
└── scripts/
    └── init_db.sql                # Alternative standalone init script
```

## 8. Phase 1 Scope

- [x] Database schema design
- [x] Project structure design
- [x] Fetcher interface abstraction + pluggable registry
- [x] Configurable source activation via config.yaml
- [ ] Zhihu fetcher implementation
- [ ] Deduplication implementation
- [ ] File storage implementation
- [ ] Daily cron scheduler
- [ ] Basic CLI interface
- [ ] Unit tests for dedup + registry wiring

## 9. Adding a New Data Source (Checklist)

1. Create `internal/fetcher/sources/{name}/fetcher.go`
2. Implement `Fetcher` interface: `Name()`, `Fetch(ctx, keyword)`, `Configure(rawConfig)`
3. Register in `internal/fetcher/registry.go` (or via init + build tag)
4. Add a section to `config.example.yaml` under `fetchers:`
5. Add integration tests if the source has offline fixtures

## 10. Future Extensions

- Full-text search capability (Bleve / SQLite FTS5)
- Article tagging and categorization
- Web UI for browsing
- API server for remote access
- RSS feed output
- Batch backfill (re-fetch last N days)
