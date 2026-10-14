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

func calendarEventCfg() config.CalendarConfig {
	return config.CalendarConfig{Timezone: "Asia/Ho_Chi_Minh", DefaultLanguage: "vi"}
}

func solarEvent(code string, order, m, d int, groups ...string) domain.CalendarEvent {
	return domain.CalendarEvent{
		Code: code, Name: code, SortOrder: order, Groups: groups,
		Rule: domain.RecurrenceRule{Kind: domain.KindSolar, SolarMonth: m, SolarDay: d},
	}
}

func newEventUsecase(t *testing.T, repo *domain.MockCalendarEventRepository, now time.Time) *CalendarEventUsecase {
	t.Helper()
	u, err := NewCalendarEventUsecase(repo, calendarEventCfg(), func() time.Time { return now }, newLogger())
	require.NoError(t, err)
	return u
}

func stubRepo(groups []domain.CalendarGroup, events []domain.CalendarEvent) *domain.MockCalendarEventRepository {
	repo := new(domain.MockCalendarEventRepository)
	repo.On("ListGroups", mock.Anything).Return(groups, nil)
	repo.On("ListActiveEvents", mock.Anything, "vi").Return(events, nil)
	return repo
}

var someGroups = []domain.CalendarGroup{
	{Code: "folk", Name: "Dân gian", SortOrder: 1},
	{Code: "vietnam", Name: "Việt Nam", SortOrder: 2},
}

func TestEventsForYear_ExplicitYear(t *testing.T) {
	repo := stubRepo(someGroups, []domain.CalendarEvent{
		solarEvent("a", 1, 4, 30, "vietnam"),
		solarEvent("b", 2, 5, 1, "vietnam", "folk"),
	})
	u := newEventUsecase(t, repo, time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC))

	res, err := u.EventsForYear(context.Background(), "2026")
	require.NoError(t, err)
	assert.Equal(t, 2026, res.Year)
	assert.Equal(t, someGroups, res.Groups)
	require.Len(t, res.Events, 2)
	require.Len(t, res.Occurrences, 2)
	assert.Equal(t, "a", res.Occurrences[0].EventCode)
	assert.Equal(t, time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), res.Occurrences[0].SolarDate)
}

func TestEventsForYear_EmptyMeansCurrentYearInConfiguredZone(t *testing.T) {
	repo := stubRepo(nil, nil)
	// 2026-12-31 20:00 UTC is already 2027-01-01 in UTC+7.
	u := newEventUsecase(t, repo, time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC))
	res, err := u.EventsForYear(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, 2027, res.Year)
}

func TestEventsForYear_InvalidYear(t *testing.T) {
	repo := new(domain.MockCalendarEventRepository)
	u := newEventUsecase(t, repo, time.Now())
	for _, y := range []string{"abc", "2026.5", " 2026", "2026 ", "+2026", "-2026", "1899", "2101", "0", "99999999999999999999", "٢٠٢٦"} {
		_, err := u.EventsForYear(context.Background(), y)
		assert.ErrorIs(t, err, domain.ErrInvalidYear, "year %q", y)
	}
	repo.AssertNotCalled(t, "ListGroups", mock.Anything)
	repo.AssertNotCalled(t, "ListActiveEvents", mock.Anything, mock.Anything)
}

func TestEventsForYear_Boundaries(t *testing.T) {
	repo := stubRepo(nil, nil)
	u := newEventUsecase(t, repo, time.Now())
	for _, y := range []string{"1900", "2100"} {
		res, err := u.EventsForYear(context.Background(), y)
		require.NoError(t, err, y)
		assert.NotEmpty(t, res.Year)
	}
}

func TestEventsForYear_OnlyEventsWithAnOccurrence(t *testing.T) {
	leapDay := solarEvent("feb29", 1, 2, 29, "folk")
	repo := stubRepo(someGroups, []domain.CalendarEvent{leapDay, solarEvent("may1", 2, 5, 1, "vietnam")})
	u := newEventUsecase(t, repo, time.Now())

	res, err := u.EventsForYear(context.Background(), "2025") // no 29/2
	require.NoError(t, err)
	require.Len(t, res.Events, 1)
	assert.Equal(t, "may1", res.Events[0].Code)
	assert.Len(t, res.Groups, 2, "every group is listed")
}

func TestEventsForYear_DropsUnknownGroupCodes(t *testing.T) {
	repo := stubRepo(someGroups, []domain.CalendarEvent{solarEvent("a", 1, 4, 30, "vietnam", "gone")})
	u := newEventUsecase(t, repo, time.Now())
	res, err := u.EventsForYear(context.Background(), "2026")
	require.NoError(t, err)
	require.Len(t, res.Events, 1)
	assert.Equal(t, []string{"vietnam"}, res.Events[0].Groups)
}

func TestEventsForYear_SkipsInvalidRuleAndKeepsTheRest(t *testing.T) {
	bad := domain.CalendarEvent{Code: "bad", Rule: domain.RecurrenceRule{Kind: "weekly"}, Groups: []string{"folk"}}
	repo := stubRepo(someGroups, []domain.CalendarEvent{bad, solarEvent("ok", 2, 4, 30, "folk")})
	logger := newLogger()
	u, err := NewCalendarEventUsecase(repo, calendarEventCfg(), time.Now, logger)
	require.NoError(t, err)

	res, err := u.EventsForYear(context.Background(), "2026")
	require.NoError(t, err)
	require.Len(t, res.Events, 1)
	assert.Equal(t, "ok", res.Events[0].Code)
	logger.AssertCalled(t, "Warn", mock.Anything, mock.Anything)
}

func TestEventsForYear_RepositoryErrorsPropagate(t *testing.T) {
	boom := errors.New("db down")

	repo := new(domain.MockCalendarEventRepository)
	repo.On("ListGroups", mock.Anything).Return(nil, boom)
	_, err := newEventUsecase(t, repo, time.Now()).EventsForYear(context.Background(), "2026")
	assert.ErrorIs(t, err, boom)

	repo = new(domain.MockCalendarEventRepository)
	repo.On("ListGroups", mock.Anything).Return(someGroups, nil)
	repo.On("ListActiveEvents", mock.Anything, "vi").Return(nil, boom)
	_, err = newEventUsecase(t, repo, time.Now()).EventsForYear(context.Background(), "2026")
	assert.ErrorIs(t, err, boom)
}

func TestEventsForYear_EmptyCatalogueGivesEmptySlices(t *testing.T) {
	u := newEventUsecase(t, stubRepo(nil, nil), time.Now())
	res, err := u.EventsForYear(context.Background(), "2026")
	require.NoError(t, err)
	assert.NotNil(t, res.Groups)
	assert.NotNil(t, res.Events)
	assert.Empty(t, res.Occurrences)
}

func TestNewCalendarEventUsecase_Validates(t *testing.T) {
	repo := new(domain.MockCalendarEventRepository)
	for name, cfg := range map[string]config.CalendarConfig{
		"no language":  {Timezone: "Asia/Ho_Chi_Minh"},
		"no timezone":  {DefaultLanguage: "vi"},
		"bad timezone": {Timezone: "Mars/Olympus", DefaultLanguage: "vi"},
	} {
		_, err := NewCalendarEventUsecase(repo, cfg, time.Now, newLogger())
		assert.Error(t, err, name)
	}
}
