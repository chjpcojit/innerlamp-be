package http

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// sutraUsecase is the subset of *usecase.SutraUsecase this handler depends on,
// declared here (the consumer) so the handler is tested with a mock.
type sutraUsecase interface {
	List(ctx context.Context, in domain.SutraFilterInput, page int) (*domain.SutraPage, error)
	Get(ctx context.Context, id string) (*domain.Sutra, error)
	Lines(ctx context.Context, sutraID, key string, page int, pageSize *int) (*domain.SutraLinesPage, error)
	ListByDeity(ctx context.Context, deityID string, role *string, page int) (*domain.DeitySutraPage, error)
}

// SutraHandler serves the read-only kinh / kệ / chú library. All routes are public:
// reading needs no account.
type SutraHandler struct {
	usecase sutraUsecase
	imageHost
}

// NewSutraHandler wires the handler.
func NewSutraHandler(usecase sutraUsecase, opts ...ImageOption) *SutraHandler {
	return &SutraHandler{usecase: usecase, imageHost: newImageHost(opts)}
}

// sutraDeityResponse is a deity linked to a sutra, with what a card needs to show it.
type sutraDeityResponse struct {
	DeityID      string `json:"deity_id"`
	Role         string `json:"role"`
	NameVI       string `json:"name_vi"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// sutraTranslationSummaryResponse is a translation as the list shows it: enough to
// pick one, no sources.
type sutraTranslationSummaryResponse struct {
	Key        string `json:"key"`
	Lang       string `json:"lang"`
	Variant    string `json:"variant"`
	Translator string `json:"translator"`
	License    string `json:"license"`
	LineCount  int    `json:"line_count"`
}

type sutraSectionResponse struct {
	Name      string `json:"name"`
	StartN    int    `json:"start_n"`
	LineCount int    `json:"line_count"`
}

// sutraTranslationDetailResponse adds the attribution and the table of contents.
type sutraTranslationDetailResponse struct {
	sutraTranslationSummaryResponse
	SourceName  string                 `json:"source_name"`
	SourceURL   string                 `json:"source_url"`
	LicenseURL  string                 `json:"license_url"`
	LicenseNote string                 `json:"license_note"`
	Note        string                 `json:"note"`
	Sections    []sutraSectionResponse `json:"sections"`
}

// sutraHeadResponse is what every sutra shape shares. duration_minutes is JSON null
// when no estimate was made; every array is [] rather than null.
type sutraHeadResponse struct {
	ID              string               `json:"id"`
	Tradition       string               `json:"tradition"`
	Kind            string               `json:"kind"`
	Collection      string               `json:"collection"`
	Reference       string               `json:"reference"`
	Titles          map[string]string    `json:"titles"`
	DailyChant      bool                 `json:"daily_chant"`
	Purposes        []string             `json:"purposes"`
	DurationMinutes *int                 `json:"duration_minutes"`
	AudioURL        string               `json:"audio_url"`
	Summary         string               `json:"summary"`
	Deities         []sutraDeityResponse `json:"deities"`
}

// sutraSummaryResponse is the card shape shared by the list and the by-deity list.
type sutraSummaryResponse struct {
	sutraHeadResponse
	Translations []sutraTranslationSummaryResponse `json:"translations"`
}

// sutraDetailResponse is the head plus what only the detail carries: the review
// status, the reference split into its own fields (reference itself stays in the head
// as the bare citation code), and per translation the sources,
// licence and sections. han_translation_year is JSON null when the year is unknown;
// every string is "" when there is nothing to say.
type sutraDetailResponse struct {
	sutraHeadResponse
	ReviewStatus       string                           `json:"review_status"`
	OriginalTitle      string                           `json:"original_title"`
	Author             string                           `json:"author"`
	HanTranslator      string                           `json:"han_translator"`
	HanTranslationYear *int                             `json:"han_translation_year"`
	ReferenceNote      string                           `json:"reference_note"`
	Translations       []sutraTranslationDetailResponse `json:"translations"`
}

// sutraByDeityItemResponse is a summary plus the role the asked-about deity plays.
type sutraByDeityItemResponse struct {
	sutraSummaryResponse
	Role string `json:"role"`
}

type sutraListResponse struct {
	Items      []sutraSummaryResponse `json:"items"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	Total      int                    `json:"total"`
	TotalPages int                    `json:"total_pages"`
}

