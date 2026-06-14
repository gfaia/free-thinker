package zhihu

import (
	"context"
	"fmt"
	"time"

	"github.com/gfaia/free-thinker/internal/fetcher"
)

type Fetcher struct {
	maxPages     int
	rateLimitMS  int
	userAgent    string
	cookie       string
	httpClient   interface{}
}

func New() *Fetcher {
	return &Fetcher{
		maxPages:    1,
		rateLimitMS: 800,
		userAgent:   "Mozilla/5.0 (compatible; free-thinker/0.1)",
	}
}

func (f *Fetcher) Name() string { return "zhihu" }

func (f *Fetcher) Configure(rawConfig map[string]interface{}) error {
	if rawConfig == nil {
		return nil
	}
	if v, ok := rawConfig["max_pages"]; ok {
		if n, ok := asInt(v); ok {
			f.maxPages = n
		}
	}
	if v, ok := rawConfig["rate_limit_ms"]; ok {
		if n, ok := asInt(v); ok {
			f.rateLimitMS = n
		}
	}
	if v, ok := rawConfig["user_agent"]; ok {
		if s, ok := v.(string); ok {
			f.userAgent = s
		}
	}
	if v, ok := rawConfig["cookie"]; ok {
		if s, ok := v.(string); ok {
			f.cookie = s
		}
	}
	return nil
}

func (f *Fetcher) Fetch(ctx context.Context, keyword string) ([]fetcher.Article, error) {
	if keyword == "" {
		return nil, fmt.Errorf("empty keyword")
	}
	_ = ctx
	_ = keyword
	_ = f.rateLimitMS
	return nil, fmt.Errorf("zhihu fetcher: not implemented yet")
}

func asInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

var _ fetcher.Fetcher = (*Fetcher)(nil)

var _ = time.Now
