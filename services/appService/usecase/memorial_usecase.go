package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
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
	cfg       config.MemorialConfig
	now       func() time.Time
}

// NewMemorialUsecase wires the use case.
func NewMemorialUsecase(memorials domain.MemorialRepository, cfg config.MemorialConfig) *MemorialUsecase {
	return &MemorialUsecase{memorials: memorials, cfg: cfg, now: time.Now}
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
	return items, nil
}

// Get returns one of the user's memorials, or domain.ErrMemorialNotFound.
func (u *MemorialUsecase) Get(ctx context.Context, userID, id int64) (*domain.Memorial, error) {
	m, err := u.memorials.FindByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("get memorial %d: %w", id, err)
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
	return updated, nil
}

// Delete removes one of the user's memorials, or returns domain.ErrMemorialNotFound.
func (u *MemorialUsecase) Delete(ctx context.Context, userID, id int64) error {
	if err := u.memorials.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("delete memorial %d: %w", id, err)
	}
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