type sutraByDeityListResponse struct {
	Items      []sutraByDeityItemResponse `json:"items"`
	Page       int                        `json:"page"`
	PageSize   int                        `json:"page_size"`
	Total      int                        `json:"total"`
	TotalPages int                        `json:"total_pages"`
}

// sutraLineResponse is one display line; section is JSON null when it belongs to none.
type sutraLineResponse struct {
	N       int     `json:"n"`
	Text    string  `json:"text"`
	Section *string `json:"section"`
}

type sutraLinesResponse struct {
	Translation string              `json:"translation"`
	Items       []sutraLineResponse `json:"items"`
	Page        int                 `json:"page"`
	PageSize    int                 `json:"page_size"`
	Total       int                 `json:"total"`
	TotalPages  int                 `json:"total_pages"`
}

func newSutraHead(h imageHost, r *http.Request, s domain.Sutra) sutraHeadResponse {
	titles := s.Titles
	if titles == nil {
		titles = map[string]string{}
	}
	purposes := make([]string, len(s.Purposes))
	for i, p := range s.Purposes {
		purposes[i] = string(p)
	}
	deities := make([]sutraDeityResponse, len(s.Deities))
	for i, d := range s.Deities {
		deities[i] = sutraDeityResponse{DeityID: d.DeityID, Role: string(d.Role), NameVI: d.NameVI, ThumbnailURL: h.url(r, d.ThumbnailURL)}
	}
	return sutraHeadResponse{
		ID: s.ID, Tradition: string(s.Tradition), Kind: string(s.Kind), Collection: s.Collection,
		Reference: s.Reference, Titles: titles, DailyChant: s.DailyChant, Purposes: purposes,
		DurationMinutes: s.DurationMinutes, AudioURL: s.AudioURL, Summary: s.Summary, Deities: deities,
	}
}

func newTranslationSummary(t domain.SutraTranslation) sutraTranslationSummaryResponse {
	return sutraTranslationSummaryResponse{
		Key: t.Key, Lang: t.Lang, Variant: string(t.Variant), Translator: t.Translator,
		License: t.License, LineCount: t.LineCount,
	}
}

func newSutraSummary(h imageHost, r *http.Request, s domain.Sutra) sutraSummaryResponse {
	trs := make([]sutraTranslationSummaryResponse, len(s.Translations))
	for i, t := range s.Translations {
		trs[i] = newTranslationSummary(t)
	}
	return sutraSummaryResponse{sutraHeadResponse: newSutraHead(h, r, s), Translations: trs}
}

func newSutraSummaries(h imageHost, r *http.Request, ss []domain.Sutra) []sutraSummaryResponse {
	out := make([]sutraSummaryResponse, 0, len(ss))
	for _, s := range ss {
		out = append(out, newSutraSummary(h, r, s))
	}
	return out
}

func newSutraDetail(h imageHost, r *http.Request, s domain.Sutra) sutraDetailResponse {
	trs := make([]sutraTranslationDetailResponse, len(s.Translations))
	for i, t := range s.Translations {
		secs := make([]sutraSectionResponse, len(t.Sections))
		for j, sec := range t.Sections {
			secs[j] = sutraSectionResponse{Name: sec.Name, StartN: sec.StartN, LineCount: sec.LineCount}
		}
		trs[i] = sutraTranslationDetailResponse{
			sutraTranslationSummaryResponse: newTranslationSummary(t),
			SourceName:                      t.SourceName, SourceURL: t.SourceURL,
			LicenseURL: t.LicenseURL, LicenseNote: t.LicenseNote, Note: t.Note, Sections: secs,
		}
	}
	return sutraDetailResponse{
		sutraHeadResponse: newSutraHead(h, r, s), ReviewStatus: s.ReviewStatus,
		OriginalTitle: s.OriginalTitle, Author: s.Author, HanTranslator: s.HanTranslator,
		HanTranslationYear: s.HanTranslationYear, ReferenceNote: s.ReferenceNote,
		Translations: trs,
	}
}

