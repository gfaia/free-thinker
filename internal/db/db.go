package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gfaia/free-thinker/pkg/models"

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

type ArticleFilter struct {
	Source       string
	QueryKeyword string
	Limit        int
	Offset       int
}

type QueryFilter struct {
	Platform string
	Status   string
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

func (s *Store) ListQueries(ctx context.Context, filter QueryFilter) ([]models.Query, error) {
	where, args := queryWhere(filter)
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, keyword, platform, last_run, status FROM queries`+where+` ORDER BY last_run DESC, platform ASC, keyword ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []models.Query
	for rows.Next() {
		q, err := scanQuery(rows)
		if err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	return queries, rows.Err()
}

func (s *Store) ListArticles(ctx context.Context, filter ArticleFilter) ([]models.Article, error) {
	filter = normalizeArticleFilter(filter)
	where, args := articleWhere(filter)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, url, title, author, published_at, source, query_keyword, content_path, summary, created_at FROM articles`+
			where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, rows.Err()
}

func (s *Store) CountArticles(ctx context.Context, filter ArticleFilter) (int, error) {
	where, args := articleWhere(filter)

	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles`+where, args...).Scan(&count)
	return count, err
}

func (s *Store) GetArticle(ctx context.Context, id int64) (*models.Article, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, url, title, author, published_at, source, query_keyword, content_path, summary, created_at FROM articles WHERE id = ?`,
		id,
	)
	a, err := scanArticle(row)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) DB() *sql.DB { return s.db }

func queryWhere(filter QueryFilter) (string, []interface{}) {
	var clauses []string
	var args []interface{}
	if filter.Platform != "" {
		clauses = append(clauses, "platform = ?")
		args = append(args, filter.Platform)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	return joinWhere(clauses), args
}

func articleWhere(filter ArticleFilter) (string, []interface{}) {
	var clauses []string
	var args []interface{}
	if filter.Source != "" {
		clauses = append(clauses, "source = ?")
		args = append(args, filter.Source)
	}
	if filter.QueryKeyword != "" {
		clauses = append(clauses, "query_keyword = ?")
		args = append(args, filter.QueryKeyword)
	}
	return joinWhere(clauses), args
}

func joinWhere(clauses []string) string {
	if len(clauses) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(clauses, " AND ")
}

func normalizeArticleFilter(filter ArticleFilter) ArticleFilter {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return filter
}

func scanQuery(scanner interface {
	Scan(dest ...interface{}) error
}) (models.Query, error) {
	var q models.Query
	var lastRun sql.NullTime
	if err := scanner.Scan(&q.ID, &q.Keyword, &q.Platform, &lastRun, &q.Status); err != nil {
		return q, err
	}
	q.LastRun = nullTimeValue(lastRun)
	return q, nil
}

func scanArticle(scanner interface {
	Scan(dest ...interface{}) error
}) (models.Article, error) {
	var a models.Article
	var publishedAt sql.NullTime
	var createdAt sql.NullTime
	if err := scanner.Scan(
		&a.ID,
		&a.URL,
		&a.Title,
		&a.Author,
		&publishedAt,
		&a.Source,
		&a.QueryKeyword,
		&a.ContentPath,
		&a.Summary,
		&createdAt,
	); err != nil {
		return a, err
	}
	a.PublishedAt = nullTimeValue(publishedAt)
	a.CreatedAt = nullTimeValue(createdAt)
	return a, nil
}

func nullTimeValue(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}
