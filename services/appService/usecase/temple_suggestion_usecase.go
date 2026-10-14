package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// TempleSuggestionUsecase accepts a temple suggestion from an app user. Review
// (listing, converting to a temple, rejecting) lives in backofficeService.
type TempleSuggestionUsecase struct {
	suggestions domain.TempleSuggestionRepository
	wards       domain.WardResolver
}

// NewTempleSuggestionUsecase wires the use case. wards is usually the
// administrative-units repository.
func NewTempleSuggestionUsecase(suggestions domain.TempleSuggestionRepository, wards domain.WardResolver) *TempleSuggestionUsecase {
	return &TempleSuggestionUsecase{suggestions: suggestions, wards: wards}
}

// Create validates the submitted values and stores them as a pending
// suggestion. name and address are trimmed and otherwise kept exactly as typed,
// so staff see what the user sent. The province and ward arrive as current-scheme
// codes: they are checked against the administrative-units dataset (the ward
// must belong to the province) and their names are copied from it, never taken
// from the client. It deliberately does not look for duplicates: that would tell
// an anonymous caller what the catalogue and queue contain.
func (u *TempleSuggestionUsecase) Create(ctx context.Context, in domain.TempleSuggestionInput) (*domain.TempleSuggestion, error) {
	s := &domain.TempleSuggestion{
		Name:         strings.TrimSpace(in.Name),
		ProvinceCode: strings.TrimSpace(in.ProvinceCode),
		WardCode:     strings.TrimSpace(in.WardCode),
		Address:      strings.TrimSpace(in.Address),
	}
	// Order is part of the contract: details.field names the first offender.
	for _, f := range []struct {
		field, value string
		max          int // 0 = no length limit (codes are checked by lookup)
	}{
		{"name", s.Name, domain.TempleSuggestionNameMax},
		{"province_code", s.ProvinceCode, 0},
		{"ward_code", s.WardCode, 0},
		{"address", s.Address, domain.TempleSuggestionAddressMax},
	} {
		if f.value == "" {
			return nil, &domain.TempleSuggestionValidationError{Field: f.field, Message: f.field + " is required"}
		}
		if f.max > 0 && utf8.RuneCountInString(f.value) > f.max {
			return nil, &domain.TempleSuggestionValidationError{
				Field: f.field, Message: fmt.Sprintf("%s must be at most %d characters", f.field, f.max),
			}
		}
	}

	resolved, err := u.wards.ResolveCurrentWard(ctx, s.ProvinceCode, s.WardCode)
	switch {
	case errors.Is(err, domain.ErrAdministrativeUnitNotFound):
		return nil, &domain.TempleSuggestionValidationError{
			Field: "province_code", Message: "province_code is not a province of the new scheme",
		}
	case errors.Is(err, domain.ErrWardNotInProvince):
		return nil, &domain.TempleSuggestionValidationError{
			Field: "ward_code", Message: "ward_code is not a ward of this province",
		}
	case err != nil:
		return nil, fmt.Errorf("resolve ward for temple suggestion: %w", err)
	}
	s.Province = resolved.ProvinceName
	s.Ward = resolved.WardFullName

	created, err := u.suggestions.Create(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("create temple suggestion: %w", err)
	}
	return created, nil
}
