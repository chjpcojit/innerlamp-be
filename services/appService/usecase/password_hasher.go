package usecase

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/unicode/norm"
)

// NormalizePassword returns pw in Unicode NFC. Nothing is trimmed. Every
// password check (length rule, hashing, comparison) goes through this so a
// Vietnamese letter typed decomposed on one keyboard equals the precomposed
// form typed on another (BR-ONB-06).
func NormalizePassword(pw string) string {
	return norm.NFC.String(pw)
}

// ValidatePassword applies BR-ONB-06 to a password exactly as typed: NFC, then
// domain.ValidatePasswordLength (8–64 characters). It returns
// domain.ErrInvalidPassword when the length is out of range.
func ValidatePassword(pw string) error {
	return domain.ValidatePasswordLength(NormalizePassword(pw))
}

// BcryptPasswordHasher implements domain.PasswordHasher as
// bcrypt(base64(SHA-256(NFC(pw)))) at bcrypt.DefaultCost (design.md
// "Layers & components").
//
// The SHA-256 pre-hash exists because bcrypt reads only the first 72 bytes: a
// 64-character Vietnamese password can be well over 72 bytes, and without the
// pre-hash its last characters would not count. base64 keeps the digest free of
// NUL bytes, which some bcrypt implementations treat as a terminator; the
// 44-byte encoding is under the 72-byte limit.
//
// Shared by registration, login and password reset, so all three hash and
// compare the same way.
type BcryptPasswordHasher struct{}

var _ domain.PasswordHasher = BcryptPasswordHasher{}

// NewBcryptPasswordHasher returns the production password hasher.
func NewBcryptPasswordHasher() BcryptPasswordHasher { return BcryptPasswordHasher{} }

// prehash is base64(SHA-256(NFC(pw))).
func prehash(pw string) []byte {
	sum := sha256.Sum256([]byte(NormalizePassword(pw)))
	out := make([]byte, base64.StdEncoding.EncodedLen(len(sum)))
	base64.StdEncoding.Encode(out, sum[:])
	return out
}

// Hash returns the bcrypt hash of the pre-hashed password.
func (BcryptPasswordHasher) Hash(pw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(prehash(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// Compare returns nil when pw matches hash. A mismatch returns
// bcrypt.ErrMismatchedHashAndPassword; a malformed hash returns bcrypt's own
// error. Callers that must not reveal which (login) map any error to
// domain.ErrInvalidCredentials.
func (BcryptPasswordHasher) Compare(hash, pw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), prehash(pw))
}
