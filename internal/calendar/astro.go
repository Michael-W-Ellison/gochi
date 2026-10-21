// Package calendar works out the dates of holidays that move each year:
// Easter, and those set by the Moon or by the Hebrew or Islamic calendars.
// Lunar dates come from the Sun's and Moon's positions, computed with the
// shorter series from Jean Meeus, Astronomical Algorithms (accurate to a
// few minutes), so a date can differ by a day from the one observed where
// the Moon is sighted by eye. Holiday windows are two weeks long, which
// covers that.
package calendar

import (
	"math"
	"time"
)

// Date is a day in the Gregorian calendar
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// Time returns the start of the day in UTC
func (d Date) Time() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

// AddDays returns the date days later (or earlier)
func (d Date) AddDays(days int) Date {
	return DateOf(d.Time().AddDate(0, 0, days))
}

// Before reports whether d is before e
func (d Date) Before(e Date) bool {
	return d.Time().Before(e.Time())
}

// DateOf returns the date of a time, in the time's location
func DateOf(t time.Time) Date {
	y, m, day := t.Date()
	return Date{y, m, day}
}

// julianDay returns the Julian Day of a time
func julianDay(t time.Time) float64 {
	return float64(t.UnixNano())/86400e9 + 2440587.5
}

// timeOfJulianDay returns the time of a Julian Day
func timeOfJulianDay(jd float64) time.Time {
	return time.Unix(0, int64((jd-2440587.5)*86400e9)).UTC()
}

// deltaT is the difference between dynamical time (which the series
// give) and universal time, close enough for this century
const deltaT = 70.0 / 86400

func rad(deg float64) float64 { return deg * math.Pi / 180 }

// sunLongitude returns the Sun's apparent longitude (degrees, 0 to 360)
// at a Julian Day
func sunLongitude(jd float64) float64 {
	t := (jd - 2451545) / 36525
	l0 := 280.46646 + 36000.76983*t + 0.0003032*t*t
	m := rad(357.52911 + 35999.05029*t - 0.0001537*t*t)
	c := (1.914602-0.004817*t-0.000014*t*t)*math.Sin(m) + (0.019993-0.000101*t)*math.Sin(2*m) +
		0.000289*math.Sin(3*m)
	omega := rad(125.04 - 1934.136*t)
	return math.Mod(math.Mod(l0+c-0.00569-0.00478*math.Sin(omega), 360)+360, 360)
}

// SunReaches returns when, in year, the Sun's apparent longitude reaches
// degrees (0 is the March equinox, 90 the June solstice, and so on)
func SunReaches(year int, degrees float64) time.Time {
	// The Sun moves about a degree a day, starting from 0 near March 20;
	// longitudes from about 285° are reached early in the year
	days := degrees * 365.2422 / 360
	if days > 286 {
		days -= 365.2422
	}
	guess := time.Date(year, 3, 20, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(days))
	jd := julianDay(guess)
	for i := 0; i < 8; i++ {
		diff := math.Mod(degrees-sunLongitude(jd)+540, 360) - 180
		jd += diff * 365.2422 / 360
	}
	return timeOfJulianDay(jd - deltaT)
}

// moonPhase returns the time of the new moon (phase 0) or full moon
// (phase 0.5) numbered k (k = 0 was the new moon of January 6, 2000)
func moonPhase(k float64) time.Time {
	t := k / 1236.85
	jde := 2451550.09766 + 29.530588861*k + 0.00015437*t*t - 0.000000150*t*t*t + 0.00000000073*t*t*t*t
	e := 1 - 0.002516*t - 0.0000074*t*t
	m := rad(2.5534 + 29.10535670*k - 0.0000014*t*t - 0.00000011*t*t*t)
	mp := rad(201.5643 + 385.81693528*k + 0.0107582*t*t + 0.00001238*t*t*t - 0.000000058*t*t*t*t)
	f := rad(160.7108 + 390.67050284*k - 0.0016118*t*t - 0.00000227*t*t*t + 0.000000011*t*t*t*t)
	om := rad(124.7746 - 1.56375588*k + 0.0020672*t*t + 0.00000215*t*t*t)

	first := []float64{-0.40720, 0.17241, 0.01608, 0.01039, 0.00739, -0.00514, 0.00208} // New moon
	if math.Mod(k, 1) != 0 {
		first = []float64{-0.40614, 0.17302, 0.01614, 0.01043, 0.00734, -0.00515, 0.00209} // Full moon
	}
	c := first[0]*math.Sin(mp) + first[1]*e*math.Sin(m) + first[2]*math.Sin(2*mp) + first[3]*math.Sin(2*f) +
		first[4]*e*math.Sin(mp-m) + first[5]*e*math.Sin(mp+m) + first[6]*e*e*math.Sin(2*m) -
		0.00111*math.Sin(mp-2*f) - 0.00057*math.Sin(mp+2*f) + 0.00056*e*math.Sin(2*mp+m) -
		0.00042*math.Sin(3*mp) + 0.00042*e*math.Sin(m+2*f) + 0.00038*e*math.Sin(m-2*f) -
		0.00024*e*math.Sin(2*mp-m) - 0.00017*math.Sin(om) - 0.00007*math.Sin(mp+2*m) +
		0.00004*math.Sin(2*mp-2*f) + 0.00004*math.Sin(3*m) + 0.00003*math.Sin(mp+m-2*f) +
		0.00003*math.Sin(2*mp+2*f) - 0.00003*math.Sin(mp+m+2*f) + 0.00003*math.Sin(mp-m+2*f) -
		0.00002*math.Sin(mp-m-2*f) - 0.00002*math.Sin(3*mp+m) + 0.00002*math.Sin(4*mp)
	return timeOfJulianDay(jde + c - deltaT)
}

// moons returns the new moons (full is false) or full moons between from
// and to, in order
func moons(from, to time.Time, full bool) []time.Time {
	years := float64(from.Year()-2000) + float64(from.YearDay())/365.25
	k := math.Floor(years*12.3685) - 2
	if full {
		k += 0.5
	}
	var found []time.Time
	for ; ; k++ {
		t := moonPhase(k)
		if t.After(to) {
			return found
		}
		if !t.Before(from) {
			found = append(found, t)
		}
	}
}

// NewMoons returns the new moons between from and to
func NewMoons(from, to time.Time) []time.Time { return moons(from, to, false) }

// FullMoons returns the full moons between from and to
func FullMoons(from, to time.Time) []time.Time { return moons(from, to, true) }

// Time zones whose dates set lunar holidays
var (
	chinaTime  = time.FixedZone("China", 8*3600)
	indiaTime  = time.FixedZone("India", 5*3600+1800)
	tehranTime = time.FixedZone("Tehran", 3*3600+1800)
)

// lastMoonOnOrBefore returns the date, in zone, of the last new (or full)
// moon whose date there is on or before day
func lastMoonOnOrBefore(day Date, zone *time.Location, full bool) Date {
	end := time.Date(day.Year, day.Month, day.Day, 23, 59, 59, 0, zone)
	found := moons(end.AddDate(0, 0, -40), end, full)
	return DateOf(found[len(found)-1].In(zone))
}
