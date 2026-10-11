package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// TempleSuggestionUsecase reviews temple suggestions: it lists and opens them and
// turns one into an official temple. Who may call it is decided at the route;
// the only role rule here is the one that needs the suggestion's status
// (reopening a rejected one is admin only) and that lives in the repository.
type TempleSuggestionUsecase struct {
	repo     domain.TempleSuggestionRepository
	wards    domain.WardResolver
	pageSize int
	now      func() time.Time
}

// NewTempleSuggestionUsecase wires the use case. pageSize is the queue page
// size (config temple_suggestion.page_size).
func NewTempleSuggestionUsecase(repo domain.TempleSuggestionRepository, wards domain.WardResolver, pageSize int) *TempleSuggestionUsecase {
	return &TempleSuggestionUsecase{repo: repo, wards: wards, pageSize: pageSize, now: time.Now}
}

// List returns one page of suggestions of status (empty = pending), oldest first.
// A page past the end is an empty page, not an error.
func (u *TempleSuggestionUsecase) List(ctx context.Context, status string, page int) (*domain.SuggestionPage, error) {
	if status == "" {
		status = domain.TempleSuggestionPending
	}
	if !domain.ValidSuggestionStatus(status) {
		return nil, domain.ErrInvalidStatus
	}
	if page < 1 {
		return nil, domain.ErrInvalidPage
	}
	items, total, err := u.repo.List(ctx, status, (page-1)*u.pageSize, u.pageSize)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.TempleSuggestion{}
	}
	return &domain.SuggestionPage{
		Items: items, Page: page, PageSize: u.pageSize, Total: total,
		TotalPages: (total + u.pageSize - 1) / u.pageSize,
	}, nil
}

// Get returns one suggestion or domain.ErrSuggestionNotFound.
func (u *TempleSuggestionUsecase) Get(ctx context.Context, id int64) (*domain.TempleSuggestion, error) {
	return u.repo.FindByID(ctx, id)
}

// Column limits of temples that the form must respect, so a too-long value is a
// 400 rather than a database error.
const (
	maxTempleName     = 255
	maxTempleSect     = 64
	maxTemplePerson   = 255
	maxTempleHours    = 64
	maxTempleAddress  = 500
	maxTempleCodeSize = 16
)

var (
	templeTypes   = map[string]bool{"CHUA": true, "THIEN_VIEN": true, "DAO_TRANG": true, "TINH_XA": true, "THIEN_XA": true}
	templeRegions = map[string]bool{"north": true, "central": true, "south": true}
)

// CreateTemple validates the form, checks the address codes against the
// administrative dataset, derives heritage_priority and asks the repository to
// create the temple and mark the suggestion created in one transaction.
//
// Validation and the dataset read happen before any write (AC19, AC18a), so a
// refused form touches nothing.
func (u *TempleSuggestionUsecase) CreateTemple(ctx context.Context, suggestionID, adminID int64, role string, in domain.NewTemple) (*domain.CreatedTemple, error) {
	t, priority, err := u.validate(in)
	if err != nil {
		return nil, err
	}

	// Names come from the dataset, never from the request (AC18b).
	rw, err := u.wards.ResolveCurrentWard(ctx, t.ProvinceCode, t.WardCode)
	switch {
	case errors.Is(err, domain.ErrAdministrativeUnitNotFound):
		return nil, inputErr("province_code", "province_code is not a province of the current scheme")
	case errors.Is(err, domain.ErrWardNotInProvince):
		return nil, inputErr("ward_code", "ward_code is not a ward of province_code")
	case err != nil:
		return nil, fmt.Errorf("resolve ward: %w", err)
	}
	t.Province, t.Ward = rw.ProvinceName, rw.WardFullName

	return u.repo.CreateTemple(ctx, suggestionID, adminID, role, t, priority)
}

func inputErr(field, msg string) error { return &domain.TempleInputError{Field: field, Message: msg} }

