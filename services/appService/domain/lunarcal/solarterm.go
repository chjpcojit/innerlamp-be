package lunarcal

import (
	"math"
	"time"
)

// SolarTermDate returns the solar date, in UTC+7, on which the Sun's
// ecliptic longitude reaches longitude degrees (a multiple of 15 in
// [0, 345]) within Gregorian year y, as a UTC-midnight time. ok is false
// when longitude is not a valid term or no crossing falls in y.
func SolarTermDate(y int, longitude int) (date time.Time, ok bool) {
	if longitude < 0 || longitude >= 360 || longitude%15 != 0 {
		return time.Time{}, false
	}
	target := float64(longitude) * math.Pi / 180
	// f is the angular distance past target: in [π, 2π) just before the
	// crossing, in [0, π) just after it.
	f := func(jdn int) float64 {
		l := sunLongitudeRad(float64(jdn) - 0.5 - float64(TimeZone)/24)
		d := l - target
		return d - 2*math.Pi*math.Floor(d/(2*math.Pi))
	}
	first := jdFromDate(1, 1, y)
	last := jdFromDate(31, 12, y)
	for jdn := first; jdn <= last; jdn++ {
		if f(jdn) >= math.Pi && f(jdn+1) < math.Pi {
			// The crossing happens between local midnight of jdn and of
			// jdn+1, i.e. during local day jdn.
			return dateFromJD(jdn), true
		}
	}
	return time.Time{}, false
}
