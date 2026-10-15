package domain

import "time"

// User is a registered account. It maps the users table.
//
// Plain data: no json/db tag. The wire shape belongs to delivery's response
// structs and the DB shape to repository's SQL — a tag here would force one
// JSON shape on every consumer and make the entity change whenever an
// unrelated API contract moves.
//
// EmailVerifiedAt is set to the account-creation time by registration, which
// verifies the email with a code (onboarding v8.0, AC14). It is nullable only
// for accounts created before migration 0037.
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