func (u *TempleSuggestionUsecase) validate(in domain.NewTemple) (domain.NewTemple, int, error) {
	t := in
	for _, p := range []*string{&t.Name, &t.Type, &t.Region, &t.Sect, &t.Founder, &t.CurrentAbbot,
		&t.HeritageRank, &t.ProvinceCode, &t.WardCode, &t.Address, &t.OperatingHours, &t.ImageURL, &t.ThumbnailURL,
		&t.Introduction, &t.ArchitectureHeritage} {
		*p = strings.TrimSpace(*p)
	}
	// Names are the dataset's to fill; whatever the caller sent is dropped.
	t.Province, t.Ward = "", ""

	required := []struct {
		field, value string
		max          int
	}{
		{"name", t.Name, maxTempleName},
		{"type", t.Type, 0},
		{"region", t.Region, 0},
		{"sect", t.Sect, maxTempleSect},
	}
	for _, r := range required {
		if r.value == "" {
			return t, 0, inputErr(r.field, r.field+" is required")
		}
		if r.max > 0 && utf8.RuneCountInString(r.value) > r.max {
			return t, 0, inputErr(r.field, fmt.Sprintf("%s must be at most %d characters", r.field, r.max))
		}
	}
	if !templeTypes[t.Type] {
		return t, 0, inputErr("type", "type is not a known temple type")
	}
	if !templeRegions[t.Region] {
		return t, 0, inputErr("region", "region must be one of north, central, south")
	}
	if t.EstablishedYear <= 0 || t.EstablishedYear > u.now().Year() {
		return t, 0, inputErr("established_year", "established_year must be a positive year not in the future")
	}
	if t.HeritageRank == "" {
		return t, 0, inputErr("heritage_rank", "heritage_rank is required")
	}
	priority, ok := domain.HeritagePriority(t.HeritageRank)
	if !ok {
		return t, 0, inputErr("heritage_rank", "heritage_rank is not a known rank")
	}
	if t.Latitude < -90 || t.Latitude > 90 {
		return t, 0, inputErr("latitude", "latitude must be between -90 and 90")
	}
	if t.Longitude < -180 || t.Longitude > 180 {
		return t, 0, inputErr("longitude", "longitude must be between -180 and 180")
	}
	if t.ProvinceCode == "" {
		return t, 0, inputErr("province_code", "province_code is required")
	}
	if t.WardCode == "" {
		return t, 0, inputErr("ward_code", "ward_code is required")
	}
	if utf8.RuneCountInString(t.ProvinceCode) > maxTempleCodeSize || utf8.RuneCountInString(t.WardCode) > maxTempleCodeSize {
		// Longer than any dataset code, so it cannot match; say which field.
		field := "province_code"
		if utf8.RuneCountInString(t.ProvinceCode) <= maxTempleCodeSize {
			field = "ward_code"
		}
		return t, 0, inputErr(field, field+" is too long")
	}
	if t.Address == "" {
		return t, 0, inputErr("address", "address is required")
	}
	if utf8.RuneCountInString(t.Address) > maxTempleAddress {
		return t, 0, inputErr("address", fmt.Sprintf("address must be at most %d characters", maxTempleAddress))
	}
	for _, p := range []struct{ field, value string }{
		{"founder", t.Founder}, {"current_abbot", t.CurrentAbbot},
	} {
		if utf8.RuneCountInString(p.value) > maxTemplePerson {
			return t, 0, inputErr(p.field, fmt.Sprintf("%s must be at most %d characters", p.field, maxTemplePerson))
		}
	}
	if utf8.RuneCountInString(t.OperatingHours) > maxTempleHours {
		return t, 0, inputErr("operating_hours", fmt.Sprintf("operating_hours must be at most %d characters", maxTempleHours))
	}
	for _, p := range []struct{ field, value string }{{"image_url", t.ImageURL}, {"thumbnail_url", t.ThumbnailURL}} {
		if p.value != "" && !validImageRef(p.value) {
			return t, 0, inputErr(p.field, p.field+" must start with /static/images/ or be an http(s) URL")
		}
	}

	if t.Aliases == nil {
		t.Aliases = []string{}
	}
	if t.VisitingNotes == nil {
		t.VisitingNotes = []string{}
	}
	return t, priority, nil
}

func validImageRef(v string) bool {
	return strings.HasPrefix(v, "/static/images/") || strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")
}
