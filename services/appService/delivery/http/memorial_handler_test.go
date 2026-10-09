package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	jwtclaims "chjpcoj.com.innerlamp/pkg/jwt/claims"
	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newMemorialMux registers the routes the way cmd/main.go does (minus the auth
// middleware, whose claims the test requests carry directly).
func newMemorialMux(uc *mockMemorialUsecase) *http.ServeMux {
	h := NewMemorialHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/memorials", h.Create)
	mux.HandleFunc("GET /v1/memorials", h.List)
	mux.HandleFunc("GET /v1/memorials/{id}", h.Get)
	mux.HandleFunc("PUT /v1/memorials/{id}", h.Update)
	mux.HandleFunc("DELETE /v1/memorials/{id}", h.Delete)
	return mux
}

// memorialReq builds a request that carries the claims the auth middleware would
// have put in the context. userID "" means no claims at all.
func memorialReq(method, target, body, userID string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if userID == "" {
		return req
	}
	claims := &jwtclaims.CustomClaims{UserID: userID, TokenType: jwtclaims.TokenTypeAccess, RegisteredClaims: jwt.RegisteredClaims{}}
	return req.WithContext(jwtcontext.WithClaims(req.Context(), claims))
}

func serve(mux *http.ServeMux, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	body := decode(t, rec)
	assert.Equal(t, false, body["success"])
	return body["error"].(map[string]any)["code"].(string)
}

func sampleMemorial() *domain.Memorial {
	d := time.Date(2015, 3, 4, 0, 0, 0, 0, time.UTC)
	note := "nhớ ơn"
	return &domain.Memorial{
		ID: 9, UserID: 7, FullName: "Nguyễn Văn A", Relationship: domain.RelationshipFather,
		DeathCalendar: domain.DeathCalendarLunar, DeathDate: &d, Note: &note,
		Reminder:        &domain.Reminder{DaysBefore: 3, Time: "07:30", Timezone: "Asia/Ho_Chi_Minh"},
		AddToOSReminder: true,
		CreatedAt:       time.Date(2026, 10, 7, 1, 2, 3, 0, time.UTC), UpdatedAt: time.Date(2026, 10, 7, 1, 2, 3, 0, time.UTC),
	}
}

const fullBody = `{"full_name":"Nguyễn Văn A","relationship":"FATHER","death_calendar":"lunar","death_date":"2015-03-04",
	"note":"nhớ ơn","reminder":{"days_before":3,"time":"07:30"},"add_to_os_reminder":true}`

func TestMemorialHandler_Create_Success(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Create", mock.Anything, int64(7), mock.Anything).Return(sampleMemorial(), nil)

	rec := serve(newMemorialMux(uc), memorialReq(http.MethodPost, "/v1/memorials", fullBody, "7"))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, float64(9), data["id"])
	assert.Equal(t, "2015-03-04", data["death_date"])
	assert.Equal(t, "lunar", data["death_calendar"])
	assert.Nil(t, data["death_day"], "year known: partial fields are null")
	assert.Contains(t, data, "death_day")
	assert.Nil(t, data["photo_url"], "photo_url is always null until the photo feature exists")
	assert.Contains(t, data, "photo_url")
	assert.Equal(t, map[string]any{"days_before": float64(3), "time": "07:30", "timezone": "Asia/Ho_Chi_Minh"}, data["reminder"])
	assert.Equal(t, true, data["add_to_os_reminder"])
	assert.NotContains(t, data, "user_id", "the owner id is not part of the wire shape")

	// The decoded request reached the use case with the owner from the claims.
	in := uc.Calls[0].Arguments.Get(2).(domain.MemorialInput)
	assert.Equal(t, "2015-03-04", in.DeathDate)
	assert.Equal(t, 3, *in.Reminder.DaysBefore)
	assert.True(t, in.AddToOSReminder)
}

