package lunarcal

import (
	"testing"
	"time"
)

// Published Vietnam-time (UTC+7) solar term dates.
func TestSolarTermDate(t *testing.T) {
	cases := []struct {
		year, lon int
		want      time.Time
	}{
		{2024, 315, day(2024, 2, 4)},   // Lập xuân
		{2024, 0, day(2024, 3, 20)},    // Xuân phân
		{2024, 15, day(2024, 4, 4)},    // Thanh minh
		{2024, 90, day(2024, 6, 21)},   // Hạ chí
		{2024, 180, day(2024, 9, 22)},  // Thu phân
		{2024, 270, day(2024, 12, 21)}, // Đông chí
		{2025, 270, day(2025, 12, 21)},
		{2026, 315, day(2026, 2, 4)},
		{2026, 0, day(2026, 3, 20)},
		{2026, 15, day(2026, 4, 5)},
		{2026, 90, day(2026, 6, 21)},
	}
	for _, c := range cases {
		got, ok := SolarTermDate(c.year, c.lon)
		if !ok || !got.Equal(c.want) {
			t.Errorf("SolarTermDate(%d,%d) = %s, %v; want %s", c.year, c.lon,
				got.Format("2006-01-02"), ok, c.want.Format("2006-01-02"))
		}
	}
}

func TestSolarTermDateInvalid(t *testing.T) {
	for _, lon := range []int{-15, 7, 360, 400} {
		if _, ok := SolarTermDate(2026, lon); ok {
			t.Errorf("longitude %d accepted", lon)
		}
	}
}

func TestSolarTermsEveryYear(t *testing.T) {
	for y := MinYear; y <= MaxYear; y++ {
		var prev time.Time
		for lon := 0; lon < 360; lon += 15 {
			d, ok := SolarTermDate(y, lon)
			if !ok {
				t.Fatalf("%d/%d: no date", y, lon)
			}
			_ = prev
			prev = d
		}
	}
}
