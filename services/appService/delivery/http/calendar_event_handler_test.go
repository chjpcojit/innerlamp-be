package http

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newCalendarEventMux(uc *mockCalendarEventUsecase) *http.ServeMux {
	h := NewCalendarEventHandler(uc, 24*time.Hour)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/calendar/events", h.Events)
	return mux
}

func sampleEventsResult() domain.CalendarEventsResult {
	image := "https://cdn.example/x.jpg"
	return domain.CalendarEventsResult{
		Year:   2026,
		Groups: []domain.CalendarGroup{{Code: "vietnam", Name: "Việt Nam"}, {Code: "folk", Name: "Dân gian"}},
		Events: []domain.CalendarEvent{
			{Code: "tet", Name: "Tết Nguyên Đán", ShortDescription: "s", LongDescription: "l", Guidance: "g",
				ImageURL: &image, Groups: []string{"vietnam", "folk"}, IsPublicHoliday: true},
			{Code: "ram", Name: "Rằm", ShortDescription: "s", LongDescription: "l", Guidance: "g"},
		},
		Occurrences: []domain.CalendarOccurrence{
			{EventCode: "tet", SolarDate: time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC),
				LunarDate: domain.CalendarLunarDate{Day: 1, Month: 1, Year: 2026}},
			{EventCode: "ram", SolarDate: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
				LunarDate: domain.CalendarLunarDate{Day: 15, Month: 1, Year: 2026, IsLeapMonth: false}},
		},
	}
}

func TestCalendarEventHandler_ExplicitYear(t *testing.T) {
	uc := new(mockCalendarEventUsecase)
	uc.On("EventsForYear", mock.Anything, "2026").Return(sampleEventsResult(), nil)

	rec := doGet(newCalendarEventMux(uc), "/v1/calendar/events?year=2026")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "private, max-age=86400", rec.Header().Get("Cache-Control"))
	data := decode(t, rec)["data"].(map[string]any)
	assert.EqualValues(t, 2026, data["year"])

	groups := data["groups"].([]any)
	require.Len(t, groups, 2)
	assert.Equal(t, map[string]any{"code": "vietnam", "name": "Việt Nam"}, groups[0])

	events := data["events"].([]any)
	require.Len(t, events, 2)
	tet := events[0].(map[string]any)
	assert.Equal(t, "tet", tet["code"])
	assert.Equal(t, "https://cdn.example/x.jpg", tet["image_url"])
	assert.Equal(t, []any{"vietnam", "folk"}, tet["groups"])
	assert.Equal(t, true, tet["is_public_holiday"])
	ram := events[1].(map[string]any)
	assert.Nil(t, ram["image_url"], "image_url is null, not absent")
	assert.Contains(t, ram, "image_url")
	assert.Equal(t, []any{}, ram["groups"], "no groups is [], not null")

	occ := data["occurrences"].([]any)
	require.Len(t, occ, 2)
	first := occ[0].(map[string]any)
	assert.Equal(t, "tet", first["event_code"])
	assert.Equal(t, "2026-02-17", first["solar_date"])
	assert.Equal(t, map[string]any{"day": float64(1), "month": float64(1), "year": float64(2026), "is_leap_month": false}, first["lunar_date"])
}

func TestCalendarEventHandler_NoYearIsNotCached(t *testing.T) {
	uc := new(mockCalendarEventUsecase)
	uc.On("EventsForYear", mock.Anything, "").Return(sampleEventsResult(), nil)

	rec := doGet(newCalendarEventMux(uc), "/v1/calendar/events")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
}

func TestCalendarEventHandler_InvalidYear(t *testing.T) {
	uc := new(mockCalendarEventUsecase)
	uc.On("EventsForYear", mock.Anything, "abc").Return(domain.CalendarEventsResult{}, domain.ErrInvalidYear)

	rec := doGet(newCalendarEventMux(uc), "/v1/calendar/events?year=abc")

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	e := decodeErrorBody(t, rec)
	assert.Equal(t, "INVALID_YEAR", e.Code)
	assert.Equal(t, "year", e.Details["field"])
	assert.Empty(t, rec.Header().Get("Cache-Control"))
}

func TestCalendarEventHandler_InternalError(t *testing.T) {
	uc := new(mockCalendarEventUsecase)
	uc.On("EventsForYear", mock.Anything, "2026").Return(domain.CalendarEventsResult{}, errors.New("db down"))

	rec := doGet(newCalendarEventMux(uc), "/v1/calendar/events?year=2026")

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "INTERNAL_ERROR", decodeErrorBody(t, rec).Code)
	assert.NotContains(t, rec.Body.String(), "db down")
}
