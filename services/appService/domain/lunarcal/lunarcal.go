// Package lunarcal converts between the Gregorian and the Vietnamese lunar
// calendar and computes solar terms. It is pure computation: standard
// library only, no I/O, no clock.
//
// The algorithm is Hồ Ngọc Đức's (2004): new moons and the Sun's ecliptic
// longitude from Julian-day series, at UTC+7 (the Vietnamese calendar's time
// zone since 1968; the algorithm applies UTC+7 to the whole range). The
// verified range is lunar/solar years 1900..2100 (docs/specs/features/calendar,
// E20); outside it the series degrade and results are not guaranteed.
package lunarcal

import (
	"errors"
	"math"
	"time"
)

// TimeZone is the offset, in hours, at which the calendar is computed.
const TimeZone = 7

// MinYear and MaxYear bound the range the converter is verified for.
const (
	MinYear = 1900
	MaxYear = 2100
)

// ErrInvalidLunarDate is returned when a lunar date does not exist: a leap
// month the year does not have, or a day beyond the month's length.
var ErrInvalidLunarDate = errors.New("lunarcal: lunar date does not exist")

// Date is a lunar date. Year is the lunar year (not the solar year the day
// falls in: lunar 29/12 can be in January of the next solar year).
type Date struct {
	Day         int
	Month       int
	Year        int
	IsLeapMonth bool
}

// Month describes one lunar month: its number, whether it is the leap month,
// its length (29 or 30) and the solar date of its first day (UTC midnight).
type Month struct {
	Month int
	Leap  bool
	Days  int
	Start time.Time
}

const synodicMonth = 29.530588853

// jdFromDate returns the Julian day number of a Gregorian civil date.
func jdFromDate(d, m, y int) int {
	a := (14 - m) / 12
	yy := y + 4800 - a
	mm := m + 12*a - 3
	return d + (153*mm+2)/5 + 365*yy + yy/4 - yy/100 + yy/400 - 32045
}