func TestMemorialHandler_Create_PartialDateAndNullsRoundTrip(t *testing.T) {
	uc := new(mockMemorialUsecase)
	day, month, leap := 15, 7, true
	uc.On("Create", mock.Anything, int64(7), mock.Anything).Return(&domain.Memorial{
		ID: 1, FullName: "Bà", Relationship: domain.RelationshipGrandmotherMaternal, DeathCalendar: domain.DeathCalendarLunar,
		DeathDay: &day, DeathMonth: &month, DeathLeapMonth: &leap,
	}, nil)
	body := `{"full_name":"Bà","relationship":"GRANDMOTHER_MATERNAL","death_calendar":"lunar","death_day":15,"death_month":7,"death_leap_month":true}`

	rec := serve(newMemorialMux(uc), memorialReq(http.MethodPost, "/v1/memorials", body, "7"))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	data := decode(t, rec)["data"].(map[string]any)
	assert.Nil(t, data["death_date"])
	assert.Contains(t, data, "death_date")
	assert.Equal(t, float64(15), data["death_day"])
	assert.Equal(t, true, data["death_leap_month"])
	assert.Nil(t, data["note"])
	assert.Nil(t, data["reminder"])
	assert.Contains(t, data, "reminder")
	assert.Equal(t, false, data["add_to_os_reminder"])
}

