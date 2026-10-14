package http

import (
	"context"
	"net/http"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// templeEventUsecase is the subset of *usecase.TempleEventUsecase this handler
// depends on, declared here (the consumer) so the handler is tested with a mock.
type templeEventUsecase interface {
	EventsForTemple(ctx context.Context, templeID, from, to string) (domain.TempleEventsResult, error)
}

// TempleEventHandler serves GET /v1/temples/{id}/events. Unlike the public
// /v1/temples routes it requires a valid access token; cmd/main.go puts the JWT
// middleware in front of it, so the handler does not look at the caller.
type TempleEventHandler struct {
	usecase templeEventUsecase
}

// NewTempleEventHandler wires the handler.
func NewTempleEventHandler(usecase templeEventUsecase) *TempleEventHandler {
	return &TempleEventHandler{usecase: usecase}
}

type templeEventRecurrenceResponse struct {
	Kind      string  `json:"kind"`
	LunarDays []int   `json:"lunar_days"`
	Weekday   *int    `json:"weekday"`
	Until     *string `json:"until"`
}

type templeEventResponse struct {
	ID           string                        `json:"id"`
	Title        string                        `json:"title"`
	Description  string                        `json:"description"`
	Kind         string                        `json:"kind"`
	IsAllDay     bool                          `json:"is_all_day"`
	Status       string                        `json:"status"`
	AudienceNote string                        `json:"audience_note"`
	ContactInfo  string                        `json:"contact_info"`
	ImageURL     *string                       `json:"image_url"`
	Recurrence   templeEventRecurrenceResponse `json:"recurrence"`
}

type templeOccurrenceResponse struct {
	EventID   string                    `json:"event_id"`
	StartsAt  string                    `json:"starts_at"`
	EndsAt    string                    `json:"ends_at"`
	SolarDate string                    `json:"solar_date"`
	LunarDate calendarLunarDateResponse `json:"lunar_date"`
}

type templeEventsResponse struct {
	TempleID    string                     `json:"temple_id"`
	From        string                     `json:"from"`
	To          string                     `json:"to"`
	Events      []templeEventResponse      `json:"events"`
	Occurrences []templeOccurrenceResponse `json:"occurrences"`
}

// Events serves GET /v1/temples/{id}/events?from=YYYY-MM-DD&to=YYYY-MM-DD. Both
// query values are optional. The answer depends on "now" (events that already
// ended are left out), so it is never cacheable.
func (h *TempleEventHandler) Events(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := h.usecase.EventsForTemple(r.Context(), r.PathValue("id"), q.Get("from"), q.Get("to"))
	if err != nil {
		writeDomainError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	const dateLayout = "2006-01-02"
	out := templeEventsResponse{
		TempleID:    res.TempleID,
		From:        res.From.Format(dateLayout),
		To:          res.To.Format(dateLayout),
		Events:      make([]templeEventResponse, 0, len(res.Events)),
		Occurrences: make([]templeOccurrenceResponse, 0, len(res.Occurrences)),
	}
	for _, e := range res.Events {
		days := e.LunarDays
		if days == nil {
			days = []int{}
		}
		var until *string
		if e.Until != nil {
			s := e.Until.Format(dateLayout)
			until = &s
		}
		out.Events = append(out.Events, templeEventResponse{
			ID: e.ID, Title: e.Title, Description: e.Description, Kind: string(e.Kind),
			IsAllDay: e.IsAllDay, Status: string(e.Status), AudienceNote: e.AudienceNote,
			ContactInfo: e.ContactInfo, ImageURL: e.ImageURL,
			Recurrence: templeEventRecurrenceResponse{
				Kind: string(e.Recurrence), LunarDays: days, Weekday: e.Weekday, Until: until,
			},
		})
	}
	for _, o := range res.Occurrences {
		out.Occurrences = append(out.Occurrences, templeOccurrenceResponse{
			EventID:   o.EventID,
			StartsAt:  o.StartsAt.Format(time.RFC3339),
			EndsAt:    o.EndsAt.Format(time.RFC3339),
			SolarDate: o.SolarDate.Format(dateLayout),
			LunarDate: calendarLunarDateResponse{
				Day: o.LunarDate.Day, Month: o.LunarDate.Month, Year: o.LunarDate.Year, IsLeapMonth: o.LunarDate.IsLeapMonth,
			},
		})
	}
	utils.WriteSuccessResponse(w, out)
}
