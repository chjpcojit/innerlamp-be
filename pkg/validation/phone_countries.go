package validation

import "strings"

// PhoneCountry is one entry of the shared country list behind BR-ONB-04
// (docs/product/onboarding/product.md): the calling code a national number is
// prefixed with, and the exact number of national digits that country takes
// once one trunk 0 has been dropped.
type PhoneCountry struct {
	// CallingCode is the E.164 country calling code without the "+", e.g. "84".
	CallingCode string
	// Digits is the exact national-number length, after dropping one leading 0.
	Digits int
}

// PhoneCountries is the 12-country list shared by the backend, the website, iOS
// and Android (BR-ONB-04; from iOS ILCountry.swift), keyed by ISO 3166-1
// alpha-2 code. A country outside this map is refused, not guessed.
//
// Treat it as read-only; it is exported so a handler can list the supported
// codes, not so callers can add to it at runtime.
var PhoneCountries = map[string]PhoneCountry{
	"VN": {CallingCode: "84", Digits: 9},
	"US": {CallingCode: "1", Digits: 10},
	"JP": {CallingCode: "81", Digits: 10},
	"KR": {CallingCode: "82", Digits: 9},
	"TH": {CallingCode: "66", Digits: 9},
	"SG": {CallingCode: "65", Digits: 8},
	"MY": {CallingCode: "60", Digits: 9},
	"TW": {CallingCode: "886", Digits: 9},
	"AU": {CallingCode: "61", Digits: 9},
	"DE": {CallingCode: "49", Digits: 10},
	"FR": {CallingCode: "33", Digits: 9},
	"CA": {CallingCode: "1", Digits: 10},
}

// NormalizePhoneForCountry turns a national number typed with a country picked
// from PhoneCountries into E.164 (BR-ONB-04):
//
//   - spaces, dots and dashes are ignored; any other non-digit is invalid;
//   - exactly ONE leading 0 (the trunk prefix) is dropped — "00912345678" for
//     VN keeps its second 0 and is therefore one digit too long;
//   - the remaining digits must be exactly the country's length.
//
// ok is false (and e164 "") for an unknown country, an empty number, any other
// character, or the wrong digit count. No "+" or country code is accepted in
// typed: the country comes from the picker, never from the number field.
func NormalizePhoneForCountry(country, typed string) (e164 string, ok bool) {
	c, known := PhoneCountries[strings.ToUpper(strings.TrimSpace(country))]
	if !known {
		return "", false
	}

	var digits strings.Builder
	for i := 0; i < len(typed); i++ {
		switch ch := typed[i]; {
		case ch >= '0' && ch <= '9':
			digits.WriteByte(ch)
		case ch == ' ' || ch == '.' || ch == '-':
			// separator, ignored
		default:
			return "", false
		}
	}

	national := strings.TrimPrefix(digits.String(), "0")
	if len(national) != c.Digits {
		return "", false
	}
	return "+" + c.CallingCode + national, true
}
