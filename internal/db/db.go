package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type ArticleRecord struct {
	ID           int64
	URL          string
	Title        string
	Author       string
	PublishedAt  time.Time
	Source       string
	QueryKeyword string
	ContentPath  string
	Summary      string
	CreatedAt    time.Time
}

func Open(driver, dsn string) (*Store, error) {
	if driver != "sqlite" {
		return nil, fmt.Errorf("only sqlite driver is implemented (got %q)", driver)
	}
	if dir := filepath.Dir(dsn); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS articles (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    url           TEXT NOT NULL,
    title         TEXT NOT NULL,
    author        TEXT,
    published_at  DATETIME,
    source        TEXT NOT NULL,
    query_keyword TEXT,
    content_path  TEXT,
    summary       TEXT,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(url, title)
);
CREATE INDEX IF NOT EXISTS idx_articles_source_query ON articles(source, query_keyword);

CREATE TABLE IF NOT EXISTS queries (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    keyword  TEXT NOT NULL,
    platform TEXT NOT NULL,
    last_run DATETIME,
    status   TEXT,
    UNIQUE(keyword, platform)
);`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) UpsertArticle(ctx context.Context, a *ArticleRecord) (isDuplicate bool, err error) {
	if a == nil {
		return false, fmt.Errorf("nil article")
	}
	if a.URL == "" || a.Title == "" {
		return false, fmt.Errorf("url and title are required")
	}

	var existingID int64
	err = s.db.QueryRowContext(ctx,
		`SELECT id FROM articles WHERE url = ? AND title = ?`,
		a.URL, a.Title).Scan(&existingID)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}

	ts := a.CreatedAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO articles(url, title, author, published_at, source, query_keyword, content_path, summary, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.URL, a.Title, a.Author, nullTime(a.PublishedAt), a.Source, a.QueryKeyword, a.ContentPath, a.Summary, ts,
	)
	if err != nil {
		return false, err
	}
	a.ID, _ = res.LastInsertId()
	return false, nil
}

func (s *Store) UpdateQuery(ctx context.Context, keyword, platform, status string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO queries(keyword, platform, last_run, status) VALUES (?, ?, ?, ?)
		 ON CONFLICT(keyword, platform) DO UPDATE SET last_run=excluded.last_run, status=excluded.status`,
		keyword, platform, now, status,
	)
	return err
}

func (s *Store) DB() *sql.DB { return s.db }

func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}
