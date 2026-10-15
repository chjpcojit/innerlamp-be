package usecase

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"golang.org/x/crypto/bcrypt"
)

// TestValidatePasswordRules runs every vector of
// docs/product/onboarding/rules/password.json (BR-ONB-06 / BR-LGN-05).
func TestValidatePasswordRules(t *testing.T) {
	var cases []struct {
		Name  string `json:"name"`
		Input struct {
			Password string `json:"password"`
		} `json:"input"`
		Expected struct {
			Valid bool `json:"valid"`
		} `json:"expected"`
	}
	loadRuleCases(t, "password.json", &cases)
	if len(cases) == 0 {
		t.Fatal("password.json has no cases")
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			err := ValidatePassword(tc.Input.Password)
			if tc.Expected.Valid && err != nil {
				t.Fatalf("ValidatePassword(%q) = %v, want nil", tc.Input.Password, err)
			}
			if !tc.Expected.Valid && !errors.Is(err, domain.ErrInvalidPassword) {
				t.Fatalf("ValidatePassword(%q) = %v, want ErrInvalidPassword", tc.Input.Password, err)
			}
		})
	}
}

func TestBcryptPasswordHasher(t *testing.T) {
	h := NewBcryptPasswordHasher()

	t.Run("hash then compare round-trips", func(t *testing.T) {
		hash, err := h.Hash("abcdefgh")
		if err != nil {
			t.Fatal(err)
		}
		if err := h.Compare(hash, "abcdefgh"); err != nil {
			t.Fatalf("Compare = %v, want nil", err)
		}
		if err := h.Compare(hash, "abcdefgH"); !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			t.Fatalf("Compare(wrong) = %v, want ErrMismatchedHashAndPassword", err)
		}
	})

	t.Run("uses bcrypt.DefaultCost", func(t *testing.T) {
		hash, err := h.Hash("abcdefgh")
		if err != nil {
			t.Fatal(err)
		}
		cost, err := bcrypt.Cost([]byte(hash))
		if err != nil || cost != bcrypt.DefaultCost {
			t.Fatalf("cost = %d (%v), want %d", cost, err, bcrypt.DefaultCost)
		}
	})

	t.Run("last character of a 64-char accented password (>72 bytes) counts", func(t *testing.T) {
		pw := strings.Repeat("ẩ", 63) + "a"
		other := strings.Repeat("ẩ", 63) + "b"
		if utf8.RuneCountInString(pw) != 64 || len(pw) <= 72 {
			t.Fatalf("fixture: %d runes, %d bytes; want 64 runes and > 72 bytes", utf8.RuneCountInString(pw), len(pw))
		}
		if err := ValidatePassword(pw); err != nil {
			t.Fatalf("ValidatePassword(64 accented) = %v, want nil", err)
		}
		hash, err := h.Hash(pw)
		if err != nil {
			t.Fatal(err)
		}
		if err := h.Compare(hash, pw); err != nil {
			t.Fatalf("Compare(same) = %v, want nil", err)
		}
		if err := h.Compare(hash, other); err == nil {
			t.Fatal("Compare(differs only in the 64th character) = nil; the last character was ignored")
		}
	})

	t.Run("decomposed input matches precomposed hash", func(t *testing.T) {
		precomposed := "mật khẩu đẹp"     // NFC
		decomposed := "mật khẩu đẹp" // NFD-ish spelling
		if precomposed == decomposed {
			t.Fatal("fixture: spellings must differ byte-wise")
		}
		hash, err := h.Hash(precomposed)
		if err != nil {
			t.Fatal(err)
		}
		if err := h.Compare(hash, decomposed); err != nil {
			t.Fatalf("Compare(decomposed) = %v, want nil", err)
		}
	})

	t.Run("nothing is trimmed", func(t *testing.T) {
		hash, err := h.Hash(" abcdefg")
		if err != nil {
			t.Fatal(err)
		}
		if err := h.Compare(hash, "abcdefg"); err == nil {
			t.Fatal("Compare without the leading space = nil; spaces must count")
		}
	})
}
