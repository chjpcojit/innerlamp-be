package validation

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

// PhoneTag is the struct-tag name bound to PhoneRule. Register it once at the
// composition root before validating anything:
//
//	if err := validate.Register(validation.PhoneTag, validation.PhoneRule); err != nil {
//		log.Fatalf("failed to register %s validation: %v", validation.PhoneTag, err)
//	}
//
// Then use it as a struct tag:
//
//	PhoneNumber string `json:"phone_number" validate:"required,phone"`
const PhoneTag = "phone"

const (
	phoneMinDigits = 8
	phoneMaxDigits = 15
)

// PhoneRule is the rule behind PhoneTag. It is exported so services can
// register it themselves; nothing is registered implicitly by New.
//
// An empty string is NOT valid, so pair the tag with required when the field
// is mandatory. Nil pointers never reach this function — the library skips
// them by default — so "missing value" is expressed with required/omitempty,
// not inside the rule.
func PhoneRule(fl validator.FieldLevel) bool {
	return IsValidPhone(fl.Field().String())
}

// IsValidPhone reports whether raw looks like a phone number: an optional
// leading "+", then 8-15 ASCII digits. Separators people type by hand — space,
// "-", ".", "(", ")" — are skipped anywhere. No country is assumed: this
// checks the shape of a number, not whether the number exists.
//
// Valid: "+84912345678", "+84 912 345 678", "(912) 345-678".
//
// The library's built-in e164 tag only checks the general shape; phone is
// stricter in that it rejects letters, a second "+", and control characters.
func IsValidPhone(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return false
	}

	if s[0] == '+' {
		s = s[1:]
	}

	digits := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			digits++
		case c == ' ' || c == '-' || c == '.' || c == '(' || c == ')':
			// separator, ignored
		default:
			return false
		}
	}

	return digits >= phoneMinDigits && digits <= phoneMaxDigits
}
