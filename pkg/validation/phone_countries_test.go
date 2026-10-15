package validation

import "testing"

// TestNormalizePhoneForCountryRules runs every vector of
// docs/product/onboarding/rules/phone-number.json (BR-ONB-04).
func TestNormalizePhoneForCountryRules(t *testing.T) {
	var cases []struct {
		Name  string `json:"name"`
		Input struct {
			Country string `json:"country"`
			Typed   string `json:"typed"`
		} `json:"input"`
		Expected struct {
			Valid  bool    `json:"valid"`
			Number *string `json:"number"`
		} `json:"expected"`
	}
	loadRuleCases(t, "phone-number.json", &cases)
	if len(cases) == 0 {
		t.Fatal("phone-number.json has no cases")
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			got, ok := NormalizePhoneForCountry(tc.Input.Country, tc.Input.Typed)
			if ok != tc.Expected.Valid {
				t.Fatalf("NormalizePhoneForCountry(%q, %q) ok = %v, want %v", tc.Input.Country, tc.Input.Typed, ok, tc.Expected.Valid)
			}
			want := ""
			if tc.Expected.Number != nil {
				want = *tc.Expected.Number
			}
			if got != want {
				t.Fatalf("NormalizePhoneForCountry(%q, %q) = %q, want %q", tc.Input.Country, tc.Input.Typed, got, want)
			}
		})
	}
}

func TestPhoneCountriesHasTwelveEntries(t *testing.T) {
	if got := len(PhoneCountries); got != 12 {
		t.Fatalf("len(PhoneCountries) = %d, want 12 (BR-ONB-04)", got)
	}
}

func TestNormalizePhoneForCountryRejectsPlusAndCountryCode(t *testing.T) {
	for _, typed := range []string{"+84912345678", "(091) 234 5678"} {
		if got, ok := NormalizePhoneForCountry("VN", typed); ok {
			t.Errorf("NormalizePhoneForCountry(VN, %q) = %q, true; want invalid", typed, got)
		}
	}
}

func TestNormalizePhoneForCountryCodeIsCaseInsensitive(t *testing.T) {
	if got, ok := NormalizePhoneForCountry("vn", "0912345678"); !ok || got != "+84912345678" {
		t.Fatalf("NormalizePhoneForCountry(vn, ...) = %q, %v", got, ok)
	}
}
