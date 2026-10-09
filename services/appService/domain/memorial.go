package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Relationship is how the deceased relates to the memorial's owner. The set is
// fixed by the memorials.relationship CHECK, so these constants are schema, not
// configuration.
type Relationship string

const (
	RelationshipGrandfatherPaternal Relationship = "GRANDFATHER_PATERNAL" // ông nội
	RelationshipGrandfatherMaternal Relationship = "GRANDFATHER_MATERNAL" // ông ngoại
	RelationshipGrandmotherPaternal Relationship = "GRANDMOTHER_PATERNAL" // bà nội
	RelationshipGrandmotherMaternal Relationship = "GRANDMOTHER_MATERNAL" // bà ngoại
	RelationshipFather              Relationship = "FATHER"               // cha
	RelationshipMother              Relationship = "MOTHER"               // mẹ
	RelationshipSpouse              Relationship = "SPOUSE"               // vợ/chồng
	RelationshipSibling             Relationship = "SIBLING"              // anh chị em
	RelationshipChild               Relationship = "CHILD"                // con
	RelationshipOther               Relationship = "OTHER"                // khác
)

// Valid reports whether r is one of the ten allowed relationships.
func (r Relationship) Valid() bool {
	switch r {
	case RelationshipGrandfatherPaternal, RelationshipGrandfatherMaternal,
		RelationshipGrandmotherPaternal, RelationshipGrandmotherMaternal,
		RelationshipFather, RelationshipMother, RelationshipSpouse,
		RelationshipSibling, RelationshipChild, RelationshipOther:
		return true
	}
	return false
}

// DeathCalendar is the calendar the user typed the death date in. The server
// stores solar dates only when the year is known; the calendar is kept so the
// client can show the date back the way it was entered.
type DeathCalendar string

const (
	DeathCalendarSolar DeathCalendar = "solar"
	DeathCalendarLunar DeathCalendar = "lunar"
)

// Reminder is the reminder setting the app turns into a local notification.
// The server never fires it.
type Reminder struct {
	// DaysBefore is how many days before the anniversary to remind, 0..30.
	DaysBefore int
	// Time is the wall-clock time "HH:mm" in Timezone.
	Time string
	// Timezone is an IANA zone name.
	Timezone string
}

// Memorial is one person in a user's death-anniversary book. It maps the
// memorials table.
//
// Plain data: no json/db tags — the wire shape is delivery's, the SQL shape is
// repository's.
//
// The death date has two mutually exclusive forms:
//   - year known:   DeathDate is the solar date; DeathDay/DeathMonth/DeathLeapMonth are nil.
//   - year unknown: DeathDate is nil; DeathDay/DeathMonth (and DeathLeapMonth) are in DeathCalendar.
type Memorial struct {
	ID     int64
	UserID int64

	FullName      string
	Relationship  Relationship
	DeathCalendar DeathCalendar

	DeathDate      *time.Time
	DeathDay       *int
	DeathMonth     *int
	DeathLeapMonth *bool

	// Note is nil when there is none.
	Note *string
	// PhotoObjectKey is nil when there is no photo. Always nil until the photo
	// feature (tasks.md Phase 6) exists.
	PhotoObjectKey *string

	// Reminder is nil when the user set no reminder.
	Reminder *Reminder
	// AddToOSReminder records the user's intent to mirror the reminder into the
	// OS reminders. It is not the real state on any device.
	AddToOSReminder bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// MemorialInput is what the user submits to create or replace a memorial: the
// raw request, before validation. Optional parts are pointers/empty strings so
// the use case can tell "absent" from "zero".
type MemorialInput struct {
	FullName      string
	Relationship  string
	DeathCalendar string

	// DeathDate is "YYYY-MM-DD" (solar) or "" when the year is unknown.
	DeathDate      string
	DeathDay       *int
	DeathMonth     *int
	DeathLeapMonth *bool

	Note *string

	Reminder        *ReminderInput
	AddToOSReminder bool
}

// ReminderInput is the raw reminder object. DaysBefore is a pointer so a
// missing value is distinguishable from 0.
type ReminderInput struct {
	DaysBefore *int
	Time       string
	// Timezone is optional; the use case defaults it.
	Timezone string
}

var (
	// ErrMemorialNotFound — no memorial with this id belongs to this user. The
	// same error covers "does not exist" and "belongs to someone else", so a
	// caller cannot probe which ids exist.
	ErrMemorialNotFound = errors.New("memorial not found")

	// ErrMemorialLimitReached — the user already owns the maximum number of memorials.
	ErrMemorialLimitReached = errors.New("memorial limit reached")

	// ErrMemorialInvalid — a submitted field is not allowed. Use
	// MemorialValidationError to say which one; errors.Is still matches this.
	ErrMemorialInvalid = errors.New("invalid memorial")
)

// MemorialValidationError names the request field that was rejected and why.
type MemorialValidationError struct {
	// Field is the request field path, e.g. "full_name" or "reminder.time".
	Field   string
	Message string
}

func (e *MemorialValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Unwrap lets errors.Is(err, ErrMemorialInvalid) match.
func (e *MemorialValidationError) Unwrap() error { return ErrMemorialInvalid }

// MemorialRepository is the port the memorial use case stores through.
//
// Every single-row method takes the owner's userID as well as the id, and the
// repository must filter on both: ownership is enforced where the data is read,
// not only above it. A row that exists but belongs to another user is reported
// as ErrMemorialNotFound.
type MemorialRepository interface {
	// Create inserts m (ID, CreatedAt and UpdatedAt are ignored and filled in)
	// unless the user already owns max memorials, atomically — concurrent calls
	// for one user can never exceed max. Returns ErrMemorialLimitReached then.
	Create(ctx context.Context, m *Memorial, max int) (*Memorial, error)

	// ListByUser returns the user's memorials by created_at DESC, id DESC.
	ListByUser(ctx context.Context, userID int64) ([]Memorial, error)

	// FindByIDAndUser returns the memorial or ErrMemorialNotFound.
	FindByIDAndUser(ctx context.Context, id, userID int64) (*Memorial, error)

	// Update replaces the editable fields of m (looked up by m.ID and m.UserID),
	// refreshes UpdatedAt and returns the stored row, or ErrMemorialNotFound.
	// It never touches the owner, created_at or the photo.
	Update(ctx context.Context, m *Memorial) (*Memorial, error)

	// Delete removes the memorial or returns ErrMemorialNotFound.
	Delete(ctx context.Context, id, userID int64) error
}
