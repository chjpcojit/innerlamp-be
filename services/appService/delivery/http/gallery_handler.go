package http

import (
	"context"
	"net/http"
	"strconv"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// galleryUsecase is the subset of *usecase.GalleryUsecase this handler depends
// on, declared here (the consumer) so the handler is tested with a mock.
type galleryUsecase interface {
	Top(ctx context.Context) ([]domain.GalleryImage, error)
	List(ctx context.Context, imageType string, page int) (*domain.GalleryPage, error)
	Get(ctx context.Context, id string) (*domain.GalleryImage, error)
}

// GalleryHandler serves the read-only Buddhist photo gallery. All routes are
// public: browsing needs no account.
type GalleryHandler struct {
	usecase galleryUsecase
	imageHost
}

// NewGalleryHandler wires the handler.
func NewGalleryHandler(usecase galleryUsecase, opts ...ImageOption) *GalleryHandler {
	return &GalleryHandler{usecase: usecase, imageHost: newImageHost(opts)}
}

// gallerySummaryResponse is the card shape shared by top and list.
type gallerySummaryResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	Location     string `json:"location"`
	ThumbnailURL string `json:"thumbnail_url"`
	ImageURL     string `json:"image_url"`
}

// galleryDetailResponse is the summary plus the attribution the image's licence requires.
type galleryDetailResponse struct {
	gallerySummaryResponse
	Author    string `json:"author"`
	License   string `json:"license"`
	SourceURL string `json:"source_url"`
}

type galleryTopResponse struct {
	Items []gallerySummaryResponse `json:"items"`
}

type galleryListResponse struct {
	Items      []gallerySummaryResponse `json:"items"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	Total      int                      `json:"total"`
	TotalPages int                      `json:"total_pages"`
}

func newGallerySummary(h imageHost, r *http.Request, g domain.GalleryImage) gallerySummaryResponse {
	return gallerySummaryResponse{
		ID: g.ID, Type: g.Type, Title: g.Title, Location: g.Location,
		ThumbnailURL: h.url(r, g.ThumbnailURL), ImageURL: h.url(r, g.ImageURL),
	}
}

func newGallerySummaries(h imageHost, r *http.Request, gs []domain.GalleryImage) []gallerySummaryResponse {
	out := make([]gallerySummaryResponse, 0, len(gs))
	for _, g := range gs {
		out = append(out, newGallerySummary(h, r, g))
	}
	return out
}

// Top serves GET /v1/gallery/top.
func (h *GalleryHandler) Top(w http.ResponseWriter, r *http.Request) {
	items, err := h.usecase.Top(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, galleryTopResponse{Items: newGallerySummaries(h.imageHost, r, items)})
}

// List serves GET /v1/gallery?type=T&page=N. type is optional (empty = every
// type); page defaults to 1 and anything that is not an integer is 400 INVALID_PAGE.
func (h *GalleryHandler) List(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil {
			writeDomainError(w, domain.ErrInvalidPage)
			return
		}
		page = p
	}

	result, err := h.usecase.List(r.Context(), r.URL.Query().Get("type"), page)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, galleryListResponse{
		Items: newGallerySummaries(h.imageHost, r, result.Items), Page: result.Page, PageSize: result.PageSize,
		Total: result.Total, TotalPages: result.TotalPages,
	})
}

// Get serves GET /v1/gallery/{id}.
func (h *GalleryHandler) Get(w http.ResponseWriter, r *http.Request) {
	g, err := h.usecase.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, galleryDetailResponse{
		gallerySummaryResponse: newGallerySummary(h.imageHost, r, *g),
		Author:                 g.Author, License: g.License, SourceURL: g.SourceURL,
	})
}
