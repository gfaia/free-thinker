package models

import "time"

type Article struct {
	ID           int64
	URL          string
	Title        string
	Author       string
	PublishedAt  time.Time
	Source       string
	QueryKeyword string
	Summary      string
	ContentPath  string
	CreatedAt    time.Time
}

type Query struct {
	ID       int64
	Keyword  string
	Platform string
	LastRun  time.Time
	Status   string
}
