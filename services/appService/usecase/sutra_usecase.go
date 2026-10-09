package usecase

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// SutraUsecase serves the read-only kinh / kệ / chú catalogue: a filtered, paged
// list, a detail lookup, a paged read of a translation's lines, the sutras linked
// to a deity, and the one-time first-run seed. Ordering lives in the repository
// contract (domain.SutraRepository); this layer owns filter validation, paging
// arithmetic and the sizes it is given by config.
type SutraUsecase struct {
	sutras  domain.SutraRepository
	deities domain.DeityRepository
	cfg     config.SutraConfig
}

// NewSutraUsecase wires the use case. deities is only used to check that a deity
// exists before listing its sutras.
func NewSutraUsecase(sutras domain.SutraRepository, deities domain.DeityRepository, cfg config.SutraConfig) *SutraUsecase {
	return &SutraUsecase{sutras: sutras, deities: deities, cfg: cfg}
}

// List returns one 1-based page of cfg.PageSize sutras matching the filters. A page
// past the end is a valid, empty page. page < 1, or so large that the offset would
// overflow, is domain.ErrInvalidPage; a filter value that is not allowed is
// *domain.InvalidSutraFilterError naming the parameter.
func (u *SutraUsecase) List(ctx context.Context, in domain.SutraFilterInput, page int) (*domain.SutraPage, error) {
	if page < 1 || page > math.MaxInt32 {
		return nil, domain.ErrInvalidPage
	}
	filter, err := parseSutraFilter(in)
	if err != nil {
		return nil, err
	}

	size := u.cfg.PageSize
	total, err := u.sutras.Count(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count sutras: %w", err)
	}
	items, err := u.sutras.List(ctx, filter, (page-1)*size, size)
	if err != nil {
		return nil, fmt.Errorf("list sutras: %w", err)
	}
	if items == nil {
		items = []domain.Sutra{}
	}
	return &domain.SutraPage{
		Items: items, Page: page, PageSize: size, Total: total, TotalPages: ceilDiv(total, size),
	}, nil
}

// Get returns one sutra by id, or domain.ErrSutraNotFound.
func (u *SutraUsecase) Get(ctx context.Context, id string) (*domain.Sutra, error) {
	s, err := u.sutras.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sutra %q: %w", id, err)
	}
	return s, nil
}

// Lines returns one 1-based page of a translation's lines. pageSize is the number
// of lines per page: nil means cfg.LinesPageSize, a value above cfg.LinesMaxPageSize
// is served at the cap, and a value below 1 is domain.ErrInvalidPageSize. A page past
// the end is a valid, empty page. The sutra is checked before the translation, so an
// unknown sutra is domain.ErrSutraNotFound and an unknown key on a known sutra is
// domain.ErrSutraTranslationNotFound.
func (u *SutraUsecase) Lines(ctx context.Context, sutraID, key string, page int, pageSize *int) (*domain.SutraLinesPage, error) {
	if key == "" {
		return nil, domain.ErrInvalidTranslation
	}
	if page < 1 || page > math.MaxInt32 {
		return nil, domain.ErrInvalidPage
	}
	size := u.cfg.LinesPageSize
	if pageSize != nil {
		if *pageSize < 1 {
			return nil, domain.ErrInvalidPageSize
		}
		size = min(*pageSize, u.cfg.LinesMaxPageSize)
	}

	tr, err := u.sutras.FindTranslation(ctx, sutraID, key)
	if err != nil {
		return nil, fmt.Errorf("find translation %q of sutra %q: %w", key, sutraID, err)
	}

	// N runs 1..LineCount without gaps, so page p of size s is exactly the
	// lines with (p-1)*s < N <= p*s. int64: page * size can exceed int32.
	fromN := int64(page-1) * int64(size)
	toN := int64(page) * int64(size)
	items, err := u.sutras.ListLines(ctx, tr.ID, fromN, toN)
	if err != nil {
		return nil, fmt.Errorf("list lines of translation %q of sutra %q: %w", key, sutraID, err)
	}
	if items == nil {
		items = []domain.SutraLine{}
	}
	return &domain.SutraLinesPage{
		Translation: tr.Key, Items: items, Page: page, PageSize: size,
		Total: tr.LineCount, TotalPages: ceilDiv(tr.LineCount, size),
	}, nil
}

// ListByDeity returns one 1-based page of cfg.PageSize sutras linked to a deity,
// optionally only those where the deity has the given role. An unknown deity is
// domain.ErrDeityNotFound; a deity with no sutras is a valid, empty page.
func (u *SutraUsecase) ListByDeity(ctx context.Context, deityID string, role *string, page int) (*domain.DeitySutraPage, error) {
	if page < 1 || page > math.MaxInt32 {
		return nil, domain.ErrInvalidPage
	}
	var r *domain.DeityRole
	if role != nil {
		dr := domain.DeityRole(*role)
		if !dr.Valid() {
			return nil, &domain.InvalidSutraFilterError{Field: "role"}
		}
		r = &dr
	}

	if _, err := u.deities.FindByID(ctx, deityID); err != nil {
		return nil, fmt.Errorf("find deity %q: %w", deityID, err)
	}

	size := u.cfg.PageSize
	total, err := u.sutras.CountByDeity(ctx, deityID, r)
	if err != nil {
		return nil, fmt.Errorf("count sutras of deity %q: %w", deityID, err)
	}
	items, err := u.sutras.ListByDeity(ctx, deityID, r, (page-1)*size, size)
	if err != nil {
		return nil, fmt.Errorf("list sutras of deity %q: %w", deityID, err)
	}
	if items == nil {
		items = []domain.DeitySutra{}
	}
	return &domain.DeitySutraPage{
		Items: items, Page: page, PageSize: size, Total: total, TotalPages: ceilDiv(total, size),
	}, nil
}

