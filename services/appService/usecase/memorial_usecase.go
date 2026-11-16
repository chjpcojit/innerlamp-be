package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
	"chjpcoj.com.innerlamp/services/appService/photo"
)

// Field limits fixed by the spec / the memorials schema (VARCHAR lengths, CHECK
// ranges). They are schema, not tuning knobs, so they are constants; the only
// tunable (the per-user cap) comes from config.MemorialConfig.
const (
	memorialFullNameMax    = 100
	memorialNoteMax        = 1000
	memorialMinDeathYear   = 1900
	memorialMaxReminderDay = 30
	memorialDefaultZone    = "Asia/Ho_Chi_Minh"
)

// memorialClockTime matches a 24h "HH:mm" with two digits each.
var memorialClockTime = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// daysInSolarMonth lets February take 29: with no year there is no way to know
// whether it is a leap year, so the day is not rejected.
var daysInSolarMonth = [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// vietnamZone is UTC+7, used for "today" when bounding the death date. A fixed
// zone avoids depending on tzdata being present in the runtime image.
var vietnamZone = time.FixedZone("ICT", 7*60*60)

// MemorialUsecase manages a user's death-anniversary book: create, read, replace
// and delete. Every method takes the owner's userID and passes it down so the
// repository can enforce ownership.
type MemorialUsecase struct {
	memorials domain.MemorialRepository
	photos    domain.MemorialPhotoStore
	cfg       config.MemorialConfig
	now       func() time.Time
}

// NewMemorialUsecase wires the use case.
func NewMemorialUsecase(memorials domain.MemorialRepository, cfg config.MemorialConfig) *MemorialUsecase {
	return &MemorialUsecase{memorials: memorials, cfg: cfg, now: time.Now}
}

// WithPhotoStore enables the photo routes. Without it SetPhoto and RemovePhoto
// return an error instead of writing to a nil store, so a deployment that has
// not configured storage fails loudly rather than silently dropping files.
func (u *MemorialUsecase) WithPhotoStore(store domain.MemorialPhotoStore) *MemorialUsecase {
	u.photos = store
	return u
}

// Create validates in and stores it as a memorial owned by userID. It returns
// domain.ErrMemorialLimitReached when the user is at the cap.
func (u *MemorialUsecase) Create(ctx context.Context, userID int64, in domain.MemorialInput) (*domain.Memorial, error) {
	m, err := u.build(in)
	if err != nil {
		return nil, err
	}
	m.UserID = userID

	created, err := u.memorials.Create(ctx, m, u.cfg.MaxPerUser)
	if err != nil {
		return nil, fmt.Errorf("create memorial: %w", err)
	}
	_ = u.attachPhotoURL(ctx, created)
	return created, nil
}

// List returns the user's memorials, newest first.
func (u *MemorialUsecase) List(ctx context.Context, userID int64) ([]domain.Memorial, error) {
	items, err := u.memorials.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list memorials: %w", err)
	}
	if items == nil {
		items = []domain.Memorial{}
	}
	for i := range items {
		_ = u.attachPhotoURL(ctx, &items[i])
	}
	return items, nil
}

