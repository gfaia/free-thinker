# Portal Server Plan

## Context

The crawler already runs on a cron schedule, executes configured query keywords, records query execution status in the `queries` table, and stores article metadata plus optional raw HTML paths in the `articles` table. There is currently no way to inspect that data except by opening SQLite directly. This change adds a new portal command that starts a local HTTP server for viewing query task status and stored articles.

The portal should be implemented with a path toward a future front-end/back-end separated architecture. The first version can keep a simple server-rendered fallback page, but the main backend surface should be JSON APIs that a future SPA can consume.

Recommended stack direction:

- Backend: Go + `net/http` + `chi` router.
- Frontend later: Vue 3 + Vite + Element Plus.
- API style: REST JSON.
- Production deployment later: prefer embedding built frontend `dist` into the Go portal binary with `embed`, while still allowing independent frontend deployment if needed.

## Recommended Approach

### 1. Add portal configuration

Modify `internal/config/config.go`:

- Add a `Portal` config struct with `Addr string `yaml:"addr"``.
- Add `Portal Portal `yaml:"portal"`` to `Config`.
- Default `Portal.Addr` to `127.0.0.1:8080` in `applyDefaults()`.

Use localhost as the default because the portal has no authentication yet and may expose scraped content paths/status.

Update `config.example.yaml` with:

```yaml
portal:
  addr: "127.0.0.1:8080"
```

Future config extensions can include CORS origins, frontend static directory, or auth settings, but keep the initial config minimal.

### 2. Use `chi` for the backend router

Add `github.com/go-chi/chi/v5` as the HTTP router dependency.

Reasoning:

- It is lightweight and close to Go's standard `net/http` model.
- It provides cleaner route parameters and middleware than raw `http.ServeMux`.
- It fits a future REST API server better than ad-hoc path parsing.
- It is less framework-heavy than Gin and keeps the existing project style relatively simple.

Avoid adopting a full backend framework at this stage. The server should still be structured around standard `http.Handler`, request contexts, and explicit dependencies.

### 3. Add typed read APIs to the database store

Modify `internal/db/db.go` so portal handlers do not issue raw SQL through `Store.DB()`.

Reuse the existing `Store`, schema, and `pkg/models.Article` / `pkg/models.Query` shapes. Add:

- `ArticleFilter` with `Source`, `QueryKeyword`, `Limit`, and `Offset`.
- `QueryFilter` with `Platform` and `Status`.
- `ListQueries(ctx, filter)` ordered by `last_run DESC, platform ASC, keyword ASC`.
- `ListArticles(ctx, filter)` ordered by `created_at DESC, id DESC`.
- `CountArticles(ctx, filter)` using the same article filters.
- `GetArticle(ctx, id)` returning `sql.ErrNoRows` for missing rows.

Pagination behavior for article listing:

- Default limit: `50`.
- Maximum limit: `200`.
- Negative offsets become `0`.

Handle nullable DB timestamps (`published_at`, `last_run`) by scanning into `sql.NullTime` and converting invalid values to zero `time.Time`. Avoid changing `pkg/models` in the first implementation.

### 4. Add safe raw-content reading to file storage

Modify `internal/storage/file.go`:

- Add `Read(path string) ([]byte, error)` to `FileStore`.
- Reject empty paths.
- Resolve and clean the storage root and requested path.
- Ensure the requested file is inside `FileStore.Root()` before reading.
- Support the existing behavior where `Save` stores an absolute or root-joined full path in `articles.content_path`.

This lets the portal display stored raw article HTML without allowing arbitrary file reads if a `content_path` value is malformed or malicious.

### 5. Add an internal portal API server package

Create `internal/portal` with a small HTTP server wrapper:

- `NewServer(store *db.Store, files *storage.FileStore) *Server`
- `Handler() http.Handler`

Use `chi` for routing and standard library packages for everything else where possible: `net/http`, `encoding/json`, `html/template` for fallback pages, and related helpers.

Recommended API routes:

- `GET /api/health` — basic health check.
- `GET /api/queries` — JSON query status list.
  - Filters: `platform`, `status`.
- `GET /api/articles` — JSON article list.
  - Filters: `source`, `query`, `limit`, `offset`.
- `GET /api/articles/{id}` — JSON article detail.
- `GET /api/articles/{id}/content` — stored raw content as `text/plain; charset=utf-8`.

Recommended initial HTML routes:

- `GET /` — redirect to `/queries` or render a minimal index page.
- `GET /queries` — minimal HTML table backed by the same DB methods.
- `GET /articles` — minimal HTML article list.
- `GET /articles/{id}` — minimal HTML article detail.

The HTML pages are useful immediately, but keep them thin so they can later be replaced by the Vue frontend without rewriting backend data access.

Handler behavior:

- Return consistent JSON for API success and error responses.
- Map `sql.ErrNoRows` to `404`.
- Return `405` for unsupported methods.
- Parse route params with `chi.URLParam`.
- Parse filters and pagination from query parameters.
- Use `r.Context()` for DB calls.

