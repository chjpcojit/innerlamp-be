package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// dharmaTalkUsecase is the subset of *usecase.DharmaTalkUsecase this handler
// depends on, declared here (the consumer) so the handler is tested with a mock.
type dharmaTalkUsecase interface {
	List(ctx context.Context, page int) (*domain.DharmaTalkPage, error)
	Get(ctx context.Context, id string) (*domain.DharmaTalk, error)
}

// DharmaTalkHandler serves the read-only pháp thoại catalogue. Both routes are
// public: browsing needs no account.
type DharmaTalkHandler struct {
	usecase dharmaTalkUsecase
}

// NewDharmaTalkHandler wires the handler.
func NewDharmaTalkHandler(usecase dharmaTalkUsecase) *DharmaTalkHandler {
	return &DharmaTalkHandler{usecase: usecase}
}

// dharmaTalkSummaryResponse is the card shape of the list. published_date
// ("YYYY-MM-DD"), duration_minutes and season are JSON null when unknown.
type dharmaTalkSummaryResponse struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Speaker         string   `json:"speaker"`
	SpeakerInitials string   `json:"speaker_initials"`
	Category        string   `json:"category"`
	Season          *string  `json:"season"`
	PublishedDate   *string  `json:"published_date"`
	DurationMinutes *int     `json:"duration_minutes"`
	Summary         string   `json:"summary"`
	Tags            []string `json:"tags"`
	MediaType       string   `json:"media_type"`
}

type dharmaTalkChapterResponse struct {
	StartSeconds int    `json:"start_seconds"`
	Title        string `json:"title"`
}

type followUpMeditationResponse struct {
	Title        string  `json:"title"`
	Minutes      int     `json:"minutes"`
	MeditationID *string `json:"meditation_id"`
}

// dharmaTalkDetailResponse is the summary plus the media, chapters and attribution.
// youtube_id / audio_url are null unless media_type says that medium is used;
// chapters is [] (never null) so a client can hide the section by emptiness.
type dharmaTalkDetailResponse struct {
	dharmaTalkSummaryResponse
	KeyPoints          []string                    `json:"key_points"`
	Quote              string                      `json:"quote"`
	QuoteSource        string                      `json:"quote_source"`
	Language           string                      `json:"language"`
	YouTubeID          *string                     `json:"youtube_id"`
	AudioURL           *string                     `json:"audio_url"`
	Chapters           []dharmaTalkChapterResponse `json:"chapters"`
	FollowUpMeditation *followUpMeditationResponse `json:"follow_up_meditation"`
	SourceName         string                      `json:"source_name"`
	SourceURL          string                      `json:"source_url"`
	License            string                      `json:"license"`
}

type dharmaTalkListResponse struct {
	Items      []dharmaTalkSummaryResponse `json:"items"`
	Page       int                         `json:"page"`
	PageSize   int                         `json:"page_size"`
	Total      int                         `json:"total"`
	TotalPages int                         `json:"total_pages"`
}

// nullIfEmpty maps "" (no value) to JSON null.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func newDharmaTalkSummary(t domain.DharmaTalk) dharmaTalkSummaryResponse {
	r := dharmaTalkSummaryResponse{
		ID: t.ID, Title: t.Title, Speaker: t.Speaker, SpeakerInitials: t.SpeakerInitials, Category: t.Category,
		Season: nullIfEmpty(t.Season), Summary: t.Summary, Tags: nonNil(t.Tags), MediaType: string(t.MediaType),
	}
	if t.PublishedDate != nil {
		d := t.PublishedDate.Format(time.DateOnly)
		r.PublishedDate = &d
	}
	if t.DurationMinutes > 0 {
		m := t.DurationMinutes
		r.DurationMinutes = &m
	}
	return r
}

func newDharmaTalkSummaries(ts []domain.DharmaTalk) []dharmaTalkSummaryResponse {
	out := make([]dharmaTalkSummaryResponse, 0, len(ts))
	for _, t := range ts {
		out = append(out, newDharmaTalkSummary(t))
	}
	return out
}

func newDharmaTalkDetail(t domain.DharmaTalk) dharmaTalkDetailResponse {
	chapters := make([]dharmaTalkChapterResponse, 0, len(t.Chapters))
	for _, c := range t.Chapters {
		chapters = append(chapters, dharmaTalkChapterResponse{StartSeconds: c.StartSeconds, Title: c.Title})
	}
	var followUp *followUpMeditationResponse
	if f := t.FollowUpMeditation; f != nil {
		followUp = &followUpMeditationResponse{Title: f.Title, Minutes: f.Minutes, MeditationID: nullIfEmpty(f.MeditationID)}
	}
	return dharmaTalkDetailResponse{
		dharmaTalkSummaryResponse: newDharmaTalkSummary(t),
		KeyPoints:                 nonNil(t.KeyPoints),
		Quote:                     t.Quote, QuoteSource: t.QuoteSource, Language: t.Language,
		YouTubeID: nullIfEmpty(t.YouTubeID), AudioURL: nullIfEmpty(t.AudioURL),
		Chapters: chapters, FollowUpMeditation: followUp,
		SourceName: t.SourceName, SourceURL: t.SourceURL, License: t.License,
	}
}

// List serves GET /v1/dharma-talks?page=N. page defaults to 1; anything that is
// not a positive integer is 400 INVALID_PAGE.
func (h *DharmaTalkHandler) List(w http.ResponseWriter, r *http.Request) {
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
	utils.WriteSuccessResponse(w, dharmaTalkListResponse{
		Items: newDharmaTalkSummaries(result.Items), Page: result.Page, PageSize: result.PageSize,
		Total: result.Total, TotalPages: result.TotalPages,
	})
}

// Get serves GET /v1/dharma-talks/{id}.
func (h *DharmaTalkHandler) Get(w http.ResponseWriter, r *http.Request) {
	t, err := h.usecase.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newDharmaTalkDetail(*t))
}