// dateFromJD is the inverse of jdFromDate, as a UTC-midnight time.
func dateFromJD(jd int) time.Time {
	a := jd + 32044
	b := (4*a + 3) / 146097
	c := a - b*146097/4
	d := (4*c + 3) / 1461
	e := c - 1461*d/4
	m := (5*e + 2) / 153
	day := e - (153*m+2)/5 + 1
	month := m + 3 - 12*(m/10)
	year := b*100 + d - 4800 + m/10
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// newMoon returns the Julian date (UT) of the k-th new moon since 1900-01-01.
func newMoon(k int) float64 {
	const dr = math.Pi / 180
	kf := float64(k)
	t := kf / 1236.85
	t2, t3 := t*t, t*t*t
	jd1 := 2415020.75933 + synodicMonth*kf + 0.0001178*t2 - 0.000000155*t3
	jd1 += 0.00033 * math.Sin((166.56+132.87*t-0.009173*t2)*dr)
	m := 359.2242 + 29.10535608*kf - 0.0000333*t2 - 0.00000347*t3
	mpr := 306.0253 + 385.81691806*kf + 0.0107306*t2 + 0.00001236*t3
	f := 21.2964 + 390.67050646*kf - 0.0016528*t2 - 0.00000239*t3
	c1 := (0.1734-0.000393*t)*math.Sin(m*dr) + 0.0021*math.Sin(2*dr*m)
	c1 -= 0.4068*math.Sin(mpr*dr) - 0.0161*math.Sin(dr*2*mpr)
	c1 -= 0.0004 * math.Sin(dr*3*mpr)
	c1 += 0.0104*math.Sin(dr*2*f) - 0.0051*math.Sin(dr*(m+mpr))
	c1 += -0.0074*math.Sin(dr*(m-mpr)) + 0.0004*math.Sin(dr*(2*f+m))
	c1 += -0.0004*math.Sin(dr*(2*f-m)) - 0.0006*math.Sin(dr*(2*f+mpr))
	c1 += 0.0010*math.Sin(dr*(2*f-mpr)) + 0.0005*math.Sin(dr*(2*mpr+m))
	var deltat float64
	if t < -11 {
		deltat = 0.001 + 0.000839*t + 0.0002261*t2 - 0.00000845*t3 - 0.000000081*t*t3
	} else {
		deltat = -0.000278 + 0.000265*t + 0.000262*t2
	}
	return jd1 + c1 - deltat
}

// sunLongitudeRad returns the Sun's ecliptic longitude, in radians in
// [0, 2π), at the Julian date jd (UT).
func sunLongitudeRad(jd float64) float64 {
	const dr = math.Pi / 180
	t := (jd - 2451545.0) / 36525
	t2 := t * t
	m := 357.52910 + 35999.05030*t - 0.0001559*t2 - 0.00000048*t*t2
	l0 := 280.46645 + 36000.76983*t + 0.0003032*t2
	dl := (1.914600-0.004817*t-0.000014*t2)*math.Sin(dr*m) +
		(0.019993-0.000101*t)*math.Sin(dr*2*m) + 0.000290*math.Sin(dr*3*m)
	l := (l0 + dl) * dr
	return l - 2*math.Pi*math.Floor(l/(2*math.Pi))
}

// newMoonDay returns the Julian day number (local) on which the k-th new moon falls.
func newMoonDay(k int) int {
	return int(math.Floor(newMoon(k) + 0.5 + float64(TimeZone)/24))
}

// sunSector returns which 30° sector (0..11) the Sun is in at local midnight
// starting local day jdn.
func sunSector(jdn int) int {
	return int(sunLongitudeRad(float64(jdn)-0.5-float64(TimeZone)/24) / math.Pi * 6)
}

// lunarMonth11 returns the Julian day number of the first day of lunar
// month 11 of the lunar year that starts in solar year y (the month holding
// the winter solstice).
func lunarMonth11(y int) int {
	off := jdFromDate(31, 12, y) - 2415021
	k := int(math.Floor(float64(off) / synodicMonth))
	nm := newMoonDay(k)
	if sunSector(nm) >= 9 {
		nm = newMoonDay(k - 1)
	}
	return nm
}

// leapMonthOffset returns, for the lunar year whose month 11 starts at
// a11, the lunation offset (from a11) of the leap month. Only meaningful
// when that year has 13 months.
func leapMonthOffset(a11 int) int {
	k := int(math.Floor(float64(a11-2415021)/synodicMonth + 0.5))
	last := 0
	i := 1
	arc := sunSector(newMoonDay(k + i))
	for {
		last = arc
		i++
		arc = sunSector(newMoonDay(k + i))
		if arc == last || i >= 14 {
			break
		}
	}
	return i - 1
}

// SolarToLunar converts the civil date of t (its year, month and day; the
// clock part is ignored) to the lunar date.
func SolarToLunar(t time.Time) Date {
	y, mo, d := t.Date()
	dayNumber := jdFromDate(d, int(mo), y)
	// The mean-lunation estimate can be off by one lunation near month
	// boundaries (new moons jitter by up to ~14h), so settle k exactly.
	k := int(math.Floor(float64(dayNumber-2415021)/synodicMonth)) + 1
	for newMoonDay(k) > dayNumber {
		k--
	}
	for newMoonDay(k+1) <= dayNumber {
		k++
	}
	monthStart := newMoonDay(k)
	a11 := lunarMonth11(y)
	b11 := a11
	lunarYear := y
	if a11 >= monthStart {
		a11 = lunarMonth11(y - 1)
	} else {
		lunarYear = y + 1
		b11 = lunarMonth11(y + 1)
	}
	lunarDay := dayNumber - monthStart + 1
	diff := (monthStart - a11) / 29
	leap := false
	lunarMonth := diff + 11
	if b11-a11 > 365 {
		leapOff := leapMonthOffset(a11)
		if diff >= leapOff {
			lunarMonth = diff + 10
			if diff == leapOff {
				leap = true
			}
		}
	}
	if lunarMonth > 12 {
		lunarMonth -= 12
	}
	if lunarMonth >= 11 && diff < 4 {
		lunarYear--
	}
	return Date{Day: lunarDay, Month: lunarMonth, Year: lunarYear, IsLeapMonth: leap}
}

// lunationsFrom lists the lunations from the month 11 starting at a11 up to,
// not including, the month 11 starting at next, numbering them (leap months
// included). Offset 0 is month 11, offset 1 month 12 (or a leap 11), and so on.
func lunationsFrom(a11, next int) []Month {
	k := int(math.Floor(float64(a11-2415021)/synodicMonth + 0.5))
	n := int(math.Floor(float64(next-a11)/synodicMonth + 0.5))
	leapOff := -1
	if n == 13 {
		leapOff = leapMonthOffset(a11)
	}
	out := make([]Month, 0, n)
	for j := 0; j < n; j++ {
		start, after := newMoonDay(k+j), newMoonDay(k+j+1)
		num := j + 11
		if leapOff >= 0 && j >= leapOff {
			num = j + 10
		}
		if num > 12 {
			num -= 12
		}
		out = append(out, Month{
			Month: num,
			Leap:  leapOff >= 0 && j == leapOff,
			Days:  after - start,
			Start: dateFromJD(start),
		})
	}
	return out
}

// LunarMonths returns the 12 or 13 months of lunar year y in order, with the
// leap month (if any) placed after the month it repeats. A leap month 11 or
// 12 belongs to the lunar year of the month it repeats.
func LunarMonths(y int) []Month {
	a11 := lunarMonth11(y - 1)
	b11 := lunarMonth11(y)
	c11 := lunarMonth11(y + 1)
	var months []Month
	for _, m := range lunationsFrom(a11, b11) {
		if m.Month <= 10 {
			months = append(months, m)
		}
	}
	for _, m := range lunationsFrom(b11, c11) {
		if m.Month >= 11 {
			months = append(months, m)
		}
	}
	return months
}

// LeapMonth returns the number of the leap month of lunar year y, or 0 when
// the year has none.
func LeapMonth(y int) int {
	for _, m := range LunarMonths(y) {
		if m.Leap {
			return m.Month
		}
	}
	return 0
}

// DaysInLunarMonth returns the length (29 or 30) of month m of lunar year y.
// It returns ErrInvalidLunarDate when the month does not exist (leap month in
// a year that lacks it).
func DaysInLunarMonth(y, m int, leap bool) (int, error) {
	for _, lm := range LunarMonths(y) {
		if lm.Month == m && lm.Leap == leap {
			return lm.Days, nil
		}
	}
	return 0, ErrInvalidLunarDate
}

// LunarToSolar converts a lunar date to its solar date (UTC midnight). It
// returns ErrInvalidLunarDate when the month or the day does not exist.
func LunarToSolar(l Date) (time.Time, error) {
	if l.Day < 1 {
		return time.Time{}, ErrInvalidLunarDate
	}
	for _, lm := range LunarMonths(l.Year) {
		if lm.Month == l.Month && lm.Leap == l.IsLeapMonth {
			if l.Day > lm.Days {
				return time.Time{}, ErrInvalidLunarDate
			}
			return lm.Start.AddDate(0, 0, l.Day-1), nil
		}
	}
	return time.Time{}, ErrInvalidLunarDate
}