func TestMemorialHandler_Create_BadBodies(t *testing.T) {
	tests := []struct{ name, body string }{
		{"empty body", ``},
		{"not json", `nope`},
		{"unknown field", `{"full_name":"A","bogus":1}`},
		{"server-owned field photo_url", `{"full_name":"A","photo_url":"x"}`},
		{"server-owned field id", `{"full_name":"A","id":5}`},
		{"server-owned field user_id", `{"full_name":"A","user_id":5}`},
		{"wrong type", `{"full_name":5}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := new(mockMemorialUsecase)
			rec := serve(newMemorialMux(uc), memorialReq(http.MethodPost, "/v1/memorials", tt.body, "7"))

			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec))
			uc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestMemorialHandler_DomainErrorsMapToStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
		field  string
	}{
		{"validation", &domain.MemorialValidationError{Field: "reminder.time", Message: "time must be HH:mm (24h)"}, 400, "VALIDATION_ERROR", "reminder.time"},
		{"limit", domain.ErrMemorialLimitReached, 409, "MEMORIAL_LIMIT_REACHED", ""},
		{"not found", domain.ErrMemorialNotFound, 404, "MEMORIAL_NOT_FOUND", ""},
		{"infrastructure never leaks its cause", errors.New("pq: connection refused"), 500, "INTERNAL_ERROR", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := new(mockMemorialUsecase)
			uc.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil, tt.err)

			rec := serve(newMemorialMux(uc), memorialReq(http.MethodPost, "/v1/memorials", fullBody, "7"))

			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Equal(t, tt.code, errCode(t, rec))
			body := decode(t, rec)
			if tt.field != "" {
				assert.Equal(t, tt.field, body["details"].(map[string]any)["field"])
			}
			if tt.status == 500 {
				assert.NotContains(t, rec.Body.String(), "connection refused")
			}
		})
	}
}

func TestMemorialHandler_List(t *testing.T) {
	t.Run("items and total", func(t *testing.T) {
		uc := new(mockMemorialUsecase)
		uc.On("List", mock.Anything, int64(7)).Return([]domain.Memorial{*sampleMemorial(), *sampleMemorial()}, nil)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodGet, "/v1/memorials", "", "7"))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Len(t, data["items"], 2)
		assert.Equal(t, float64(2), data["total"])
	})
	t.Run("none is [] not null", func(t *testing.T) {
		uc := new(mockMemorialUsecase)
		uc.On("List", mock.Anything, int64(7)).Return([]domain.Memorial{}, nil)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodGet, "/v1/memorials", "", "7"))

		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, []any{}, data["items"])
		assert.Equal(t, float64(0), data["total"])
	})
}

func TestMemorialHandler_Get(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		uc := new(mockMemorialUsecase)
		uc.On("Get", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodGet, "/v1/memorials/9", "", "7"))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, float64(9), decode(t, rec)["data"].(map[string]any)["id"])
	})
	t.Run("not found", func(t *testing.T) {
		uc := new(mockMemorialUsecase)
		uc.On("Get", mock.Anything, int64(7), int64(9)).Return(nil, domain.ErrMemorialNotFound)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodGet, "/v1/memorials/9", "", "7"))

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "MEMORIAL_NOT_FOUND", errCode(t, rec))
	})
	for _, id := range []string{"abc", "0", "-3", "1.5", "99999999999999999999"} {
		t.Run("id "+id+" is the same 404 and never reaches the use case", func(t *testing.T) {
			uc := new(mockMemorialUsecase)

			rec := serve(newMemorialMux(uc), memorialReq(http.MethodGet, "/v1/memorials/"+id, "", "7"))

			assert.Equal(t, http.StatusNotFound, rec.Code)
			assert.Equal(t, "MEMORIAL_NOT_FOUND", errCode(t, rec))
			uc.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestMemorialHandler_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		uc := new(mockMemorialUsecase)
		uc.On("Update", mock.Anything, int64(7), int64(9), mock.Anything).Return(sampleMemorial(), nil)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodPut, "/v1/memorials/9", fullBody, "7"))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, float64(9), decode(t, rec)["data"].(map[string]any)["id"])
	})
	t.Run("not found", func(t *testing.T) {
		uc := new(mockMemorialUsecase)
		uc.On("Update", mock.Anything, int64(8), int64(9), mock.Anything).Return(nil, domain.ErrMemorialNotFound)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodPut, "/v1/memorials/9", fullBody, "8"))

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "MEMORIAL_NOT_FOUND", errCode(t, rec))
	})
	t.Run("bad id is 404 before the body is looked at", func(t *testing.T) {
		uc := new(mockMemorialUsecase)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodPut, "/v1/memorials/x", `nope`, "7"))

		assert.Equal(t, http.StatusNotFound, rec.Code)
		uc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
	t.Run("unknown field", func(t *testing.T) {
		uc := new(mockMemorialUsecase)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodPut, "/v1/memorials/9", `{"user_id":1}`, "7"))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		uc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestMemorialHandler_Delete(t *testing.T) {
	t.Run("success is 200 with no data", func(t *testing.T) {
		uc := new(mockMemorialUsecase)
		uc.On("Delete", mock.Anything, int64(7), int64(9)).Return(nil)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodDelete, "/v1/memorials/9", "", "7"))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		body := decode(t, rec)
		assert.Equal(t, true, body["success"])
		assert.NotContains(t, body, "data")
	})
	t.Run("not found", func(t *testing.T) {
		uc := new(mockMemorialUsecase)
		uc.On("Delete", mock.Anything, int64(8), int64(9)).Return(domain.ErrMemorialNotFound)

		rec := serve(newMemorialMux(uc), memorialReq(http.MethodDelete, "/v1/memorials/9", "", "8"))

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "MEMORIAL_NOT_FOUND", errCode(t, rec))
	})
}

// Whatever the middleware lets through, a handler without a usable user id must
// refuse rather than act for user 0.
func TestMemorialHandler_NoUsableUserID_Is401(t *testing.T) {
	routes := []struct{ method, target, body string }{
		{http.MethodPost, "/v1/memorials", fullBody},
		{http.MethodGet, "/v1/memorials", ""},
		{http.MethodGet, "/v1/memorials/9", ""},
		{http.MethodPut, "/v1/memorials/9", fullBody},
		{http.MethodDelete, "/v1/memorials/9", ""},
	}
	for _, userID := range []string{"", "not-a-number"} {
		for _, rt := range routes {
			t.Run(rt.method+" "+rt.target+" user="+userID, func(t *testing.T) {
				uc := new(mockMemorialUsecase)

				rec := serve(newMemorialMux(uc), memorialReq(rt.method, rt.target, rt.body, userID))

				assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
				assert.Equal(t, "UNAUTHORIZED", errCode(t, rec))
				assert.Empty(t, uc.Calls, "no use case call without an owner")
			})
		}
	}
}
