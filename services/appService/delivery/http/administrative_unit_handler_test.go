package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

const testAdminUnitsCacheAge = 24 * time.Hour

// registerAdministrativeUnitRoutes registers the nine routes with the same
// paths cmd/main.go uses, so {code} extraction and the literal-vs-wildcard
// precedence are exercised for real. wrap is how main.go would wrap each one.
func registerAdministrativeUnitRoutes(mux *http.ServeMux, h *AdministrativeUnitHandler, wrap func(http.HandlerFunc) http.HandlerFunc) {
	const p = "GET /v1/administrative-units/"
	mux.HandleFunc(p+"new/provinces", wrap(h.CurrentProvinces))
	mux.HandleFunc(p+"new/provinces/{code}/wards", wrap(h.CurrentWards))
	mux.HandleFunc(p+"new/provinces/{code}/conversion", wrap(h.ConvertCurrentProvince))
	mux.HandleFunc(p+"new/wards/{code}/conversion", wrap(h.ConvertCurrentWard))
	mux.HandleFunc(p+"old/provinces", wrap(h.LegacyProvinces))
	mux.HandleFunc(p+"old/provinces/{code}/districts", wrap(h.LegacyDistricts))
	mux.HandleFunc(p+"old/provinces/{code}/conversion", wrap(h.ConvertLegacyProvince))
	mux.HandleFunc(p+"old/districts/{code}/wards", wrap(h.LegacyWards))
	mux.HandleFunc(p+"old/wards/{code}/conversion", wrap(h.ConvertLegacyWard))
}

func newAdminUnitMux(uc *mockAdministrativeUnitUsecase) *http.ServeMux {
	mux := http.NewServeMux()
	registerAdministrativeUnitRoutes(mux, NewAdministrativeUnitHandler(uc, testAdminUnitsCacheAge),
		func(f http.HandlerFunc) http.HandlerFunc { return f })
	return mux
}

