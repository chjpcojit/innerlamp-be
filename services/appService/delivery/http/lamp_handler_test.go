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

// newLampMux registers the routes the way cmd/main.go does (minus the auth
// middleware, whose claims the test requests carry directly).
func newLampMux(uc *mockLampUsecase) *http.ServeMux {
	h := NewLampHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/lamps", h.Light)
	mux.HandleFunc("GET /v1/lamps", h.Garden)
	return mux
}

func strp(s string) *string { return &s }

func sampleLamp() *domain.Lamp {
	return &domain.Lamp{
		ID: 9, UserID: 7, LampType: domain.LampTypeLotusCandle, Intention: domain.IntentionGratitude,
		Recipient: "Thầy", Prayer: strp("Tri ân"), DurationDays: 7, IsAnonymous: false,
		LighterName: strp("Thích Minh An"),
		LitAt:       time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC),
		EndAt:       time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC),
	}
}

const lampBody = `{"lamp_type":"LOTUS_CANDLE","intention":"GRATITUDE","recipient":"Thầy","prayer":"Tri ân","duration_days":7,"is_anonymous":false}`

func TestLampHandler_Light_Created(t *testing.T) {
	uc := new(mockLampUsecase)
	var got domain.LampInput
	uc.On("Light", mock.Anything, int64(7), mock.Anything).
		Run(func(args mock.Arguments) { got = args.Get(2).(domain.LampInput) }).
		Return(sampleLamp(), true, nil)

	rec := serve(newLampMux(uc), memorialReq(http.MethodPost, "/v1/lamps", lampBody, "7"))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, domain.LampInput{
		LampType: "LOTUS_CANDLE", Intention: "GRATITUDE", Recipient: "Thầy", Prayer: strp("Tri ân"), DurationDays: 7,
	}, got)

	body := decode(t, rec)
	assert.Equal(t, true, body["success"])
	data := body["data"].(map[string]any)
	assert.Equal(t, float64(9), data["id"])
	assert.Equal(t, "LOTUS_CANDLE", data["lamp_type"])
	assert.Equal(t, "GRATITUDE", data["intention"])
	assert.Equal(t, "Thầy", data["recipient"])
	assert.Equal(t, "Tri ân", data["prayer"])
	assert.Equal(t, float64(7), data["duration_days"])
	assert.Equal(t, false, data["is_anonymous"])
	assert.Equal(t, "Thích Minh An", data["lighter_name"])
	assert.Equal(t, "2026-10-10T12:00:00Z", data["lit_at"])
	assert.Equal(t, "2026-10-17T12:00:00Z", data["end_at"])
	assert.NotContains(t, data, "user_id", "the owner id is never on the wire")
}

