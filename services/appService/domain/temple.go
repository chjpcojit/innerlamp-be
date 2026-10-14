package domain

import (
	"context"
	"errors"
	"strings"
)

// Temple is one place of worship in the browse catalogue (chùa, thiền viện,
// đạo tràng, tịnh xá, thiền xá). It maps the temples table.
//
// Plain data: no json/db tag — the wire shape is delivery's, the SQL shape is
// repository's. ImageURL and ThumbnailURL are empty strings when the source
// data has no image yet (currently all of them); delivery turns "" into null.
type Temple struct {
	ID          string
	Name        string
	Type        string
	Region      string
	ImageURL    string
	Thumbnail   string
	Aliases     []string
	Sect        string
	Founder     string
	Abbot       string
	Established int
	Heritage    string
	// HeritagePriority is derived from Heritage via HeritagePriority(); lower is
	// more prominent. Stored so ranking is a column sort.
	HeritagePriority int
	Latitude         float64
	Longitude        float64
	// Address is the part below ward level (house number, street, hamlet) for a
	// temple loaded with a structured address (spec temple v1.7); for a temple
	// not re-seeded yet it is still the old full display string. Province is the
	// separate, stored short province/city name (no prefix, e.g. "Hà Nội") —
	// never parsed from Address.
	//
	// ProvinceCode, WardCode and Ward are the structured address (migration
	// 0032): current-scheme unit codes from administrative-units and the ward's
	// full name as it was in the dataset when the temple was written. "" means
	// the temple has not been re-seeded yet (all three are then empty together);
	// delivery turns "" into null. Use FullAddress for display.
	Address        string
	Province       string
	ProvinceCode   string
	WardCode       string
	Ward           string
	OperatingHours string
	Amenities      TempleAmenities

	// Editorial content for the detail screen. Empty until researched.
	Introduction         string
	ArchitectureHeritage string
	VisitingNotes        []string
}

// FullAddress is the address as shown to a person (AC 33 of spec temple v1.7):
// when the temple has a Ward, the non-empty parts Address, Ward and Province
// joined by ", "; when it has none (not re-seeded yet), the stored Address
// unchanged — it is already the full string and appending Province would repeat
// the province.
func (t Temple) FullAddress() string {
	if t.Ward == "" {
		return t.Address
	}
	var parts []string
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

// TempleOption is the minimal projection of a temple for pick-lists (the
// registration screen): just enough to show a choice and send its id back.
// Province is the temple's stored province/city name and ProvinceCode its
// current-scheme code ("" until the temple is re-seeded; delivery turns it into
// null). Plain data, no tag — the wire shape is delivery's.
type TempleOption struct {
	ID           string
	Name         string
	Province     string
	ProvinceCode string
}

// TempleOptionPage is one page of TempleOption plus what a client needs to know
// whether another page exists.
type TempleOptionPage struct {
	Items      []TempleOption
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// TemplePage is one page of the ordered catalogue plus what a client needs to
// know whether another page exists.
type TemplePage struct {
	Items      []Temple
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// heritagePriority orders heritage ranks from most to least prominent. The
// order is fixed by the schema (chk_temples_heritage_rank lists the same set);
// a new rank is a migration plus an entry here, not configuration.
var heritagePriority = map[string]int{
	"DI_SAN_THE_GIOI":           1,
	"DI_TICH_QUOC_GIA_DAC_BIET": 2,
	"KHU_DU_LICH_QUOC_GIA":      3,
	"DI_TICH_QUOC_GIA":          4,
	"DANH_LAM_THANG_CANH":       5,
	"DI_TICH_CAP_TINH":          6,
	"DI_TICH_CAP_THANH_PHO":     7,
	"CHUA_XEP_HANG":             8,
}

// HeritagePriority returns the ranking weight of a heritage_rank value and
// whether the value is known.
func HeritagePriority(rank string) (int, bool) {
	p, ok := heritagePriority[rank]
	return p, ok
}

// Sentinel errors for the temple catalogue.
var (
	// ErrTempleNotFound — no temple has the requested id.
	ErrTempleNotFound = errors.New("temple not found")

	// ErrInvalidPage — the requested page number is not a positive integer.
	ErrInvalidPage = errors.New("page must be a positive integer")

	// ErrInvalidTempleQuery — the temple search term is present but empty or
	// blank after trimming, or longer than the allowed length.
	ErrInvalidTempleQuery = errors.New("invalid search term")
)

// TempleRepository is the port the temple use case reads through, and the
// seeder writes through.
//
// Ordering is part of the contract: every read returns temples by
// heritage_priority ASC, established_year ASC, id ASC — deterministic, so
// paging never repeats or skips a row.
type TempleRepository interface {
	// ListTop returns the first limit temples in ranking order.
	ListTop(ctx context.Context, limit int) ([]Temple, error)

	// List returns limit temples starting at offset in ranking order, and the
	// total number of temples. An offset past the end returns an empty slice,
	// not an error.
	List(ctx context.Context, offset, limit int) ([]Temple, int, error)

	// ListOptions returns limit temples as TempleOption starting at offset in the
	// same ranking order as ListTop and List, and the total number of matching
	// temples. A non-empty query keeps only temples whose name contains it,
	// ignoring case and Vietnamese diacritics; "" keeps every temple. An offset
	// past the end, or no match, returns an empty slice, not an error.
	ListOptions(ctx context.Context, query string, offset, limit int) ([]TempleOption, int, error)

	// FindByID returns the temple or ErrTempleNotFound.
	FindByID(ctx context.Context, id string) (*Temple, error)

	// Upsert inserts or updates temples keyed by id, idempotently.
	Upsert(ctx context.Context, temples []Temple) error
}