func TestAdministrativeUnitHandler_Lists(t *testing.T) {
	tests := []struct {
		name   string
		target string
		stub   func(*mockAdministrativeUnitUsecase)
		want   []map[string]any // first-level fields expected on the single item
	}{
		{
			name: "new provinces", target: "/v1/administrative-units/new/provinces",
			stub: func(uc *mockAdministrativeUnitUsecase) {
				uc.On("ListCurrentProvinces", mock.Anything).Return(
					[]domain.CurrentProvince{{Code: "79", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city", SortOrder: 1}}, nil)
			},
			want: []map[string]any{{"code": "79", "name": "Hồ Chí Minh", "full_name": "Thành phố Hồ Chí Minh", "type": "city"}},
		},
		{
			name: "new wards of a province", target: "/v1/administrative-units/new/provinces/79/wards",
			stub: func(uc *mockAdministrativeUnitUsecase) {
				uc.On("ListCurrentWards", mock.Anything, "79").Return(
					[]domain.CurrentWard{{Code: "26737", ProvinceCode: "79", Name: "Tân Định", FullName: "Phường Tân Định", Type: "ward", SortOrder: 5}}, nil)
			},
			want: []map[string]any{{"code": "26737", "name": "Tân Định", "full_name": "Phường Tân Định", "type": "ward", "province_code": "79"}},
		},
		{
			name: "old provinces", target: "/v1/administrative-units/old/provinces",
			stub: func(uc *mockAdministrativeUnitUsecase) {
				uc.On("ListLegacyProvinces", mock.Anything).Return(
					[]domain.LegacyProvince{{Code: "74", Name: "Bình Dương", FullName: "Tỉnh Bình Dương", Type: "province"}}, nil)
			},
			want: []map[string]any{{"code": "74", "name": "Bình Dương", "full_name": "Tỉnh Bình Dương", "type": "province"}},
		},
		{
			name: "old districts of a province", target: "/v1/administrative-units/old/provinces/79/districts",
			stub: func(uc *mockAdministrativeUnitUsecase) {
				uc.On("ListLegacyDistricts", mock.Anything, "79").Return(
					[]domain.LegacyDistrict{{Code: "760", ProvinceCode: "79", Name: "Quận 1", FullName: "Quận 1", Type: "urban_district"}}, nil)
			},
			want: []map[string]any{{"code": "760", "name": "Quận 1", "full_name": "Quận 1", "type": "urban_district", "province_code": "79"}},
		},
		{
			name: "old wards of a district", target: "/v1/administrative-units/old/districts/760/wards",
			stub: func(uc *mockAdministrativeUnitUsecase) {
				uc.On("ListLegacyWards", mock.Anything, "760").Return(
					[]domain.LegacyWard{{Code: "26734", DistrictCode: "760", ProvinceCode: "79", Name: "Tân Định", FullName: "Phường Tân Định", Type: "ward"}}, nil)
			},
			want: []map[string]any{{"code": "26734", "name": "Tân Định", "full_name": "Phường Tân Định", "type": "ward", "district_code": "760", "province_code": "79"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := &mockAdministrativeUnitUsecase{}
			tc.stub(uc)

			rec := doGet(newAdminUnitMux(uc), tc.target)

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.Equal(t, "public, max-age=86400", rec.Header().Get("Cache-Control"))
			body := decode(t, rec)
			assert.Equal(t, true, body["success"])
			items := body["data"].(map[string]any)["items"].([]any)
			require.Len(t, items, len(tc.want))
			assert.Equal(t, tc.want[0], items[0], "exactly the documented fields, no sort_order")
			uc.AssertExpectations(t)
		})
	}
}

func TestAdministrativeUnitHandler_EmptyListIs200WithEmptyArray(t *testing.T) {
	uc := &mockAdministrativeUnitUsecase{}
	uc.On("ListLegacyWards", mock.Anything, "318").Return([]domain.LegacyWard{}, nil)

	rec := doGet(newAdminUnitMux(uc), "/v1/administrative-units/old/districts/318/wards")

	require.Equal(t, http.StatusOK, rec.Code)
	items := decode(t, rec)["data"].(map[string]any)["items"]
	assert.Equal(t, []any{}, items, "an island district has no wards: [] not null")
}

func TestAdministrativeUnitHandler_ConvertLegacyWard(t *testing.T) {
	t.Run("unique conversion carries ward, province, mapping type and note", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("ConvertLegacyWard", mock.Anything, "26761").Return(domain.Conversion[domain.CurrentWardCandidate]{
			Candidates: []domain.CurrentWardCandidate{{
				Ward:        domain.CurrentWard{Code: "26758", ProvinceCode: "79", Name: "Cầu Ông Lãnh", FullName: "Phường Cầu Ông Lãnh", Type: "ward"},
				Province:    domain.CurrentProvince{Code: "79", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city"},
				MappingType: "merged", Note: "Nhập toàn bộ",
			}},
		}, nil)

		rec := doGet(newAdminUnitMux(uc), "/v1/administrative-units/old/wards/26761/conversion")

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "public, max-age=86400", rec.Header().Get("Cache-Control"))
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, true, data["unique"])
		cands := data["candidates"].([]any)
		require.Len(t, cands, 1)
		c := cands[0].(map[string]any)
		assert.Equal(t, "merged", c["mapping_type"])
		assert.Equal(t, "Nhập toàn bộ", c["note"])
		assert.Equal(t, map[string]any{
			"code": "26758", "name": "Cầu Ông Lãnh", "full_name": "Phường Cầu Ông Lãnh", "type": "ward", "province_code": "79",
		}, c["ward"])
		assert.Equal(t, map[string]any{
			"code": "79", "name": "Hồ Chí Minh", "full_name": "Thành phố Hồ Chí Minh", "type": "city",
		}, c["province"])
	})

	t.Run("several candidates are all returned and unique is false", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("ConvertLegacyWard", mock.Anything, "26737").Return(domain.Conversion[domain.CurrentWardCandidate]{
			Candidates: []domain.CurrentWardCandidate{
				{Ward: domain.CurrentWard{Code: "26740", Name: "Sài Gòn"}, MappingType: "ambiguous"},
				{Ward: domain.CurrentWard{Code: "26737", Name: "Tân Định"}, MappingType: "ambiguous"},
			},
		}, nil)

		rec := doGet(newAdminUnitMux(uc), "/v1/administrative-units/old/wards/26737/conversion")

		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, false, data["unique"])
		assert.Len(t, data["candidates"], 2, "the server never chooses for the caller")
	})

	t.Run("not found maps to 404 ADMINISTRATIVE_UNIT_NOT_FOUND and carries no cache header", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("ConvertLegacyWard", mock.Anything, "nope").Return(
			domain.Conversion[domain.CurrentWardCandidate]{}, domain.ErrAdministrativeUnitNotFound)

		rec := doGet(newAdminUnitMux(uc), "/v1/administrative-units/old/wards/nope/conversion")

		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Empty(t, rec.Header().Get("Cache-Control"), "an error must not be cached")
		body := decode(t, rec)
		assert.Equal(t, false, body["success"])
		assert.Equal(t, "ADMINISTRATIVE_UNIT_NOT_FOUND", body["error"].(map[string]any)["code"])
	})
}

