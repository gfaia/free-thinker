package models

import "time"

type Article struct {
	ID           int64     `db:"id"            json:"id"`
	URL          string    `db:"url"           json:"url"`
	Title        string    `db:"title"         json:"title"`
	Author       string    `db:"author"        json:"author"`
	PublishedAt  time.Time `db:"published_at"  json:"published_at"`
	Source       string    `db:"source"        json:"source"`
	QueryKeyword string    `db:"query_keyword" json:"query_keyword"`
	ContentPath  string    `db:"content_path"  json:"content_path"`
	Summary      string    `db:"summary"       json:"summary"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}

type Query struct {
	ID        int64     `db:"id"         json:"id"`
	Keyword   string    `db:"keyword"    json:"keyword"`
	Platform  string    `db:"platform"   json:"platform"`
	LastRun   time.Time `db:"last_run"   json:"last_run"`
	Status    string    `db:"status"     json:"status"`
}
