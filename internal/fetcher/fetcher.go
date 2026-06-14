package fetcher

import "context"

type Article struct {
	URL          string
	Title        string
	Author       string
	PublishedAt  int64
	Summary      string
	RawHTML      string
	Source       string
	QueryKeyword string
}

type Fetcher interface {
	Name() string
	Fetch(ctx context.Context, keyword string) ([]Article, error)
	Configure(rawConfig map[string]interface{}) error
}
