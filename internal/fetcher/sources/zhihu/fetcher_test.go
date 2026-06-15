package zhihu

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestConfigure(t *testing.T) {
	f := New()
	err := f.Configure(map[string]interface{}{
		"max_pages":     float64(0),
		"rate_limit_ms": float64(-10),
		"user_agent":    "  custom-agent  ",
		"cookie":        "  a=b  ",
	})
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	if f.maxPages != 1 {
		t.Fatalf("maxPages = %d, want 1", f.maxPages)
	}
	if f.rateLimitMS != 0 {
		t.Fatalf("rateLimitMS = %d, want 0", f.rateLimitMS)
	}
	if f.userAgent != "custom-agent" {
		t.Fatalf("userAgent = %q, want custom-agent", f.userAgent)
	}
	if f.cookie != "a=b" {
		t.Fatalf("cookie = %q, want a=b", f.cookie)
	}
}

func TestFetchRejectsEmptyKeyword(t *testing.T) {
	_, err := New().Fetch(context.Background(), "  ")
	if err == nil || !strings.Contains(err.Error(), "empty keyword") {
		t.Fatalf("Fetch(empty) error = %v, want empty keyword", err)
	}
}

func TestFetchParsesSearchResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/search_v3" {
			t.Fatalf("path = %s, want /api/v4/search_v3", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != "golang" {
			t.Fatalf("query q = %q, want golang", got)
		}
		if got := r.Header.Get("User-Agent"); got != "test-agent" {
			t.Fatalf("User-Agent = %q, want test-agent", got)
		}
		if got := r.Header.Get("Cookie"); got != "z=1" {
			t.Fatalf("Cookie = %q, want z=1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"object": {
						"id": 123,
						"type": "article",
						"url": "https://www.zhihu.com/api/v4/articles/123",
						"title": "<em>Go</em> &amp; Zhihu",
						"excerpt": "<p>short&nbsp;summary</p>",
						"content": "<p>raw content</p>",
						"author": {"name": "Alice"},
						"created": 1710000000
					}
				},
				{
					"object": {
						"id": "888",
						"type": "answer",
						"url": "https://www.zhihu.com/api/v4/answers/888",
						"question": {
							"id": 777,
							"title": "Question title",
							"url": "https://www.zhihu.com/api/v4/questions/777"
						},
						"author": {"name": "Bob"},
						"created_time": 1710000001
					}
				},
				{"object": {"url": "https://example.com/no-title"}}
			],
			"paging": {"is_end": true}
		}`))
	}))
	defer server.Close()

	f := New()
	f.baseURL = server.URL
	if err := f.Configure(map[string]interface{}{
		"max_pages":     3,
		"rate_limit_ms": 0,
		"user_agent":    "test-agent",
		"cookie":        "z=1",
	}); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	articles, err := f.Fetch(context.Background(), "golang")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(articles) != 2 {
		t.Fatalf("len(articles) = %d, want 2: %#v", len(articles), articles)
	}
	if got, want := articles[0].URL, "https://zhuanlan.zhihu.com/p/123"; got != want {
		t.Fatalf("article[0].URL = %q, want %q", got, want)
	}
	if got, want := articles[0].Title, "Go & Zhihu"; got != want {
		t.Fatalf("article[0].Title = %q, want %q", got, want)
	}
	if got, want := articles[0].Summary, "short summary"; got != want {
		t.Fatalf("article[0].Summary = %q, want %q", got, want)
	}
	if articles[0].Author != "Alice" || articles[0].PublishedAt != 1710000000 || articles[0].RawHTML == "" {
		t.Fatalf("article[0] fields = %#v", articles[0])
	}
	if got, want := articles[1].URL, "https://www.zhihu.com/question/777/answer/888"; got != want {
		t.Fatalf("article[1].URL = %q, want %q", got, want)
	}
}

func TestFetchRespectsMaxPages(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requests, 1)
		if got, want := r.URL.Query().Get("offset"), strconv.Itoa((int(n)-1)*pageSize); got != want {
			t.Fatalf("offset = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"object":{"id":"` + strconv.Itoa(int(n)) + `","type":"article","url":"https://www.zhihu.com/api/v4/articles/` + strconv.Itoa(int(n)) + `","title":"title ` + strconv.Itoa(int(n)) + `"}}],"paging":{"is_end":false}}`))
	}))
	defer server.Close()

	f := New()
	f.baseURL = server.URL
	f.maxPages = 2
	f.rateLimitMS = 0

	articles, err := f.Fetch(context.Background(), "golang")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if len(articles) != 2 {
		t.Fatalf("len(articles) = %d, want 2", len(articles))
	}
}

func TestFetchStopsOnPagingEnd(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"paging":{"is_end":true}}`))
	}))
	defer server.Close()

	f := New()
	f.baseURL = server.URL
	f.maxPages = 3
	f.rateLimitMS = 0

	_, err := f.Fetch(context.Background(), "golang")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestFetchReturnsHTTPStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "blocked", http.StatusForbidden)
	}))
	defer server.Close()

	f := New()
	f.baseURL = server.URL
	_, err := f.Fetch(context.Background(), "golang")
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("Fetch() error = %v, want status 403", err)
	}
}

func TestFetchStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New().Fetch(ctx, "golang")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch() error = %v, want context.Canceled", err)
	}
}

func TestFetchCancelsRateLimitSleep(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"paging":{"is_end":false}}`))
	}))
	defer server.Close()

	f := New()
	f.baseURL = server.URL
	f.maxPages = 2
	f.rateLimitMS = 5000

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := f.Fetch(ctx, "golang")
		done <- err
	}()

	for atomic.LoadInt32(&requests) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Fetch() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Fetch() did not stop after context cancellation")
	}
}
