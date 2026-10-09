package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// articleUsecase is the subset of *usecase.ArticleUsecase this handler depends
// on, declared here (the consumer) so the handler is tested with a mock.
type articleUsecase interface {
	Top(ctx context.Context) ([]domain.Article, error)
	List(ctx context.Context, page int) (*domain.ArticlePage, error)
	Get(ctx context.Context, id string) (*domain.Article, error)
}

// ArticleHandler serves the read-only Phật pháp article catalogue. All routes
// are public: browsing needs no account.
type ArticleHandler struct {
	usecase articleUsecase
	imageHost
}

// NewArticleHandler wires the handler.
func NewArticleHandler(usecase articleUsecase, opts ...ImageOption) *ArticleHandler {
	return &ArticleHandler{usecase: usecase, imageHost: newImageHost(opts)}
}

// articleSummaryResponse is the card shape shared by top and list: no body.
// published_date ("YYYY-MM-DD"), image_url and thumbnail_url are JSON null when unknown.
type articleSummaryResponse struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Category       string   `json:"category"`
	Author         string   `json:"author"`
	AuthorInitials string   `json:"author_initials"`
	PublishedDate  *string  `json:"published_date"`
	ReadMinutes    int      `json:"read_minutes"`
	Summary        string   `json:"summary"`
	Tags           []string `json:"tags"`
	ImageURL       *string  `json:"image_url"`
	ThumbnailURL   *string  `json:"thumbnail_url"`
}

// articleDetailResponse is the summary plus the full text and attribution.
type articleDetailResponse struct {
	articleSummaryResponse
	BodyParagraphs []string `json:"body_paragraphs"`
	Quote          string   `json:"quote"`
	QuoteSource    string   `json:"quote_source"`
	Language       string   `json:"language"`
	SourceName     string   `json:"source_name"`
	SourceURL      string   `json:"source_url"`
	License        string   `json:"license"`
}

type articleTopResponse struct {
	Items []articleSummaryResponse `json:"items"`
}

type articleListResponse struct {
	Items      []articleSummaryResponse `json:"items"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	Total      int                      `json:"total"`
	TotalPages int                      `json:"total_pages"`
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func newArticleSummary(h imageHost, r *http.Request, a domain.Article) articleSummaryResponse {
	out := articleSummaryResponse{
		ID: a.ID, Title: a.Title, Category: a.Category, Author: a.Author, AuthorInitials: a.AuthorInitials,
		ReadMinutes: a.ReadMinutes, Summary: a.Summary, Tags: nonNil(a.Tags),
	}
	if a.PublishedDate != nil {
		d := a.PublishedDate.Format(time.DateOnly)
		out.PublishedDate = &d
	}
	out.ImageURL = h.ptr(r, a.ImageURL)
	out.ThumbnailURL = h.ptr(r, a.ThumbnailURL)
	return out
}

func newArticleSummaries(h imageHost, r *http.Request, as []domain.Article) []articleSummaryResponse {
	out := make([]articleSummaryResponse, 0, len(as))
	for _, a := range as {
		out = append(out, newArticleSummary(h, r, a))
	}
	return out
}

// Top serves GET /v1/articles/top.
func (h *ArticleHandler) Top(w http.ResponseWriter, r *http.Request) {
	items, err := h.usecase.Top(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, articleTopResponse{Items: newArticleSummaries(h.imageHost, r, items)})
}

// List serves GET /v1/articles?page=N. page defaults to 1; anything that is not
// a positive integer is 400 INVALID_PAGE.
func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil {
			writeDomainError(w, domain.ErrInvalidPage)
			return
		}
		page = p
	}

	result, err := h.usecase.List(r.Context(), page)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, articleListResponse{
		Items: newArticleSummaries(h.imageHost, r, result.Items), Page: result.Page, PageSize: result.PageSize,
		Total: result.Total, TotalPages: result.TotalPages,
	})
}

// Get serves GET /v1/articles/{id}.
func (h *ArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
	a, err := h.usecase.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, articleDetailResponse{
		articleSummaryResponse: newArticleSummary(h.imageHost, r, *a),
		BodyParagraphs:         nonNil(a.BodyParagraphs),
		Quote:                  a.Quote, QuoteSource: a.QuoteSource, Language: a.Language,
		SourceName: a.SourceName, SourceURL: a.SourceURL, License: a.License,
	})
}
