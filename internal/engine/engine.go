package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gfaia/free-thinker/internal/config"
	"github.com/gfaia/free-thinker/internal/db"
	"github.com/gfaia/free-thinker/internal/fetcher"
	"github.com/gfaia/free-thinker/internal/storage"
)

type Engine struct {
	registry *fetcher.Registry
	store    *db.Store
	fstore   *storage.FileStore
	fetchers []config.FetcherEntry
}

func New(
	registry *fetcher.Registry,
	store *db.Store,
	fstore *storage.FileStore,
	fetchers []config.FetcherEntry,
) *Engine {
	return &Engine{registry: registry, store: store, fstore: fstore, fetchers: fetchers}
}

func (e *Engine) RunOnce(ctx context.Context) error {
	total := 0
	dups := 0
	for _, entry := range e.fetchers {
		if !entry.Enabled {
			continue
		}
		f, ok := e.registry.Get(entry.Name)
		if !ok {
			log.Printf("[engine] fetcher %q not registered, skip", entry.Name)
			continue
		}
		if err := f.Configure(entry.Config); err != nil {
			log.Printf("[engine] configure %s failed: %v", entry.Name, err)
			continue
		}
		for _, keyword := range entry.Queries {
			count, dupCount, err := e.runOne(ctx, f, keyword)
			if err != nil {
				log.Printf("[engine] %s query %q failed: %v", entry.Name, keyword, err)
				_ = e.store.UpdateQuery(ctx, keyword, entry.Name, "failed")
				continue
			}
			total += count
			dups += dupCount
			_ = e.store.UpdateQuery(ctx, keyword, entry.Name, "completed")
			log.Printf("[engine] %s:%s fetched=%d dup=%d", entry.Name, keyword, count, dupCount)
		}
	}
	log.Printf("[engine] run complete: fetched=%d dup=%d", total, dups)
	return nil
}

func (e *Engine) runOne(ctx context.Context, f fetcher.Fetcher, keyword string) (int, int, error) {
	articles, err := f.Fetch(ctx, keyword)
	if err != nil {
		return 0, 0, err
	}
	dups := 0
	for i := range articles {
		articles[i].Source = f.Name()
		articles[i].QueryKeyword = keyword

		contentPath := ""
		if articles[i].RawHTML != "" {
			p, werr := e.fstore.Save(f.Name(), articles[i].RawHTML)
			if werr != nil {
				log.Printf("[engine] store content failed: %v", werr)
			} else {
				contentPath = p
			}
		}

		rec := &db.ArticleRecord{
			URL:          articles[i].URL,
			Title:        articles[i].Title,
			Author:       articles[i].Author,
			PublishedAt:  unixZero(articles[i].PublishedAt),
			Source:       articles[i].Source,
			QueryKeyword: articles[i].QueryKeyword,
			ContentPath:  contentPath,
			Summary:      articles[i].Summary,
		}
		isDup, err := e.store.UpsertArticle(ctx, rec)
		if err != nil {
			log.Printf("[engine] upsert failed: %v", err)
			continue
		}
		if isDup {
			dups++
		}
	}
	return len(articles), dups, nil
}

func unixZero(sec int64) time.Time {
	if sec == 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}

var _ = fmt.Sprintf
