# Implement Zhihu Crawler

## Context

Running `go run ./cmd/crawler -config config.example.yaml -once` currently reaches the crawler entrypoint and engine, but every query fails because `internal/fetcher/sources/zhihu/fetcher.go` returns `zhihu fetcher: not implemented yet`. The intended outcome is for the existing crawler pipeline to fetch Zhihu search results for configured keywords, normalize them into `fetcher.Article`, and let the existing engine persist metadata/content to SQLite and file storage.

## Recommended approach

Implement the Zhihu fetcher with Zhihu's JSON search API using only Go standard library packages. Avoid adding scraping dependencies in the first pass: the project currently has a small dependency surface, and JSON extraction is a better fit than scraping JS-heavy rendered HTML.

## Files to modify

- `internal/fetcher/sources/zhihu/fetcher.go`
  - Replace the stubbed `Fetch` implementation.
  - Type the HTTP client for request injection/testing.
  - Add private JSON response structs and parser/helper functions.
- `internal/fetcher/sources/zhihu/fetcher_test.go`
  - Add offline unit tests with `httptest.Server`.

No `go.mod` or `go.sum` changes are expected for the recommended stdlib-only implementation.

## Existing code to reuse

- `fetcher.Article` and `fetcher.Fetcher` from `internal/fetcher/fetcher.go`.
- Existing config keys parsed by `Configure` in `internal/fetcher/sources/zhihu/fetcher.go`: `max_pages`, `rate_limit_ms`, `user_agent`, `cookie`.
- Existing engine behavior in `internal/engine/engine.go`: `RunOnce` calls `Fetch`, sets `Source`/`QueryKeyword`, saves non-empty `RawHTML`, and writes DB records.
- Existing DB/file storage behavior in `internal/db/db.go` and `internal/storage/file.go`; fetcher only needs to return normalized articles with non-empty `URL` and `Title`.

## Implementation steps

1. Update `Fetcher` in `internal/fetcher/sources/zhihu/fetcher.go`:
   - Replace `httpClient interface{}` with a small `httpDoer` interface or `*http.Client`.
   - Add a `baseURL` field defaulting to `https://www.zhihu.com` so tests can point to `httptest.Server`.
   - Initialize the client with a timeout in `New()`.

2. Harden `Configure` while preserving existing config compatibility:
   - Parse `max_pages`, `rate_limit_ms`, `user_agent`, and `cookie`.
   - Clamp `max_pages <= 0` to `1`.
   - Clamp negative `rate_limit_ms` to `0`.
   - Ignore blank `user_agent`; trim `cookie`.

3. Implement `Fetch(ctx, keyword)`:
   - Trim and validate `keyword`.
   - Loop up to `maxPages`, with a page size such as `20`.
   - Build `GET /api/v4/search_v3` requests with query params including `t=general`, `q`, `offset`, `limit`, `lc_idx`, `correction=1`, and `show_all_topics=0`.
   - Use `http.NewRequestWithContext`.
   - Set headers: configured `User-Agent`, optional `Cookie`, `Accept`, `Referer`, `Origin`, and `Accept-Language`.
   - Return clear errors for request failures, non-2xx statuses, JSON decode errors, and context cancellation.
   - Respect `rate_limit_ms` between pages with a cancelable `select` on `ctx.Done()`.

4. Parse and normalize results:
   - Decode tolerant JSON structs for top-level `data`, optional `paging.is_end`, each item's `object`, `question`, `author`, `highlight`, `excerpt`, `content`, and timestamps.
   - Map items to `fetcher.Article`:
     - `URL`: prefer object URL/answer URL, normalize known Zhihu API URLs to browser URLs where practical.
     - `Title`: prefer object title, then question title, then highlight title; clean HTML/entities.
     - `Author`: use `author.name` when present.
     - `PublishedAt`: prefer created/created_time, then updated_time.
     - `Summary`: use excerpt/highlight/content text, cleaned and truncated.
     - `RawHTML`: keep returned content HTML when available.
   - Skip malformed individual items lacking URL or title.
   - Deduplicate within one fetch by `(url, title)` before returning.
   - Stop early if response `paging.is_end` is true.

5. Add tests in `internal/fetcher/sources/zhihu/fetcher_test.go`:
   - `Configure` parsing/clamping.
   - Empty keyword error.
   - Parsing representative Zhihu-like JSON into `fetcher.Article` fields.
   - `max_pages` request count and `paging.is_end` early stop.
   - HTTP status error includes status code.
   - Context cancellation behavior.

## Verification

1. Run the unit suite:

   ```bash
   go test ./...
   ```

2. Run the crawler end-to-end:

   ```bash
   go run ./cmd/crawler -config config.example.yaml -once
   ```

3. Expected result:
   - The previous `zhihu fetcher: not implemented yet` error is gone.
   - Successful live access logs per-query lines such as `zhihu:golang fetched=N dup=M`.
   - SQLite data is written to `data/aggregator.db`.
   - Raw content files may be written under `data/articles/zhihu/<date>/` when API results include content HTML.

4. If live Zhihu returns anti-bot/login/captcha responses, the run should fail with a clear HTTP status/body snippet rather than silently returning empty results; users can then provide a `cookie` in config.
