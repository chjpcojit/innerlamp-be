package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// calendarEventUsecase is the subset of *usecase.CalendarEventUsecase this
// handler depends on, declared here (the consumer) so the handler is tested
// with a mock.
type calendarEventUsecase interface {
	EventsForYear(ctx context.Context, year string) (domain.CalendarEventsResult, error)
}

// CalendarEventHandler serves GET /v1/calendar/events. The route is public (v3.0):
// the wiring (cmd/main.go) puts the web API key check and the per-IP limit in
// front of it, and the handler does not look at the caller.
type CalendarEventHandler struct {
	usecase calendarEventUsecase
	// explicitYearCacheControl is sent when the request names a year.
	explicitYearCacheControl string
}

// NewCalendarEventHandler wires the handler. cacheMaxAge is how long a client
// (and a shared cache) may reuse the events of an explicit year (config: calendar.events_cache_max_age).
func NewCalendarEventHandler(usecase calendarEventUsecase, cacheMaxAge time.Duration) *CalendarEventHandler {
	return &CalendarEventHandler{
		usecase:                  usecase,
		explicitYearCacheControl: "public, max-age=" + strconv.Itoa(int(cacheMaxAge/time.Second)),
	}
}

type calendarGroupResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type calendarEventResponse struct {
	Code             string   `json:"code"`
	Name             string   `json:"name"`
	ShortDescription string   `json:"short_description"`
	LongDescription  string   `json:"long_description"`
	Guidance         string   `json:"guidance"`
	ImageURL         *string  `json:"image_url"`
	Groups           []string `json:"groups"`
	IsPublicHoliday  bool     `json:"is_public_holiday"`
	IsMajor          bool     `json:"is_major"`
	SortOrder        int      `json:"sort_order"`
	// Recurrence is the kind of the repeat rule (solar, lunar, lunar_monthly, lunar_last_day,
	// solar_term); clients use it to tell Rằm / Mùng 1 (lunar_monthly) apart (BR-CAL-15).
	Recurrence string `json:"recurrence"`
}

type calendarLunarDateResponse struct {
	Day         int  `json:"day"`
	Month       int  `json:"month"`
	Year        int  `json:"year"`
	IsLeapMonth bool `json:"is_leap_month"`
}

type calendarOccurrenceResponse struct {
	EventCode string                    `json:"event_code"`
	SolarDate string                    `json:"solar_date"`
	LunarDate calendarLunarDateResponse `json:"lunar_date"`
}

type calendarEventsResponse struct {
	Year        int                          `json:"year"`
	Groups      []calendarGroupResponse      `json:"groups"`
	Events      []calendarEventResponse      `json:"events"`
	Occurrences []calendarOccurrenceResponse `json:"occurrences"`
}

// Events serves GET /v1/calendar/events?year=YYYY. year is optional (empty =
// the current year). An explicit year is cacheable by the client and by a shared cache; an
// omitted one changes when the year turns, so it is not.
func (h *CalendarEventHandler) Events(w http.ResponseWriter, r *http.Request) {
	requested := r.URL.Query().Get("year")
	res, err := h.usecase.EventsForYear(r.Context(), requested)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	if requested != "" {
		w.Header().Set("Cache-Control", h.explicitYearCacheControl)
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	out := calendarEventsResponse{
		Year:        res.Year,
		Groups:      make([]calendarGroupResponse, 0, len(res.Groups)),
		Events:      make([]calendarEventResponse, 0, len(res.Events)),
		Occurrences: make([]calendarOccurrenceResponse, 0, len(res.Occurrences)),
	}
	for _, g := range res.Groups {
		out.Groups = append(out.Groups, calendarGroupResponse{Code: g.Code, Name: g.Name})
	}
	for _, e := range res.Events {
		groups := e.Groups
		if groups == nil {
			groups = []string{}
		}
		out.Events = append(out.Events, calendarEventResponse{
			Code: e.Code, Name: e.Name, ShortDescription: e.ShortDescription,
			LongDescription: e.LongDescription, Guidance: e.Guidance, ImageURL: e.ImageURL,
			Groups: groups, IsPublicHoliday: e.IsPublicHoliday,
			IsMajor: e.IsMajor, SortOrder: e.SortOrder, Recurrence: string(e.Rule.Kind),
		})
	}
	for _, o := range res.Occurrences {
		out.Occurrences = append(out.Occurrences, calendarOccurrenceResponse{
			EventCode: o.EventCode,
			SolarDate: o.SolarDate.Format("2006-01-02"),
			LunarDate: calendarLunarDateResponse{
				Day: o.LunarDate.Day, Month: o.LunarDate.Month,
				Year: o.LunarDate.Year, IsLeapMonth: o.LunarDate.IsLeapMonth,
			},
		})
	}
	utils.WriteSuccessResponse(w, out)
}
