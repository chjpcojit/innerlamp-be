package http

import (
	"context"
	"net/http"
	"strconv"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// deityUsecase is the subset of *usecase.DeityUsecase this handler depends on,
// declared here (the consumer) so the handler is tested with a mock.
type deityUsecase interface {
	Top(ctx context.Context) ([]domain.Deity, error)
	ListDeities(ctx context.Context, page int) (*domain.DeityPage, error)
	Get(ctx context.Context, id string) (*domain.Deity, error)
}

// DeityHandler serves the read-only Phật / Bồ tát catalogue. All routes are
// public: browsing needs no account.
type DeityHandler struct {
	usecase deityUsecase
	imageHost
}

// NewDeityHandler wires the handler.
func NewDeityHandler(usecase deityUsecase, opts ...ImageOption) *DeityHandler {
	return &DeityHandler{usecase: usecase, imageHost: newImageHost(opts)}
}

// deitySummaryResponse is the card shape shared by top and list.
type deitySummaryResponse struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	NameVI         string `json:"name_vi"`
	NameSanskrit   string `json:"name_sanskrit"`
	NameChinese    string `json:"name_chinese"`
	TitleHonorific string `json:"title_honorific"`
	ThumbnailURL   string `json:"thumbnail_url"`
	ImageURL       string `json:"image_url"`
	CoreVirtue     string `json:"core_virtue"`
	PureLand       string `json:"pure_land"`
}

type memorialDayResponse struct {
	LunarDay   int    `json:"lunar_day"`
	LunarMonth int    `json:"lunar_month"`
	EventName  string `json:"event_name"`
}

type iconographyResponse struct {
	Posture         string   `json:"posture"`
	MountOrThrone   string   `json:"mount_or_throne"`
	SacredArtifacts []string `json:"sacred_artifacts"`
	SymbolicMeaning string   `json:"symbolic_meaning"`
}

type mantrasResponse struct {
	ShortMantraSanskrit       string `json:"short_mantra_sanskrit"`
	ShortMantraSinoVietnamese string `json:"short_mantra_sino_vietnamese"`
	LongDharaniName           string `json:"long_dharani_name"`
	DharaniPhoneticSnippet    string `json:"dharani_phonetic_snippet"`
}

// deityDetailResponse is the summary plus everything the detail screen shows.
type deityDetailResponse struct {
	deitySummaryResponse
	CanonicalSource   string                `json:"canonical_source"`
	LunarMemorialDays []memorialDayResponse `json:"lunar_memorial_days"`
	Iconography       iconographyResponse   `json:"iconography"`
	BiographySummary  string                `json:"biography_summary"`
	GreatVows         []string              `json:"great_vows"`
	Mantras           mantrasResponse       `json:"mantras"`
}

type deityTopResponse struct {
	Items []deitySummaryResponse `json:"items"`
}

type deityListResponse struct {
	Items      []deitySummaryResponse `json:"items"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	Total      int                    `json:"total"`
	TotalPages int                    `json:"total_pages"`
}

func newDeitySummary(h imageHost, r *http.Request, d domain.Deity) deitySummaryResponse {
	return deitySummaryResponse{
		ID: d.ID, Type: d.Type, NameVI: d.NameVI, NameSanskrit: d.NameSanskrit, NameChinese: d.NameChinese,
		TitleHonorific: d.TitleHonorific, ThumbnailURL: h.url(r, d.ThumbnailURL), ImageURL: h.url(r, d.ImageURL),
		CoreVirtue: d.CoreVirtue, PureLand: d.PureLand,
	}
}

func newDeitySummaries(h imageHost, r *http.Request, ds []domain.Deity) []deitySummaryResponse {
	out := make([]deitySummaryResponse, 0, len(ds))
	for _, d := range ds {
		out = append(out, newDeitySummary(h, r, d))
	}
	return out
}

// Top serves GET /v1/deities/top.
func (h *DeityHandler) Top(w http.ResponseWriter, r *http.Request) {
	items, err := h.usecase.Top(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, deityTopResponse{Items: newDeitySummaries(h.imageHost, r, items)})
}

// ListDeities serves GET /v1/deities?page=N: Buddhas and bodhisattvas in one
// paged list (each item carries its type). page defaults to 1; anything that
// is not a positive integer is 400 INVALID_PAGE.
func (h *DeityHandler) ListDeities(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil {
			writeDomainError(w, domain.ErrInvalidPage)
			return
		}
		page = p
	}

	result, err := h.usecase.ListDeities(r.Context(), page)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, deityListResponse{
		Items: newDeitySummaries(h.imageHost, r, result.Items), Page: result.Page, PageSize: result.PageSize,
		Total: result.Total, TotalPages: result.TotalPages,
	})
}

// Get serves GET /v1/deities/{id}.
func (h *DeityHandler) Get(w http.ResponseWriter, r *http.Request) {
	d, err := h.usecase.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	days := make([]memorialDayResponse, 0, len(d.LunarMemorialDays))
	for _, m := range d.LunarMemorialDays {
		days = append(days, memorialDayResponse{LunarDay: m.LunarDay, LunarMonth: m.LunarMonth, EventName: m.EventName})
	}
	artifacts, vows := d.Iconography.SacredArtifacts, d.GreatVows
	if artifacts == nil {
		artifacts = []string{}
	}
	if vows == nil {
		vows = []string{}
	}
	utils.WriteSuccessResponse(w, deityDetailResponse{
		deitySummaryResponse: newDeitySummary(h.imageHost, r, *d),
		CanonicalSource:      d.CanonicalSource,
		LunarMemorialDays:    days,
		Iconography: iconographyResponse{
			Posture: d.Iconography.Posture, MountOrThrone: d.Iconography.MountOrThrone,
			SacredArtifacts: artifacts, SymbolicMeaning: d.Iconography.SymbolicMeaning,
		},
		BiographySummary: d.BiographySummary,
		GreatVows:        vows,
		Mantras: mantrasResponse{
			ShortMantraSanskrit: d.Mantras.ShortSanskrit, ShortMantraSinoVietnamese: d.Mantras.ShortSinoVietnamese,
			LongDharaniName: d.Mantras.LongDharaniName, DharaniPhoneticSnippet: d.Mantras.DharaniPhoneticSnippet,
		},
	})
}
