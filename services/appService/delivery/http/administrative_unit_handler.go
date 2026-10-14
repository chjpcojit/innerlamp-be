package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// administrativeUnitUsecase is the subset of *usecase.AdministrativeUnitUsecase
// this handler depends on, declared here (the consumer) so the handler is tested
// with a mock.
type administrativeUnitUsecase interface {
	ListCurrentProvinces(ctx context.Context) ([]domain.CurrentProvince, error)
	ListCurrentWards(ctx context.Context, provinceCode string) ([]domain.CurrentWard, error)
	SearchCurrentWards(ctx context.Context, provinceCode, query string) ([]domain.CurrentWard, error)
	ListLegacyProvinces(ctx context.Context) ([]domain.LegacyProvince, error)
	ListLegacyDistricts(ctx context.Context, provinceCode string) ([]domain.LegacyDistrict, error)
	ListLegacyWards(ctx context.Context, districtCode string) ([]domain.LegacyWard, error)
	ConvertLegacyWard(ctx context.Context, wardCode string) (domain.Conversion[domain.CurrentWardCandidate], error)
	ConvertCurrentWard(ctx context.Context, wardCode string) (domain.Conversion[domain.LegacyWardCandidate], error)
	ConvertLegacyProvince(ctx context.Context, provinceCode string) (domain.Conversion[domain.CurrentProvince], error)
	ConvertCurrentProvince(ctx context.Context, provinceCode string) (domain.Conversion[domain.LegacyProvince], error)
}

// AdministrativeUnitHandler serves the read-only administrative-units lookups.
// The routes are public — the caller may be mid-registration with no account —
// and are put behind the web API key and a per-IP limit in cmd/main.go, not
// here. The two schemes are named in the path ("new" = the two-level scheme in
// force since 2025-07-01, "old" = the three-level scheme before it); inside the
// code the same two are called current and legacy.
type AdministrativeUnitHandler struct {
	usecase administrativeUnitUsecase
	// cacheControl is the Cache-Control value sent on every successful answer;
	// the data only changes when an operator loads a new dataset.
	cacheControl string
}

// NewAdministrativeUnitHandler wires the handler. cacheMaxAge is how long a
// client may reuse a successful answer (config: administrative_units.cache_max_age).
func NewAdministrativeUnitHandler(usecase administrativeUnitUsecase, cacheMaxAge time.Duration) *AdministrativeUnitHandler {
	return &AdministrativeUnitHandler{
		usecase:      usecase,
		cacheControl: "public, max-age=" + strconv.Itoa(int(cacheMaxAge/time.Second)),
	}
}

// ---- response shapes -------------------------------------------------------
// Plain DTOs: domain types carry no json tag, and sort_order is an ordering
// detail the client never needs.

// provinceResponse is a province-level unit of either scheme.
type provinceResponse struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Type     string `json:"type"`
}

// currentWardResponse is a ward-level unit of the new scheme.
type currentWardResponse struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Type         string `json:"type"`
	ProvinceCode string `json:"province_code"`
}

// legacyDistrictResponse is a district of the old scheme.
type legacyDistrictResponse struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Type         string `json:"type"`
	ProvinceCode string `json:"province_code"`
}

// legacyWardResponse is a ward-level unit of the old scheme.
type legacyWardResponse struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Type         string `json:"type"`
	DistrictCode string `json:"district_code"`
	ProvinceCode string `json:"province_code"`
}

// currentWardCandidateResponse is one possible new-scheme result of converting
// an old ward: the ward, its province and how the old ward relates to it.
type currentWardCandidateResponse struct {
	Ward        currentWardResponse `json:"ward"`
	Province    provinceResponse    `json:"province"`
	MappingType string              `json:"mapping_type"`
	Note        string              `json:"note"`
}

