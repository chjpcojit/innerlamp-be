package validation

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"local number with leading zero", "0901234567", "+84901234567"},
		{"already E.164", "+84912345678", "+84912345678"},
		{"foreign country code kept", "+14155552671", "+14155552671"},
		{"separators stripped, country code kept", "+81 90-1234-5678", "+819012345678"},
		{"separators anywhere", "090 123-4567", "+84901234567"},
		{"parens and bare national number", "(912) 345-678", "+84912345678"},
		{"surrounding whitespace", "  +84912345678  ", "+84912345678"},
		{"84 prefix without plus", "84912345678", "+84912345678"},
		{"bare national, no leading zero", "912345678", "+84912345678"},

		{"empty stays empty", "", ""},
		{"whitespace only", "   ", ""},
		{"letters", "abc", ""},
		{"plus with non-digits", "+84abc", ""},
		{"second plus rejected", "++84912345678", ""},
		{"trailing letter", "0901234567x", ""},
		{"too many digits", "0901234567890123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizePhone(tt.input); got != tt.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNormalizePhoneOutputAlwaysPassesPhoneRule pins the contract the caller
// relies on: a non-empty NormalizePhone result is always accepted by
// IsValidPhone, so the two layers never disagree (the caller validates the
// *normalized* value, and PhoneRule rejects "" on its own).
func TestNormalizePhoneOutputAlwaysPassesPhoneRule(t *testing.T) {
	inputs := []string{
		"0901234567",
		"+84912345678",
		"+14155552671",
		"+81 90-1234-5678",
		"090 123-4567",
		"(912) 345-678",
		"  +84912345678  ",
		"84912345678",
		"912345678",
	}

	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			got := NormalizePhone(in)
			if got == "" {
				t.Fatalf("NormalizePhone(%q) = \"\", want a normalized value", in)
			}
			if !IsValidPhone(got) {
				t.Errorf("NormalizePhone(%q) = %q, which IsValidPhone rejects", in, got)
			}
		})
	}
}