func TestAdministrativeUnitHandler_ConvertCurrentWard(t *testing.T) {
	uc := &mockAdministrativeUnitUsecase{}
	uc.On("ConvertCurrentWard", mock.Anything, "26758").Return(domain.Conversion[domain.LegacyWardCandidate]{
		Candidates: []domain.LegacyWardCandidate{
			{
				Ward:        domain.LegacyWard{Code: "26755", DistrictCode: "760", ProvinceCode: "79", Name: "Cô Giang", FullName: "Phường Cô Giang", Type: "ward"},
				District:    domain.LegacyDistrict{Code: "760", ProvinceCode: "79", Name: "Quận 1", FullName: "Quận 1", Type: "urban_district"},
				Province:    domain.LegacyProvince{Code: "79", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city"},
				MappingType: "merged", Note: "Nhập toàn bộ",
			},
			{Ward: domain.LegacyWard{Code: "26761", Name: "Cầu Kho"}, MappingType: "merged"},
		},
	}, nil)

	rec := doGet(newAdminUnitMux(uc), "/v1/administrative-units/new/wards/26758/conversion")

	require.Equal(t, http.StatusOK, rec.Code)
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, false, data["unique"])
	first := data["candidates"].([]any)[0].(map[string]any)
	assert.Equal(t, "Phường Cô Giang", first["ward"].(map[string]any)["full_name"])
	assert.Equal(t, "760", first["district"].(map[string]any)["code"])
	assert.Equal(t, "Hồ Chí Minh", first["province"].(map[string]any)["name"])
}

func TestAdministrativeUnitHandler_SpecialZoneWithNoLegacyWardIsAnEmptyUnconvertedAnswer(t *testing.T) {
	uc := &mockAdministrativeUnitUsecase{}
	uc.On("ConvertCurrentWard", mock.Anything, "11948").Return(
		domain.Conversion[domain.LegacyWardCandidate]{Candidates: []domain.LegacyWardCandidate{}}, nil)

	rec := doGet(newAdminUnitMux(uc), "/v1/administrative-units/new/wards/11948/conversion")

	require.Equal(t, http.StatusOK, rec.Code)
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, false, data["unique"])
	assert.Equal(t, []any{}, data["candidates"])
}

func TestAdministrativeUnitHandler_ProvinceConversions(t *testing.T) {
	t.Run("old → new is exactly one", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("ConvertLegacyProvince", mock.Anything, "74").Return(domain.Conversion[domain.CurrentProvince]{
			Candidates: []domain.CurrentProvince{{Code: "79", Name: "Hồ Chí Minh", FullName: "Thành phố Hồ Chí Minh", Type: "city"}},
		}, nil)

		rec := doGet(newAdminUnitMux(uc), "/v1/administrative-units/old/provinces/74/conversion")

		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, true, data["unique"])
		assert.Equal(t, "79", data["candidates"].([]any)[0].(map[string]any)["code"])
	})

	t.Run("new → old lists every merged province", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("ConvertCurrentProvince", mock.Anything, "79").Return(domain.Conversion[domain.LegacyProvince]{
			Candidates: []domain.LegacyProvince{{Code: "74"}, {Code: "77"}, {Code: "79"}},
		}, nil)

		rec := doGet(newAdminUnitMux(uc), "/v1/administrative-units/new/provinces/79/conversion")

		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, false, data["unique"])
		assert.Len(t, data["candidates"], 3)
	})
}