func TestLampHandler_Light_DoubleTapReturnsOK(t *testing.T) {
	uc := new(mockLampUsecase)
	uc.On("Light", mock.Anything, int64(7), mock.Anything).Return(sampleLamp(), false, nil)

	rec := serve(newLampMux(uc), memorialReq(http.MethodPost, "/v1/lamps", lampBody, "7"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, float64(9), decode(t, rec)["data"].(map[string]any)["id"])
}

func TestLampHandler_Light_NullablesAreJSONNull(t *testing.T) {
	l := sampleLamp()
	l.Prayer, l.LighterName, l.IsAnonymous = nil, nil, true
	uc := new(mockLampUsecase)
	uc.On("Light", mock.Anything, int64(7), mock.Anything).Return(l, true, nil)

	rec := serve(newLampMux(uc), memorialReq(http.MethodPost, "/v1/lamps",
		`{"lamp_type":"OIL_LAMP","intention":"PEACE","recipient":"r","duration_days":1,"is_anonymous":true}`, "7"))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	data := decode(t, rec)["data"].(map[string]any)
	assert.Contains(t, data, "prayer")
	assert.Nil(t, data["prayer"], "no prayer is JSON null")
	assert.Equal(t, "Vô danh", data["lighter_name"])
	assert.Equal(t, true, data["is_anonymous"])
}

func TestLampHandler_Light_Errors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		userID     string
		ucErr      error
		wantStatus int
		wantCode   string
		wantField  string
		callsUC    bool
	}{
		{"no claims", lampBody, "", nil, http.StatusUnauthorized, "UNAUTHORIZED", "", false},
		{"non numeric user id claim", lampBody, "abc", nil, http.StatusUnauthorized, "UNAUTHORIZED", "", false},
		{"empty body", ``, "7", nil, http.StatusBadRequest, "VALIDATION_ERROR", "", false},
		{"not json", `{nope`, "7", nil, http.StatusBadRequest, "VALIDATION_ERROR", "", false},
		{"unknown field", `{"lamp_type":"OIL_LAMP","extra":1}`, "7", nil, http.StatusBadRequest, "VALIDATION_ERROR", "", false},
		{"server owned id", `{"id":5,"lamp_type":"OIL_LAMP"}`, "7", nil, http.StatusBadRequest, "VALIDATION_ERROR", "", false},
		{"server owned end_at", `{"end_at":"2030-01-01T00:00:00Z"}`, "7", nil, http.StatusBadRequest, "VALIDATION_ERROR", "", false},
		{"server owned lighter_name", `{"lighter_name":"x"}`, "7", nil, http.StatusBadRequest, "VALIDATION_ERROR", "", false},
		{"is_anonymous not a boolean", `{"lamp_type":"OIL_LAMP","is_anonymous":"yes"}`, "7", nil, http.StatusBadRequest, "VALIDATION_ERROR", "is_anonymous", false},
		{"field rejected by the use case", lampBody, "7",
			&domain.LampValidationError{Field: "recipient", Message: "recipient is required"},
			http.StatusBadRequest, "VALIDATION_ERROR", "recipient", true},
		{"limit reached", lampBody, "7", domain.ErrLampLimitReached, http.StatusConflict, "LAMP_LIMIT_REACHED", "", true},
		{"unexpected error", lampBody, "7", errors.New("db down"), http.StatusInternalServerError, "INTERNAL_ERROR", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := new(mockLampUsecase)
			if tt.callsUC {
				uc.On("Light", mock.Anything, int64(7), mock.Anything).Return(nil, false, tt.ucErr)
			}

			rec := serve(newLampMux(uc), memorialReq(http.MethodPost, "/v1/lamps", tt.body, tt.userID))

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			assert.Equal(t, tt.wantCode, errCode(t, rec))
			if tt.wantField != "" {
				details := decode(t, rec)["details"].(map[string]any)
				assert.Equal(t, tt.wantField, details["field"])
			}
			if !tt.callsUC {
				uc.AssertNotCalled(t, "Light", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

func sampleGarden() *domain.GardenPage {
	name := "Thích Minh An"
	next := "MTc5MDAwMDAwMDAwMDAwMC40MQ"
	return &domain.GardenPage{
		Items: []domain.GardenLamp{
			{ID: 41, LampType: domain.LampTypeLotusCandle, Recipient: "Thầy", LighterName: &name,
				LitAt: time.Date(2026, 10, 10, 11, 0, 0, 0, time.UTC), EndAt: time.Date(2026, 10, 17, 18, 0, 0, 0, time.UTC)},
			{ID: 40, LampType: domain.LampTypeOilLamp, Recipient: "Mẹ", LighterName: nil,
				LitAt: time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC), EndAt: time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)},
		},
		Total:   10,
		Counts:  map[domain.LampType]int{domain.LampTypeOilLamp: 5, domain.LampTypeLotusCandle: 3, domain.LampTypeFlowerLantern: 2},
		HasMore: true, NextCursor: &next,
	}
}

func TestLampHandler_Garden_Success(t *testing.T) {
	uc := new(mockLampUsecase)
	uc.On("Garden", mock.Anything, mock.Anything).Return(sampleGarden(), nil)

	rec := serve(newLampMux(uc), memorialReq(http.MethodGet, "/v1/lamps", "", "7"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := decode(t, rec)["data"].(map[string]any)

	items := data["items"].([]any)
	require.Len(t, items, 2)
	first := items[0].(map[string]any)
	assert.Equal(t, map[string]any{
		"id": float64(41), "lamp_type": "LOTUS_CANDLE", "recipient": "Thầy",
		"lighter_name": "Thích Minh An", "end_at": "2026-10-17T18:00:00Z",
	}, first, "a garden item carries exactly these five fields")
	assert.Equal(t, "Vô danh", items[1].(map[string]any)["lighter_name"])

	assert.Equal(t, float64(10), data["total"])
	assert.Equal(t, map[string]any{"OIL_LAMP": float64(5), "LOTUS_CANDLE": float64(3), "FLOWER_LANTERN": float64(2)}, data["counts"])
	assert.Equal(t, true, data["has_more"])
	assert.Equal(t, "MTc5MDAwMDAwMDAwMDAwMC40MQ", data["next_cursor"])
}

func TestLampHandler_Garden_NeverExposesPrivateFields(t *testing.T) {
	uc := new(mockLampUsecase)
	uc.On("Garden", mock.Anything, mock.Anything).Return(sampleGarden(), nil)

	rec := serve(newLampMux(uc), memorialReq(http.MethodGet, "/v1/lamps", "", "7"))

	for _, forbidden := range []string{"intention", "prayer", "user_id", "is_anonymous", "lit_at"} {
		assert.NotContains(t, rec.Body.String(), `"`+forbidden+`"`, forbidden)
	}
}

func TestLampHandler_Garden_EmptyIsEmptyArrayAndZeroCounts(t *testing.T) {
	uc := new(mockLampUsecase)
	uc.On("Garden", mock.Anything, mock.Anything).Return(&domain.GardenPage{}, nil)

	rec := serve(newLampMux(uc), memorialReq(http.MethodGet, "/v1/lamps", "", "7"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, []any{}, data["items"], "items is [] not null")
	assert.Equal(t, float64(0), data["total"])
	assert.Equal(t, map[string]any{"OIL_LAMP": float64(0), "LOTUS_CANDLE": float64(0), "FLOWER_LANTERN": float64(0)}, data["counts"])
	assert.Equal(t, false, data["has_more"])
	assert.Nil(t, data["next_cursor"])
	assert.Contains(t, data, "next_cursor", "next_cursor is present as null")
}

func TestLampHandler_Garden_PassesQueryParameters(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   domain.GardenFilter
	}{
		{"none", "/v1/lamps", domain.GardenFilter{}},
		{"all three", "/v1/lamps?lamp_type=OIL_LAMP&limit=5&cursor=abc", domain.GardenFilter{LampType: strp("OIL_LAMP"), Limit: strp("5"), Cursor: strp("abc")}},
		{"empty values stay present so the use case can reject them", "/v1/lamps?lamp_type=&limit=&cursor=", domain.GardenFilter{LampType: strp(""), Limit: strp(""), Cursor: strp("")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := new(mockLampUsecase)
			var got domain.GardenFilter
			uc.On("Garden", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) { got = args.Get(1).(domain.GardenFilter) }).
				Return(&domain.GardenPage{}, nil)

			rec := serve(newLampMux(uc), memorialReq(http.MethodGet, tt.target, "", "7"))

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLampHandler_Garden_Errors(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		ucErr      error
		wantStatus int
		wantCode   string
		wantField  string
		callsUC    bool
	}{
		{"no claims", "", nil, http.StatusUnauthorized, "UNAUTHORIZED", "", false},
		{"bad filter", "7", domain.ErrInvalidLampFilter, http.StatusBadRequest, "INVALID_FILTER", "lamp_type", true},
		{"bad limit", "7", domain.ErrInvalidLimit, http.StatusBadRequest, "INVALID_LIMIT", "limit", true},
		{"bad cursor", "7", domain.ErrInvalidCursor, http.StatusBadRequest, "INVALID_CURSOR", "cursor", true},
		{"unexpected error", "7", errors.New("db down"), http.StatusInternalServerError, "INTERNAL_ERROR", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := new(mockLampUsecase)
			if tt.callsUC {
				uc.On("Garden", mock.Anything, mock.Anything).Return(nil, tt.ucErr)
			}

			rec := serve(newLampMux(uc), memorialReq(http.MethodGet, "/v1/lamps", "", tt.userID))

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			assert.Equal(t, tt.wantCode, errCode(t, rec))
			if tt.wantField != "" {
				assert.Equal(t, tt.wantField, decode(t, rec)["details"].(map[string]any)["field"])
			}
			if !tt.callsUC {
				uc.AssertNotCalled(t, "Garden", mock.Anything, mock.Anything)
			}
		})
	}
}
