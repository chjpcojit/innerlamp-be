package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func goodCatalogue() CalendarCatalogue {
	return CalendarCatalogue{
		Lang:   "vi",
		Groups: []CalendarGroup{{Code: "folk", Name: "Dân gian", SortOrder: 1}},
		Events: []CalendarEvent{{
			Code: "a", Name: "A", ShortDescription: "s", LongDescription: "l", Guidance: "g",
			Groups: []string{"folk"}, SortOrder: 1,
			Rule: RecurrenceRule{Kind: KindSolar, SolarMonth: 4, SolarDay: 30},
		}},
	}
}

func TestCatalogueVerifyOK(t *testing.T) {
	assert.Empty(t, goodCatalogue().Verify())
}

func TestCatalogueVerifyMonthlyLastOK(t *testing.T) {
	c := goodCatalogue()
	c.Events = append(c.Events, CalendarEvent{
		Code: "ram", Name: "R", ShortDescription: "s", LongDescription: "l", Guidance: "g",
		Groups: []string{"folk"}, SortOrder: 2, Rule: RecurrenceRule{Kind: KindLunarMonthly, LunarDay: 15}})
	assert.Empty(t, c.Verify())
}

func TestCatalogueVerifyProblems(t *testing.T) {
	long := strings.Repeat("x", 300)
	cases := []struct {
		name   string
		mutate func(c *CalendarCatalogue)
		want   string
	}{
		{"empty lang", func(c *CalendarCatalogue) { c.Lang = "" }, "lang is empty"},
		{"no events", func(c *CalendarCatalogue) { c.Events = nil }, "no events"},
		{"duplicate event", func(c *CalendarCatalogue) { c.Events = append(c.Events, c.Events[0]) }, "duplicate code"},
		{"empty event code", func(c *CalendarCatalogue) { c.Events[0].Code = "" }, "code is empty"},
		{"duplicate group", func(c *CalendarCatalogue) { c.Groups = append(c.Groups, c.Groups[0]) }, "group \"folk\": duplicate"},
		{"empty name", func(c *CalendarCatalogue) { c.Events[0].Name = "" }, "name is empty"},
		{"long name", func(c *CalendarCatalogue) { c.Events[0].Name = long }, "name is longer"},
		{"empty guidance", func(c *CalendarCatalogue) { c.Events[0].Guidance = "" }, "guidance is empty"},
		{"unknown kind", func(c *CalendarCatalogue) { c.Events[0].Rule = RecurrenceRule{Kind: "weekly"} }, "unknown kind"},
		{"missing params", func(c *CalendarCatalogue) { c.Events[0].Rule = RecurrenceRule{Kind: KindSolar} }, "solar_month"},
		{"term not multiple of 15", func(c *CalendarCatalogue) {
			c.Events[0].Rule = RecurrenceRule{Kind: KindSolarTerm, SolarTermLongitude: 10}
		}, "multiple of 15"},
		{"leap on non-lunar", func(c *CalendarCatalogue) { c.Events[0].Rule.IsLeapMonth = true }, "is_leap_month"},
		{"no group", func(c *CalendarCatalogue) { c.Events[0].Groups = nil }, "belongs to no group"},
		{"unknown group", func(c *CalendarCatalogue) { c.Events[0].Groups = []string{"nope"} }, "does not exist"},
		{"group twice", func(c *CalendarCatalogue) { c.Events[0].Groups = []string{"folk", "folk"} }, "listed twice"},
		{"sort order", func(c *CalendarCatalogue) { c.Events[0].SortOrder = 0 }, "sort_order must be >= 1"},
		{"monthly not last", func(c *CalendarCatalogue) {
			c.Events = append(c.Events, CalendarEvent{
				Code: "ram", Name: "R", ShortDescription: "s", LongDescription: "l", Guidance: "g",
				Groups: []string{"folk"}, SortOrder: 1, Rule: RecurrenceRule{Kind: KindLunarMonthly, LunarDay: 15}})
		}, "lunar_monthly sort_order"},
		{"empty image", func(c *CalendarCatalogue) { e := ""; c.Events[0].ImageURL = &e }, "image_url"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := goodCatalogue()
			tc.mutate(&c)
			issues := c.Verify()
			if assert.NotEmpty(t, issues) {
				assert.Contains(t, strings.Join(issues, "\n"), tc.want)
			}
		})
	}
}
