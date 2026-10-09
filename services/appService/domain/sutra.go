package domain

import (
	"context"
	"errors"
	"fmt"
)

// Tradition is the Buddhist school a sutra belongs to. The set is fixed by the
// CHECK constraint on sutras.tradition (migrations/0013).
type Tradition string

const (
	TraditionTheravada Tradition = "THERAVADA" // Nam Tông
	TraditionMahayana  Tradition = "MAHAYANA"  // Bắc Tông
	TraditionVajrayana Tradition = "VAJRAYANA" // Kim Cang Thừa
)

// Valid reports whether t is one of the three known traditions.
func (t Tradition) Valid() bool {
	switch t {
	case TraditionTheravada, TraditionMahayana, TraditionVajrayana:
		return true
	}
	return false
}

// Kind is what sort of text a sutra row holds (CHECK on sutras.kind).
type Kind string

const (
	KindSutra   Kind = "SUTRA"   // prose discourse
	KindVerse   Kind = "VERSE"   // kệ / verse collections
	KindDharani Kind = "DHARANI" // chú, chanted by sound
)

// Valid reports whether k is one of the known kinds.
func (k Kind) Valid() bool {
	switch k {
	case KindSutra, KindVerse, KindDharani:
		return true
	}
	return false
}

// Variant is how a translation relates to the source text (CHECK on
// sutra_translations.variant).
type Variant string

const (
	VariantTranslation Variant = "translation"
	VariantChant       Variant = "chant"   // Hán-Việt / transliteration, as chanted
	VariantMeaning     Variant = "meaning" // dịch nghĩa
	VariantPoem        Variant = "poem"
)

// Valid reports whether v is one of the known variants.
func (v Variant) Valid() bool {
	switch v {
	case VariantTranslation, VariantChant, VariantMeaning, VariantPoem:
		return true
	}
	return false
}

// Licence codes of a translation (sutra_translations.license). The closed set is
// checked by the seed parser, not by a CHECK, so a code can be added without a migration.
const (
	LicenseCC0          = "CC0-1.0"
	LicenseCCBYNC       = "CC-BY-NC-4.0"
	LicensePublicDomain = "PUBLIC_DOMAIN"
	LicenseCopyAllowed  = "COPY_ALLOWED" // the page says copying is welcome; not a standard licence
	LicenseUnknown      = "UNKNOWN"
)

// ValidLicense reports whether code is one of the known licence codes.
func ValidLicense(code string) bool {
	switch code {
	case LicenseCC0, LicenseCCBYNC, LicensePublicDomain, LicenseCopyAllowed, LicenseUnknown:
		return true
	}
	return false
}

// DeityRole is how a Buddha / bodhisattva relates to a sutra (CHECK on
// sutra_deities.role).
type DeityRole string

const (
	RoleSpeaker   DeityRole = "SPEAKER"   // delivers the sutra
	RoleSubject   DeityRole = "SUBJECT"   // the text is about them
	RoleMentioned DeityRole = "MENTIONED" // only named in it
)

// Valid reports whether r is one of the known roles.
func (r DeityRole) Valid() bool {
	switch r {
	case RoleSpeaker, RoleSubject, RoleMentioned:
		return true
	}
	return false
}

// Purpose is what a text is chanted or read for. The set is closed on purpose
// so a typo in the data cannot silently create a new filter value.
type Purpose string

const (
	PurposeCauAn       Purpose = "CAU_AN"
	PurposeCauSieu     Purpose = "CAU_SIEU"
	PurposeTriChu      Purpose = "TRI_CHU"
	PurposeSamHoi      Purpose = "SAM_HOI"
	PurposeHoiHuong    Purpose = "HOI_HUONG"
	PurposePhatNguyen  Purpose = "PHAT_NGUYEN"
	PurposeBaoHieu     Purpose = "BAO_HIEU"
	PurposeThienDinh   Purpose = "THIEN_DINH"
	PurposeHocGiaoLy   Purpose = "HOC_GIAO_LY"
	PurposeQuyYThoGioi Purpose = "QUY_Y_THO_GIOI"
)

// Valid reports whether p is one of the known purposes.
func (p Purpose) Valid() bool {
	switch p {
	case PurposeCauAn, PurposeCauSieu, PurposeTriChu, PurposeSamHoi, PurposeHoiHuong,
		PurposePhatNguyen, PurposeBaoHieu, PurposeThienDinh, PurposeHocGiaoLy, PurposeQuyYThoGioi:
		return true
	}
	return false
}

