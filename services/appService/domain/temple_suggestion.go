package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Suggestion states. The set is fixed by chk_temple_suggestions_status.
const (
	TempleSuggestionPending  = "pending"
	TempleSuggestionCreated  = "created"
	TempleSuggestionRejected = "rejected"
)

// Length limits fixed by the temple_suggestions columns. They count characters
// (runes), not bytes: Vietnamese text is mostly multi-byte.
const (
	TempleSuggestionNameMax     = 255
	TempleSuggestionProvinceMax = 100
	TempleSuggestionAddressMax  = 500
)

// TempleSuggestion is a temple an app user proposes for the catalogue. It maps
// the temple_suggestions table. The app side only ever creates one; review
// (created / rejected) belongs to the back-office service.
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
}

// TempleSuggestionInput is what a submitter provides, untrimmed.
type TempleSuggestionInput struct {
	Name     string
	Province string
	Address  string
}

// ErrTempleSuggestionInvalid — a submitted value is not allowed. Use
// TempleSuggestionValidationError to say which one; errors.Is still matches this.
var ErrTempleSuggestionInvalid = errors.New("invalid temple suggestion")

// TempleSuggestionValidationError names the request field that was rejected and why.
type TempleSuggestionValidationError struct {
	// Field is the request field, e.g. "name".
	Field   string
	Message string
}

func (e *TempleSuggestionValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Unwrap lets errors.Is(err, ErrTempleSuggestionInvalid) match.
func (e *TempleSuggestionValidationError) Unwrap() error { return ErrTempleSuggestionInvalid }

// TempleSuggestionRepository is the port the suggestion use case stores through.
type TempleSuggestionRepository interface {
	// Create inserts s as pending (ID, Status and CreatedAt are ignored and
	// filled in) and returns the stored row.
	Create(ctx context.Context, s *TempleSuggestion) (*TempleSuggestion, error)
}
