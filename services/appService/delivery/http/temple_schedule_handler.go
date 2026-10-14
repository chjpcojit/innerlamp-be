package http

import (
	"context"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// templeScheduleUsecase is the subset of *usecase.TempleScheduleUsecase this
// handler depends on, declared here (the consumer) so the handler is tested
// with a mock.
type templeScheduleUsecase interface {
	ScheduleForTemple(ctx context.Context, templeID string) ([]domain.TempleScheduleItem, error)
}

// TempleScheduleHandler serves GET /v1/temples/{id}/schedule. Unlike the public
// /v1/temples routes it requires a valid access token; cmd/main.go puts the JWT
// middleware in front of it, so the handler does not look at the caller.
type TempleScheduleHandler struct {
	usecase templeScheduleUsecase
}

// NewTempleScheduleHandler wires the handler.
func NewTempleScheduleHandler(usecase templeScheduleUsecase) *TempleScheduleHandler {
	return &TempleScheduleHandler{usecase: usecase}
}

type templeScheduleStepResponse struct {
	Position int     `json:"position"`
	Time     *string `json:"time"`
	Text     string  `json:"text"`
}

type templeScheduleItemResponse struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	StartTime   string                       `json:"start_time"`
	EndTime     *string                      `json:"end_time"`
	Weekdays    []int                        `json:"weekdays"`
	Description string                       `json:"description"`
	Instructor  string                       `json:"instructor"`
	Location    string                       `json:"location"`
	Preparation string                       `json:"preparation"`
	Steps       []templeScheduleStepResponse `json:"steps"`
}

type templeScheduleResponse struct {
	TempleID string                       `json:"temple_id"`
	Items    []templeScheduleItemResponse `json:"items"`
}

// Schedule serves GET /v1/temples/{id}/schedule.
func (h *TempleScheduleHandler) Schedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	items, err := h.usecase.ScheduleForTemple(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	out := templeScheduleResponse{TempleID: id, Items: make([]templeScheduleItemResponse, 0, len(items))}
	for _, it := range items {
		weekdays := it.Weekdays
		if weekdays == nil {
			weekdays = []int{}
		}
		steps := make([]templeScheduleStepResponse, 0, len(it.Steps))
		for _, s := range it.Steps {
			steps = append(steps, templeScheduleStepResponse{Position: s.Position, Time: timeOfDayPtr(s.Time), Text: s.Text})
		}
		out.Items = append(out.Items, templeScheduleItemResponse{
			ID: it.ID, Name: it.Name, StartTime: it.StartTime.String(), EndTime: timeOfDayPtr(it.EndTime),
			Weekdays: weekdays, Description: it.Description, Instructor: it.Instructor,
			Location: it.Location, Preparation: it.Preparation, Steps: steps,
		})
	}
	utils.WriteSuccessResponse(w, out)
}

func timeOfDayPtr(t *domain.TimeOfDay) *string {
	if t == nil {
		return nil
	}
	s := t.String()
	return &s
}