// Sutra is one text (kinh, kệ or chú). It maps the sutras table plus the rows
// hanging off it.
//
// Plain data: no json/db tags — the wire shape is delivery's, the SQL shape is
// repository's.
type Sutra struct {
	ID        string
	SortOrder int
	Tradition Tradition
	Kind      Kind
	// Collection is the bộ / nhóm the text belongs to (Trung Bộ, Bát Nhã, ...).
	Collection string
	// Reference is the citation code only ("SN 56.11", "Taishō T0450"); "" when the
	// source has none. The Hán title, author, Hán translator, year and any remark live
	// in the fields below, never packed into this one.
	Reference string
	// OriginalTitle is the full Hán-Việt / source title, or the scripture a dharani is
	// drawn from; "" when none.
	OriginalTitle string
	// Author is who composed the text (not who translated it); "" when none.
	Author string
	// HanTranslator is who translated the text into Hán; "" when none.
	HanTranslator string
	// HanTranslationYear is the year (CE) of the Hán translation; nil when unknown.
	HanTranslationYear *int
	// ReferenceNote is whatever the source said about the citation that fits no other field.
	ReferenceNote string
	// Summary is a short Vietnamese description of the text; "" until one is written.
	Summary string
	// Titles maps a language code to the title. It always holds "vi".
	Titles     map[string]string
	DailyChant bool
	Purposes   []Purpose
	// DurationMinutes is an editorial estimate, not a measurement; nil when none was made.
	DurationMinutes *int
	// AudioURL is the recording of the text; "" when there is none (never NULL).
	AudioURL     string
	ReviewStatus string

	Deities      []SutraDeity
	Translations []SutraTranslation
}

// SutraDeity links a sutra to a deity. NameVI and ThumbnailURL are read from the
// deities table so a client needs no second request to show the link.
type SutraDeity struct {
	DeityID      string
	Role         DeityRole
	NameVI       string
	ThumbnailURL string
}

// SutraTranslation describes one text of a sutra in one language. Its lines are
// read separately (SutraRepository.ListLines) because a text can run to
// thousands of lines.
type SutraTranslation struct {
	// ID is the storage id; it is not part of the API and changes when a sutra is reloaded.
	ID         int64
	Key        string
	Lang       string
	Variant    Variant
	Translator string
	SourceName string
	SourceURL  string
	// License is a License code; LicenseURL and LicenseNote carry the URL and the
	// source's own words (both "" when the source gave none).
	License     string
	LicenseURL  string
	LicenseNote string
	Note        string
	LineCount   int
	Sections    []SutraSection
}

// SutraSection is a named run of consecutive lines (a phẩm, a chapter).
type SutraSection struct {
	Name string
	// StartN is the N of the section's first line.
	StartN    int
	LineCount int
}

// SutraLine is one display line. N runs 1..LineCount without gaps within one
// translation, so line N always sits on page floor((N-1)/pageSize)+1.
type SutraLine struct {
	N    int
	Text string
	// Section is "" when the line belongs to no section.
	Section string
}

// DeitySutra is a sutra together with the role one particular deity plays in it.
type DeitySutra struct {
	Sutra
	Role DeityRole
}

