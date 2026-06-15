package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestListQueries(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	if err := store.UpdateQuery(ctx, "golang", "zhihu", "completed"); err != nil {
		t.Fatalf("update query: %v", err)
	}
	if err := store.UpdateQuery(ctx, "rust", "zhihu", "failed"); err != nil {
		t.Fatalf("update query: %v", err)
	}

	queries, err := store.ListQueries(ctx, QueryFilter{Platform: "zhihu", Status: "completed"})
	if err != nil {
		t.Fatalf("list queries: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("got %d queries, want 1", len(queries))
	}
	if queries[0].Keyword != "golang" || queries[0].Status != "completed" || queries[0].LastRun.IsZero() {
		t.Fatalf("unexpected query: %+v", queries[0])
	}
}

func TestArticlesReadAPIs(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	published := time.Date(2026, 6, 14, 1, 2, 3, 0, time.UTC)

	first := &ArticleRecord{
		URL:          "https://example.com/1",
		Title:        "First",
		Author:       "Alice",
		PublishedAt:  published,
		Source:       "zhihu",
		QueryKeyword: "golang",
		ContentPath:  "content.html",
		Summary:      "summary",
	}
	if duplicate, err := store.UpsertArticle(ctx, first); err != nil || duplicate {
		t.Fatalf("upsert first duplicate=%v err=%v", duplicate, err)
	}
	second := &ArticleRecord{URL: "https://example.com/2", Title: "Second", Source: "zhihu", QueryKeyword: "rust"}
	if duplicate, err := store.UpsertArticle(ctx, second); err != nil || duplicate {
		t.Fatalf("upsert second duplicate=%v err=%v", duplicate, err)
	}

	articles, err := store.ListArticles(ctx, ArticleFilter{Source: "zhihu", QueryKeyword: "golang", Limit: 10})
	if err != nil {
		t.Fatalf("list articles: %v", err)
	}
	if len(articles) != 1 || articles[0].Title != "First" || !articles[0].PublishedAt.Equal(published) {
		t.Fatalf("unexpected articles: %+v", articles)
	}

	count, err := store.CountArticles(ctx, ArticleFilter{Source: "zhihu"})
	if err != nil {
		t.Fatalf("count articles: %v", err)
	}
	if count != 2 {
		t.Fatalf("count=%d, want 2", count)
	}

	got, err := store.GetArticle(ctx, first.ID)
	if err != nil {
		t.Fatalf("get article: %v", err)
	}
	if got.Title != "First" || got.Author != "Alice" {
		t.Fatalf("unexpected article: %+v", got)
	}

	_, err = store.GetArticle(ctx, 9999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing article err=%v, want sql.ErrNoRows", err)
	}
}

func TestListArticlesPaginationDefaults(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	for i := 0; i < 3; i++ {
		rec := &ArticleRecord{URL: "https://example.com/p/" + string(rune('a'+i)), Title: string(rune('A' + i)), Source: "zhihu"}
		if _, err := store.UpsertArticle(ctx, rec); err != nil {
			t.Fatalf("upsert article: %v", err)
		}
	}

	articles, err := store.ListArticles(ctx, ArticleFilter{Limit: -1, Offset: -1})
	if err != nil {
		t.Fatalf("list articles: %v", err)
	}
	if len(articles) != 3 {
		t.Fatalf("got %d articles, want 3", len(articles))
	}
}
