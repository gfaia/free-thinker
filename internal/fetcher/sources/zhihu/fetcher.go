package zhihu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gfaia/free-thinker/internal/fetcher"
)

const (
	defaultBaseURL = "https://www.zhihu.com"
	pageSize       = 20
	maxSummaryLen  = 500
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Fetcher struct {
	maxPages    int
	rateLimitMS int
	userAgent   string
	cookie      string
	httpClient  httpDoer
	baseURL     string
}

func New() *Fetcher {
	return &Fetcher{
		maxPages:    1,
		rateLimitMS: 800,
		userAgent:   "Mozilla/5.0 (compatible; free-thinker/0.1)",
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		baseURL:     defaultBaseURL,
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
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			f.userAgent = strings.TrimSpace(s)
		}
	}
	if v, ok := rawConfig["cookie"]; ok {
		if s, ok := v.(string); ok {
			f.cookie = strings.TrimSpace(s)
		}
	}
	if f.maxPages <= 0 {
		f.maxPages = 1
	}
	if f.rateLimitMS < 0 {
		f.rateLimitMS = 0
	}
	return nil
}

func (f *Fetcher) Fetch(ctx context.Context, keyword string) ([]fetcher.Article, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("empty keyword")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.httpClient == nil {
		f.httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if f.baseURL == "" {
		f.baseURL = defaultBaseURL
	}

	maxPages := f.maxPages
	if maxPages <= 0 {
		maxPages = 1
	}
	rateLimit := time.Duration(f.rateLimitMS) * time.Millisecond
	if rateLimit < 0 {
		rateLimit = 0
	}

	articles := make([]fetcher.Article, 0, maxPages*pageSize)
	seen := make(map[string]struct{})
	for page := 0; page < maxPages; page++ {
		if err := ctx.Err(); err != nil {
			return articles, err
		}

		req, err := f.newSearchRequest(ctx, keyword, page*pageSize, pageSize)
		if err != nil {
			return articles, err
		}
		resp, err := f.httpClient.Do(req)
		if err != nil {
			return articles, fmt.Errorf("zhihu search request: %w", err)
		}

		pageArticles, isEnd, err := parseSearchResponse(resp)
		if err != nil {
			if page == 0 && isSearchFallbackError(err) {
				return f.fetchSuggestions(ctx, keyword)
			}
			return articles, err
		}
		for _, article := range pageArticles {
			key := article.URL + "\x00" + article.Title
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			articles = append(articles, article)
		}
		if isEnd || page == maxPages-1 {
			break
		}
		if rateLimit > 0 {
			select {
			case <-ctx.Done():
				return articles, ctx.Err()
			case <-time.After(rateLimit):
			}
		}
	}
	return articles, nil
}

func (f *Fetcher) newSearchRequest(ctx context.Context, keyword string, offset, limit int) (*http.Request, error) {
	base, err := url.Parse(f.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse zhihu base url: %w", err)
	}
	base.Path = "/api/v4/search_v3"
	q := base.Query()
	q.Set("t", "general")
	q.Set("q", keyword)
	q.Set("correction", "1")
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("lc_idx", strconv.Itoa(offset))
	q.Set("show_all_topics", "0")
	base.RawQuery = q.Encode()
	return f.newJSONRequest(ctx, base.String(), keyword)
}

func (f *Fetcher) newSuggestRequest(ctx context.Context, keyword string) (*http.Request, error) {
	base, err := url.Parse(f.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse zhihu base url: %w", err)
	}
	base.Path = "/api/v4/search/suggest"
	q := base.Query()
	q.Set("q", keyword)
	base.RawQuery = q.Encode()
	return f.newJSONRequest(ctx, base.String(), keyword)
}

func (f *Fetcher) newJSONRequest(ctx context.Context, target, keyword string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("create zhihu request: %w", err)
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Origin", defaultBaseURL)
	req.Header.Set("Referer", defaultBaseURL+"/search?type=content&q="+url.QueryEscape(keyword))
	if f.cookie != "" {
		req.Header.Set("Cookie", f.cookie)
	}
	return req, nil
}

type statusError struct {
	StatusCode int
	Body       string
}

func (e statusError) Error() string {
	return fmt.Sprintf("zhihu search status %d: %s", e.StatusCode, e.Body)
}

type searchResponse struct {
	Data   []searchItem `json:"data"`
	Paging struct {
		IsEnd bool `json:"is_end"`
	} `json:"paging"`
}

type searchItem struct {
	Object    searchObject    `json:"object"`
	Highlight searchHighlight `json:"highlight"`
}

type searchHighlight struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type searchObject struct {
	ID          flexibleID    `json:"id"`
	Type        string        `json:"type"`
	URL         string        `json:"url"`
	Title       string        `json:"title"`
	Name        string        `json:"name"`
	Excerpt     string        `json:"excerpt"`
	Content     string        `json:"content"`
	Author      searchAuthor  `json:"author"`
	Question    *searchObject `json:"question"`
	Created     int64         `json:"created"`
	CreatedTime int64         `json:"created_time"`
	UpdatedTime int64         `json:"updated_time"`
}

type searchAuthor struct {
	Name string `json:"name"`
}

type suggestResponse struct {
	Suggest []suggestItem `json:"suggest"`
}

type suggestItem struct {
	Query       string     `json:"query"`
	RawID       flexibleID `json:"raw_id"`
	TargetURL   string     `json:"target_url"`
	TabType     string     `json:"tab_type"`
	AttachedRaw string     `json:"attached_info"`
}

type flexibleID string

func (id *flexibleID) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*id = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*id = flexibleID(s)
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		*id = flexibleID(strconv.FormatInt(n, 10))
		return nil
	}
	return nil
}

