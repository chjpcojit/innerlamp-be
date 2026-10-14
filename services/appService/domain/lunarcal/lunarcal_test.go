package lunarcal

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Mùng 1 Tết (lunar 1/1) per published Vietnamese calendars.
var tetTable = []struct {
	year int
	date time.Time
}{
	{1900, day(1900, 1, 31)},
	{2000, day(2000, 2, 5)},
	{2015, day(2015, 2, 19)},
	{2016, day(2016, 2, 8)},
	{2017, day(2017, 1, 28)},
	{2018, day(2018, 2, 16)},
	{2019, day(2019, 2, 5)},
	{2020, day(2020, 1, 25)},
	{2021, day(2021, 2, 12)},
	{2022, day(2022, 2, 1)},
	{2023, day(2023, 1, 22)},
	{2024, day(2024, 2, 10)},
	{2025, day(2025, 1, 29)},
	{2026, day(2026, 2, 17)},
}

func TestTetDates(t *testing.T) {
	for _, c := range tetTable {
		got, err := LunarToSolar(Date{Day: 1, Month: 1, Year: c.year})
		if err != nil {
			t.Fatalf("%d: %v", c.year, err)
		}
		if !got.Equal(c.date) {
			t.Errorf("Tết %d: got %s, want %s", c.year, got.Format("2006-01-02"), c.date.Format("2006-01-02"))
		}
		back := SolarToLunar(c.date)
		if back != (Date{Day: 1, Month: 1, Year: c.year}) {
			t.Errorf("SolarToLunar(%s) = %+v", c.date.Format("2006-01-02"), back)
		}
	}
}

func TestLeapMonth(t *testing.T) {
	cases := map[int]int{
		2004: 2, 2006: 7, 2009: 5, 2012: 4, 2014: 9, 2017: 6, 2020: 4, 2023: 2, 2025: 6,
		2021: 0, 2022: 0, 2024: 0, 2026: 0,
	}
	for y, want := range cases {
		if got := LeapMonth(y); got != want {
			t.Errorf("LeapMonth(%d) = %d, want %d", y, got, want)
		}
	}
}

func TestLeapMonthStartDates(t *testing.T) {
	// Nhuận tháng 2 năm 2023 bắt đầu 22/3/2023; nhuận tháng 6 năm 2025 bắt đầu 25/7/2025.
	cases := []struct {
		d    Date
		want time.Time
	}{
		{Date{1, 2, 2023, true}, day(2023, 3, 22)},
		{Date{1, 6, 2025, true}, day(2025, 7, 25)},
	}
	for _, c := range cases {
		got, err := LunarToSolar(c.d)
		if err != nil || !got.Equal(c.want) {
			t.Errorf("LunarToSolar(%+v) = %v, %v; want %v", c.d, got, err, c.want)
		}
	}
}

func TestLunarToSolarInvalid(t *testing.T) {
	cases := []Date{
		{1, 3, 2024, true}, // 2024 has no leap month
		{1, 5, 2023, true}, // leap month 5 of 2023 does not exist
		{31, 1, 2024, false},
		{0, 1, 2024, false},
		{1, 13, 2024, false},
	}
	for _, c := range cases {
		if _, err := LunarToSolar(c); err != ErrInvalidLunarDate {
			t.Errorf("LunarToSolar(%+v) err = %v, want ErrInvalidLunarDate", c, err)
		}
	}
}

func TestMonthStructure(t *testing.T) {
	for y := MinYear; y <= MaxYear; y++ {
		ms := LunarMonths(y)
		if len(ms) != 12 && len(ms) != 13 {
			t.Fatalf("%d: %d months", y, len(ms))
		}
		leaps := 0
		for i, m := range ms {
			if m.Days != 29 && m.Days != 30 {
				t.Fatalf("%d month %d: %d days", y, m.Month, m.Days)
			}
			if m.Leap {
				leaps++
			}
			if i > 0 && !ms[i-1].Start.AddDate(0, 0, ms[i-1].Days).Equal(m.Start) {
				t.Fatalf("%d: months not contiguous at %d", y, i)
			}
		}
		if (len(ms) == 13) != (leaps == 1) || leaps > 1 {
			t.Fatalf("%d: %d months but %d leap", y, len(ms), leaps)
		}
		if ms[0].Month != 1 || ms[0].Leap {
			t.Fatalf("%d: first month is %d", y, ms[0].Month)
		}
	}
}

func TestRoundTripEveryDay(t *testing.T) {
	for d := day(MinYear, 1, 1); d.Year() <= MaxYear; d = d.AddDate(0, 0, 1) {
		l := SolarToLunar(d)
		back, err := LunarToSolar(l)
		if err != nil || !back.Equal(d) {
			t.Fatalf("%s -> %+v -> %v (%v)", d.Format("2006-01-02"), l, back, err)
		}
	}
}

func TestGiaoThuaLength(t *testing.T) {
	// Tháng Chạp năm Giáp Thìn (2024) có 29 ngày: Giao thừa 28/1/2025 là 29/12.
	got := SolarToLunar(day(2025, 1, 28))
	if got != (Date{Day: 29, Month: 12, Year: 2024}) {
		t.Errorf("got %+v", got)
	}
	n, err := DaysInLunarMonth(2024, 12, false)
	if err != nil || n != 29 {
		t.Errorf("DaysInLunarMonth(2024,12) = %d, %v", n, err)
	}
}

func TestMidAutumnAndVuLan(t *testing.T) {
	cases := []struct {
		d    Date
		want time.Time
	}{
		{Date{15, 8, 2026, false}, day(2026, 9, 25)},
		{Date{15, 7, 2025, false}, day(2025, 9, 6)},
	}
	for _, c := range cases {
		got, err := LunarToSolar(c.d)
		if err != nil || !got.Equal(c.want) {
			t.Errorf("LunarToSolar(%+v) = %v, %v; want %v", c.d, got, err, c.want)
		}
	}
}
