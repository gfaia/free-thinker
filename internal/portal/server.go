package portal

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/gfaia/free-thinker/internal/db"
	"github.com/gfaia/free-thinker/internal/storage"
	"github.com/gfaia/free-thinker/pkg/models"
)

type Server struct {
	store     *db.Store
	files     *storage.FileStore
	templates *template.Template
}

func NewServer(store *db.Store, files *storage.FileStore) *Server {
	return &Server{
		store:     store,
		files:     files,
		templates: template.Must(template.New("portal").Parse(pageTemplates)),
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Get("/", s.handleIndex)
	r.Get("/queries", s.handleQueriesPage)
	r.Get("/articles", s.handleArticlesPage)
	r.Get("/articles/{id}", s.handleArticlePage)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Get("/queries", s.handleQueriesAPI)
		r.Get("/articles", s.handleArticlesAPI)
		r.Get("/articles/{id}", s.handleArticleAPI)
		r.Get("/articles/{id}/content", s.handleArticleContent)
	})

	return r
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/queries", http.StatusFound)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleQueriesAPI(w http.ResponseWriter, r *http.Request) {
	queries, err := s.store.ListQueries(r.Context(), queryFilter(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": queries})
}

func (s *Server) handleArticlesAPI(w http.ResponseWriter, r *http.Request) {
	filter := articleFilter(r)
	articles, err := s.store.ListArticles(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	count, err := s.store.CountArticles(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  articles,
		"total":  count,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

func (s *Server) handleArticleAPI(w http.ResponseWriter, r *http.Request) {
	article, ok := s.articleByID(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, article)
}

func (s *Server) handleArticleContent(w http.ResponseWriter, r *http.Request) {
	article, ok := s.articleByID(w, r)
	if !ok {
		return
	}
	if article.ContentPath == "" {
		writeErrorMessage(w, http.StatusNotFound, "article content not found")
		return
	}
	b, err := s.files.Read(article.ContentPath)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (s *Server) handleQueriesPage(w http.ResponseWriter, r *http.Request) {
	queries, err := s.store.ListQueries(r.Context(), queryFilter(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "queries", map[string]interface{}{"Queries": queries})
}

func (s *Server) handleArticlesPage(w http.ResponseWriter, r *http.Request) {
	filter := articleFilter(r)
	articles, err := s.store.ListArticles(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	count, err := s.store.CountArticles(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "articles", map[string]interface{}{
		"Articles": articles,
		"Total":    count,
		"Limit":    filter.Limit,
		"Offset":   filter.Offset,
	})
}

func (s *Server) handleArticlePage(w http.ResponseWriter, r *http.Request) {
	article, ok := s.articleByID(w, r)
	if !ok {
		return
	}
	s.render(w, "article", map[string]interface{}{"Article": article})
}

func (s *Server) articleByID(w http.ResponseWriter, r *http.Request) (*models.Article, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeErrorMessage(w, http.StatusBadRequest, "invalid article id")
		return nil, false
	}
	article, err := s.store.GetArticle(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeErrorMessage(w, http.StatusNotFound, "article not found")
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	return article, true
}

func (s *Server) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func queryFilter(r *http.Request) db.QueryFilter {
	q := r.URL.Query()
	return db.QueryFilter{
		Platform: q.Get("platform"),
		Status:   q.Get("status"),
	}
}

func articleFilter(r *http.Request) db.ArticleFilter {
	q := r.URL.Query()
	return db.ArticleFilter{
		Source:       q.Get("source"),
		QueryKeyword: q.Get("query"),
		Limit:        parseInt(q.Get("limit")),
		Offset:       parseInt(q.Get("offset")),
	}
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeErrorMessage(w, status, err.Error())
}

func writeErrorMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
