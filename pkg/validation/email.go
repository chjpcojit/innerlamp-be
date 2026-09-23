package validation

import (
	"net/mail"
	"strings"
)

// IsValidEmail reports whether raw is usable as an account email address.
//
// It exists because NormalizeEmail deliberately does not check shape — it only
// lowercases — and the use case must reject a malformed address *before* it
// touches the database (US-001 AC7/AC8). The delivery layer's `email` struct
// tag is the outer gate; this is the inner one, so a caller that reaches the
// use case without passing through a handler is still checked. The two agree on
// every ordinary address; where they differ, this rule is the stricter of the
// pair (see the dotted-domain note below), so the use case never accepts what
// delivery rejected.
//
// Stricter than mail.ParseAddress alone in three ways:
//
//   - a display name or angle-addr wrapper ("Lamp <a@b.com>") is rejected: the
//     stored value must be the bare address the user typed, not a different
//     shape that merely parses to it;
//   - the domain must contain a dot, so "user@localhost" is not an account
//     address. This is the one place it diverges from the library's `email`
//     tag, which accepts a dotless domain;
//   - a leading, trailing or doubled dot in the domain is rejected.
//
// It checks shape only: no DNS lookup, no MX record, no plus-address policy.
// Callers pass the *normalized* value (NormalizeEmail) so the check is applied
// to the string that will actually be stored.
func IsValidEmail(raw string) bool {
	address := strings.TrimSpace(raw)
	if address == "" {
		return false
	}

	parsed, err := mail.ParseAddress(address)
	if err != nil {
		return false
	}
	// ParseAddress also accepts "Name <a@b.com>" and normalizes a bare
	// address, so equality with the input is what proves nothing was
	// rewritten. Anything else is rejected rather than stored in a shape the
	// user never typed.
	if parsed.Address != address {
		return false
	}

	at := strings.LastIndex(address, "@")
	if at < 0 {
		// Unreachable: ParseAddress rejects a string with no "@". Kept so the
		// slice below can never panic on a future mail.ParseAddress change.
		return false
	}

	domain := address[at+1:]
	switch {
	case !strings.Contains(domain, "."):
		return false
	case strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, "."):
		return false
	case strings.Contains(domain, ".."):
		return false
	}

	return true
}
