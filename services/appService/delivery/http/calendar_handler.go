package http

import (
	"net/http"
	"strings"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// calendarUsecase is the subset of *usecase.CalendarUsecase this handler depends
// on, declared here (the consumer) so the handler is tested with a mock.
type calendarUsecase interface {
	ImageForDate(date string) (string, domain.CalendarImage, error)
}

// CalendarHandler serves the lunar-calendar photo. The route is public.
type CalendarHandler struct {
	usecase calendarUsecase
	// imageBaseURL is the absolute URL file names are appended to; empty = derive it
	// from the request (dev only, see config.CalendarConfig.ImageBaseURL).
	imageBaseURL string
}

// NewCalendarHandler wires the handler.
func NewCalendarHandler(usecase calendarUsecase, imageBaseURL string) *CalendarHandler {
	return &CalendarHandler{usecase: usecase, imageBaseURL: strings.TrimRight(imageBaseURL, "/")}
}

type calendarImageResponse struct {
	Date       string `json:"date"`
	ID         string `json:"id"`
	ImageURL   string `json:"image_url"`
	Author     string `json:"author"`
	License    string `json:"license"`
	LicenseURL string `json:"license_url"`
	SourceURL  string `json:"source_url"`
}

// Image serves GET /v1/calendar/image?date=YYYY-MM-DD. date is optional (empty = today).
func (h *CalendarHandler) Image(w http.ResponseWriter, r *http.Request) {
	requested := r.URL.Query().Get("date")
	date, img, err := h.usecase.ImageForDate(requested)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// An explicit date maps to one image forever; "today" changes at midnight.
	if requested != "" {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	utils.WriteSuccessResponse(w, calendarImageResponse{
		Date: date, ID: img.ID, ImageURL: h.imageBase(r) + "/" + img.File,
		Author: img.Author, License: img.License, LicenseURL: img.LicenseURL, SourceURL: img.SourceURL,
	})
}

// imageBase returns the URL prefix of the image files: the configured one, else this
// request's own origin plus the dev static route.
func (h *CalendarHandler) imageBase(r *http.Request) string {
	if h.imageBaseURL != "" {
		return h.imageBaseURL
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/static/images/home-calendar"
}
