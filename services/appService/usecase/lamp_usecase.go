package usecase

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// LampUsecase lights lamps and serves the garden of lit lamps.
type LampUsecase struct {
	lamps domain.LampRepository
	cfg   config.LampConfig
	now   func() time.Time
}

// NewLampUsecase wires the use case.
func NewLampUsecase(lamps domain.LampRepository, cfg config.LampConfig) *LampUsecase {
	return &LampUsecase{lamps: lamps, cfg: cfg, now: time.Now}
}

// clock returns the server time at the precision Postgres stores, so a value
// compared in Go and read back from the database are the same instant.
func (u *LampUsecase) clock() time.Time {
	return u.now().UTC().Truncate(time.Microsecond)
}

func invalidLamp(field, message string) error {
	return &domain.LampValidationError{Field: field, Message: message}
}

// normalizeLampText trims surrounding whitespace and composes to NFC, so a
// Vietnamese word typed with combining accents is counted and stored the same
// as the precomposed form.
func normalizeLampText(s string) string {
	return norm.NFC.String(strings.TrimSpace(s))
}

// Light validates in and lights it as a lamp owned by userID. created is false
// when an identical lamp the same user lit moments ago is returned instead (a
// double tap). It returns domain.ErrLampLimitReached when the user is at the cap.
func (u *LampUsecase) Light(ctx context.Context, userID int64, in domain.LampInput) (lamp *domain.Lamp, created bool, err error) {
	l, err := u.build(in)
	if err != nil {
		return nil, false, err
	}
	l.UserID = userID

	lamp, created, err = u.lamps.Light(ctx, domain.LightLampParams{
		Lamp: *l, MaxActive: u.cfg.MaxActivePerUser, DedupeWindow: u.cfg.DedupeWindow.Duration(),
	})
	if err != nil {
		return nil, false, fmt.Errorf("light lamp: %w", err)
	}
	return lamp, created, nil
}

// build validates the raw input and turns it into an entity (without ID/owner).
// Validation follows the request's field order so the first reported field is
// the first broken one.
func (u *LampUsecase) build(in domain.LampInput) (*domain.Lamp, error) {
	lampType := domain.LampType(in.LampType)
	if !lampType.Valid() {
		return nil, invalidLamp("lamp_type", "lamp_type must be one of OIL_LAMP, LOTUS_CANDLE, FLOWER_LANTERN")
	}
	intention := domain.Intention(in.Intention)
	if !intention.Valid() {
		return nil, invalidLamp("intention", "intention must be one of PEACE, SALVATION, MERIT_DEDICATION, GRATITUDE")
	}

	// A blank recipient is not an error since v2.0: the lamp is dedicated to all
	// beings and stored without one (BR-LMP-06).
	// VI: Từ v2.0, recipient trống không còn là lỗi: đèn dành cho tất cả chúng sanh
	// và được lưu không có người nhận (BR-LMP-06).
	var recipient *string
	if in.Recipient != nil {
		if r := normalizeLampText(*in.Recipient); r != "" {
			if utf8.RuneCountInString(r) > u.cfg.RecipientMaxChars {
				return nil, invalidLamp("recipient", fmt.Sprintf("recipient must be at most %d characters", u.cfg.RecipientMaxChars))
			}
			recipient = &r
		}
	}

	var prayer *string
	if in.Prayer != nil {
		if p := normalizeLampText(*in.Prayer); p != "" {
			if utf8.RuneCountInString(p) > u.cfg.PrayerMaxChars {
				return nil, invalidLamp("prayer", fmt.Sprintf("prayer must be at most %d characters", u.cfg.PrayerMaxChars))
			}
			prayer = &p
		}
	}

	if !u.durationAllowed(in.DurationDays) {
		return nil, invalidLamp("duration_days", "duration_days is not an allowed burn duration")
	}

	litAt := u.clock()
	return &domain.Lamp{
		LampType: lampType, Intention: intention, Recipient: recipient, Prayer: prayer,
		DurationDays: in.DurationDays, IsAnonymous: in.IsAnonymous,
		LitAt: litAt, EndAt: litAt.Add(time.Duration(in.DurationDays) * 24 * time.Hour),
	}, nil
}

func (u *LampUsecase) durationAllowed(days int) bool {
	for _, d := range u.cfg.DurationsDays {
		if d == days {
			return true
		}
	}
	return false
}

// Garden returns one page of the lit lamps of all users plus the total and
// per-type counts. A bad filter, limit or cursor is reported before any read.
// viewerID only marks the viewer's own lamps (IsMine); it never filters.
// VI: viewerID chỉ dùng để đánh dấu đèn của chính người xem (IsMine), không lọc.
func (u *LampUsecase) Garden(ctx context.Context, viewerID int64, f domain.GardenFilter) (*domain.GardenPage, error) {
	q := domain.GardenQuery{Limit: u.cfg.PageSize, Now: u.clock(), ViewerID: viewerID}

	if f.LampType != nil {
		t := domain.LampType(*f.LampType)
		if !t.Valid() {
			return nil, domain.ErrInvalidLampFilter
		}
		q.LampType = &t
	}

	if f.Limit != nil {
		n, err := strconv.Atoi(*f.Limit)
		if err != nil || n < 1 {
			return nil, domain.ErrInvalidLimit
		}
		if n < q.Limit {
			q.Limit = n
		}
	}

	if f.Cursor != nil {
		c, ok := decodeLampCursor(*f.Cursor)
		if !ok {
			return nil, domain.ErrInvalidCursor
		}
		q.After = c
	}

	page, err := u.lamps.Garden(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("lamp garden: %w", err)
	}

	if len(page.Items) > q.Limit {
		page.Items = page.Items[:q.Limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		next := encodeLampCursor(domain.Cursor{LitAt: last.LitAt, ID: last.ID})
		page.NextCursor = &next
	}
	if page.Items == nil {
		page.Items = []domain.GardenLamp{}
	}
	if page.Counts == nil {
		page.Counts = make(map[domain.LampType]int, len(domain.AllLampTypes))
	}
	for _, t := range domain.AllLampTypes {
		if _, ok := page.Counts[t]; !ok {
			page.Counts[t] = 0
		}
	}
	return page, nil
}

// The cursor is "<lit_at unix microseconds>.<id>" in base64url. It is opaque to
// clients; only this file reads it back.
func encodeLampCursor(c domain.Cursor) string {
	raw := strconv.FormatInt(c.LitAt.UnixMicro(), 10) + "." + strconv.FormatInt(c.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeLampCursor(s string) (*domain.Cursor, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, false
	}
	micro, id, found := strings.Cut(string(raw), ".")
	if !found {
		return nil, false
	}
	m, err := strconv.ParseInt(micro, 10, 64)
	if err != nil {
		return nil, false
	}
	i, err := strconv.ParseInt(id, 10, 64)
	if err != nil || i < 1 {
		return nil, false
	}
	return &domain.Cursor{LitAt: time.UnixMicro(m).UTC(), ID: i}, true
}
