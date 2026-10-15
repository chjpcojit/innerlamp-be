package domain

import "unicode/utf8"

// Password length bounds from BR-ONB-06 (docs/product/onboarding/product.md),
// shared with login's new-password rule BR-LGN-05. They are product rules that
// every platform enforces identically, not deployment settings, so they are
// constants here rather than config.
const (
	PasswordMinChars = 8
	PasswordMaxChars = 64
)

// ValidatePasswordLength reports ErrInvalidPassword unless pw has between
// PasswordMinChars and PasswordMaxChars characters (runes). Nothing is trimmed:
// a leading or trailing space is part of the password and counts.
//
// The caller must pass the NFC-normalised password. BR-ONB-06 counts
// characters after NFC so a Vietnamese letter typed decomposed is one
// character, but NFC needs golang.org/x/text and domain is stdlib-only, so the
// normalisation lives in the usecase layer (usecase.ValidatePassword /
// usecase.NormalizePassword) and this function only counts.
func ValidatePasswordLength(pw string) error {
	n := utf8.RuneCountInString(pw)
	if n < PasswordMinChars || n > PasswordMaxChars {
		return ErrInvalidPassword
	}
	return nil
}

// PasswordHasher turns a password into a storable hash and checks a password
// against one. Implementations normalise the password the same way in Hash and
// Compare, so a password typed decomposed on one device matches the hash made
// from the precomposed form on another.
type PasswordHasher interface {
	// Hash returns the storable hash of pw.
	Hash(pw string) (string, error)
	// Compare returns nil when pw matches hash, and a non-nil error otherwise
	// (a mismatch and a malformed hash are both non-nil).
	Compare(hash, pw string) error
}
