package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The product spec ships its computational rules as vectors every platform must pass
// (docs/product/calendar/rules/). They are read from docs/product/, not copied, so a changed
// vector fails here until the resolver agrees (calendar E27).
const calendarRulesDir = "../../../docs/product/calendar/rules"

func readRules(t *testing.T, name string, into any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(calendarRulesDir, name))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, into))
}

func TestEventPlacementVectors(t *testing.T) {
	var file struct {
		Cases []struct {
			Name  string `json:"name"`
			Input struct {
				Rule struct {
					Kind      string `json:"kind"`
					Month     int    `json:"month"`
					Day       int    `json:"day"`
					LeapMonth bool   `json:"leap_month"`
					Longitude int    `json:"longitude"`
				} `json:"rule"`
				Year int `json:"year"`
			} `json:"input"`
			Expected struct {
				Occurrences []struct {
					SolarDate string `json:"solar_date"`
					Lunar     struct {
						Day       int  `json:"day"`
						Month     int  `json:"month"`
						Year      int  `json:"year"`
						LeapMonth bool `json:"leap_month"`
					} `json:"lunar"`
				} `json:"occurrences"`
			} `json:"expected"`
		} `json:"cases"`
	}
	readRules(t, "event-placement.json", &file)
	require.NotEmpty(t, file.Cases)

	for _, tc := range file.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			r := tc.Input.Rule
			var rule RecurrenceRule
			switch r.Kind {
			case "solar":
				rule = RecurrenceRule{Kind: KindSolar, SolarMonth: r.Month, SolarDay: r.Day}
			case "lunar":
				rule = RecurrenceRule{Kind: KindLunar, LunarMonth: r.Month, LunarDay: r.Day, IsLeapMonth: r.LeapMonth}
			case "lunar_last_day":
				rule = RecurrenceRule{Kind: KindLunarLastDay, LunarMonth: r.Month}
			case "every_lunar_month":
				rule = RecurrenceRule{Kind: KindLunarMonthly, LunarDay: r.Day}
			case "solar_term":
				rule = RecurrenceRule{Kind: KindSolarTerm, SolarTermLongitude: r.Longitude}
			default:
				t.Fatalf("vector uses a rule kind this test does not know: %q", r.Kind)
			}

			got := ResolveOccurrences([]CalendarEvent{{Code: "e", Rule: rule}}, tc.Input.Year)

			require.Len(t, got, len(tc.Expected.Occurrences))
			for i, want := range tc.Expected.Occurrences {
				assert.Equal(t, want.SolarDate, got[i].SolarDate.Format("2006-01-02"), "occurrence %d", i)
				assert.Equal(t, CalendarLunarDate{
					Day: want.Lunar.Day, Month: want.Lunar.Month, Year: want.Lunar.Year, IsLeapMonth: want.Lunar.LeapMonth,
				}, got[i].LunarDate, "occurrence %d lunar date", i)
			}
		})
	}
}

// TestOrderingVectors checks BR-CAL-12: solar date, then display order, then code. The vectors
// give events as occurrences; one single-day solar event per entry reproduces each of them.
func TestOrderingVectors(t *testing.T) {
	var file struct {
		Cases []struct {
			Name  string `json:"name"`
			Input struct {
				Occurrences []struct {
					Event        string `json:"event"`
					DisplayOrder int    `json:"display_order"`
					SolarDate    string `json:"solar_date"`
				} `json:"occurrences"`
			} `json:"input"`
			Expected struct {
				Order []string `json:"order"`
			} `json:"expected"`
		} `json:"cases"`
	}
	readRules(t, "ordering.json", &file)
	require.NotEmpty(t, file.Cases)

	for _, tc := range file.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			var events []CalendarEvent
			year := 0
			for _, o := range tc.Input.Occurrences {
				d, err := time.Parse("2006-01-02", o.SolarDate)
				require.NoError(t, err)
				year = d.Year()
				events = append(events, CalendarEvent{
					Code: o.Event, SortOrder: o.DisplayOrder,
					Rule: RecurrenceRule{Kind: KindSolar, SolarMonth: int(d.Month()), SolarDay: d.Day()},
				})
			}

			got := ResolveOccurrences(events, year)

			var order []string
			for _, o := range got {
				order = append(order, o.EventCode)
			}
			assert.Equal(t, tc.Expected.Order, order)
		})
	}
}
