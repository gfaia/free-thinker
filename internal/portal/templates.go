package portal

const pageTemplates = `
{{define "layout-start"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Free Thinker Portal</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 2rem; color: #17202a; }
    nav { margin-bottom: 1.5rem; }
    nav a { margin-right: 1rem; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #d5d8dc; padding: .5rem; text-align: left; vertical-align: top; }
    th { background: #f4f6f7; }
    .muted { color: #7f8c8d; }
    .summary { max-width: 48rem; }
  </style>
</head>
<body>
<nav><a href="/queries">Queries</a><a href="/articles">Articles</a><a href="/api/health">API Health</a></nav>
{{end}}

{{define "layout-end"}}
</body>
</html>
{{end}}

{{define "queries"}}
{{template "layout-start" .}}
<h1>Query Tasks</h1>
<table>
  <thead><tr><th>ID</th><th>Keyword</th><th>Platform</th><th>Last Run</th><th>Status</th></tr></thead>
  <tbody>
  {{range .Queries}}
    <tr><td>{{.ID}}</td><td>{{.Keyword}}</td><td>{{.Platform}}</td><td>{{.LastRun}}</td><td>{{.Status}}</td></tr>
  {{else}}
    <tr><td colspan="5" class="muted">No query task records found.</td></tr>
  {{end}}
  </tbody>
</table>
{{template "layout-end" .}}
{{end}}

{{define "articles"}}
{{template "layout-start" .}}
<h1>Articles</h1>
<p class="muted">Total: {{.Total}}, limit: {{.Limit}}, offset: {{.Offset}}</p>
<table>
  <thead><tr><th>ID</th><th>Title</th><th>Source</th><th>Query</th><th>Author</th><th>Created</th></tr></thead>
  <tbody>
  {{range .Articles}}
    <tr>
      <td>{{.ID}}</td>
      <td><a href="/articles/{{.ID}}">{{.Title}}</a><br><a href="{{.URL}}">{{.URL}}</a></td>
      <td>{{.Source}}</td><td>{{.QueryKeyword}}</td><td>{{.Author}}</td><td>{{.CreatedAt}}</td>
    </tr>
  {{else}}
    <tr><td colspan="6" class="muted">No articles found.</td></tr>
  {{end}}
  </tbody>
</table>
{{template "layout-end" .}}
{{end}}

{{define "article"}}
{{template "layout-start" .}}
<h1>{{.Article.Title}}</h1>
<p><a href="{{.Article.URL}}">{{.Article.URL}}</a></p>
<dl>
  <dt>ID</dt><dd>{{.Article.ID}}</dd>
  <dt>Source</dt><dd>{{.Article.Source}}</dd>
  <dt>Query</dt><dd>{{.Article.QueryKeyword}}</dd>
  <dt>Author</dt><dd>{{.Article.Author}}</dd>
  <dt>Published</dt><dd>{{.Article.PublishedAt}}</dd>
  <dt>Created</dt><dd>{{.Article.CreatedAt}}</dd>
</dl>
<p class="summary">{{.Article.Summary}}</p>
{{if .Article.ContentPath}}<p><a href="/api/articles/{{.Article.ID}}/content">View stored raw content as text</a></p>{{end}}
{{template "layout-end" .}}
{{end}}
`
