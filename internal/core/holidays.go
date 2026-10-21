package core

import (
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/calendar"
)

// Holidays from many cultures and faiths. Each has a few items (a
// decoration, a cosmetic, a pet-safe treat) sold from a week before to a
// week after it, and a lesson about the holiday's real dangers to pets.
// Items celebrate shared customs (lights, flowers, food, gatherings) and
// avoid religious symbols. A parent can turn holidays off (SetHolidays);
// a holiday that is off has no shop, its decorations aren't shown and its
// cosmetics can't be worn, until it is turned on again.

// holidayWindow is how many days before and after a holiday its items are sold
const holidayWindow = 7

// Holiday is a celebration with its own items
type Holiday struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Lesson string `json:"lesson"`

	// dates returns when it falls in a year: first and last day of each
	// time (most last one day)
	dates func(year int) [][2]calendar.Date
}

// once makes a one-day holiday from a rule
func once(rule func(int) calendar.Date) func(int) [][2]calendar.Date {
	return func(year int) [][2]calendar.Date {
		d := rule(year)
		return [][2]calendar.Date{{d, d}}
	}
}

// fixed makes a holiday on the same date each year, lasting days days
func fixed(month time.Month, day, days int) func(int) [][2]calendar.Date {
	return func(year int) [][2]calendar.Date {
		start := calendar.Date{Year: year, Month: month, Day: day}
		return [][2]calendar.Date{{start, start.AddDays(days - 1)}}
	}
}

// each makes a holiday from a rule that can fall more than once a year
func each(rule func(int) []calendar.Date) func(int) [][2]calendar.Date {
	return func(year int) [][2]calendar.Date {
		var spans [][2]calendar.Date
		for _, d := range rule(year) {
			spans = append(spans, [2]calendar.Date{d, d})
		}
		return spans
	}
}

