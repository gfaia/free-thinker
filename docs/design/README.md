# Information Aggregator System Design

## 1. Overview

A daily information aggregation system that fetches content from platforms (starting with Zhihu) based on user-defined queries, stores articles with deduplication.

## 2. Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Scheduler (Cron)                    │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                      Crawler Engine                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Zhihu    │  │   Future    │  │   Future    │     │
│  │   Module   │  │   Modules   │  │   Modules   │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    Deduplication                        │
│              (URL + Title as unique key)                │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                      Storage Layer                       │
│  ┌──────────────────┐    ┌──────────────────────────┐   │
│  │  SQLite/PGDB    │    │  Local File Storage      │   │
│  │  (Metadata)      │    │  (Article Content)       │   │
│  └──────────────────┘    └──────────────────────────────┘
└─────────────────────────────────────────────────────────┘
```

## 3. Database Schema

### Table: articles

| Column        | Type         | Constraints                | Description              |
|---------------|--------------|----------------------------|--------------------------|
| id            | BIGSERIAL    | PRIMARY KEY                | Auto-increment ID        |
| url           | VARCHAR(512) | NOT NULL                   | Article URL              |
| title         | VARCHAR(1024)| NOT NULL                   | Article title            |
| author        | VARCHAR(256) |                            | Author name              |
| published_at  | TIMESTAMP    |                            | Publication date         |
| source        | VARCHAR(64)  | NOT NULL, DEFAULT 'zhihu' | Data source platform     |
| query_keyword | VARCHAR(256) |                            | Keyword that triggered   |
| content_path  | VARCHAR(512) |                            | Local content file path  |
| created_at    | TIMESTAMP    | DEFAULT NOW()              | Record creation time     |

**Unique Constraint**: (url, title) - Deduplication key

### Table: queries

| Column     | Type         | Constraints         | Description              |
|------------|--------------|---------------------|--------------------------|
| id         | BIGSERIAL    | PRIMARY KEY         | Auto-increment ID        |
| keyword    | VARCHAR(256) | NOT NULL            | Search keyword           |
| platform   | VARCHAR(64)  | NOT NULL            | Target platform          |
| last_run   | TIMESTAMP    |                     | Last execution time      |
| status     | VARCHAR(32)  |                     | pending/running/completed/failed |

## 4. Storage Strategy

- **Metadata**: Stored in relational database (SQLite for simplicity, PostgreSQL for production)
- **Article Content**: Stored as local files in `data/articles/{source}/{date}/{id}.html`
- **Content Path**: Stored in `content_path` column, linking metadata to actual content

## 5. Deduplication Logic

1. Before inserting a new article, check if (url, title) combination already exists
2. Use database unique constraint as final guard
3. Return early if duplicate found, skip storage

## 6. Project Structure

```
free-thinker/
├── cmd/
│   └── crawler/
│       └── main.go           # Entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── db/
│   │   ├── db.go             # Database connection
│   │   └── migrations/       # SQL migrations
│   ├── crawler/
│   │   └── zhihu/
│   │       └── crawler.go    # Zhihu crawler implementation
│   ├── storage/
│   │   └── file.go           # File storage operations
│   └── dedup/
│       └── dedup.go          # Deduplication logic
├── pkg/
│   └── models/
│       └── article.go        # Data models
├── data/
│   └── articles/             # Article content storage
├── docs/
│   └── design/
│       └── README.md         # This document
└── scripts/
    └── init_db.sql           # Database initialization
```

## 7. Phase 1 Scope (Zhihu Only)

- [x] Database schema design
- [x] Project structure design
- [ ] Zhihu search API integration
- [ ] Article content extraction
- [ ] Deduplication implementation
- [ ] File storage implementation
- [ ] Daily cron scheduler
- [ ] Basic CLI interface

## 8. Future Extensions

- Support more platforms (Weibo, Douban, etc.)
- Full-text search capability
- Article tagging and categorization
- Web UI for browsing
- API server for remote access
