package usecase

import (
	"errors"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Limits used for docs/product/onboarding/rules/names.json (BR-ONB-07). In the
// running service they come from ProfileConfig (profile.full_name_max_chars /
// profile.dharma_name_max_chars in config.<env>.yaml, 300 / 255); the vectors
// pin the product values, so the test pins them too.
const (
	rulesFullNameMaxChars   = 300
	rulesDharmaNameMaxChars = 255
)

// TestNameRules runs every vector of docs/product/onboarding/rules/names.json
// against the existing normalizeFullName / normalizeDharmaName helpers.
func TestNameRules(t *testing.T) {
	var cases []struct {
		Name  string `json:"name"`
		Input struct {
			Field string `json:"field"`
			Typed string `json:"typed"`
		} `json:"input"`
		Expected struct {
			Valid bool    `json:"valid"`
			Value *string `json:"value"`
		} `json:"expected"`
	}
	loadRuleCases(t, "names.json", &cases)
	if len(cases) == 0 {
		t.Fatal("names.json has no cases")
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var (
				got     *string
				err     error
				wantErr error
			)
			switch tc.Input.Field {
			case "full_name":
				var v string
				v, err = normalizeFullName(tc.Input.Typed, rulesFullNameMaxChars)
				if err == nil {
					got = &v
				}
				wantErr = domain.ErrInvalidFullName
			case "dharma_name":
				got, err = normalizeDharmaName(tc.Input.Typed, rulesDharmaNameMaxChars)
				wantErr = domain.ErrInvalidDharmaName
			default:
				t.Fatalf("unknown field %q", tc.Input.Field)
			}

			if !tc.Expected.Valid {
				if !errors.Is(err, wantErr) {
					t.Fatalf("err = %v, want %v", err, wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			switch {
			case tc.Expected.Value == nil && got != nil:
				t.Fatalf("value = %q, want absent", *got)
			case tc.Expected.Value != nil && (got == nil || *got != *tc.Expected.Value):
				t.Fatalf("value = %v, want %q", got, *tc.Expected.Value)
			}
		})
	}
}