// SeedIfEmpty loads the seed sutras on the service's first run: only when the table
// holds no sutra at all, so a restart never resurrects a deleted sutra or overwrites
// an edited one. load is called only when seeding is needed, so a normal start does
// not parse the seed. Every deity the seed links to must already exist: seeding stops
// with domain.ErrSutraSeedDeitiesMissing otherwise, because the links could never be
// added later (the seed does not run again once the table has rows). It returns what
// was written (all zero when the table already had data).
// TD-004: this is the only way content enters; there is no update path after the first seed.
func (u *SutraUsecase) SeedIfEmpty(ctx context.Context, load func() ([]domain.SutraSeed, error)) (domain.SutraSeedResult, error) {
	n, err := u.sutras.Count(ctx, domain.SutraFilter{})
	if err != nil {
		return domain.SutraSeedResult{}, fmt.Errorf("count sutras: %w", err)
	}
	if n > 0 {
		return domain.SutraSeedResult{}, nil
	}

	seeds, err := load()
	if err != nil {
		return domain.SutraSeedResult{}, fmt.Errorf("load sutra seed: %w", err)
	}

	if ids := seedDeityIDs(seeds); len(ids) > 0 {
		missing, err := u.sutras.MissingDeities(ctx, ids)
		if err != nil {
			return domain.SutraSeedResult{}, fmt.Errorf("check seed deities: %w", err)
		}
		if len(missing) > 0 {
			return domain.SutraSeedResult{}, fmt.Errorf("%w: %s (run cmd/seeddeities first)",
				domain.ErrSutraSeedDeitiesMissing, strings.Join(missing, ", "))
		}
	}

	res, err := u.sutras.InsertAll(ctx, seeds)
	if err != nil {
		return domain.SutraSeedResult{}, fmt.Errorf("seed sutras: %w", err)
	}
	return res, nil
}

// parseSutraFilter validates the raw list filters. A parameter that is present but
// empty is as invalid as one with an unknown value.
// maxSutraQueryRunes caps the search term; no title or reference is longer.
const maxSutraQueryRunes = 100

func parseSutraFilter(in domain.SutraFilterInput) (domain.SutraFilter, error) {
	var f domain.SutraFilter
	invalid := func(field string) error { return &domain.InvalidSutraFilterError{Field: field} }

	if in.Tradition != nil {
		t := domain.Tradition(*in.Tradition)
		if !t.Valid() {
			return f, invalid("tradition")
		}
		f.Tradition = &t
	}
	if in.Kind != nil {
		k := domain.Kind(*in.Kind)
		if !k.Valid() {
			return f, invalid("kind")
		}
		f.Kind = &k
	}
	if in.Collection != nil {
		if *in.Collection == "" {
			return f, invalid("collection")
		}
		c := *in.Collection
		f.Collection = &c
	}
	if in.Purpose != nil {
		p := domain.Purpose(*in.Purpose)
		if !p.Valid() {
			return f, invalid("purpose")
		}
		f.Purpose = &p
	}
	if in.Daily != nil {
		switch *in.Daily {
		case "true":
			d := true
			f.Daily = &d
		case "false":
			d := false
			f.Daily = &d
		default:
			return f, invalid("daily")
		}
	}
	if in.MaxMinutes != nil {
		m, err := strconv.Atoi(*in.MaxMinutes)
		if err != nil || m < 1 || m > math.MaxInt32 {
			return f, invalid("max_minutes")
		}
		f.MaxMinutes = &m
	}
	if in.Lang != nil {
		if *in.Lang == "" {
			return f, invalid("lang")
		}
		l := *in.Lang
		f.Lang = &l
	}
	if in.Q != nil {
		q := strings.TrimSpace(*in.Q)
		if q == "" || utf8.RuneCountInString(q) > maxSutraQueryRunes {
			return f, invalid("q")
		}
		f.Query = &q
	}
	return f, nil
}

// seedDeityIDs returns the distinct deity ids a seed links to, sorted so the check
// (and its error message) is deterministic.
func seedDeityIDs(seeds []domain.SutraSeed) []string {
	set := map[string]struct{}{}
	for _, s := range seeds {
		for _, d := range s.Sutra.Deities {
			set[d.DeityID] = struct{}{}
		}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ceilDiv is ceil(n / size) for size >= 1; 0 when n is 0.
func ceilDiv(n, size int) int { return (n + size - 1) / size }