// Get returns one of the user's memorials, or domain.ErrMemorialNotFound.
func (u *MemorialUsecase) Get(ctx context.Context, userID, id int64) (*domain.Memorial, error) {
	m, err := u.memorials.FindByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("get memorial %d: %w", id, err)
	}
	if err := u.attachPhotoURL(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Update replaces the editable fields of one of the user's memorials with in.
func (u *MemorialUsecase) Update(ctx context.Context, userID, id int64, in domain.MemorialInput) (*domain.Memorial, error) {
	m, err := u.build(in)
	if err != nil {
		return nil, err
	}
	m.ID = id
	m.UserID = userID

	updated, err := u.memorials.Update(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("update memorial %d: %w", id, err)
	}
	_ = u.attachPhotoURL(ctx, updated)
	return updated, nil
}

// Delete removes one of the user's memorials, or returns domain.ErrMemorialNotFound.
func (u *MemorialUsecase) Delete(ctx context.Context, userID, id int64) error {
	key, err := u.memorials.Delete(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("delete memorial %d: %w", id, err)
	}
	// The row is gone; a leftover object must not turn a successful delete into
	// an error (tasks.md Phase 6, Step D-3).
	if key != nil && u.photos != nil {
		_ = u.photos.Delete(ctx, *key)
	}
	return nil
}

// SetPhoto decodes one uploaded image, stores the normalised JPEG and points
// the memorial at it. The previous object is removed only after the database
// accepted the new key.
//
// The client's content type is never trusted: the format is taken from the
// bytes themselves, and what is stored is the re-encoded copy, not the upload.
func (u *MemorialUsecase) SetPhoto(ctx context.Context, userID, id int64, r io.Reader) (*domain.Memorial, error) {
	if u.photos == nil {
		return nil, errors.New("memorial photo store is not configured")
	}
	maxBytes := u.cfg.PhotoMaxBytes
	if maxBytes <= 0 {
		maxBytes = 5 << 20
	}
	// Ownership comes before the body: the handler already checks it, and this
	// second lookup keeps the use case safe when it is called directly, so an
	// id the caller does not own never reaches the decoder.
	if _, err := u.memorials.FindByIDAndUser(ctx, id, userID); err != nil {
		return nil, fmt.Errorf("get memorial %d: %w", id, err)
	}

	// One byte over the cap is enough to know the body is too large, without
	// buffering an unbounded upload.
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read memorial photo: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, domain.ErrPhotoTooLarge
	}
	normalized, err := photo.Normalize(data)
	if err != nil {
		if errors.Is(err, photo.ErrTooLarge) {
			return nil, domain.ErrPhotoTooLarge
		}
		return nil, domain.ErrPhotoUnsupportedType
	}

	key := fmt.Sprintf("%s%d/%d/%s.jpg", u.photoPrefix(), userID, id, uuid.NewString())
	if err := u.photos.Put(ctx, key, bytes.NewReader(normalized), "image/jpeg", int64(len(normalized))); err != nil {
		return nil, fmt.Errorf("put memorial photo: %w", err)
	}
	oldKey, err := u.memorials.SetPhotoKey(ctx, id, userID, &key)
	if err != nil {
		_ = u.photos.Delete(ctx, key)
		return nil, fmt.Errorf("set memorial photo key: %w", err)
	}
	if oldKey != nil {
		_ = u.photos.Delete(ctx, *oldKey)
	}
	return u.Get(ctx, userID, id)
}

// RemovePhoto clears the photo. A memorial without a photo is already in the
// requested state, so it succeeds.
func (u *MemorialUsecase) RemovePhoto(ctx context.Context, userID, id int64) (*domain.Memorial, error) {
	if u.photos == nil {
		return nil, errors.New("memorial photo store is not configured")
	}
	oldKey, err := u.memorials.SetPhotoKey(ctx, id, userID, nil)
	if err != nil {
		return nil, fmt.Errorf("clear memorial photo key: %w", err)
	}
	if oldKey != nil {
		_ = u.photos.Delete(ctx, *oldKey)
	}
	return u.Get(ctx, userID, id)
}

func (u *MemorialUsecase) photoPrefix() string {
	if u.cfg.PhotoPrefix != "" {
		return u.cfg.PhotoPrefix
	}
	return "memorials/"
}

// attachPhotoURL fills m.PhotoURL with a fresh signed URL. An expired signature
// is returned so the caller can answer 403 and make the client ask again; any
// other failure only leaves the URL empty, because a photo that cannot be signed
// must not hide the rest of the memorial.
func (u *MemorialUsecase) attachPhotoURL(ctx context.Context, m *domain.Memorial) error {
	if m == nil || m.PhotoObjectKey == nil || u.photos == nil {
		return nil
	}
	ttl := u.cfg.PhotoURLTTL.Duration()
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	url, err := u.photos.SignedURL(ctx, *m.PhotoObjectKey, ttl)
	if err != nil {
		if errors.Is(err, domain.ErrPhotoURLExpired) {
			return err
		}
		return nil
	}
	m.PhotoURL = &url
	return nil
}

func invalid(field, message string) error {
	return &domain.MemorialValidationError{Field: field, Message: message}
}