// holidays lists every holiday in calendar order
var holidays = []Holiday{
	{ID: "new_year", Name: "New Year", Icon: "🎆", dates: fixed(time.January, 1, 1),
		Lesson: "Fireworks frighten pets: keep them indoors with a den."},
	{ID: "lunar_new_year", Name: "Lunar New Year", Icon: "🏮", dates: once(calendar.LunarNewYear),
		Lesson: "Firecrackers frighten pets; keep them inside and away from the noise."},
	{ID: "valentines", Name: "Valentine's Day", Icon: "💝", dates: fixed(time.February, 14, 1),
		Lesson: "Chocolate and sweets with xylitol are poisonous to pets."},
	{ID: "nowruz", Name: "Nowruz", Icon: "🌸", dates: once(calendar.Nowruz),
		Lesson: "A goldfish needs a large filtered tank and a lid, away from the cat."},
	{ID: "holi", Name: "Holi", Icon: "🎨", dates: once(calendar.Holi),
		Lesson: "Holi color powders can be toxic: keep pets away and wash any off at once."},
	{ID: "easter", Name: "Easter", Icon: "🐣", dates: func(year int) [][2]calendar.Date {
		w, o := calendar.Easter(year), calendar.OrthodoxEaster(year)
		if o.Before(w) {
			w, o = o, w
		}
		return [][2]calendar.Date{{w, o}} // Western and Orthodox, and the days between
	}, Lesson: "Lilies are deadly to cats, and chocolate eggs are poisonous."},
	{ID: "eid_al_fitr", Name: "Eid al-Fitr", Icon: "🌙", dates: each(calendar.EidAlFitr),
		Lesson: "Rich festive food and bones upset pets; give them a quiet room when guests visit."},
	{ID: "earth_day", Name: "Earth Day", Icon: "🌍", dates: fixed(time.April, 22, 1),
		Lesson: "Pet-safe plants and natural cleaners keep a home safe."},
	{ID: "vesak", Name: "Vesak", Icon: "🪷", dates: once(calendar.Vesak),
		Lesson: "Keep candles and lanterns out of a pet's reach."},
	{ID: "eid_al_adha", Name: "Eid al-Adha", Icon: "✨", dates: each(calendar.EidAlAdha),
		Lesson: "Cooked bones splinter and can tear a pet's insides."},
	{ID: "cat_day", Name: "International Cat Day", Icon: "🐈", dates: fixed(time.August, 8, 1),
		Lesson: "Cats need daily play as much as dogs do."},
	{ID: "dog_day", Name: "International Dog Day", Icon: "🐕", dates: fixed(time.August, 26, 1),
		Lesson: "Check peanut butter for xylitol before giving it."},
	{ID: "mid_autumn", Name: "Mid-Autumn Festival", Icon: "🥮", dates: once(calendar.MidAutumn),
		Lesson: "Mooncakes are too rich and sugary for pets."},
	{ID: "world_animal_day", Name: "World Animal Day", Icon: "🐾", dates: fixed(time.October, 4, 1),
		Lesson: "Shelters care for animals waiting for homes."},
	{ID: "diwali", Name: "Diwali", Icon: "🪔", dates: once(calendar.Diwali),
		Lesson: "Fireworks frighten pets, and open flames burn: use electric lights near pets."},
	{ID: "halloween", Name: "Halloween", Icon: "🎃", dates: fixed(time.October, 31, 1),
		Lesson: "Keep sweets away; costumes must not restrict movement or breathing."},
	{ID: "day_of_the_dead", Name: "Day of the Dead", Icon: "🏵️", dates: fixed(time.November, 1, 2),
		Lesson: "Remembering pets who have died is part of loving them. Marigolds can upset a pet's stomach."},
	{ID: "hanukkah", Name: "Hanukkah", Icon: "🕯️", dates: once(calendar.Hanukkah),
		Lesson: "Keep candles out of reach; fried foods upset pets' stomachs."},
	{ID: "christmas", Name: "Christmas", Icon: "🎄", dates: fixed(time.December, 25, 1),
		Lesson: "Tinsel, ribbons and some festive plants are dangerous to swallow."},
	{ID: "kwanzaa", Name: "Kwanzaa", Icon: "🧺", dates: fixed(time.December, 26, 7),
		Lesson: "Keep candles out of reach of tails and paws."},
}

// Holidays returns every holiday
func Holidays() []Holiday {
	return append([]Holiday(nil), holidays...)
}

// FindHoliday returns a holiday by ID
func FindHoliday(id string) (Holiday, bool) {
	for _, h := range holidays {
		if h.ID == id {
			return h, true
		}
	}
	return Holiday{}, false
}

// Next returns the first and last day of the holiday's next time (or the
// current one, while it is on), from day
func (h Holiday) Next(day calendar.Date) (calendar.Date, calendar.Date) {
	for year := day.Year - 1; year <= day.Year+1; year++ {
		for _, span := range h.dates(year) {
			if !span[1].Before(day) {
				return span[0], span[1]
			}
		}
	}
	return calendar.Date{}, calendar.Date{}
}

// OpenOn reports whether a holiday's items are sold on day: from a week
// before it to a week after
func (h Holiday) OpenOn(day calendar.Date) bool {
	for year := day.Year - 1; year <= day.Year+1; year++ {
		for _, span := range h.dates(year) {
			if !day.Before(span[0].AddDays(-holidayWindow)) && !span[1].AddDays(holidayWindow).Before(day) {
				return true
			}
		}
	}
	return false
}

// HolidayOpen reports whether a holiday's items are sold today: it is
// turned on and today is within a week of it
func HolidayOpen(id string) bool {
	h, ok := FindHoliday(id)
	return ok && HolidayOn(id) && h.OpenOn(calendar.DateOf(now()))
}

// OpenHolidays returns the holidays whose items are sold today
func OpenHolidays() []Holiday {
	var open []Holiday
	for _, h := range holidays {
		if HolidayOpen(h.ID) {
			open = append(open, h)
		}
	}
	return open
}
