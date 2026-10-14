package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

var teZone = time.FixedZone("ICT", 7*3600)

func templeEventCfg() config.TempleEventConfig {
	return config.TempleEventConfig{Timezone: "Asia/Ho_Chi_Minh", DefaultWindowMonths: 3, MaxWindowMonths: 3}
}

// 2026-10-13 09:00 in Vietnam.
var teNow = time.Date(2026, 10, 13, 9, 0, 0, 0, teZone)

func teDay(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func newEventsUsecase(t *testing.T, temples *domain.MockTempleRepository, events *domain.MockTempleEventRepository) *TempleEventUsecase {
	t.Helper()
	u, err := NewTempleEventUsecase(temples, events, templeEventCfg(), func() time.Time { return teNow }, newLogger())
	require.NoError(t, err)
	return u
}

func existingTemple() *domain.MockTempleRepository {
	r := new(domain.MockTempleRepository)
	r.On("FindByID", mock.Anything, "t1").Return(&domain.Temple{ID: "t1"}, nil)
	return r
}

func oneOff(id string, start time.Time) domain.TempleEvent {
	return domain.TempleEvent{
		ID: id, TempleID: "t1", Title: id, Kind: domain.TempleEventKhac, Status: domain.TempleEventScheduled,
		StartsAt: start, EndsAt: start.Add(2 * time.Hour), Recurrence: domain.RecurrenceNone,
	}
}

func TestNewTempleEventUsecase_RejectsBadConfig(t *testing.T) {
	cfg := templeEventCfg()
	cfg.Timezone = "Mars/Olympus"
	_, err := NewTempleEventUsecase(existingTemple(), new(domain.MockTempleEventRepository), cfg, time.Now, newLogger())
	assert.Error(t, err)

	cfg = templeEventCfg()
	cfg.Timezone = ""
	_, err = NewTempleEventUsecase(existingTemple(), new(domain.MockTempleEventRepository), cfg, time.Now, newLogger())
	assert.Error(t, err)
}

func TestEventsForTemple_Window(t *testing.T) {
	ctx := context.Background()

	t.Run("defaults: today to today + default_window_months", func(t *testing.T) {
		events := new(domain.MockTempleEventRepository)
		events.On("ListActiveByTemple", mock.Anything, "t1",
			domain.TempleEventWindow{From: teDay(2026, 10, 13), To: teDay(2027, 1, 13)}).Return([]domain.TempleEvent{}, nil)

		res, err := newEventsUsecase(t, existingTemple(), events).EventsForTemple(ctx, "t1", "", "")

		require.NoError(t, err)
		assert.Equal(t, teDay(2026, 10, 13), res.From)
		assert.Equal(t, teDay(2027, 1, 13), res.To)
		assert.NotNil(t, res.Events)
		assert.NotNil(t, res.Occurrences)
		assert.Equal(t, "t1", res.TempleID)
	})

	t.Run("from only: to is from + default window", func(t *testing.T) {
		events := new(domain.MockTempleEventRepository)
		events.On("ListActiveByTemple", mock.Anything, "t1",
			domain.TempleEventWindow{From: teDay(2026, 11, 1), To: teDay(2027, 2, 1)}).Return([]domain.TempleEvent{}, nil)

		res, err := newEventsUsecase(t, existingTemple(), events).EventsForTemple(ctx, "t1", "2026-11-01", "")

		require.NoError(t, err)
		assert.Equal(t, teDay(2027, 2, 1), res.To)
	})

	t.Run("to only: from is today", func(t *testing.T) {
		events := new(domain.MockTempleEventRepository)
		events.On("ListActiveByTemple", mock.Anything, "t1",
			domain.TempleEventWindow{From: teDay(2026, 10, 13), To: teDay(2026, 11, 1)}).Return([]domain.TempleEvent{}, nil)

		_, err := newEventsUsecase(t, existingTemple(), events).EventsForTemple(ctx, "t1", "", "2026-11-01")

		require.NoError(t, err)
	})

	t.Run("exactly the maximum window is accepted", func(t *testing.T) {
		events := new(domain.MockTempleEventRepository)
		events.On("ListActiveByTemple", mock.Anything, "t1", mock.Anything).Return([]domain.TempleEvent{}, nil)

		_, err := newEventsUsecase(t, existingTemple(), events).EventsForTemple(ctx, "t1", "2026-10-13", "2027-01-13")

		assert.NoError(t, err)
	})

	bad := []struct {
		name, from, to string
		want           error
		field          string
	}{
		{"from not a date", "tomorrow", "", domain.ErrInvalidDate, "from"},
		{"from impossible date", "2026-02-30", "", domain.ErrInvalidDate, "from"},
		{"to not a date", "", "2026/11/01", domain.ErrInvalidDate, "to"},
		{"from year too low", "1899-12-31", "", domain.ErrInvalidDate, "from"},
		{"to year too high", "", "2101-01-01", domain.ErrInvalidDate, "to"},
		{"to before from", "2026-11-01", "2026-10-31", domain.ErrInvalidDateRange, ""},
		{"window over the maximum", "2026-10-13", "2027-01-14", domain.ErrDateRangeTooLarge, ""},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			temples := new(domain.MockTempleRepository)
			events := new(domain.MockTempleEventRepository)

			_, err := newEventsUsecase(t, temples, events).EventsForTemple(ctx, "t1", tc.from, tc.to)

			require.ErrorIs(t, err, tc.want)
			if tc.field != "" {
				fe, ok := errors.AsType[*domain.InvalidDateFieldError](err)
				require.True(t, ok)
				assert.Equal(t, tc.field, fe.Field)
			}
			temples.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
			events.AssertNotCalled(t, "ListActiveByTemple", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestEventsForTemple_Reads(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("db down")

	t.Run("unknown temple", func(t *testing.T) {
		temples := new(domain.MockTempleRepository)
		temples.On("FindByID", mock.Anything, "t1").Return(nil, domain.ErrTempleNotFound)
		events := new(domain.MockTempleEventRepository)

		_, err := newEventsUsecase(t, temples, events).EventsForTemple(ctx, "t1", "", "")

		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
		events.AssertNotCalled(t, "ListActiveByTemple", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("temple lookup failure", func(t *testing.T) {
		temples := new(domain.MockTempleRepository)
		temples.On("FindByID", mock.Anything, "t1").Return(nil, dbErr)

		_, err := newEventsUsecase(t, temples, new(domain.MockTempleEventRepository)).EventsForTemple(ctx, "t1", "", "")

		assert.ErrorIs(t, err, dbErr)
		assert.NotErrorIs(t, err, domain.ErrTempleNotFound)
	})

	t.Run("event read failure", func(t *testing.T) {
		events := new(domain.MockTempleEventRepository)
		events.On("ListActiveByTemple", mock.Anything, "t1", mock.Anything).Return(nil, dbErr)

		_, err := newEventsUsecase(t, existingTemple(), events).EventsForTemple(ctx, "t1", "", "")

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("returns only events that have an occurrence, sorted; drops past and invalid", func(t *testing.T) {
		past := oneOff("past", time.Date(2026, 10, 13, 6, 0, 0, 0, teZone)) // ended 08:00, now is 09:00
		later := oneOff("later", time.Date(2026, 10, 20, 19, 0, 0, 0, teZone))
		sooner := oneOff("sooner", time.Date(2026, 10, 14, 8, 0, 0, 0, teZone))
		invalid := oneOff("invalid", time.Date(2026, 10, 15, 8, 0, 0, 0, teZone))
		invalid.Kind = "NOPE"
		farOut := oneOff("far", time.Date(2026, 12, 25, 8, 0, 0, 0, teZone))
		farOut.StartsAt = time.Date(2027, 6, 1, 8, 0, 0, 0, teZone) // outside the window though returned by the coarse read
		farOut.EndsAt = farOut.StartsAt.Add(time.Hour)
		events := new(domain.MockTempleEventRepository)
		events.On("ListActiveByTemple", mock.Anything, "t1", mock.Anything).
			Return([]domain.TempleEvent{later, past, invalid, farOut, sooner}, nil)

		res, err := newEventsUsecase(t, existingTemple(), events).EventsForTemple(ctx, "t1", "", "")

		require.NoError(t, err)
		require.Len(t, res.Occurrences, 2)
		assert.Equal(t, "sooner", res.Occurrences[0].EventID)
		assert.Equal(t, "later", res.Occurrences[1].EventID)
		require.Len(t, res.Events, 2)
		ids := []string{res.Events[0].ID, res.Events[1].ID}
		assert.ElementsMatch(t, []string{"sooner", "later"}, ids)
	})

	t.Run("a recurring event appears once in events with several occurrences", func(t *testing.T) {
		weekly := oneOff("w", time.Date(2026, 10, 4, 8, 0, 0, 0, teZone)) // a Sunday
		weekly.Recurrence = domain.RecurrenceWeekly
		sunday := 0
		weekly.Weekday = &sunday
		events := new(domain.MockTempleEventRepository)
		events.On("ListActiveByTemple", mock.Anything, "t1", mock.Anything).Return([]domain.TempleEvent{weekly}, nil)

		res, err := newEventsUsecase(t, existingTemple(), events).EventsForTemple(ctx, "t1", "2026-10-13", "2026-11-02")

		require.NoError(t, err)
		assert.Len(t, res.Events, 1)
		assert.Len(t, res.Occurrences, 3) // 10-18, 10-25, 11-01
	})

	t.Run("window ending before today is empty, not an error", func(t *testing.T) {
		events := new(domain.MockTempleEventRepository)
		events.On("ListActiveByTemple", mock.Anything, "t1", mock.Anything).
			Return([]domain.TempleEvent{oneOff("old", time.Date(2026, 9, 1, 8, 0, 0, 0, teZone))}, nil)

		res, err := newEventsUsecase(t, existingTemple(), events).EventsForTemple(ctx, "t1", "2026-09-01", "2026-09-30")

		require.NoError(t, err)
		assert.Empty(t, res.Events)
		assert.Empty(t, res.Occurrences)
		assert.NotNil(t, res.Events)
	})
}