// queryParam returns a pointer to the parameter's value when it is present — even
// empty — and nil when it is absent, so the use case can reject "?tradition=" while
// treating no tradition at all as no filter.
func queryParam(q url.Values, key string) *string {
	vals, ok := q[key]
	if !ok {
		return nil
	}
	v := vals[0]
	return &v
}

// pageParam reads ?page: absent or empty is page 1; anything that is not an integer
// is domain.ErrInvalidPage (range is the use case's call).
func pageParam(q url.Values) (int, error) {
	raw := q.Get("page")
	if raw == "" {
		return 1, nil
	}
	p, err := strconv.Atoi(raw)
	if err != nil {
		return 0, domain.ErrInvalidPage
	}
	return p, nil
}

// List serves GET /v1/sutras?page=N&tradition=&kind=&collection=&purpose=&daily=&max_minutes=&lang=&q=.
func (h *SutraHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, err := pageParam(q)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	filter := domain.SutraFilterInput{
		Tradition: queryParam(q, "tradition"), Kind: queryParam(q, "kind"), Collection: queryParam(q, "collection"),
		Purpose: queryParam(q, "purpose"), Daily: queryParam(q, "daily"), MaxMinutes: queryParam(q, "max_minutes"),
		Lang: queryParam(q, "lang"), Q: queryParam(q, "q"),
	}

	result, err := h.usecase.List(r.Context(), filter, page)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, sutraListResponse{
		Items: newSutraSummaries(h.imageHost, r, result.Items), Page: result.Page, PageSize: result.PageSize,
		Total: result.Total, TotalPages: result.TotalPages,
	})
}

// Get serves GET /v1/sutras/{id}.
func (h *SutraHandler) Get(w http.ResponseWriter, r *http.Request) {
	s, err := h.usecase.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newSutraDetail(h.imageHost, r, *s))
}

// Lines serves GET /v1/sutras/{id}/lines?translation=<key>&page=N&page_size=M. The
// translation key is required; page defaults to 1 and page_size to the configured
// size. A-001: "page by lines" is read as page + page_size, not a cursor.
func (h *SutraHandler) Lines(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	key := q.Get("translation")
	if key == "" {
		writeDomainError(w, domain.ErrInvalidTranslation)
		return
	}
	page, err := pageParam(q)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	var pageSize *int
	if raw := q.Get("page_size"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeDomainError(w, domain.ErrInvalidPageSize)
			return
		}
		pageSize = &n
	}

	result, err := h.usecase.Lines(r.Context(), r.PathValue("id"), key, page, pageSize)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	items := make([]sutraLineResponse, 0, len(result.Items))
	for _, l := range result.Items {
		line := sutraLineResponse{N: l.N, Text: l.Text}
		if l.Section != "" {
			sec := l.Section
			line.Section = &sec
		}
		items = append(items, line)
	}
	utils.WriteSuccessResponse(w, sutraLinesResponse{
		Translation: result.Translation, Items: items, Page: result.Page, PageSize: result.PageSize,
		Total: result.Total, TotalPages: result.TotalPages,
	})
}

// ListByDeity serves GET /v1/deities/{id}/sutras?page=N&role=.
func (h *SutraHandler) ListByDeity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, err := pageParam(q)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	result, err := h.usecase.ListByDeity(r.Context(), r.PathValue("id"), queryParam(q, "role"), page)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	items := make([]sutraByDeityItemResponse, 0, len(result.Items))
	for _, ds := range result.Items {
		items = append(items, sutraByDeityItemResponse{sutraSummaryResponse: newSutraSummary(h.imageHost, r, ds.Sutra), Role: string(ds.Role)})
	}
	utils.WriteSuccessResponse(w, sutraByDeityListResponse{
		Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total, TotalPages: result.TotalPages,
	})
}
