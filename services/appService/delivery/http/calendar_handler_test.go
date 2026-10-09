package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newCalendarMux(uc *mockCalendarUsecase, baseURL string) *http.ServeMux {
	h := NewCalendarHandler(uc, baseURL)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/calendar/image", h.Image)
	return mux
}

func sampleCalendarImage() domain.CalendarImage {
	return domain.CalendarImage{
		ID: "buddha_008", File: "buddha_008.jpg", Author: "Jakub Hałun", License: "CC BY-SA 4.0",
		LicenseURL: "https://creativecommons.org/licenses/by-sa/4.0", SourceURL: "https://commons.wikimedia.org/wiki/File:X.jpg",
	}
}

func TestCalendarHandler_Image_ExplicitDate(t *testing.T) {
	uc := new(mockCalendarUsecase)
	uc.On("ImageForDate", "2026-10-07").Return("2026-10-07", sampleCalendarImage(), nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/calendar/image?date=2026-10-07", nil)
	req.Host = "192.168.1.186:8080"
	rec := httptest.NewRecorder()
	newCalendarMux(uc, "").ServeHTTP(rec, req) // no Authorization header: public

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "public, max-age=86400", rec.Header().Get("Cache-Control"))
	body := decode(t, rec)
	assert.Equal(t, true, body["success"])
	data := body["data"].(map[string]any)
	assert.Equal(t, "2026-10-07", data["date"])
	assert.Equal(t, "buddha_008", data["id"])
	assert.Equal(t, "http://192.168.1.186:8080/static/images/home-calendar/buddha_008.jpg", data["image_url"])
	assert.Equal(t, "Jakub Hałun", data["author"])
	assert.Equal(t, "CC BY-SA 4.0", data["license"])
	assert.Equal(t, "https://creativecommons.org/licenses/by-sa/4.0", data["license_url"])
	assert.Equal(t, "https://commons.wikimedia.org/wiki/File:X.jpg", data["source_url"])
}

func TestCalendarHandler_Image_NoDateIsNotCached(t *testing.T) {
	uc := new(mockCalendarUsecase)
	uc.On("ImageForDate", "").Return("2026-10-08", sampleCalendarImage(), nil)

	rec := doGet(newCalendarMux(uc, ""), "/v1/calendar/image")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "2026-10-08", decode(t, rec)["data"].(map[string]any)["date"], "echoes the resolved date")
}

func TestCalendarHandler_Image_ConfiguredBaseURL(t *testing.T) {
	uc := new(mockCalendarUsecase)
	uc.On("ImageForDate", "2026-10-07").Return("2026-10-07", sampleCalendarImage(), nil)

	rec := doGet(newCalendarMux(uc, "https://cdn.example.com/cal/"), "/v1/calendar/image?date=2026-10-07")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://cdn.example.com/cal/buddha_008.jpg", decode(t, rec)["data"].(map[string]any)["image_url"])
}

func TestCalendarHandler_Image_ForwardedHTTPS(t *testing.T) {
	uc := new(mockCalendarUsecase)
	uc.On("ImageForDate", "2026-10-07").Return("2026-10-07", sampleCalendarImage(), nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/calendar/image?date=2026-10-07", nil)
	req.Host = "api.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	newCalendarMux(uc, "").ServeHTTP(rec, req)

	assert.Equal(t, "https://api.example.com/static/images/home-calendar/buddha_008.jpg",
		decode(t, rec)["data"].(map[string]any)["image_url"])
}

func TestCalendarHandler_Image_InvalidDate(t *testing.T) {
	uc := new(mockCalendarUsecase)
	uc.On("ImageForDate", "2026-02-30").Return("", domain.CalendarImage{}, domain.ErrInvalidDate)

	rec := doGet(newCalendarMux(uc, ""), "/v1/calendar/image?date=2026-02-30")

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	e := decodeErrorBody(t, rec)
	assert.Equal(t, "INVALID_DATE", e.Code)
	assert.Equal(t, "date", e.Details["field"])
}

func TestCalendarHandler_Image_UnexpectedError(t *testing.T) {
	uc := new(mockCalendarUsecase)
	uc.On("ImageForDate", "").Return("", domain.CalendarImage{}, errors.New("boom"))

	rec := doGet(newCalendarMux(uc, ""), "/v1/calendar/image")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "boom")
}
