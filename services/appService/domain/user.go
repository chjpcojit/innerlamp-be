package domain

import "time"

// User is a registered account. It maps the users table.
//
// Plain data: no json/db tag. The wire shape belongs to delivery's response
// structs and the DB shape to repository's SQL — a tag here would force one
// JSON shape on every consumer and make the entity change whenever an
// unrelated API contract moves.
//
// EmailVerifiedAt is deliberately nullable and stays NULL at registration:
// US-001 verifies the *phone number* by OTP, not the email, so the column
// records an email-verification step that does not exist yet. Nothing in this
// feature depends on it being set.
type User struct {
	ID              int64
	Email           string
	EmailVerifiedAt *time.Time
	PhoneNumber     string
	PasswordHash    string
	// FullName is the account holder's full name; nil only for accounts created
	// before registration required it (migration 0033).
	FullName   *string
	DharmaName *string
	// OrdinationTempleID is temples.id of the temple where the user took
	// refuge or was ordained; nil when they have not chosen one (v7.0 —
	// replaces the free-text ordination_place).
	OrdinationTempleID *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