// legacyWardCandidateResponse is one possible old-scheme result of converting a
// new ward.
type legacyWardCandidateResponse struct {
	Ward        legacyWardResponse     `json:"ward"`
	District    legacyDistrictResponse `json:"district"`
	Province    provinceResponse       `json:"province"`
	MappingType string                 `json:"mapping_type"`
	Note        string                 `json:"note"`
}

type itemsResponse[T any] struct {
	Items []T `json:"items"`
}

// conversionResponse lists every candidate; unique says whether there is
// exactly one, so a client can skip the "which one?" step.
type conversionResponse[T any] struct {
	Unique     bool `json:"unique"`
	Candidates []T  `json:"candidates"`
}

func newProvince(code, name, fullName, typ string) provinceResponse {
	return provinceResponse{Code: code, Name: name, FullName: fullName, Type: typ}
}

func mapSlice[S, D any](in []S, f func(S) D) []D {
	out := make([]D, 0, len(in))
	for _, v := range in {
		out = append(out, f(v))
	}
	return out
}

func newCurrentWard(w domain.CurrentWard) currentWardResponse {
	return currentWardResponse{Code: w.Code, Name: w.Name, FullName: w.FullName, Type: w.Type, ProvinceCode: w.ProvinceCode}
}

func newLegacyDistrict(d domain.LegacyDistrict) legacyDistrictResponse {
	return legacyDistrictResponse{Code: d.Code, Name: d.Name, FullName: d.FullName, Type: d.Type, ProvinceCode: d.ProvinceCode}
}

func newLegacyWard(w domain.LegacyWard) legacyWardResponse {
	return legacyWardResponse{Code: w.Code, Name: w.Name, FullName: w.FullName, Type: w.Type,
		DistrictCode: w.DistrictCode, ProvinceCode: w.ProvinceCode}
}

func provinceFromCurrent(p domain.CurrentProvince) provinceResponse {
	return newProvince(p.Code, p.Name, p.FullName, p.Type)
}

func provinceFromLegacy(p domain.LegacyProvince) provinceResponse {
	return newProvince(p.Code, p.Name, p.FullName, p.Type)
}

// ok writes a successful answer with the cache header. The header is set only
// here, so an error response (404, 5xx) never carries it.
func (h *AdministrativeUnitHandler) ok(w http.ResponseWriter, data any) {
	w.Header().Set("Cache-Control", h.cacheControl)
	utils.WriteSuccessResponse(w, data)
}

// ---- pick-lists ------------------------------------------------------------

