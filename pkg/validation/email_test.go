package validation

import "testing"

func TestIsValidEmail(t *testing.T) {
	valid := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.co.uk",
		"u_ser-1@sub.example.com",
		"USER@EXAMPLE.COM",
		"  user@example.com  ",
		"user@example.museum",
	}

	for _, email := range valid {
		t.Run("valid/"+email, func(t *testing.T) {
			if !IsValidEmail(email) {
				t.Errorf("IsValidEmail(%q) = false, want true", email)
			}
		})
	}

	invalid := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"no at sign", "not-an-email"},
		{"no domain", "user@"},
		{"no local part", "@example.com"},
		{"no dot in domain", "user@localhost"},
		{"trailing dot in domain", "user@example.com."},
		{"leading dot in domain", "user@.example.com"},
		{"doubled dot in domain", "user@example..com"},
		{"display name wrapper", "Lamp <user@example.com>"},
		{"angle addr only", "<user@example.com>"},
		{"spaces inside", "user name@example.com"},
		{"two at signs", "user@@example.com"},
		{"scheme prefix", "mailto:user@example.com"},
	}

	for _, tt := range invalid {
		t.Run("invalid/"+tt.name, func(t *testing.T) {
			if IsValidEmail(tt.email) {
				t.Errorf("IsValidEmail(%q) = true, want false", tt.email)
			}
		})
	}
}

// TestIsValidEmailAcceptsNormalizedOutput pins the contract the use case relies
// on: it normalizes first and validates second, so anything NormalizeEmail
// produces from a well-formed address must survive the check. Without this, a
// lowercasing step could quietly turn a valid address into a rejected one.
func TestIsValidEmailAcceptsNormalizedOutput(t *testing.T) {
	raw := []string{
		"  User@Example.com  ",
		"USER.NAME+TAG@EXAMPLE.COM",
		"\tuser@example.com\n",
	}

	for _, input := range raw {
		t.Run(input, func(t *testing.T) {
			normalized := NormalizeEmail(input)
			if !IsValidEmail(normalized) {
				t.Errorf("IsValidEmail(NormalizeEmail(%q)) = false for %q, want true", input, normalized)
			}
		})
	}
}