func TestAdministrativeUnitHandler_NotFoundOnEveryListRoute(t *testing.T) {
	routes := []struct {
		target string
		method string
	}{
		{"/v1/administrative-units/new/provinces/99/wards", "ListCurrentWards"},
		{"/v1/administrative-units/old/provinces/99/districts", "ListLegacyDistricts"},
		{"/v1/administrative-units/old/districts/99/wards", "ListLegacyWards"},
	}
	for _, rt := range routes {
		t.Run(rt.method, func(t *testing.T) {
			uc := &mockAdministrativeUnitUsecase{}
			uc.On(rt.method, mock.Anything, "99").Return(nil, domain.ErrAdministrativeUnitNotFound)

			rec := doGet(newAdminUnitMux(uc), rt.target)

			require.Equal(t, http.StatusNotFound, rec.Code)
			assert.Empty(t, rec.Header().Get("Cache-Control"))
			assert.Equal(t, "ADMINISTRATIVE_UNIT_NOT_FOUND", decode(t, rec)["error"].(map[string]any)["code"])
		})
	}
}

func TestAdministrativeUnitHandler_UnexpectedErrorIs500AndLeaksNothing(t *testing.T) {
	uc := &mockAdministrativeUnitUsecase{}
	uc.On("ListCurrentProvinces", mock.Anything).Return(nil, errors.New("pq: password authentication failed"))

	rec := doGet(newAdminUnitMux(uc), "/v1/administrative-units/new/provinces")

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "password")
	assert.Empty(t, rec.Header().Get("Cache-Control"))
}

func TestAdministrativeUnitHandler_CacheControlFollowsConfig(t *testing.T) {
	uc := &mockAdministrativeUnitUsecase{}
	uc.On("ListLegacyProvinces", mock.Anything).Return([]domain.LegacyProvince{}, nil)
	mux := http.NewServeMux()
	registerAdministrativeUnitRoutes(mux, NewAdministrativeUnitHandler(uc, 90*time.Second),
		func(f http.HandlerFunc) http.HandlerFunc { return f })

	rec := doGet(mux, "/v1/administrative-units/old/provinces")

	assert.Equal(t, "public, max-age=90", rec.Header().Get("Cache-Control"))
}

