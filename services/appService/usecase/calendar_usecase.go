package usecase

import (
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

const (
	calendarDateLayout = "2006-01-02"
	calendarMinYear    = 1900
	calendarMaxYear    = 2100
	secondsPerDay      = 24 * 60 * 60
)

// CalendarUsecase picks the lunar-calendar photo for a date. The choice is a pure
// function of the date and the image list — no state, no randomness — so the same
// date always yields the same image.
type CalendarUsecase struct {
	images []domain.CalendarImage
	loc    *time.Location
	now    func() time.Time
}

// NewCalendarUsecase wires the use case. images must be non-empty and ordered
// deterministically (repository.ParseCalendarManifest sorts them); cfg.Timezone must be
// a valid IANA zone. now is the clock, injected so tests can fix "today".
func NewCalendarUsecase(images []domain.CalendarImage, cfg config.CalendarConfig, now func() time.Time) (*CalendarUsecase, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("calendar: no images")
	}
	if cfg.Timezone == "" { // LoadLocation("") silently means UTC
		return nil, fmt.Errorf("calendar timezone is required")
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("calendar timezone %q: %w", cfg.Timezone, err)
	}
	return &CalendarUsecase{images: images, loc: loc, now: now}, nil
}

// ImageForDate returns the date it resolved (YYYY-MM-DD) and its image. An empty date
// means today in the configured zone. A date that is not a real YYYY-MM-DD day, or whose
// year is outside 1900..2100, is domain.ErrInvalidDate.
func (u *CalendarUsecase) ImageForDate(date string) (string, domain.CalendarImage, error) {
	if date == "" {
		date = u.now().In(u.loc).Format(calendarDateLayout)
	}
	day, err := time.Parse(calendarDateLayout, date)
	if err != nil || day.Year() < calendarMinYear || day.Year() > calendarMaxYear {
		return "", domain.CalendarImage{}, domain.ErrInvalidDate
	}

	n := int64(len(u.images))
	dayNumber := day.Unix() / secondsPerDay // exact: day is UTC midnight
	idx := ((dayNumber % n) + n) % n
	return date, u.images[idx], nil
}
