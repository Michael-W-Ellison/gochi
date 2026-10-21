package calendar

import (
	"testing"
	"time"
)

// within reports whether a date is within days of the expected one
func within(got, want Date, days int) bool {
	diff := got.Time().Sub(want.Time()).Hours() / 24
	return diff >= -float64(days) && diff <= float64(days)
}

func d(y int, m time.Month, day int) Date { return Date{y, m, day} }

// Published dates (days is how far the computed date may be from them)
func TestHolidayDates(t *testing.T) {
	for _, c := range []struct {
		name string
		rule func(int) Date
		want []Date
		days int
	}{
		{"Easter", Easter, []Date{d(2024, 3, 31), d(2025, 4, 20), d(2026, 4, 5), d(2027, 3, 28), d(2038, 4, 25)}, 0},
		{"Orthodox Easter", OrthodoxEaster, []Date{d(2024, 5, 5), d(2025, 4, 20), d(2026, 4, 12), d(2027, 5, 2)}, 0},
		{"Lunar New Year", LunarNewYear, []Date{d(2023, 1, 22), d(2024, 2, 10), d(2025, 1, 29), d(2026, 2, 17), d(2027, 2, 6), d(2028, 1, 26), d(2033, 1, 31)}, 0},
		{"Mid-Autumn", MidAutumn, []Date{d(2023, 9, 29), d(2024, 9, 17), d(2025, 10, 6), d(2026, 9, 25), d(2027, 9, 15)}, 0},
		{"Nowruz", Nowruz, []Date{d(2024, 3, 20), d(2025, 3, 21), d(2026, 3, 21)}, 1},
		{"Holi", Holi, []Date{d(2023, 3, 8), d(2024, 3, 25), d(2025, 3, 14), d(2026, 3, 4), d(2027, 3, 22)}, 1},
		{"Diwali", Diwali, []Date{d(2023, 11, 12), d(2024, 11, 1), d(2025, 10, 20), d(2026, 11, 8)}, 1},
		{"Vesak", Vesak, []Date{d(2024, 5, 23), d(2025, 5, 12)}, 1},
		{"Hanukkah", Hanukkah, []Date{d(2023, 12, 8), d(2024, 12, 26), d(2025, 12, 15), d(2026, 12, 5)}, 0},
	} {
		for _, want := range c.want {
			if got := c.rule(want.Year); !within(got, want, c.days) {
				t.Errorf("%s %d = %v, want %v", c.name, want.Year, got, want)
			}
		}
	}
	for _, c := range []struct {
		name string
		rule func(int) []Date
		want []Date
	}{
		{"Eid al-Fitr", EidAlFitr, []Date{d(2023, 4, 21), d(2024, 4, 10), d(2025, 3, 30), d(2026, 3, 20)}},
		{"Eid al-Adha", EidAlAdha, []Date{d(2023, 6, 28), d(2024, 6, 16), d(2025, 6, 6), d(2026, 5, 27)}},
	} {
		for _, want := range c.want {
			got := c.rule(want.Year)
			if len(got) == 0 || !within(got[0], want, 2) {
				t.Errorf("%s %d = %v, want %v", c.name, want.Year, got, want)
			}
		}
	}
	// The Islamic year is shorter, so an Eid falls twice in some years
	if len(EidAlFitr(2033)) != 2 {
		t.Errorf("Eid al-Fitr falls twice in 2033: %v", EidAlFitr(2033))
	}
}

func TestMoonPhases(t *testing.T) {
	for _, c := range []struct {
		got  []time.Time
		want time.Time
	}{
		{NewMoons(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC)),
			time.Date(2025, 1, 29, 12, 36, 0, 0, time.UTC)},
		{FullMoons(time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC), time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)),
			time.Date(2025, 3, 14, 6, 55, 0, 0, time.UTC)},
	} {
		if len(c.got) != 1 || c.got[0].Sub(c.want).Abs() > 10*time.Minute {
			t.Errorf("Moon phase = %v, want %v", c.got, c.want)
		}
	}
	if eq := SunReaches(2025, 0); eq.Sub(time.Date(2025, 3, 20, 9, 1, 0, 0, time.UTC)).Abs() > 20*time.Minute {
		t.Errorf("March equinox 2025 = %v", eq)
	}
}

// Every rule gives a date in its usual months, for the rest of the century
func TestHolidayDatesStayInRange(t *testing.T) {
	ranges := []struct {
		name     string
		rule     func(int) Date
		from, to time.Month
	}{
		{"Easter", Easter, time.March, time.April},
		{"Orthodox Easter", OrthodoxEaster, time.April, time.May},
		{"Lunar New Year", LunarNewYear, time.January, time.February},
		{"Mid-Autumn", MidAutumn, time.September, time.October},
		{"Nowruz", Nowruz, time.March, time.March},
		{"Holi", Holi, time.February, time.March},
		{"Diwali", Diwali, time.October, time.November},
		{"Vesak", Vesak, time.May, time.May},
		{"Hanukkah", Hanukkah, time.November, time.December},
	}
	for year := 2025; year <= 2100; year++ {
		for _, r := range ranges {
			got := r.rule(year)
			if got.Year != year || got.Month < r.from || got.Month > r.to {
				t.Errorf("%s %d = %v", r.name, year, got)
			}
		}
		if n := len(EidAlFitr(year)) + len(EidAlAdha(year)); n < 2 || n > 3 {
			t.Errorf("Eids in %d: %v %v", year, EidAlFitr(year), EidAlAdha(year))
		}
	}
}
