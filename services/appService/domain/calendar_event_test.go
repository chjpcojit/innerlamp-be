package domain

import (
	"errors"
	"testing"
)

func TestRecurrenceRuleValidate(t *testing.T) {
	ok := []RecurrenceRule{
		{Kind: KindSolar, SolarMonth: 2, SolarDay: 29},
		{Kind: KindLunar, LunarMonth: 12, LunarDay: 30},
		{Kind: KindLunar, LunarMonth: 6, LunarDay: 15, IsLeapMonth: true},
		{Kind: KindLunarMonthly, LunarDay: 1},
		{Kind: KindLunarLastDay, LunarMonth: 12},
		{Kind: KindSolarTerm, SolarTermLongitude: 0},
		{Kind: KindSolarTerm, SolarTermLongitude: 345},
	}
	for _, r := range ok {
		if err := r.Validate(); err != nil {
			t.Errorf("%+v: %v", r, err)
		}
	}
	bad := []RecurrenceRule{
		{},
		{Kind: "weekly"},
		{Kind: KindSolar, SolarMonth: 0, SolarDay: 1},
		{Kind: KindSolar, SolarMonth: 13, SolarDay: 1},
		{Kind: KindSolar, SolarMonth: 2, SolarDay: 30},
		{Kind: KindSolar, SolarMonth: 4, SolarDay: 31},
		{Kind: KindLunar, LunarMonth: 1, LunarDay: 31},
		{Kind: KindLunar, LunarMonth: 0, LunarDay: 1},
		{Kind: KindLunarMonthly, LunarDay: 0},
		{Kind: KindLunarLastDay, LunarMonth: 13},
		{Kind: KindSolarTerm, SolarTermLongitude: 10},
		{Kind: KindSolarTerm, SolarTermLongitude: 360},
		{Kind: KindSolarTerm, SolarTermLongitude: -15},
	}
	for _, r := range bad {
		if err := r.Validate(); !errors.Is(err, ErrInvalidRecurrenceRule) {
			t.Errorf("%+v: err = %v", r, err)
		}
	}
}
