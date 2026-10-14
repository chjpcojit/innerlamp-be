package domain

import (
	"testing"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain/lunarcal"
)

func ymd(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func ev(code string, order int, r RecurrenceRule) CalendarEvent {
	return CalendarEvent{Code: code, SortOrder: order, Rule: r}
}

func resolveOne(r RecurrenceRule, year int) []CalendarOccurrence {
	return ResolveOccurrences([]CalendarEvent{ev("x", 1, r)}, year)
}

// scanLunar is an oracle: it walks every solar day of year and keeps the days
// whose lunar date satisfies keep.
func scanLunar(year int, keep func(today, tomorrow lunarcal.Date) bool) []time.Time {
	var out []time.Time
	for d := ymd(year, 1, 1); d.Year() == year; d = d.AddDate(0, 0, 1) {
		if keep(lunarcal.SolarToLunar(d), lunarcal.SolarToLunar(d.AddDate(0, 0, 1))) {
			out = append(out, d)
		}
	}
	return out
}

func dates(occ []CalendarOccurrence) []time.Time {
	out := make([]time.Time, len(occ))
	for i, o := range occ {
		out[i] = o.SolarDate
	}
	return out
}

func equalDates(a, b []time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

func TestResolveSolar(t *testing.T) {
	occ := resolveOne(RecurrenceRule{Kind: KindSolar, SolarMonth: 4, SolarDay: 30}, 2026)
	if len(occ) != 1 || !occ[0].SolarDate.Equal(ymd(2026, 4, 30)) {
		t.Fatalf("got %+v", occ)
	}
	if occ[0].LunarDate.Day == 0 || occ[0].LunarDate.Year != 2026 {
		t.Errorf("lunar date missing: %+v", occ[0].LunarDate)
	}
}

func TestResolveSolarFeb29(t *testing.T) {
	r := RecurrenceRule{Kind: KindSolar, SolarMonth: 2, SolarDay: 29}
	if got := resolveOne(r, 2025); len(got) != 0 {
		t.Errorf("2025 should have no 29/2, got %+v", got)
	}
	if got := resolveOne(r, 2024); len(got) != 1 || !got[0].SolarDate.Equal(ymd(2024, 2, 29)) {
		t.Errorf("2024: %+v", got)
	}
}

func TestResolveLunarTet(t *testing.T) {
	occ := resolveOne(RecurrenceRule{Kind: KindLunar, LunarMonth: 1, LunarDay: 1}, 2026)
	if len(occ) != 1 || !occ[0].SolarDate.Equal(ymd(2026, 2, 17)) {
		t.Fatalf("got %+v", occ)
	}
	if occ[0].LunarDate != (CalendarLunarDate{Day: 1, Month: 1, Year: 2026}) {
		t.Errorf("lunar date %+v", occ[0].LunarDate)
	}
}

// E11: tháng Chạp events belong to the solar year of their solar date, whose
// lunar year is the previous one.
func TestResolveLunarDecemberBelongsToSolarYear(t *testing.T) {
	r := RecurrenceRule{Kind: KindLunar, LunarMonth: 12, LunarDay: 23}
	for year := 2000; year <= 2040; year++ {
		occ := resolveOne(r, year)
		want := scanLunar(year, func(d, _ lunarcal.Date) bool { return d.Month == 12 && d.Day == 23 && !d.IsLeapMonth })
		if !equalDates(dates(occ), want) {
			t.Fatalf("%d: got %v want %v", year, dates(occ), want)
		}
		for _, o := range occ {
			if o.SolarDate.Year() != year {
				t.Errorf("%d: occurrence in %d", year, o.SolarDate.Year())
			}
			if o.SolarDate.Month() > 3 && o.LunarDate.Year != year {
				t.Errorf("%d: lunar year %d", year, o.LunarDate.Year)
			}
		}
	}
}

func TestResolveLunarLeapMonth(t *testing.T) {
	leap := RecurrenceRule{Kind: KindLunar, LunarMonth: 6, LunarDay: 15, IsLeapMonth: true}
	reg := RecurrenceRule{Kind: KindLunar, LunarMonth: 6, LunarDay: 15}
	// 2025 has a leap month 6.
	if got := resolveOne(leap, 2025); len(got) != 1 || !got[0].LunarDate.IsLeapMonth {
		t.Errorf("leap 2025: %+v", got)
	}
	if got := resolveOne(reg, 2025); len(got) != 1 || got[0].LunarDate.IsLeapMonth {
		t.Errorf("regular 2025: %+v", got)
	}
	// 2024 has none.
	if got := resolveOne(leap, 2024); len(got) != 0 {
		t.Errorf("leap 2024 should be empty: %+v", got)
	}
}

func TestResolveLunarMonthly(t *testing.T) {
	for _, day := range []int{1, 15, 30} {
		r := RecurrenceRule{Kind: KindLunarMonthly, LunarDay: day}
		for year := 2000; year <= 2040; year++ {
			got := dates(resolveOne(r, year))
			want := scanLunar(year, func(d, _ lunarcal.Date) bool { return d.Day == day })
			if !equalDates(got, want) {
				t.Fatalf("day %d year %d: got %d dates, want %d", day, year, len(got), len(want))
			}
		}
	}
}

func TestResolveLunarMonthlyLeapYearHasExtra(t *testing.T) {
	occ := resolveOne(RecurrenceRule{Kind: KindLunarMonthly, LunarDay: 15}, 2025)
	leaps := 0
	for _, o := range occ {
		if o.LunarDate.IsLeapMonth {
			leaps++
		}
	}
	if leaps != 1 {
		t.Errorf("2025 Rằm in leap month: %d, want 1", leaps)
	}
}

// E9: Giao thừa is day 30 of month 12 when it has 30 days, otherwise 29.
func TestResolveLunarLastDay(t *testing.T) {
	r := RecurrenceRule{Kind: KindLunarLastDay, LunarMonth: 12}
	seen := map[int]bool{}
	for year := 2000; year <= 2040; year++ {
		occ := resolveOne(r, year)
		want := scanLunar(year, func(d, tomorrow lunarcal.Date) bool {
			return d.Month == 12 && !d.IsLeapMonth && (tomorrow.Month != 12 || tomorrow.IsLeapMonth)
		})
		if !equalDates(dates(occ), want) {
			t.Fatalf("%d: got %v want %v", year, dates(occ), want)
		}
		for _, o := range occ {
			seen[o.LunarDate.Day] = true
			next := lunarcal.SolarToLunar(o.SolarDate.AddDate(0, 0, 1))
			if next.Day != 1 || next.Month != 1 {
				t.Errorf("%d: day after Giao thừa is %+v, want 1/1", year, next)
			}
		}
	}
	if !seen[29] || !seen[30] {
		t.Errorf("expected both 29 and 30 over 2000..2040, saw %v", seen)
	}
}

func TestResolveSolarTerm(t *testing.T) {
	occ := resolveOne(RecurrenceRule{Kind: KindSolarTerm, SolarTermLongitude: 15}, 2026)
	if len(occ) != 1 || !occ[0].SolarDate.Equal(ymd(2026, 4, 5)) {
		t.Fatalf("Thanh minh 2026: %+v", occ)
	}
}

// E12: ordered by solar date, then SortOrder, then code; stable across calls.
func TestResolveOrdering(t *testing.T) {
	solar := func(m, d int) RecurrenceRule { return RecurrenceRule{Kind: KindSolar, SolarMonth: m, SolarDay: d} }
	events := []CalendarEvent{
		ev("b", 2, solar(5, 1)),
		ev("z", 1, solar(5, 1)),
		ev("a", 2, solar(5, 1)),
		ev("early", 9, solar(1, 1)),
	}
	got := ResolveOccurrences(events, 2026)
	var codes []string
	for _, o := range got {
		codes = append(codes, o.EventCode)
	}
	want := []string{"early", "z", "a", "b"}
	if len(codes) != len(want) {
		t.Fatalf("got %v", codes)
	}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("got %v want %v", codes, want)
		}
	}
	again := ResolveOccurrences(events, 2026)
	for i := range got {
		if got[i] != again[i] {
			t.Fatalf("not deterministic at %d", i)
		}
	}
}

func TestResolveSkipsInvalidRules(t *testing.T) {
	events := []CalendarEvent{
		ev("unknown", 1, RecurrenceRule{Kind: "weekly"}),
		ev("bad-term", 2, RecurrenceRule{Kind: KindSolarTerm, SolarTermLongitude: 7}),
		ev("bad-day", 3, RecurrenceRule{Kind: KindSolar, SolarMonth: 4, SolarDay: 31}),
		ev("ok", 4, RecurrenceRule{Kind: KindSolar, SolarMonth: 4, SolarDay: 30}),
	}
	got := ResolveOccurrences(events, 2026)
	if len(got) != 1 || got[0].EventCode != "ok" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveEmpty(t *testing.T) {
	if got := ResolveOccurrences(nil, 2026); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}