func parseSearchResponse(resp *http.Response) ([]fetcher.Article, bool, error) {
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, false, statusError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(b))}
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, fmt.Errorf("decode zhihu search response: %w", err)
	}

	articles := make([]fetcher.Article, 0, len(result.Data))
	for _, item := range result.Data {
		article, ok := articleFromItem(item)
		if !ok {
			continue
		}
		articles = append(articles, article)
	}
	return articles, result.Paging.IsEnd, nil
}

func parseSuggestResponse(resp *http.Response) ([]fetcher.Article, error) {
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, statusError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(b))}
	}

	var result suggestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode zhihu suggest response: %w", err)
	}

	articles := make([]fetcher.Article, 0, len(result.Suggest))
	seen := make(map[string]struct{})
	for _, item := range result.Suggest {
		article, ok := articleFromSuggest(item)
		if !ok {
			continue
		}
		key := article.URL + "\x00" + article.Title
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		articles = append(articles, article)
	}
	return articles, nil
}

func (f *Fetcher) fetchSuggestions(ctx context.Context, keyword string) ([]fetcher.Article, error) {
	req, err := f.newSuggestRequest(ctx, keyword)
	if err != nil {
		return nil, err
	}
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zhihu suggest request: %w", err)
	}
	return parseSuggestResponse(resp)
}

func isSearchFallbackError(err error) bool {
	var status statusError
	if !errors.As(err, &status) {
		return false
	}
	return status.StatusCode == http.StatusBadRequest || status.StatusCode == http.StatusForbidden
}

func articleFromItem(item searchItem) (fetcher.Article, bool) {
	obj := item.Object
	question := obj.Question

	title := cleanText(firstNonEmpty(obj.Title, questionTitle(question), obj.Name, item.Highlight.Title))
	articleURL := normalizeURL(firstNonEmpty(obj.URL, questionURL(question)), string(obj.ID), questionID(question), obj.Type)
	if title == "" || articleURL == "" {
		return fetcher.Article{}, false
	}

	summary := cleanText(firstNonEmpty(obj.Excerpt, item.Highlight.Description, obj.Content))
	return fetcher.Article{
		URL:         articleURL,
		Title:       title,
		Author:      cleanText(obj.Author.Name),
		PublishedAt: firstPositive(obj.CreatedTime, obj.Created, obj.UpdatedTime),
		Summary:     truncateRunes(summary, maxSummaryLen),
		RawHTML:     obj.Content,
	}, true
}

func articleFromSuggest(item suggestItem) (fetcher.Article, bool) {
	title := cleanText(item.Query)
	if title == "" {
		return fetcher.Article{}, false
	}
	articleURL := strings.TrimSpace(item.TargetURL)
	if articleURL == "" {
		articleURL = defaultBaseURL + "/search?type=content&q=" + url.QueryEscape(title)
	}
	return fetcher.Article{
		URL:     articleURL,
		Title:   title,
		Summary: cleanText(firstNonEmpty(item.TabType, item.AttachedRaw)),
	}, true
}

var tagPattern = regexp.MustCompile(`<[^>]*>`)

func cleanText(s string) string {
	s = html.UnescapeString(s)
	s = tagPattern.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func normalizeURL(raw, objectID, questionID, objectType string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Host == "" {
		u.Host = "www.zhihu.com"
	}

	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) >= 4 && parts[0] == "api" && parts[1] == "v4" {
		switch parts[2] {
		case "articles":
			id := firstNonEmpty(objectID, parts[3])
			if id != "" {
				return "https://zhuanlan.zhihu.com/p/" + id
			}
		case "answers":
			answerID := firstNonEmpty(objectID, parts[3])
			if answerID != "" && questionID != "" {
				return "https://www.zhihu.com/question/" + questionID + "/answer/" + answerID
			}
		case "questions":
			id := firstNonEmpty(questionID, objectID, parts[3])
			if id != "" {
				return "https://www.zhihu.com/question/" + id
			}
		}
	}
	if objectType == "article" && objectID != "" && strings.Contains(u.Host, "zhihu.com") && strings.Contains(path, "articles") {
		return "https://zhuanlan.zhihu.com/p/" + objectID
	}
	return u.String()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func firstPositive(values ...int64) int64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

func questionTitle(q *searchObject) string {
	if q == nil {
		return ""
	}
	return q.Title
}

func questionURL(q *searchObject) string {
	if q == nil {
		return ""
	}
	return q.URL
}

func questionID(q *searchObject) string {
	if q == nil {
		return ""
	}
	return string(q.ID)
}

func asInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case int32:
		return int(n), true
	case uint:
		return int(n), true
	case uint64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

var _ fetcher.Fetcher = (*Fetcher)(nil)
