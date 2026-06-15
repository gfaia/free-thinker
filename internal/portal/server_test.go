package portal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gfaia/free-thinker/internal/db"
	"github.com/gfaia/free-thinker/internal/storage"
)

func newTestServer(t *testing.T) (*Server, *db.ArticleRecord) {
	t.Helper()
	ctx := context.Background()
	store, err := db.Open("sqlite", filepath.Join(t.TempDir(), "portal.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	files := storage.NewFileStore(t.TempDir())
	contentPath, err := files.Save("zhihu", "<h1>raw</h1>")
	if err != nil {
		t.Fatalf("save content: %v", err)
	}
	if err := store.UpdateQuery(ctx, "golang", "zhihu", "completed"); err != nil {
		t.Fatalf("update query: %v", err)
	}
	article := &db.ArticleRecord{
		URL:          "https://example.com/article",
		Title:        "Portal Article",
		Author:       "Alice",
		Source:       "zhihu",
		QueryKeyword: "golang",
		ContentPath:  contentPath,
		Summary:      "summary",
	}
	if _, err := store.UpsertArticle(ctx, article); err != nil {
		t.Fatalf("upsert article: %v", err)
	}
	return NewServer(store, files), article
}

func TestPortalAPIs(t *testing.T) {
	server, article := newTestServer(t)
	handler := server.Handler()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"health", "/api/health", http.StatusOK, `"status":"ok"`},
		{"queries", "/api/queries", http.StatusOK, "golang"},
		{"articles", "/api/articles", http.StatusOK, "Portal Article"},
		{"article", "/api/articles/" + itoa(article.ID), http.StatusOK, "Portal Article"},
		{"missing article", "/api/articles/9999", http.StatusNotFound, "article not found"},
		{"content", "/api/articles/" + itoa(article.ID) + "/content", http.StatusOK, "<h1>raw</h1>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status=%d, want %d, body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body=%s, want contains %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestPortalHTMLPagesAndMethods(t *testing.T) {
	server, article := newTestServer(t)
	handler := server.Handler()

	for _, path := range []string{"/queries", "/articles", "/articles/" + itoa(article.ID)} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d, want 200", path, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/queries", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/queries status=%d, want 405", rec.Code)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