// CurrentProvinces serves GET /v1/administrative-units/new/provinces.
func (h *AdministrativeUnitHandler) CurrentProvinces(w http.ResponseWriter, r *http.Request) {
	items, err := h.usecase.ListCurrentProvinces(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	h.ok(w, itemsResponse[provinceResponse]{Items: mapSlice(items, provinceFromCurrent)})
}

// CurrentWards serves GET /v1/administrative-units/new/provinces/{code}/wards
// with an optional ?q=: when q is present the list is narrowed to the wards whose
// name contains it. q is passed through as present-or-absent, so an empty ?q= is
// the use case's ErrInvalidAdministrativeUnitQuery (400 INVALID_FILTER), never
// "no search" — the same rule as GET /v1/temples/options?q=.
func (h *AdministrativeUnitHandler) CurrentWards(w http.ResponseWriter, r *http.Request) {
	var (
		items []domain.CurrentWard
		err   error
	)
	if vals, ok := r.URL.Query()["q"]; ok && len(vals) > 0 {
		items, err = h.usecase.SearchCurrentWards(r.Context(), r.PathValue("code"), vals[0])
	} else {
		items, err = h.usecase.ListCurrentWards(r.Context(), r.PathValue("code"))
	}
	if err != nil {
		writeDomainError(w, err)
		return
	}
	h.ok(w, itemsResponse[currentWardResponse]{Items: mapSlice(items, newCurrentWard)})
}

// LegacyProvinces serves GET /v1/administrative-units/old/provinces.
func (h *AdministrativeUnitHandler) LegacyProvinces(w http.ResponseWriter, r *http.Request) {
	items, err := h.usecase.ListLegacyProvinces(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	h.ok(w, itemsResponse[provinceResponse]{Items: mapSlice(items, provinceFromLegacy)})
}

// LegacyDistricts serves GET /v1/administrative-units/old/provinces/{code}/districts.
func (h *AdministrativeUnitHandler) LegacyDistricts(w http.ResponseWriter, r *http.Request) {
	items, err := h.usecase.ListLegacyDistricts(r.Context(), r.PathValue("code"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	h.ok(w, itemsResponse[legacyDistrictResponse]{Items: mapSlice(items, newLegacyDistrict)})
}

// LegacyWards serves GET /v1/administrative-units/old/districts/{code}/wards. A
// district without wards answers 200 with an empty list.
func (h *AdministrativeUnitHandler) LegacyWards(w http.ResponseWriter, r *http.Request) {
	items, err := h.usecase.ListLegacyWards(r.Context(), r.PathValue("code"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	h.ok(w, itemsResponse[legacyWardResponse]{Items: mapSlice(items, newLegacyWard)})
}

// ---- conversions -----------------------------------------------------------

// ConvertLegacyWard serves GET /v1/administrative-units/old/wards/{code}/conversion.
func (h *AdministrativeUnitHandler) ConvertLegacyWard(w http.ResponseWriter, r *http.Request) {
	c, err := h.usecase.ConvertLegacyWard(r.Context(), r.PathValue("code"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	h.ok(w, conversionResponse[currentWardCandidateResponse]{
		Unique: c.Unique(),
		Candidates: mapSlice(c.Candidates, func(x domain.CurrentWardCandidate) currentWardCandidateResponse {
			return currentWardCandidateResponse{
				Ward:        newCurrentWard(x.Ward),
				Province:    provinceFromCurrent(x.Province),
				MappingType: x.MappingType,
				Note:        x.Note,
			}
		}),
	})
}

// ConvertCurrentWard serves GET /v1/administrative-units/new/wards/{code}/conversion.
func (h *AdministrativeUnitHandler) ConvertCurrentWard(w http.ResponseWriter, r *http.Request) {
	c, err := h.usecase.ConvertCurrentWard(r.Context(), r.PathValue("code"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	h.ok(w, conversionResponse[legacyWardCandidateResponse]{
		Unique: c.Unique(),
		Candidates: mapSlice(c.Candidates, func(x domain.LegacyWardCandidate) legacyWardCandidateResponse {
			return legacyWardCandidateResponse{
				Ward:        newLegacyWard(x.Ward),
				District:    newLegacyDistrict(x.District),
				Province:    provinceFromLegacy(x.Province),
				MappingType: x.MappingType,
				Note:        x.Note,
			}
		}),
	})
}

// ConvertLegacyProvince serves GET /v1/administrative-units/old/provinces/{code}/conversion.
func (h *AdministrativeUnitHandler) ConvertLegacyProvince(w http.ResponseWriter, r *http.Request) {
	c, err := h.usecase.ConvertLegacyProvince(r.Context(), r.PathValue("code"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	h.ok(w, conversionResponse[provinceResponse]{
		Unique:     c.Unique(),
		Candidates: mapSlice(c.Candidates, provinceFromCurrent),
	})
}

// ConvertCurrentProvince serves GET /v1/administrative-units/new/provinces/{code}/conversion.
func (h *AdministrativeUnitHandler) ConvertCurrentProvince(w http.ResponseWriter, r *http.Request) {
	c, err := h.usecase.ConvertCurrentProvince(r.Context(), r.PathValue("code"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	h.ok(w, conversionResponse[provinceResponse]{
		Unique:     c.Unique(),
		Candidates: mapSlice(c.Candidates, provinceFromLegacy),
	})
}
