package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tod(t *testing.T, s string) TimeOfDay {
	t.Helper()
	v, err := ParseTimeOfDay(s)
	require.NoError(t, err)
	return v
}

func todPtr(t *testing.T, s string) *TimeOfDay { v := tod(t, s); return &v }

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		in      string
		want    TimeOfDay
		wantErr bool
	}{
		{"00:00", 0, false},
		{"08:30", 510, false},
		{"23:59", 1439, false},
		{"24:00", 0, true},
		{"7:5", 0, true},
		{"07:60", 0, true},
		{"", 0, true},
		{"08:00:00", 0, true},
		{"ab:cd", 0, true},
		{"+8:00", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseTimeOfDay(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.in, got.String())
		})
	}
}

func TestSortTempleSchedule(t *testing.T) {
	t.Run("items by start_time then sort_order then id", func(t *testing.T) {
		items := []TempleScheduleItem{
			{ID: "c", StartTime: tod(t, "09:00")},
			{ID: "b", StartTime: tod(t, "04:00"), SortOrder: 2},
			{ID: "z", StartTime: tod(t, "04:00"), SortOrder: 1},
			{ID: "a", StartTime: tod(t, "04:00"), SortOrder: 2},
		}
		got := SortTempleSchedule(items)
		var ids []string
		for _, it := range got {
			ids = append(ids, it.ID)
		}
		assert.Equal(t, []string{"z", "a", "b", "c"}, ids)
	})

	t.Run("steps by position, not by time (crossing midnight)", func(t *testing.T) {
		items := []TempleScheduleItem{{ID: "a", Steps: []TempleScheduleStep{
			{Position: 3, Time: todPtr(t, "00:15"), Text: "c"},
			{Position: 1, Time: todPtr(t, "23:00"), Text: "a"},
			{Position: 2, Time: todPtr(t, "23:45"), Text: "b"},
		}}}
		got := SortTempleSchedule(items)
		assert.Equal(t, []int{1, 2, 3}, []int{got[0].Steps[0].Position, got[0].Steps[1].Position, got[0].Steps[2].Position})
	})

	t.Run("nil steps become empty non-nil", func(t *testing.T) {
		got := SortTempleSchedule([]TempleScheduleItem{{ID: "a"}})
		assert.NotNil(t, got[0].Steps)
		assert.Empty(t, got[0].Steps)
	})

	weekdays := []struct {
		name string
		in   []int
		want []int
	}{
		{"nil is every day", nil, []int{}},
		{"duplicates removed, ascending", []int{6, 0, 6, 0}, []int{0, 6}},
		{"all seven collapse to every day", []int{3, 1, 2, 0, 6, 5, 4}, []int{}},
		{"seven with duplicates still all", []int{0, 1, 2, 3, 4, 5, 6, 0}, []int{}},
	}
	for _, tc := range weekdays {
		t.Run(tc.name, func(t *testing.T) {
			got := SortTempleSchedule([]TempleScheduleItem{{ID: "a", Weekdays: tc.in}})
			assert.Equal(t, tc.want, got[0].Weekdays)
		})
	}

	t.Run("does not mutate the caller's steps slice", func(t *testing.T) {
		steps := []TempleScheduleStep{{Position: 2, Text: "b"}, {Position: 1, Text: "a"}}
		SortTempleSchedule([]TempleScheduleItem{{ID: "a", Steps: steps}})
		assert.Equal(t, 2, steps[0].Position)
	})
}

func validItem(t *testing.T) TempleScheduleItem {
	return TempleScheduleItem{
		ID: "tsch_1", TempleID: "temple_1", Name: "Công phu sáng",
		StartTime: tod(t, "04:00"), EndTime: todPtr(t, "05:00"), Weekdays: []int{1, 2},
		Steps: []TempleScheduleStep{{Position: 1, Time: todPtr(t, "04:00"), Text: "Tụng kinh"}},
	}
}

func TestTempleScheduleCatalogueVerify(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c := TempleScheduleCatalogue{Source: "mock", Items: []TempleScheduleItem{validItem(t)}}
		assert.NoError(t, c.Verify())
	})

	mut := func(f func(*TempleScheduleItem)) TempleScheduleCatalogue {
		it := validItem(t)
		f(&it)
		return TempleScheduleCatalogue{Source: "mock", Items: []TempleScheduleItem{it}}
	}
	tests := []struct {
		name string
		c    TempleScheduleCatalogue
		want string
	}{
		{"source empty", TempleScheduleCatalogue{Items: []TempleScheduleItem{validItem(t)}}, "source is empty"},
		{"no items", TempleScheduleCatalogue{Source: "mock"}, "no items"},
		{"duplicate id", TempleScheduleCatalogue{Source: "mock", Items: []TempleScheduleItem{validItem(t), validItem(t)}}, "duplicate id"},
		{"empty id", mut(func(i *TempleScheduleItem) { i.ID = "" }), "id is empty"},
		{"empty temple", mut(func(i *TempleScheduleItem) { i.TempleID = "" }), "temple_id is empty"},
		{"blank name", mut(func(i *TempleScheduleItem) { i.Name = "  " }), "name is empty"},
		{"end equals start", mut(func(i *TempleScheduleItem) { i.EndTime = todPtr(t, "04:00") }), "end_time equals start_time"},
		{"weekday 7", mut(func(i *TempleScheduleItem) { i.Weekdays = []int{7} }), "outside 0..6"},
		{"weekday -1", mut(func(i *TempleScheduleItem) { i.Weekdays = []int{-1} }), "outside 0..6"},
		{"step text blank", mut(func(i *TempleScheduleItem) { i.Steps[0].Text = " " }), "text is empty"},
		{"step position 0", mut(func(i *TempleScheduleItem) { i.Steps[0].Position = 0 }), "position must be >= 1"},
		{"two steps same position", mut(func(i *TempleScheduleItem) {
			i.Steps = append(i.Steps, TempleScheduleStep{Position: 1, Text: "x"})
		}), "share position 1"},
		{"step time out of range", mut(func(i *TempleScheduleItem) { v := TimeOfDay(1440); i.Steps[0].Time = &v }), "time out of range"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Verify()
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidTempleSchedule)
			assert.Contains(t, err.Error(), tc.want)
		})
	}

	t.Run("end before start (crosses midnight) is valid", func(t *testing.T) {
		c := mut(func(i *TempleScheduleItem) { i.StartTime = tod(t, "23:00"); i.EndTime = todPtr(t, "01:00") })
		assert.NoError(t, c.Verify())
	})
}
