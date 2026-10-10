package usecase

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// TempleSuggestionUsecase accepts a temple suggestion from an app user. Review
// (listing, converting to a temple, rejecting) lives in backofficeService.
type TempleSuggestionUsecase struct {
	suggestions domain.TempleSuggestionRepository
}

// NewTempleSuggestionUsecase wires the use case.
func NewTempleSuggestionUsecase(suggestions domain.TempleSuggestionRepository) *TempleSuggestionUsecase {
	return &TempleSuggestionUsecase{suggestions: suggestions}
}

// Create validates the three submitted values and stores them as a pending
// suggestion. Values are trimmed and otherwise kept exactly as typed, so staff
// see what the user sent. It deliberately does not look for duplicates: that
// would tell an anonymous caller what the catalogue and queue contain.
func (u *TempleSuggestionUsecase) Create(ctx context.Context, in domain.TempleSuggestionInput) (*domain.TempleSuggestion, error) {
	s := &domain.TempleSuggestion{
		Name:     strings.TrimSpace(in.Name),
		Province: strings.TrimSpace(in.Province),
		Address:  strings.TrimSpace(in.Address),
	}
	for _, f := range []struct {
		field, value string
		max          int
	}{
		{"name", s.Name, domain.TempleSuggestionNameMax},
		{"province", s.Province, domain.TempleSuggestionProvinceMax},
		{"address", s.Address, domain.TempleSuggestionAddressMax},
	} {
		if f.value == "" {
			return nil, &domain.TempleSuggestionValidationError{Field: f.field, Message: f.field + " is required"}
		}
		if utf8.RuneCountInString(f.value) > f.max {
			return nil, &domain.TempleSuggestionValidationError{
				Field: f.field, Message: fmt.Sprintf("%s must be at most %d characters", f.field, f.max),
			}
		}
	}

	created, err := u.suggestions.Create(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("create temple suggestion: %w", err)
	}
	return created, nil
}
