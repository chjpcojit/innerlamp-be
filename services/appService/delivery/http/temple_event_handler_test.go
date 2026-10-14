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

func newTempleEventMux(uc *mockTempleEventUsecase) *http.ServeMux {
	h := NewTempleEventHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/temples/{id}/events", h.Events)
	return mux
}

var evZone = time.FixedZone("ICT", 7*3600)

func sampleTempleEventsResult() domain.TempleEventsResult {
	image := "https://cdn.example/a.jpg"
	sunday := 0
	until := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)
	return domain.TempleEventsResult{
		TempleID: "temple_north_001",
		From:     time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2027, 1, 13, 0, 0, 0, 0, time.UTC),
		Events: []domain.TempleEvent{
			{ID: "lunar", Title: "Sám hối", Description: "d", Kind: domain.TempleEventKhac,
				Status: domain.TempleEventPostponed, Recurrence: domain.RecurrenceLunarMonthly, LunarDays: []int{14, 30}},
			{ID: "weekly", Title: "Dược Sư", Kind: domain.TempleEventLeCauAn, Status: domain.TempleEventScheduled,
				IsAllDay: false, AudienceNote: "mọi người", ContactInfo: "0123", ImageURL: &image,
				Recurrence: domain.RecurrenceWeekly, Weekday: &sunday, Until: &until},
		},
		Occurrences: []domain.TempleEventOccurrence{
			{EventID: "lunar", StartsAt: time.Date(2026, 11, 23, 19, 0, 0, 0, evZone),
				EndsAt: time.Date(2026, 11, 23, 21, 0, 0, 0, evZone), SolarDate: time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC),
				LunarDate: domain.CalendarLunarDate{Day: 14, Month: 10, Year: 2026}},
		},
	}
}

func TestTempleEventHandler_Success(t *testing.T) {
	uc := new(mockTempleEventUsecase)
	uc.On("EventsForTemple", mock.Anything, "temple_north_001", "2026-10-13", "2027-01-13").Return(sampleTempleEventsResult(), nil)

	rec := doGet(newTempleEventMux(uc), "/v1/temples/temple_north_001/events?from=2026-10-13&to=2027-01-13")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, "temple_north_001", data["temple_id"])
	assert.Equal(t, "2026-10-13", data["from"])
	assert.Equal(t, "2027-01-13", data["to"])

	events := data["events"].([]any)
	require.Len(t, events, 2)
	lunar := events[0].(map[string]any)
	assert.Equal(t, "POSTPONED", lunar["status"])
	assert.Nil(t, lunar["image_url"])
	assert.Equal(t, "", lunar["audience_note"])
	assert.Equal(t, map[string]any{"kind": "LUNAR_MONTHLY", "lunar_days": []any{float64(14), float64(30)}, "weekday": nil, "until": nil},
		lunar["recurrence"])

	weekly := events[1].(map[string]any)
	assert.Equal(t, "https://cdn.example/a.jpg", weekly["image_url"])
	assert.Equal(t, map[string]any{"kind": "WEEKLY", "lunar_days": []any{}, "weekday": float64(0), "until": "2027-06-30"},
		weekly["recurrence"])

	occ := data["occurrences"].([]any)
	require.Len(t, occ, 1)
	o := occ[0].(map[string]any)
	assert.Equal(t, "lunar", o["event_id"])
	assert.Equal(t, "2026-11-23T19:00:00+07:00", o["starts_at"])
	assert.Equal(t, "2026-11-23T21:00:00+07:00", o["ends_at"])
	assert.Equal(t, "2026-11-23", o["solar_date"])
	assert.Equal(t, map[string]any{"day": float64(14), "month": float64(10), "year": float64(2026), "is_leap_month": false}, o["lunar_date"])
}

func TestTempleEventHandler_EmptyAndDefaults(t *testing.T) {
	uc := new(mockTempleEventUsecase)
	uc.On("EventsForTemple", mock.Anything, "t", "", "").Return(domain.TempleEventsResult{
		TempleID: "t", From: time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC), To: time.Date(2027, 1, 13, 0, 0, 0, 0, time.UTC),
	}, nil)

	rec := doGet(newTempleEventMux(uc), "/v1/temples/t/events")

	require.Equal(t, http.StatusOK, rec.Code)
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, []any{}, data["events"])
	assert.Equal(t, []any{}, data["occurrences"])
}

func TestTempleEventHandler_Errors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
		field  string
	}{
		{"temple not found", domain.ErrTempleNotFound, http.StatusNotFound, "TEMPLE_NOT_FOUND", ""},
		{"bad from", &domain.InvalidDateFieldError{Field: "from"}, http.StatusBadRequest, "INVALID_DATE", "from"},
		{"bad to wrapped", errors.Join(errors.New("ctx"), &domain.InvalidDateFieldError{Field: "to"}), http.StatusBadRequest, "INVALID_DATE", "to"},
		{"reversed", domain.ErrInvalidDateRange, http.StatusBadRequest, "INVALID_DATE_RANGE", ""},
		{"too large", domain.ErrDateRangeTooLarge, http.StatusBadRequest, "DATE_RANGE_TOO_LARGE", ""},
		{"internal", errors.New("db down"), http.StatusInternalServerError, "INTERNAL_ERROR", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := new(mockTempleEventUsecase)
			uc.On("EventsForTemple", mock.Anything, "t", "", "").Return(domain.TempleEventsResult{}, tc.err)

			rec := doGet(newTempleEventMux(uc), "/v1/temples/t/events")

			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			e := decodeErrorBody(t, rec)
			assert.Equal(t, tc.code, e.Code)
			if tc.field != "" {
				assert.Equal(t, tc.field, e.Details["field"])
			}
			assert.NotContains(t, rec.Body.String(), "db down")
		})
	}
}
