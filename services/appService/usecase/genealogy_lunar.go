package usecase

import (
	"math"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Lunar conversion for the genealogy feature (AC40, AC55): chronicle and death
// dates are stored as solar dates; the server derives the lunar equivalent at
// read time. Pure functions, no I/O.
//
// The algorithm is Hồ Ngọc Đức's astronomical computation of the Vietnamese
// lunar calendar (time zone UTC+7), which is what the Vietnamese calendar
// follows. It is accurate for the supported range below.

const (
	lunarMinYear  = 1900
	lunarMaxYear  = 2199
	lunarTimeZone = 7.0
)

// DaysInSolarMonth is the largest valid day of a solar month, counting February
// as 29 because the year is validated separately where it matters. It returns 0
// for a month outside 1..12.
func DaysInSolarMonth(month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		return 29
	}
	return 0
}

// SolarToLunar converts a solar date to the Vietnamese lunar date. ok is false
// when the date does not exist or the year is outside 1900..2199, in which case
// the lunar fields must be reported as null (AC40, AC55).
func SolarToLunar(year, month, day int) (domain.LunarDate, bool) {
	if year < lunarMinYear || year > lunarMaxYear || month < 1 || month > 12 || day < 1 {
		return domain.LunarDate{}, false
	}
	if day > daysInMonthOfYear(year, month) {
		return domain.LunarDate{}, false
	}
	return convertSolar2Lunar(day, month, year, lunarTimeZone), true
}

func daysInMonthOfYear(year, month int) int {
	if month == 2 {
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
	return DaysInSolarMonth(month)
}

// jdFromDate is the Julian day number of a Gregorian (or Julian before
// 1582-10-15) calendar date.
func jdFromDate(dd, mm, yy int) int {
	a := (14 - mm) / 12
	y := yy + 4800 - a
	m := mm + 12*a - 3
	jd := dd + (153*m+2)/5 + 365*y + y/4 - y/100 + y/400 - 32045
	if jd < 2299161 {
		jd = dd + (153*m+2)/5 + 365*y + y/4 - 32083
	}
	return jd
}

// newMoon returns the Julian day (fractional) of the k-th new moon since
// 1900-01-01.
func newMoon(k int) float64 {
	kf := float64(k)
	T := kf / 1236.85
	T2 := T * T
	T3 := T2 * T
	dr := math.Pi / 180
	Jd1 := 2415020.75933 + 29.53058868*kf + 0.0001178*T2 - 0.000000155*T3
	Jd1 += 0.00033 * math.Sin((166.56+132.87*T-0.009173*T2)*dr)
	M := 359.2242 + 29.10535608*kf - 0.0000333*T2 - 0.00000347*T3
	Mpr := 306.0253 + 385.81691806*kf + 0.0107306*T2 + 0.00001236*T3
	F := 21.2964 + 390.67050646*kf - 0.0016528*T2 - 0.00000239*T3
	C1 := (0.1734-0.000393*T)*math.Sin(M*dr) + 0.0021*math.Sin(2*dr*M)
	C1 -= 0.4068 * math.Sin(Mpr*dr)
	C1 += 0.0161 * math.Sin(dr*2*Mpr)
	C1 -= 0.0004 * math.Sin(dr*3*Mpr)
	C1 += 0.0104 * math.Sin(dr*2*F)
	C1 -= 0.0051 * math.Sin(dr*(M+Mpr))
	C1 -= 0.0074 * math.Sin(dr*(M-Mpr))
	C1 += 0.0004 * math.Sin(dr*(2*F+M))
	C1 -= 0.0004 * math.Sin(dr*(2*F-M))
	C1 -= 0.0006 * math.Sin(dr*(2*F+Mpr))
	C1 += 0.0010 * math.Sin(dr*(2*F-Mpr))
	C1 += 0.0005 * math.Sin(dr*(2*Mpr+M))
	var deltat float64
	if T < -11 {
		deltat = 0.001 + 0.000839*T + 0.0002261*T2 - 0.00000845*T3 - 0.000000081*T*T3
	} else {
		deltat = -0.000278 + 0.000265*T + 0.000262*T2
	}
	return Jd1 + C1 - deltat
}

func sunLongitude(jdn float64) float64 {
	T := (jdn - 2451545.0) / 36525
	T2 := T * T
	dr := math.Pi / 180
	M := 357.52910 + 35999.05030*T - 0.0001559*T2 - 0.00000048*T*T2
	L0 := 280.46645 + 36000.76983*T + 0.0003032*T2
	DL := (1.914600 - 0.004817*T - 0.000014*T2) * math.Sin(dr*M)
	DL += (0.019993-0.000101*T)*math.Sin(dr*2*M) + 0.000290*math.Sin(dr*3*M)
	L := (L0 + DL) * dr
	L -= math.Pi * 2 * math.Floor(L/(math.Pi*2))
	return L
}

func newMoonDay(k int, tz float64) int {
	return int(math.Floor(newMoon(k) + 0.5 + tz/24))
}

func sunLongitudeIndex(dayNumber int, tz float64) int {
	return int(math.Floor(sunLongitude(float64(dayNumber)-0.5-tz/24) / math.Pi * 6))
}

// lunarMonth11 is the Julian day of the new moon that starts lunar month 11 of
// the given solar year (the month containing the winter solstice).
func lunarMonth11(yy int, tz float64) int {
	off := jdFromDate(31, 12, yy) - 2415021
	k := int(math.Floor(float64(off) / 29.530588853))
	nm := newMoonDay(k, tz)
	if sunLongitudeIndex(nm, tz) >= 9 {
		nm = newMoonDay(k-1, tz)
	}
	return nm
}

// leapMonthOffset is the index (counted from month 11 at a11) of the leap month
// in a 13-month lunar year.
func leapMonthOffset(a11 int, tz float64) int {
	k := int(math.Floor((float64(a11)-2415021.076998695)/29.530588853 + 0.5))
	i := 1
	arc := sunLongitudeIndex(newMoonDay(k+i, tz), tz)
	var last int
	for {
		last = arc
		i++
		arc = sunLongitudeIndex(newMoonDay(k+i, tz), tz)
		if arc == last || i >= 14 {
			break
		}
	}
	return i - 1
}

func convertSolar2Lunar(dd, mm, yy int, tz float64) domain.LunarDate {
	dayNumber := jdFromDate(dd, mm, yy)
	k := int(math.Floor((float64(dayNumber) - 2415021.076998695) / 29.530588853))
	monthStart := newMoonDay(k+1, tz)
	if monthStart > dayNumber {
		monthStart = newMoonDay(k, tz)
	}
	a11 := lunarMonth11(yy, tz)
	b11 := a11
	if a11 >= monthStart {
		a11 = lunarMonth11(yy-1, tz)
	} else {
		b11 = lunarMonth11(yy+1, tz)
	}
	lunarDay := dayNumber - monthStart + 1
	diff := (monthStart - a11) / 29
	leap := false
	lunarMonth := diff + 11
	if b11-a11 > 365 {
		leapDiff := leapMonthOffset(a11, tz)
		if diff >= leapDiff {
			lunarMonth = diff + 10
			if diff == leapDiff {
				leap = true
			}
		}
	}
	if lunarMonth > 12 {
		lunarMonth -= 12
	}
	return domain.LunarDate{Day: lunarDay, Month: lunarMonth, Leap: leap}
}
