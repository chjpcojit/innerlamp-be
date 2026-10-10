package usecase

import (
	"context"
	"fmt"
	"strings"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// AdministrativeUnitUsecase serves the read-only administrative-units lookups:
// pick-lists for the current (two-level) and legacy (three-level) schemes and
// the conversion between them.
//
// It adds three guarantees on top of the repository: a blank code is a
// "not found" without touching the database, a list is never nil (so it
// serialises as [] not null), and a conversion lists every candidate — this
// layer never picks one for the caller. Ordering lives in the repository
// contract (domain.AdministrativeUnitRepository).
type AdministrativeUnitUsecase struct {
	units domain.AdministrativeUnitRepository
}

// NewAdministrativeUnitUsecase wires the use case. It has no configuration:
// lists are not paged and the rate limit lives in delivery.
func NewAdministrativeUnitUsecase(units domain.AdministrativeUnitRepository) *AdministrativeUnitUsecase {
	return &AdministrativeUnitUsecase{units: units}
}

// ListCurrentProvinces returns every current-scheme province.
func (u *AdministrativeUnitUsecase) ListCurrentProvinces(ctx context.Context) ([]domain.CurrentProvince, error) {
	items, err := u.units.ListCurrentProvinces(ctx)
	if err != nil {
		return nil, fmt.Errorf("list current provinces: %w", err)
	}
	return nonNil(items), nil
}

// ListCurrentWards returns every ward of a current-scheme province, or
// domain.ErrAdministrativeUnitNotFound.
func (u *AdministrativeUnitUsecase) ListCurrentWards(ctx context.Context, provinceCode string) ([]domain.CurrentWard, error) {
	code, err := normalizeUnitCode(provinceCode)
	if err != nil {
		return nil, err
	}
	items, err := u.units.ListCurrentWards(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("list current wards of province %q: %w", code, err)
	}
	return nonNil(items), nil
}

// ListLegacyProvinces returns every legacy-scheme province.
func (u *AdministrativeUnitUsecase) ListLegacyProvinces(ctx context.Context) ([]domain.LegacyProvince, error) {
	items, err := u.units.ListLegacyProvinces(ctx)
	if err != nil {
		return nil, fmt.Errorf("list legacy provinces: %w", err)
	}
	return nonNil(items), nil
}

// ListLegacyDistricts returns every district of a legacy-scheme province, or
// domain.ErrAdministrativeUnitNotFound.
func (u *AdministrativeUnitUsecase) ListLegacyDistricts(ctx context.Context, provinceCode string) ([]domain.LegacyDistrict, error) {
	code, err := normalizeUnitCode(provinceCode)
	if err != nil {
		return nil, err
	}
	items, err := u.units.ListLegacyDistricts(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("list legacy districts of province %q: %w", code, err)
	}
	return nonNil(items), nil
}

// ListLegacyWards returns every ward of a legacy-scheme district. A district
// that exists but has no wards (an island district) is an empty list, not an
// error; domain.ErrAdministrativeUnitNotFound means the district is unknown.
func (u *AdministrativeUnitUsecase) ListLegacyWards(ctx context.Context, districtCode string) ([]domain.LegacyWard, error) {
	code, err := normalizeUnitCode(districtCode)
	if err != nil {
		return nil, err
	}
	items, err := u.units.ListLegacyWards(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("list legacy wards of district %q: %w", code, err)
	}
	return nonNil(items), nil
}

// ConvertLegacyWard returns every current ward a legacy ward maps to.
func (u *AdministrativeUnitUsecase) ConvertLegacyWard(ctx context.Context, wardCode string) (domain.Conversion[domain.CurrentWardCandidate], error) {
	code, err := normalizeUnitCode(wardCode)
	if err != nil {
		return domain.Conversion[domain.CurrentWardCandidate]{}, err
	}
	cands, err := u.units.ConvertLegacyWard(ctx, code)
	if err != nil {
		return domain.Conversion[domain.CurrentWardCandidate]{}, fmt.Errorf("convert legacy ward %q: %w", code, err)
	}
	return domain.Conversion[domain.CurrentWardCandidate]{Candidates: nonNil(cands)}, nil
}

// ConvertCurrentWard returns every legacy ward that maps to a current ward. The
// candidate list is empty for the special zones that replaced whole island
// districts (they are the target of a district mapping, which is not served).
func (u *AdministrativeUnitUsecase) ConvertCurrentWard(ctx context.Context, wardCode string) (domain.Conversion[domain.LegacyWardCandidate], error) {
	code, err := normalizeUnitCode(wardCode)
	if err != nil {
		return domain.Conversion[domain.LegacyWardCandidate]{}, err
	}
	cands, err := u.units.ConvertCurrentWard(ctx, code)
	if err != nil {
		return domain.Conversion[domain.LegacyWardCandidate]{}, fmt.Errorf("convert current ward %q: %w", code, err)
	}
	return domain.Conversion[domain.LegacyWardCandidate]{Candidates: nonNil(cands)}, nil
}

// ConvertLegacyProvince returns the current province(s) a legacy province's
// wards map into — exactly one for a verified dataset.
func (u *AdministrativeUnitUsecase) ConvertLegacyProvince(ctx context.Context, provinceCode string) (domain.Conversion[domain.CurrentProvince], error) {
	code, err := normalizeUnitCode(provinceCode)
	if err != nil {
		return domain.Conversion[domain.CurrentProvince]{}, err
	}
	cands, err := u.units.ConvertLegacyProvince(ctx, code)
	if err != nil {
		return domain.Conversion[domain.CurrentProvince]{}, fmt.Errorf("convert legacy province %q: %w", code, err)
	}
	return domain.Conversion[domain.CurrentProvince]{Candidates: nonNil(cands)}, nil
}

// ConvertCurrentProvince returns every legacy province merged into a current
// province (several for a province such as Hồ Chí Minh).
func (u *AdministrativeUnitUsecase) ConvertCurrentProvince(ctx context.Context, provinceCode string) (domain.Conversion[domain.LegacyProvince], error) {
	code, err := normalizeUnitCode(provinceCode)
	if err != nil {
		return domain.Conversion[domain.LegacyProvince]{}, err
	}
	cands, err := u.units.ConvertCurrentProvince(ctx, code)
	if err != nil {
		return domain.Conversion[domain.LegacyProvince]{}, fmt.Errorf("convert current province %q: %w", code, err)
	}
	return domain.Conversion[domain.LegacyProvince]{Candidates: nonNil(cands)}, nil
}

// normalizeUnitCode trims a unit code and rejects a blank one as "not found":
// no unit has an empty code, so there is nothing to look up.
func normalizeUnitCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", domain.ErrAdministrativeUnitNotFound
	}
	return code, nil
}

// nonNil turns a nil slice into an empty one so a list serialises as [].
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