// TestAdministrativeUnitRoutes_ComposedLikeMain wraps the routes with the real
// RequireWebAPIKey and ClientIPRateLimit in the order cmd/main.go uses
// (webAPIKey first, so a refused key does not spend the caller's budget) and
// proves what the spec promises about access.
func TestAdministrativeUnitRoutes_ComposedLikeMain(t *testing.T) {
	build := func(t *testing.T, max int64) (*http.ServeMux, *mockAdministrativeUnitUsecase) {
		t.Helper()
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("ListCurrentProvinces", mock.Anything).Return([]domain.CurrentProvince{{Code: "01"}}, nil)
		webKey := NewRequireWebAPIKey("web-secret")
		limit, err := NewClientIPRateLimit(max, time.Minute, false)
		require.NoError(t, err)
		mux := http.NewServeMux()
		registerAdministrativeUnitRoutes(mux, NewAdministrativeUnitHandler(uc, testAdminUnitsCacheAge),
			func(f http.HandlerFunc) http.HandlerFunc { return webKey.WrapFunc(limit.WrapFunc(f)) })
		return mux, uc
	}
	get := func(mux *http.ServeMux, key *string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/administrative-units/new/provinces", nil)
		req.RemoteAddr = "203.0.113.7:51000"
		if key != nil {
			req.Header.Set("X-API-Key", *key)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	ptr := func(s string) *string { return &s }

	t.Run("needs no access token", func(t *testing.T) {
		mux, _ := build(t, 100)
		assert.Equal(t, http.StatusOK, get(mux, nil).Code, "no Authorization header and no key: served as mobile")
	})

	t.Run("correct web key is served", func(t *testing.T) {
		mux, _ := build(t, 100)
		assert.Equal(t, http.StatusOK, get(mux, ptr("web-secret")).Code)
	})

	t.Run("wrong web key is refused with 401 INVALID_API_KEY", func(t *testing.T) {
		mux, uc := build(t, 100)
		rec := get(mux, ptr("nope"))
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "INVALID_API_KEY", decode(t, rec)["error"].(map[string]any)["code"])
		uc.AssertNotCalled(t, "ListCurrentProvinces", mock.Anything)
	})

	t.Run("over the per-IP budget is 429 and the handler does not run", func(t *testing.T) {
		mux, uc := build(t, 2)
		assert.Equal(t, http.StatusOK, get(mux, nil).Code)
		assert.Equal(t, http.StatusOK, get(mux, nil).Code)

		rec := get(mux, nil)

		require.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Equal(t, "RATE_LIMITED", decode(t, rec)["error"].(map[string]any)["code"])
		uc.AssertNumberOfCalls(t, "ListCurrentProvinces", 2)
	})

	t.Run("a refused key does not spend the caller's budget", func(t *testing.T) {
		mux, _ := build(t, 1)
		for range 5 {
			assert.Equal(t, http.StatusUnauthorized, get(mux, ptr("nope")).Code)
		}
		assert.Equal(t, http.StatusOK, get(mux, nil).Code, "the one allowed request is still available")
	})
}

func TestAdministrativeUnitHandler_CurrentWardsSearch(t *testing.T) {
	const base = "/v1/administrative-units/new/provinces/79/wards"
	hit := []domain.CurrentWard{{Code: "26737", ProvinceCode: "79", Name: "Tân Định", FullName: "Phường Tân Định", Type: "ward"}}

	t.Run("q present searches, passes the raw term and answers in the list shape", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("SearchCurrentWards", mock.Anything, "79", "tan dinh").Return(hit, nil)

		rec := doGet(newAdminUnitMux(uc), base+"?q=tan%20dinh")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, "public, max-age=86400", rec.Header().Get("Cache-Control"))
		items := decode(t, rec)["data"].(map[string]any)["items"].([]any)
		require.Len(t, items, 1)
		assert.Equal(t, "Phường Tân Định", items[0].(map[string]any)["full_name"])
		uc.AssertNotCalled(t, "ListCurrentWards", mock.Anything, mock.Anything)
		uc.AssertExpectations(t)
	})

	t.Run("q absent still lists every ward", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("ListCurrentWards", mock.Anything, "79").Return(hit, nil)

		rec := doGet(newAdminUnitMux(uc), base)

		require.Equal(t, http.StatusOK, rec.Code)
		uc.AssertNotCalled(t, "SearchCurrentWards", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("an empty q is a 400 INVALID_FILTER, not 'no search'", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("SearchCurrentWards", mock.Anything, "79", "").Return(nil, domain.ErrInvalidAdministrativeUnitQuery)

		rec := doGet(newAdminUnitMux(uc), base+"?q=")

		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		body := decode(t, rec)
		assert.Equal(t, "INVALID_FILTER", body["error"].(map[string]any)["code"])
		assert.Equal(t, "q", body["details"].(map[string]any)["field"])
		assert.Empty(t, rec.Header().Get("Cache-Control"))
		uc.AssertNotCalled(t, "ListCurrentWards", mock.Anything, mock.Anything)
	})

	t.Run("no match is 200 with an empty array", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("SearchCurrentWards", mock.Anything, "79", "zzz").Return([]domain.CurrentWard{}, nil)

		rec := doGet(newAdminUnitMux(uc), base+"?q=zzz")

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, []any{}, decode(t, rec)["data"].(map[string]any)["items"])
	})

	t.Run("unknown province with a q is still 404", func(t *testing.T) {
		uc := &mockAdministrativeUnitUsecase{}
		uc.On("SearchCurrentWards", mock.Anything, "79", "tan").Return(nil, domain.ErrAdministrativeUnitNotFound)

		rec := doGet(newAdminUnitMux(uc), base+"?q=tan")

		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "ADMINISTRATIVE_UNIT_NOT_FOUND", decode(t, rec)["error"].(map[string]any)["code"])
	})
}
