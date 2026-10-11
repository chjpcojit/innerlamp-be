package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	ID   int64
	Name string
	// ProvinceCode, WardCode and Ward are nil together for a suggestion sent
	// before the structured address (migration 0030); Province is then the
	// string the user typed. Otherwise Province and Ward are the dataset-name
	// snapshots taken when it was sent. Address is the part below ward level.
	ProvinceCode *string
	Province     string
	WardCode     *string
	Ward         *string
	Address      string
	Status       string
	CreatedAt    time.Time

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
	// Address is the part below ward level; ProvinceCode/WardCode and the name
	// snapshots Province/Ward make up the rest (spec temple v1.7). A row seeded
	// before then has no codes and Address holds the whole string.
	Address        string
	ProvinceCode   string
	Province       string
	WardCode       string
	Ward           string
	OperatingHours string
	Amenities      TempleAmenities

	Introduction         string
	ArchitectureHeritage string
	VisitingNotes        []string
}

// FullAddress is the address to display: the non-empty parts of Address, Ward
// and Province joined by ", ". A temple with no Ward (not structured yet)
// returns Address unchanged. It mirrors appService/domain.Temple.FullAddress,
// which the temple read API uses, so a created temple reads back identically.
func (t Temple) FullAddress() string {
	if t.Ward == "" {
		return t.Address
	}
	parts := make([]string, 0, 3)
	for _, p := range []string{t.Address, t.Ward, t.Province} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
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
	// Address is the part below ward level (may not be empty: AC18).
	Address string
	// ProvinceCode and WardCode come from the client; Province and Ward are
	// filled by the use case from the administrative dataset, never by the client.
	ProvinceCode   string
	WardCode       string
	Province       string
	Ward           string
	OperatingHours string
	Amenities      TempleAmenities

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

// ResolvedWard is a ward and its province as the dataset names them.
type ResolvedWard struct {
	ProvinceName string
	WardFullName string
}

var (
	// ErrAdministrativeUnitNotFound — the province code is not a province of the
	// current scheme.
	ErrAdministrativeUnitNotFound = errors.New("administrative unit not found")

	// ErrWardNotInProvince — the ward code is unknown or belongs to another province.
	ErrWardNotInProvince = errors.New("ward not found in province")
)

// WardResolver checks a province/ward code pair against the current-scheme
// administrative dataset (addr_new_*). It is a narrow port so the use case does
// not depend on how the dataset is stored.
type WardResolver interface {
	// ResolveCurrentWard returns the names for the pair, ErrAdministrativeUnitNotFound
	// when the province does not exist, or ErrWardNotInProvince when the ward does
	// not exist in that province.
	ResolveCurrentWard(ctx context.Context, provinceCode, wardCode string) (*ResolvedWard, error)
}

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