// build validates the raw input and turns it into an entity (without ID/owner).
// Validation order follows the request's field order so the first reported
// field is the first broken one.
func (u *MemorialUsecase) build(in domain.MemorialInput) (*domain.Memorial, error) {
	m := &domain.Memorial{AddToOSReminder: in.AddToOSReminder}

	name := strings.TrimSpace(in.FullName)
	if name == "" {
		return nil, invalid("full_name", "full_name is required")
	}
	if utf8.RuneCountInString(name) > memorialFullNameMax {
		return nil, invalid("full_name", fmt.Sprintf("full_name must be at most %d characters", memorialFullNameMax))
	}
	m.FullName = name

	rel := domain.Relationship(in.Relationship)
	if !rel.Valid() {
		return nil, invalid("relationship", "relationship is not one of the allowed values")
	}
	m.Relationship = rel

	cal := domain.DeathCalendar(in.DeathCalendar)
	if cal != domain.DeathCalendarSolar && cal != domain.DeathCalendarLunar {
		return nil, invalid("death_calendar", "death_calendar must be solar or lunar")
	}
	m.DeathCalendar = cal

	if err := u.applyDeathDate(m, in); err != nil {
		return nil, err
	}

	if in.Note != nil {
		note := strings.TrimSpace(*in.Note)
		if utf8.RuneCountInString(note) > memorialNoteMax {
			return nil, invalid("note", fmt.Sprintf("note must be at most %d characters", memorialNoteMax))
		}
		if note != "" {
			m.Note = &note
		}
	}

	reminder, err := buildReminder(in.Reminder)
	if err != nil {
		return nil, err
	}
	m.Reminder = reminder

	if in.AddToOSReminder && reminder == nil {
		return nil, invalid("add_to_os_reminder", "add_to_os_reminder requires a reminder")
	}
	return m, nil
}

// applyDeathDate fills the death date in one of its two forms: a full solar date
// (year known) or day+month in the entered calendar (year unknown).
func (u *MemorialUsecase) applyDeathDate(m *domain.Memorial, in domain.MemorialInput) error {
	if in.DeathDate != "" {
		if in.DeathDay != nil || in.DeathMonth != nil || in.DeathLeapMonth != nil {
			return invalid("death_date", "death_date cannot be combined with death_day, death_month or death_leap_month")
		}
		d, err := time.Parse(time.DateOnly, in.DeathDate)
		if err != nil {
			return invalid("death_date", "death_date must be a valid YYYY-MM-DD date")
		}
		if d.Year() < memorialMinDeathYear {
			return invalid("death_date", fmt.Sprintf("death_date cannot be before %d-01-01", memorialMinDeathYear))
		}
		today := u.now().In(vietnamZone)
		if d.After(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)) {
			return invalid("death_date", "death_date cannot be in the future")
		}
		m.DeathDate = &d
		return nil
	}

	switch {
	case in.DeathDay == nil && in.DeathMonth == nil:
		return invalid("death_date", "death_date, or death_day and death_month, is required")
	case in.DeathMonth == nil:
		return invalid("death_month", "death_month is required when death_date is absent")
	case in.DeathDay == nil:
		return invalid("death_day", "death_day is required when death_date is absent")
	}

	month, day := *in.DeathMonth, *in.DeathDay
	if month < 1 || month > 12 {
		return invalid("death_month", "death_month must be 1..12")
	}
	maxDay := 30 // lunar: the server cannot know whether the month has 29 or 30 days
	if m.DeathCalendar == domain.DeathCalendarSolar {
		maxDay = daysInSolarMonth[month]
	}
	if day < 1 || day > maxDay {
		return invalid("death_day", fmt.Sprintf("death_day must be 1..%d for this month", maxDay))
	}

	leap := false
	if in.DeathLeapMonth != nil {
		leap = *in.DeathLeapMonth
	}
	if leap && m.DeathCalendar != domain.DeathCalendarLunar {
		return invalid("death_leap_month", "death_leap_month is only valid for the lunar calendar")
	}

	m.DeathDay, m.DeathMonth, m.DeathLeapMonth = &day, &month, &leap
	return nil
}

// buildReminder validates the optional reminder. nil in means no reminder.
func buildReminder(in *domain.ReminderInput) (*domain.Reminder, error) {
	if in == nil {
		return nil, nil
	}
	if in.DaysBefore == nil || *in.DaysBefore < 0 || *in.DaysBefore > memorialMaxReminderDay {
		return nil, invalid("reminder.days_before", fmt.Sprintf("days_before must be 0..%d", memorialMaxReminderDay))
	}
	if !memorialClockTime.MatchString(in.Time) {
		return nil, invalid("reminder.time", "time must be HH:mm (24h)")
	}
	zone := in.Timezone
	if zone == "" {
		zone = memorialDefaultZone
	}
	if zone == "Local" {
		return nil, invalid("reminder.timezone", "timezone must be an IANA zone name")
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return nil, invalid("reminder.timezone", "timezone must be an IANA zone name")
	}
	return &domain.Reminder{DaysBefore: *in.DaysBefore, Time: in.Time, Timezone: zone}, nil
}
