package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/jwt/claims"
	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

type mockSuggestionUsecase struct{ mock.Mock }

func (m *mockSuggestionUsecase) List(ctx context.Context, status string, page int) (*domain.SuggestionPage, error) {
	a := m.Called(ctx, status, page)
	if r := a.Get(0); r != nil {
		return r.(*domain.SuggestionPage), a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockSuggestionUsecase) Get(ctx context.Context, id int64) (*domain.TempleSuggestion, error) {
	a := m.Called(ctx, id)
	if r := a.Get(0); r != nil {
		return r.(*domain.TempleSuggestion), a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockSuggestionUsecase) CreateTemple(ctx context.Context, id, adminID int64, role string, in domain.NewTemple) (*domain.CreatedTemple, error) {
	a := m.Called(ctx, id, adminID, role, in)
	if r := a.Get(0); r != nil {
		return r.(*domain.CreatedTemple), a.Error(1)
	}
	return nil, a.Error(1)
}

func sp(s string) *string { return &s }

// serveSuggestion runs one handler through a mux so {id} is populated, with the
// admin claims the real middleware would have placed in the context.
func serveSuggestion(t *testing.T, uc *mockSuggestionUsecase, method, target, body, role string, pick func(*TempleSuggestionHandler) http.HandlerFunc, pattern string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewTempleSuggestionHandler(uc, "")
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, pick(h))
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req = req.WithContext(jwtcontext.WithClaims(req.Context(), &claims.CustomClaims{UserID: "7", Role: role}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
		Error   map[string]any `json:"error"`
		Details map[string]any `json:"details"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), rec.Body.String())
	if !env.Success {
		return map[string]any{"code": env.Error["code"], "details": env.Details}
	}
	return env.Data
}

func TestTempleSuggestionHandler_List(t *testing.T) {
	ctx := mock.Anything
	list := func(h *TempleSuggestionHandler) http.HandlerFunc { return h.List }
	created := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	t.Run("items carry the structured address, legacy rows null codes", func(t *testing.T) {
		uc := &mockSuggestionUsecase{}
		uc.On("List", ctx, "", 1).Return(&domain.SuggestionPage{
			Page: 1, PageSize: 20, Total: 2, TotalPages: 1,
			Items: []domain.TempleSuggestion{
				{ID: 1, Name: "A", ProvinceCode: sp("79"), Province: "Hồ Chí Minh", WardCode: sp("26740"), Ward: sp("Phường Sài Gòn"), Address: "12 NH", Status: "pending", CreatedAt: created},
				{ID: 2, Name: "B", Province: "Quảng Ninh", Address: "Thôn 1", Status: "pending", CreatedAt: created},
			},
		}, nil)
		rec := serveSuggestion(t, uc, http.MethodGet, "/v1/admin/temple-suggestions", "", domain.RoleAdmin, list, "GET /v1/admin/temple-suggestions")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decodeData(t, rec)
		items := data["items"].([]any)
		first, second := items[0].(map[string]any), items[1].(map[string]any)
		assert.Equal(t, "79", first["province_code"])
		assert.Equal(t, "Phường Sài Gòn", first["ward"])
		assert.Nil(t, second["province_code"])
		assert.Nil(t, second["ward_code"])
		assert.Nil(t, second["ward"])
		assert.Equal(t, "Quảng Ninh", second["province"])
		assert.EqualValues(t, 2, data["total"])
	})

	t.Run("status and page are passed through", func(t *testing.T) {
		uc := &mockSuggestionUsecase{}
		uc.On("List", ctx, "created", 3).Return(&domain.SuggestionPage{Page: 3, PageSize: 20, Items: []domain.TempleSuggestion{}}, nil)
		rec := serveSuggestion(t, uc, http.MethodGet, "/v1/admin/temple-suggestions?status=created&page=3", "", domain.RoleAdmin, list, "GET /v1/admin/temple-suggestions")
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("non-numeric page is INVALID_PAGE", func(t *testing.T) {
		uc := &mockSuggestionUsecase{}
		rec := serveSuggestion(t, uc, http.MethodGet, "/v1/admin/temple-suggestions?page=x", "", domain.RoleAdmin, list, "GET /v1/admin/temple-suggestions")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "INVALID_PAGE", decodeData(t, rec)["code"])
	})

	t.Run("bad status is INVALID_STATUS", func(t *testing.T) {
		uc := &mockSuggestionUsecase{}
		uc.On("List", ctx, "bogus", 1).Return(nil, domain.ErrInvalidStatus)
		rec := serveSuggestion(t, uc, http.MethodGet, "/v1/admin/temple-suggestions?status=bogus", "", domain.RoleAdmin, list, "GET /v1/admin/temple-suggestions")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "INVALID_STATUS", decodeData(t, rec)["code"])
	})
}

func TestTempleSuggestionHandler_Get(t *testing.T) {
	get := func(h *TempleSuggestionHandler) http.HandlerFunc { return h.Get }
	const pattern = "GET /v1/admin/temple-suggestions/{id}"

	t.Run("detail includes review fields", func(t *testing.T) {
		uc := &mockSuggestionUsecase{}
		now := time.Now()
		by := int64(3)
		uc.On("Get", mock.Anything, int64(5)).Return(&domain.TempleSuggestion{
			ID: 5, Name: "A", Province: "P", Status: "rejected", TempleID: nil, ReviewedBy: &by, ReviewedAt: &now, RejectReason: sp("rác"),
		}, nil)
		rec := serveSuggestion(t, uc, http.MethodGet, "/v1/admin/temple-suggestions/5", "", domain.RoleViewer, get, pattern)
		require.Equal(t, http.StatusOK, rec.Code)
		d := decodeData(t, rec)
		assert.Equal(t, "rác", d["reject_reason"])
		assert.EqualValues(t, 3, d["reviewed_by"])
		assert.Nil(t, d["temple_id"])
	})

	for _, id := range []string{"0", "-1", "abc"} {
		t.Run("id "+id+" is 404", func(t *testing.T) {
			rec := serveSuggestion(t, &mockSuggestionUsecase{}, http.MethodGet, "/v1/admin/temple-suggestions/"+id, "", domain.RoleAdmin, get, pattern)
			assert.Equal(t, http.StatusNotFound, rec.Code)
			assert.Equal(t, "TEMPLE_SUGGESTION_NOT_FOUND", decodeData(t, rec)["code"])
		})
	}

	t.Run("unknown id is 404", func(t *testing.T) {
		uc := &mockSuggestionUsecase{}
		uc.On("Get", mock.Anything, int64(9)).Return(nil, domain.ErrSuggestionNotFound)
		rec := serveSuggestion(t, uc, http.MethodGet, "/v1/admin/temple-suggestions/9", "", domain.RoleAdmin, get, pattern)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

const validCreateBody = `{"name":"Chùa Mới","type":"CHUA","region":"south","sect":"Bắc tông","established_year":1990,
	"heritage_rank":"DI_TICH_CAP_TINH","latitude":10.77,"longitude":106.69,
	"province_code":"79","ward_code":"26740","address":"12 Nguyễn Huệ","image_url":"/static/images/x.jpg"}`

func TestTempleSuggestionHandler_CreateTemple(t *testing.T) {
	create := func(h *TempleSuggestionHandler) http.HandlerFunc { return h.CreateTemple }
	const pattern = "POST /v1/admin/temple-suggestions/{id}/create-temple"
	const target = "/v1/admin/temple-suggestions/5/create-temple"

	t.Run("201 with the temple shaped like GET /v1/temples/{id}; ids come from the token", func(t *testing.T) {
		uc := &mockSuggestionUsecase{}
		var got domain.NewTemple
		uc.On("CreateTemple", mock.Anything, int64(5), int64(7), domain.RoleContentEditor, mock.Anything).
			Run(func(a mock.Arguments) { got = a.Get(4).(domain.NewTemple) }).
			Return(&domain.CreatedTemple{
				Suggestion: domain.TempleSuggestion{ID: 5, Status: "created", TempleID: sp("temple_south_181")},
				Temple: domain.Temple{
					ID: "temple_south_181", Name: "Chùa Mới", Type: "CHUA", Region: "south", ImageURL: "/static/images/x.jpg",
					Sect: "Bắc tông", HeritageRank: "DI_TICH_CAP_TINH", EstablishedYear: 1990, Latitude: 10.77, Longitude: 106.69,
					Address: "12 Nguyễn Huệ", ProvinceCode: "79", Province: "Hồ Chí Minh", WardCode: "26740", Ward: "Phường Sài Gòn",
				},
			}, nil)

		rec := serveSuggestion(t, uc, http.MethodPost, target, validCreateBody, domain.RoleContentEditor, create, pattern)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		assert.Equal(t, "79", got.ProvinceCode)
		assert.Equal(t, "26740", got.WardCode)
		assert.Equal(t, "12 Nguyễn Huệ", got.Address)
		assert.Empty(t, got.Province, "names are never taken from the request")

		d := decodeData(t, rec)
		assert.Equal(t, "created", d["suggestion"].(map[string]any)["status"])
		temple := d["temple"].(map[string]any)
		loc := temple["location"].(map[string]any)
		assert.Equal(t, "79", loc["province_code"])
		assert.Equal(t, "26740", loc["ward_code"])
		assert.Equal(t, "Phường Sài Gòn", loc["ward"])
		assert.Equal(t, "12 Nguyễn Huệ, Phường Sài Gòn, Hồ Chí Minh", loc["full_address"])
		assert.Equal(t, "http://example.com/static/images/x.jpg", temple["image_url"])
		assert.Nil(t, temple["thumbnail_url"])
		assert.Equal(t, []any{}, temple["aliases"])
	})

	t.Run("province in the body is an unknown field", func(t *testing.T) {
		rec := serveSuggestion(t, &mockSuggestionUsecase{}, http.MethodPost, target,
			`{"name":"x","province":"Hà Nội"}`, domain.RoleAdmin, create, pattern)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	for _, field := range []string{"established_year", "latitude", "longitude"} {
		t.Run("missing "+field, func(t *testing.T) {
			var m map[string]any
			require.NoError(t, json.Unmarshal([]byte(validCreateBody), &m))
			delete(m, field)
			b, _ := json.Marshal(m)
			rec := serveSuggestion(t, &mockSuggestionUsecase{}, http.MethodPost, target, string(b), domain.RoleAdmin, create, pattern)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			d := decodeData(t, rec)
			assert.Equal(t, field, d["details"].(map[string]any)["field"])
		})
	}

	t.Run("bad id is 404 before the body is read", func(t *testing.T) {
		rec := serveSuggestion(t, &mockSuggestionUsecase{}, http.MethodPost, "/v1/admin/temple-suggestions/0/create-temple", "not json", domain.RoleAdmin, create, pattern)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"input error names the field", &domain.TempleInputError{Field: "ward_code", Message: "bad"}, http.StatusBadRequest, "VALIDATION_ERROR"},
		{"not found", domain.ErrSuggestionNotFound, http.StatusNotFound, "TEMPLE_SUGGESTION_NOT_FOUND"},
		{"already created", domain.ErrSuggestionAlreadyCreated, http.StatusConflict, "SUGGESTION_ALREADY_CREATED"},
		{"reopen forbidden", domain.ErrReopenForbidden, http.StatusForbidden, "FORBIDDEN"},
		{"unexpected error hides the cause", assert.AnError, http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := &mockSuggestionUsecase{}
			uc.On("CreateTemple", mock.Anything, int64(5), int64(7), domain.RoleAdmin, mock.Anything).Return(nil, tc.err)
			rec := serveSuggestion(t, uc, http.MethodPost, target, validCreateBody, domain.RoleAdmin, create, pattern)
			assert.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.code, decodeData(t, rec)["code"])
			assert.NotContains(t, rec.Body.String(), assert.AnError.Error())
		})
	}
}

func TestRequireAnyRole(t *testing.T) {
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }
	run := func(role string) int {
		h := RequireAnyRole(ok, domain.RoleAdmin, domain.RoleContentEditor)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(jwtcontext.WithClaims(req.Context(), &claims.CustomClaims{UserID: "1", Role: role}))
		rec := httptest.NewRecorder()
		h(rec, req)
		return rec.Code
	}
	assert.Equal(t, http.StatusNoContent, run(domain.RoleAdmin))
	assert.Equal(t, http.StatusNoContent, run(domain.RoleContentEditor))
	assert.Equal(t, http.StatusForbidden, run(domain.RoleViewer))
}
