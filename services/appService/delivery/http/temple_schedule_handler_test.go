package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newTempleScheduleMux(uc *mockTempleScheduleUsecase) *http.ServeMux {
	h := NewTempleScheduleHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/temples/{id}/schedule", h.Schedule)
	return mux
}

func scheduleTime(minutes int) *domain.TimeOfDay { v := domain.TimeOfDay(minutes); return &v }

func TestTempleScheduleHandler_Success(t *testing.T) {
	uc := new(mockTempleScheduleUsecase)
	uc.On("ScheduleForTemple", mock.Anything, "temple_north_001").Return([]domain.TempleScheduleItem{
		{ID: "full", Name: "Lễ Dược Sư", StartTime: 480, EndTime: scheduleTime(660), Weekdays: []int{0},
			Description: "d", Instructor: "i", Location: "l", Preparation: "p",
			Steps: []domain.TempleScheduleStep{
				{Position: 1, Time: scheduleTime(480), Text: "Niệm Phật"},
				{Position: 2, Text: "Hồi hướng"},
			}},
		{ID: "min", Name: "Công phu sáng", StartTime: 240, Weekdays: []int{}, Steps: []domain.TempleScheduleStep{}},
	}, nil)

	rec := doGet(newTempleScheduleMux(uc), "/v1/temples/temple_north_001/schedule")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, "temple_north_001", data["temple_id"])
	items := data["items"].([]any)
	require.Len(t, items, 2)

	full := items[0].(map[string]any)
	assert.Equal(t, "08:00", full["start_time"])
	assert.Equal(t, "11:00", full["end_time"])
	assert.Equal(t, []any{float64(0)}, full["weekdays"])
	steps := full["steps"].([]any)
	require.Len(t, steps, 2)
	assert.Equal(t, map[string]any{"position": float64(1), "time": "08:00", "text": "Niệm Phật"}, steps[0])
	assert.Equal(t, map[string]any{"position": float64(2), "time": nil, "text": "Hồi hướng"}, steps[1])

	min := items[1].(map[string]any)
	assert.Nil(t, min["end_time"])
	assert.Equal(t, []any{}, min["weekdays"])
	assert.Equal(t, []any{}, min["steps"])
	for _, k := range []string{"description", "instructor", "location", "preparation"} {
		assert.Equal(t, "", min[k], k)
	}
}

func TestTempleScheduleHandler_EmptyItems(t *testing.T) {
	uc := new(mockTempleScheduleUsecase)
	uc.On("ScheduleForTemple", mock.Anything, "t").Return([]domain.TempleScheduleItem{}, nil)

	rec := doGet(newTempleScheduleMux(uc), "/v1/temples/t/schedule")

	require.Equal(t, http.StatusOK, rec.Code)
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, []any{}, data["items"])
}

func TestTempleScheduleHandler_NotFound(t *testing.T) {
	uc := new(mockTempleScheduleUsecase)
	uc.On("ScheduleForTemple", mock.Anything, "x").Return(nil, domain.ErrTempleNotFound)

	rec := doGet(newTempleScheduleMux(uc), "/v1/temples/x/schedule")

	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "TEMPLE_NOT_FOUND", decodeErrorBody(t, rec).Code)
}

func TestTempleScheduleHandler_InternalError(t *testing.T) {
	uc := new(mockTempleScheduleUsecase)
	uc.On("ScheduleForTemple", mock.Anything, "t").Return(nil, errors.New("db down"))

	rec := doGet(newTempleScheduleMux(uc), "/v1/temples/t/schedule")

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "db down")
}
