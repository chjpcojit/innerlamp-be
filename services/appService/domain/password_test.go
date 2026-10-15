package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestValidatePasswordLength(t *testing.T) {
	tests := []struct {
		name string
		pw   string
		ok   bool
	}{
		{"empty", "", false},
		{"7 chars", "abcdefg", false},
		{"8 chars", "abcdefgh", true},
		{"leading space counts", " abcdefg", true},
		{"7 precomposed Vietnamese chars (>8 bytes)", "mậtkhẩu", false},
		{"10 precomposed Vietnamese chars", "mậtkhẩuđẹp", true},
		{"64 chars", strings.Repeat("a", 64), true},
		{"65 chars", strings.Repeat("a", 65), false},
		{"64 accented chars (>64 bytes)", strings.Repeat("ẩ", 64), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordLength(tt.pw)
			if tt.ok && err != nil {
				t.Fatalf("ValidatePasswordLength(%q) = %v, want nil", tt.pw, err)
			}
			if !tt.ok && !errors.Is(err, ErrInvalidPassword) {
				t.Fatalf("ValidatePasswordLength(%q) = %v, want ErrInvalidPassword", tt.pw, err)
			}
		})
	}
}