Security/display rules:

- Use `html/template` for any HTML pages so fields are escaped by default.
- Do not directly inline scraped raw HTML into normal portal pages.
- Serve stored content as `text/plain` or escaped content.
- Keep default listen address on localhost until authentication/access control is added.

### 6. Prepare for future Vue frontend

Do not implement the Vue app in the first portal-server change unless explicitly requested. However, shape the backend so it is ready for it.

Future frontend recommendation:

```text
web/portal/
  package.json
  vite.config.ts
  src/
    main.ts
    App.vue
    pages/
      Queries.vue
      Articles.vue
      ArticleDetail.vue
    api/
      client.ts
      articles.ts
      queries.ts
```

Recommended frontend stack:

- Vue 3
- Vite
- Element Plus
- TypeScript

Future development mode:

- Run Go portal API on `127.0.0.1:8080`.
- Run Vite dev server separately under `web/portal`.
- Add CORS config only when needed for cross-origin dev.

Future production mode:

- Build frontend with `npm run build`.
- Embed frontend `dist` into the Go portal binary with `//go:embed`.
- Serve SPA static assets from the portal server while keeping `/api/*` handled by backend routes.

This keeps local development front-end/back-end separated while preserving simple single-binary deployment.

### 7. Add the new command

Create `cmd/portal/main.go`, mirroring the existing startup style in `cmd/crawler/main.go`:

1. Parse flags with `flag`:
   - `-config`, default `config.yaml`.
   - `-addr`, optional override for `cfg.Portal.Addr`.
2. Load config with `config.Load`.
3. Open SQLite store with `db.Open(cfg.Database.Driver, cfg.Database.DSN)`.
4. Create `storage.NewFileStore(cfg.Storage.ContentRoot)`.
5. Create `portal.NewServer(store, fstore)`.
6. Start an `http.Server` with basic timeouts.
7. Use `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`.
8. On shutdown, call `srv.Shutdown` with a timeout and close the store.

Suggested server timeouts:

- `ReadHeaderTimeout: 5s`
- `ReadTimeout: 10s`
- `WriteTimeout: 30s`
- `IdleTimeout: 60s`

## Critical Files

- `go.mod` / `go.sum` — add `github.com/go-chi/chi/v5`.
- `cmd/portal/main.go` — new portal command entrypoint.
- `internal/portal/server.go` and related handler/template files — new HTTP UI/API package.
- `internal/db/db.go` — add query/article read methods and filters.
- `internal/storage/file.go` — add safe raw content read support.
- `internal/config/config.go` — add portal config/default.
- `config.example.yaml` — document portal address.
- `pkg/models/article.go` — reuse existing `Article` and `Query` JSON/model structs; no planned changes unless implementation reveals a timestamp issue.

Future frontend files, not part of the initial implementation unless requested:

- `web/portal/package.json`
- `web/portal/vite.config.ts`
- `web/portal/src/**`

## Tests

Add tests using the standard library plus the `chi` router through the portal server handler.

### Database tests

Create `internal/db/db_test.go` with a temporary SQLite DB:

- `UpdateQuery` + `ListQueries` returns query status rows.
- `ListQueries` filters by platform/status.
- `UpsertArticle` + `ListArticles` returns article rows.
- `ListArticles` filters by source/query keyword and applies limit/offset.
- `CountArticles` matches the same filters.
- `GetArticle` returns an article by ID and returns `sql.ErrNoRows` for missing IDs.
- Nullable timestamp columns scan successfully.

### Storage tests

Create or extend `internal/storage/file_test.go`:

- `Read` can read a file saved under the storage root.
- `Read` rejects empty paths and paths outside the storage root.
- `Read` returns an error for missing files.

### Portal tests

Create `internal/portal/server_test.go` using `httptest`:

- `GET /api/health` returns `200`.
- `GET /api/queries` returns valid JSON and includes known keyword/status.
- `GET /api/articles` returns valid JSON and includes a known article title.
- `GET /api/articles/{id}` returns `200` for an existing article.
- Missing article detail returns `404`.
- Unsupported methods return `405`.
- Content route serves raw stored content as `text/plain`, not executable portal HTML.
- Minimal HTML pages return `200` if included in the first implementation.

## Verification

Run automated checks:

```sh
go test ./...
go build ./...
```

Manual verification:

```sh
go run ./cmd/portal -config config.example.yaml
```

Then in another terminal:

```sh
curl -i http://127.0.0.1:8080/
curl -i http://127.0.0.1:8080/api/health
curl -i http://127.0.0.1:8080/api/queries
curl -i http://127.0.0.1:8080/api/articles
```

If minimal HTML pages are implemented:

```sh
curl -i http://127.0.0.1:8080/queries
curl -i http://127.0.0.1:8080/articles
```

If local data is empty, first populate the database with the crawler once:

```sh
go run ./cmd/crawler -config config.example.yaml -once
```

Also verify graceful shutdown by starting the portal, pressing `Ctrl+C`, and confirming it exits cleanly after logging shutdown.
