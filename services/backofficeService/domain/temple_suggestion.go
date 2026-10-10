package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Suggestion states. The set is fixed by chk_temple_suggestions_status and is the
// same one appService writes 'pending' into.
const (
	TempleSuggestionPending  = "pending"
	TempleSuggestionCreated  = "created"
	TempleSuggestionRejected = "rejected"
)

// ValidSuggestionStatus reports whether s is a state a queue listing may filter by.
func ValidSuggestionStatus(s string) bool {
	switch s {
	case TempleSuggestionPending, TempleSuggestionCreated, TempleSuggestionRejected:
		return true
	}
	return false
}

// TempleSuggestion is a temple an app user proposed, as staff see it: what was
// submitted plus the review outcome. It maps the temple_suggestions table.
//
// Plain data: no json/db tag — the wire shape is delivery's, the SQL shape is
// repository's.
type TempleSuggestion struct {
	ID        int64
	Name      string
	Province  string
	Address   string
	Status    string
	CreatedAt time.Time

	// Review outcome; all nil/empty while pending.
	TempleID     *string
	ReviewedBy   *int64
	ReviewedAt   *time.Time
	RejectReason *string
}

// Temple is an official catalogue row. It maps the temples table that
// appService reads; this service writes it only through a suggestion.
type Temple struct {
	ID               string
	Name             string
	Type             string
	Region           string
	ImageURL         string
	ThumbnailURL     string
	Aliases          []string
	Sect             string
	Founder          string
	CurrentAbbot     string
	EstablishedYear  int
	HeritageRank     string
	HeritagePriority int
	Latitude         float64
	Longitude        float64
	Address          string
	OperatingHours   string
	Amenities        TempleAmenities

	Introduction         string
	ArchitectureHeritage string
	VisitingNotes        []string
}

// TempleAmenities are the yes/no facilities a visitor plans around.
type TempleAmenities struct {
	ParkingCar        bool
	ParkingMotorcycle bool
	VegetarianMeals   bool
	SummerRetreat     bool
}

// NewTemple is what staff submit to turn a suggestion into a temple. The ID and
// HeritagePriority are never supplied: the repository generates the id and the
// use case derives the priority from HeritageRank.
type NewTemple struct {
	Name            string
	Type            string
	Region          string
	ImageURL        string
	ThumbnailURL    string
	Aliases         []string
	Sect            string
	Founder         string
	CurrentAbbot    string
	EstablishedYear int
	HeritageRank    string
	Latitude        float64
	Longitude       float64
	Address         string
	OperatingHours  string
	Amenities       TempleAmenities

	Introduction         string
	ArchitectureHeritage string
	VisitingNotes        []string
}

// SuggestionPage is one page of the review queue.
type SuggestionPage struct {
	Items      []TempleSuggestion
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// CreatedTemple is the outcome of converting a suggestion: the updated
// suggestion and the temple that now exists.
type CreatedTemple struct {
	Suggestion TempleSuggestion
	Temple     Temple
}

var (
	// ErrSuggestionNotFound — no suggestion has the requested id.
	ErrSuggestionNotFound = errors.New("temple suggestion not found")

	// ErrSuggestionAlreadyCreated — the suggestion was already turned into a temple.
	ErrSuggestionAlreadyCreated = errors.New("temple suggestion already created")

	// ErrSuggestionAlreadyHandled — the suggestion is no longer pending (reject).
	ErrSuggestionAlreadyHandled = errors.New("temple suggestion already handled")

	// ErrReopenForbidden — only an admin may turn a rejected suggestion into a temple.
	ErrReopenForbidden = errors.New("only an admin may create a temple from a rejected suggestion")

	// ErrInvalidStatus — the status filter is not a known suggestion state.
	ErrInvalidStatus = errors.New("status must be one of pending, created, rejected")

	// ErrInvalidPage — the requested page number is not a positive integer.
	ErrInvalidPage = errors.New("page must be a positive integer")

	// ErrTempleInputInvalid — the temple form has a missing or disallowed value.
	// Use TempleInputError to say which one; errors.Is still matches this.
	ErrTempleInputInvalid = errors.New("invalid temple input")
)

// TempleInputError names the temple-form field that was rejected and why.
type TempleInputError struct {
	// Field is the request field, e.g. "heritage_rank" or "latitude".
	Field   string
	Message string
}

func (e *TempleInputError) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Message) }

// Unwrap lets errors.Is(err, ErrTempleInputInvalid) match.
func (e *TempleInputError) Unwrap() error { return ErrTempleInputInvalid }

// TempleSuggestionRepository is the port the review use case works through.
type TempleSuggestionRepository interface {
	// List returns limit suggestions of the given status starting at offset, ordered
	// created_at ASC, id ASC, and the total number with that status. An offset past
	// the end returns an empty slice, not an error.
	List(ctx context.Context, status string, offset, limit int) ([]TempleSuggestion, int, error)

	// FindByID returns the suggestion or ErrSuggestionNotFound.
	FindByID(ctx context.Context, id int64) (*TempleSuggestion, error)

	// CreateTemple, in ONE transaction, locks the suggestion row, generates the
	// temple id, inserts the temple (priority already derived by the caller in t)
	// and marks the suggestion created with adminID, now and the temple id. It
	// returns ErrSuggestionNotFound, ErrSuggestionAlreadyCreated, or
	// ErrReopenForbidden (suggestion rejected and role is not RoleAdmin), and on any
	// failure leaves suggestion and temples untouched.
	CreateTemple(ctx context.Context, suggestionID, adminID int64, role string, t NewTemple, priority int) (*CreatedTemple, error)
}