// SutraPage is one page of the sutra list.
type SutraPage struct {
	Items      []Sutra
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// DeitySutraPage is one page of the sutras linked to a deity.
type DeitySutraPage struct {
	Items      []DeitySutra
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// SutraLinesPage is one page of a translation's lines.
type SutraLinesPage struct {
	Translation string
	Items       []SutraLine
	Page        int
	PageSize    int
	Total       int
	TotalPages  int
}

// SutraFilterInput carries the raw list filters as the delivery layer read them.
// A nil field means the parameter was absent; a pointer to "" means it was
// present but empty, which is invalid (the use case rejects it).
type SutraFilterInput struct {
	Tradition  *string
	Kind       *string
	Collection *string
	Purpose    *string
	Daily      *string
	MaxMinutes *string
	Lang       *string
	Q          *string
}

// SutraFilter is the validated form of SutraFilterInput the repository
// understands. A nil field means "do not filter on it".
type SutraFilter struct {
	Tradition  *Tradition
	Kind       *Kind
	Collection *string
	Purpose    *Purpose
	Daily      *bool
	MaxMinutes *int
	Lang       *string
	// Query is the trimmed search term (accent- and case-insensitive substring).
	Query *string
}

// SutraSeed is one sutra with its full content, as loaded from the data files.
// Sutra.Translations is unused; the translations and their lines are in Translations.
type SutraSeed struct {
	Sutra        Sutra
	Translations []SutraTranslationSeed
}

// SutraTranslationSeed is a translation with its lines.
type SutraTranslationSeed struct {
	SutraTranslation
	Lines []SutraLine
}

// SutraSeedResult counts what a seed run wrote.
type SutraSeedResult struct {
	Sutras       int
	Translations int
	Lines        int
	DeityLinks   int
}

var (
	// ErrSutraNotFound — no sutra has the requested id.
	ErrSutraNotFound = errors.New("sutra not found")

	// ErrSutraTranslationNotFound — the sutra exists but has no translation with that key.
	ErrSutraTranslationNotFound = errors.New("sutra translation not found")

	// ErrInvalidSutraFilter — a filter value is not allowed. Use InvalidSutraFilterError
	// to say which parameter; errors.Is still matches this sentinel.
	ErrInvalidSutraFilter = errors.New("invalid filter value")

	// ErrInvalidPageSize — page_size is not a positive integer.
	ErrInvalidPageSize = errors.New("page_size must be a positive integer")

	// ErrInvalidTranslation — the translation query parameter is missing or empty.
	ErrInvalidTranslation = errors.New("translation is required")

	// ErrSutraSeedDeitiesMissing — the sutra seed references deities that are not in
	// the deities table. Seeding is aborted rather than dropping the links, because the
	// seed never runs again once the sutras table has rows. Run cmd/seeddeities first.
	ErrSutraSeedDeitiesMissing = errors.New("sutra seed references deities that are not seeded")
)

// InvalidSutraFilterError names the request parameter whose value was rejected.
type InvalidSutraFilterError struct {
	Field string
}

func (e *InvalidSutraFilterError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, ErrInvalidSutraFilter.Error())
}

// Unwrap lets errors.Is(err, ErrInvalidSutraFilter) match.
func (e *InvalidSutraFilterError) Unwrap() error { return ErrInvalidSutraFilter }

// SutraRepository is the port the sutra use case reads and seeds through.
//
// Ordering is part of the contract: lists return sutras by sort_order ASC, id ASC,
// except ListByDeity, which orders by the deity's role (SUBJECT, SPEAKER, MENTIONED)
// first. Deterministic, so paging never repeats or skips a row.
type SutraRepository interface {
	// Count returns the number of sutras matching the filter (the zero filter counts all).
	Count(ctx context.Context, filter SutraFilter) (int, error)

	// List returns limit sutras matching the filter starting at offset, each with its
	// translations (list shape: no sources, no sections) and deities. An offset past
	// the end returns an empty slice, not an error.
	List(ctx context.Context, filter SutraFilter, offset, limit int) ([]Sutra, error)

	// FindByID returns the sutra with its full translations (sources, sections) and
	// deities, or ErrSutraNotFound.
	FindByID(ctx context.Context, id string) (*Sutra, error)

	// FindTranslation returns the translation of a sutra by key. It returns
	// ErrSutraNotFound when the sutra does not exist and ErrSutraTranslationNotFound
	// when it exists but has no such key; the sutra is checked first.
	FindTranslation(ctx context.Context, sutraID, key string) (*SutraTranslation, error)

	// ListLines returns the lines of a translation with fromN < N <= toN, ordered by N.
	// A range past the end returns an empty slice, not an error.
	ListLines(ctx context.Context, translationID int64, fromN, toN int64) ([]SutraLine, error)

	// CountByDeity returns how many sutras the deity is linked to, optionally only with
	// the given role (nil = any role).
	CountByDeity(ctx context.Context, deityID string, role *DeityRole) (int, error)

	// ListByDeity returns limit sutras linked to the deity starting at offset, in list
	// shape, ordered by role then sort_order then id. Each element carries the
	// deity's role in that sutra.
	ListByDeity(ctx context.Context, deityID string, role *DeityRole, offset, limit int) ([]DeitySutra, error)

	// MissingDeities returns those of ids that are not in the deities table.
	MissingDeities(ctx context.Context, ids []string) ([]string, error)

	// InsertAll inserts the sutras with their translations, lines and deity links in
	// ONE transaction: all or nothing. A sutra whose id already exists is left
	// untouched and its children are skipped (ON CONFLICT DO NOTHING), so two
	// instances seeding at once cannot duplicate or overwrite each other.
	InsertAll(ctx context.Context, seeds []SutraSeed) (SutraSeedResult, error)
}
