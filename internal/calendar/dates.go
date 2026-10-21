package calendar

import "time"

// The rules for holidays that move each year. Each returns the date in
// the given Gregorian year (the second Eid, for example, can fall twice in
// one year; those return every date in the year).

// Easter returns Western Easter Sunday (the anonymous Gregorian algorithm)
func Easter(year int) Date {
	a, b, c := year%19, year/100, year%100
	d, e := b/4, b%4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := (h+l-7*m+114)%31 + 1
	return Date{year, time.Month(month), day}
}

// OrthodoxEaster returns Orthodox Easter Sunday: Easter by the Julian
// calendar, given as its Gregorian date
func OrthodoxEaster(year int) Date {
	a, b, c := year%4, year%7, year%19
	d := (19*c + 15) % 30
	e := (2*a + 4*b - d + 34) % 7
	month := (d + e + 114) / 31
	day := (d+e+114)%31 + 1
	// The Julian calendar is behind the Gregorian by 13 days from 1900 to
	// 2099, and one more each century after that (not counting those
	// divisible by 400)
	behind := year/100 - year/400 - 2
	return Date{year, time.Month(month), day}.AddDays(behind)
}

// Nowruz returns the Persian New Year: the day of the March equinox in
// Tehran, or the next day if the equinox comes after noon there
func Nowruz(year int) Date {
	equinox := SunReaches(year, 0).In(tehranTime)
	day := DateOf(equinox)
	if equinox.Hour() >= 12 {
		day = day.AddDays(1)
	}
	return day
}

// LunarNewYear returns the Chinese New Year: the first day of the month
// in which the Sun reaches longitude 330° (the solar term Rain Water), by
// the date in China
func LunarNewYear(year int) Date {
	rainWater := DateOf(SunReaches(year, 330).In(chinaTime))
	return lastMoonOnOrBefore(rainWater, chinaTime, false)
}

// MidAutumn returns the Mid-Autumn Festival: the 15th day of the month in
// which the Sun reaches longitude 180° (the autumn equinox), in China
func MidAutumn(year int) Date {
	equinox := DateOf(SunReaches(year, 180).In(chinaTime))
	return lastMoonOnOrBefore(equinox, chinaTime, false).AddDays(14)
}

// Holi returns the festival of colors: the full moon of the month before
// the Hindu new year, which begins with the new moon before the Sun
// enters Aries in the sidereal zodiac, about April 14 (dates in India)
func Holi(year int) Date {
	newYear := lastMoonOnOrBefore(Date{year, time.April, 13}, indiaTime, false)
	return lastMoonOnOrBefore(newYear.AddDays(-1), indiaTime, true)
}

// Diwali returns the festival of lights: the first new moon in India on
// or after October 16
func Diwali(year int) Date {
	from := time.Date(year, time.October, 16, 0, 0, 0, 0, indiaTime)
	return DateOf(NewMoons(from, from.AddDate(0, 0, 31))[0].In(indiaTime))
}

// Vesak returns the day of the Buddha's birth, enlightenment and death:
// the first full moon in May, as many countries keep it
func Vesak(year int) Date {
	from := time.Date(year, time.May, 1, 0, 0, 0, 0, indiaTime)
	return DateOf(FullMoons(from, from.AddDate(0, 0, 31))[0].In(indiaTime))
}

// Rata Die numbers days from January 1 of year 1 (day 1)
const unixEpochRD = 719163 // January 1, 1970

// toRD returns a date's day number
func toRD(d Date) int {
	return int(d.Time().Unix()/86400) + unixEpochRD
}

// fromRD returns the date of a day number
func fromRD(rd int) Date {
	return DateOf(time.Unix(int64(rd-unixEpochRD)*86400, 0).UTC())
}

// floorDiv divides, rounding towards minus infinity
func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// The Hebrew calendar, by its arithmetic rules (Reingold and Dershowitz,
// Calendrical Calculations)
const hebrewEpoch = -1373427

func hebrewElapsedDays(year int) int {
	months := floorDiv(235*year-234, 19)
	parts := 12084 + 13753*months
	days := 29*months + floorDiv(parts, 25920)
	if (3*(days+1))%7 < 3 {
		days++
	}
	return days
}

func hebrewNewYear(year int) int {
	ny0, ny1, ny2 := hebrewElapsedDays(year-1), hebrewElapsedDays(year), hebrewElapsedDays(year+1)
	correction := 0
	switch {
	case ny2-ny1 == 356:
		correction = 2
	case ny1-ny0 == 382:
		correction = 1
	}
	return hebrewEpoch + ny1 + correction
}

// Hanukkah returns the first day of Hanukkah (25 Kislev); it begins the
// evening before
func Hanukkah(year int) Date {
	hebrewYear := year + 3761 // The Hebrew year that begins in the autumn
	start := hebrewNewYear(hebrewYear)
	heshvan := 29
	if (hebrewNewYear(hebrewYear+1)-start)%10 == 5 { // A 355- or 385-day year
		heshvan = 30
	}
	return fromRD(start + 30 + heshvan + 24)
}

// The Islamic calendar, by its arithmetic (tabular) rules; observed dates
// can differ by a day or two, and between countries
const islamicEpoch = 227015 // July 16, 622 (Julian)

func islamicToRD(year, month, day int) int {
	return day + (59*(month-1)+1)/2 + (year-1)*354 + floorDiv(3+11*year, 30) + islamicEpoch - 1
}

// islamicDates returns the dates in a Gregorian year of a day of the
// Islamic year
func islamicDates(year, month, day int) []Date {
	var dates []Date
	first := floorDiv((year-622)*33, 32) // About the Islamic year beginning in that year
	for y := first - 1; y <= first+2; y++ {
		if d := fromRD(islamicToRD(y, month, day)); d.Year == year {
			dates = append(dates, d)
		}
	}
	return dates
}

// EidAlFitr returns the festival at the end of Ramadan (1 Shawwal)
func EidAlFitr(year int) []Date { return islamicDates(year, 10, 1) }

// EidAlAdha returns the festival of sacrifice (10 Dhu al-Hijjah)
func EidAlAdha(year int) []Date { return islamicDates(year, 12, 10) }
