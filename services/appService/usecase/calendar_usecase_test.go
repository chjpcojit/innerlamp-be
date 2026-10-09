package usecase

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func calendarImages(n int) []domain.CalendarImage {
	out := make([]domain.CalendarImage, n)
	for i := range out {
		out[i] = domain.CalendarImage{ID: fmt.Sprintf("img_%03d", i), File: fmt.Sprintf("img_%03d.jpg", i)}
	}
	return out
}

func newCalendarUsecase(t *testing.T, n int, now time.Time) *CalendarUsecase {
	t.Helper()
	u, err := NewCalendarUsecase(calendarImages(n), config.CalendarConfig{Timezone: "Asia/Ho_Chi_Minh"}, func() time.Time { return now })
	require.NoError(t, err)
	return u
}

func TestNewCalendarUsecase_RejectsBadSetup(t *testing.T) {
	now := func() time.Time { return time.Now() }
	_, err := NewCalendarUsecase(nil, config.CalendarConfig{Timezone: "Asia/Ho_Chi_Minh"}, now)
	assert.Error(t, err, "no images")
	_, err = NewCalendarUsecase(calendarImages(3), config.CalendarConfig{Timezone: "Mars/Olympus"}, now)
	assert.Error(t, err, "unknown zone")
	_, err = NewCalendarUsecase(calendarImages(3), config.CalendarConfig{}, now)
	assert.Error(t, err, "empty zone")
}

func TestCalendarUsecase_ImageForDate(t *testing.T) {
	u := newCalendarUsecase(t, 100, time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))

	t.Run("same date always gives the same image", func(t *testing.T) {
		d1, a, err := u.ImageForDate("2026-10-07")
		require.NoError(t, err)
		d2, b, err := u.ImageForDate("2026-10-07")
		require.NoError(t, err)
		assert.Equal(t, "2026-10-07", d1)
		assert.Equal(t, d1, d2)
		assert.Equal(t, a, b)
	})

	t.Run("consecutive dates give different images until the list wraps", func(t *testing.T) {
		seen := map[string]bool{}
		day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < 100; i++ {
			_, img, err := u.ImageForDate(day.AddDate(0, 0, i).Format("2006-01-02"))
			require.NoError(t, err)
			seen[img.ID] = true
		}
		assert.Len(t, seen, 100, "every image appears once per 100-day cycle")

		_, first, _ := u.ImageForDate("2026-01-01")
		_, again, _ := u.ImageForDate(day.AddDate(0, 0, 100).Format("2006-01-02"))
		assert.Equal(t, first, again, "wraps after N days")
	})

	t.Run("maps day number mod N (1970-01-01 is day 0)", func(t *testing.T) {
		_, img, err := u.ImageForDate("1970-01-01")
		require.NoError(t, err)
		assert.Equal(t, "img_000", img.ID)
		_, img, err = u.ImageForDate("1970-01-02")
		require.NoError(t, err)
		assert.Equal(t, "img_001", img.ID)
	})

	t.Run("dates before 1970 stay in range", func(t *testing.T) {
		_, img, err := u.ImageForDate("1969-12-31") // day -1 -> last image
		require.NoError(t, err)
		assert.Equal(t, "img_099", img.ID)
		_, _, err = u.ImageForDate("1900-01-01")
		require.NoError(t, err)
	})

	t.Run("year bounds are inclusive", func(t *testing.T) {
		_, _, err := u.ImageForDate("1900-01-01")
		assert.NoError(t, err)
		_, _, err = u.ImageForDate("2100-12-31")
		assert.NoError(t, err)
		_, _, err = u.ImageForDate("1899-12-31")
		assert.ErrorIs(t, err, domain.ErrInvalidDate)
		_, _, err = u.ImageForDate("2101-01-01")
		assert.ErrorIs(t, err, domain.ErrInvalidDate)
	})

	for _, bad := range []string{"2026-02-30", "2026-1-5", "07-10-2026", "2026/10/07", "abc", "2026-13-01", " 2026-10-07", "2026-10-07T00:00:00Z"} {
		t.Run("rejects "+bad, func(t *testing.T) {
			_, _, err := u.ImageForDate(bad)
			assert.ErrorIs(t, err, domain.ErrInvalidDate)
		})
	}
}

func TestCalendarUsecase_EmptyDateIsTodayInConfiguredZone(t *testing.T) {
	// 18:00 UTC on 6 Oct is already 7 Oct 01:00 in Ho Chi Minh (UTC+7).
	u := newCalendarUsecase(t, 100, time.Date(2026, 10, 6, 18, 0, 0, 0, time.UTC))

	date, img, err := u.ImageForDate("")

	require.NoError(t, err)
	assert.Equal(t, "2026-10-07", date)
	_, want, _ := u.ImageForDate("2026-10-07")
	assert.Equal(t, want, img)
}
