package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

const goodFile = `{
  "meta": {"format": 1, "source": "mock"},
  "events": [
    {"id": "e1", "temple_id": "t1", "title": "Khoá tu", "description": "d", "kind": "KHOA_TU",
     "starts_at": "2026-11-07T06:00:00+07:00", "ends_at": "2026-11-08T17:00:00+07:00",
     "is_all_day": false, "status": "SCHEDULED", "audience_note": "n", "contact_info": "c",
     "image_url": null, "recurrence": "NONE"},
    {"id": "e2", "temple_id": "t1", "title": "Lễ", "kind": "LE_CAU_AN",
     "starts_at": "2026-10-04T08:00:00+07:00", "ends_at": "2026-10-04T11:00:00+07:00",
     "status": "SCHEDULED", "recurrence": "WEEKLY", "recurrence_weekday": 0, "recurrence_until": "2027-06-30"},
    {"id": "e3", "temple_id": "t1", "title": "Sám hối", "kind": "KHAC",
     "starts_at": "2026-10-01T19:00:00+07:00", "ends_at": "2026-10-01T21:00:00+07:00",
     "status": "POSTPONED", "recurrence": "LUNAR_MONTHLY", "recurrence_lunar_days": [14, 30]}
  ]
}`

func TestParseCatalogue_Valid(t *testing.T) {
	c, rep, err := parseCatalogue([]byte(goodFile))
	require.NoError(t, err)
	assert.Equal(t, "mock", c.Source)
	assert.Equal(t, 0, rep.Normalized)
	require.Len(t, c.Events, 3)

	one := c.Events[0]
	assert.True(t, time.Date(2026, 11, 7, 6, 0, 0, 0, time.FixedZone("", 7*3600)).Equal(one.StartsAt))
	assert.Equal(t, domain.RecurrenceNone, one.Recurrence)
	assert.Nil(t, one.ImageURL)
	assert.Nil(t, one.Weekday)
	assert.Nil(t, one.Until)

	weekly := c.Events[1]
	require.NotNil(t, weekly.Weekday)
	assert.Equal(t, 0, *weekly.Weekday, "weekday 0 (Sunday) is present, not absent")
	require.NotNil(t, weekly.Until)
	assert.Equal(t, time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC), *weekly.Until)

	assert.Equal(t, []int{14, 30}, c.Events[2].LunarDays)
	assert.Equal(t, domain.TempleEventPostponed, c.Events[2].Status)

	loc := time.FixedZone("ICT", 7*3600)
	assert.NoError(t, c.Verify(loc))
}

func TestParseCatalogue_NormalisesNFC(t *testing.T) {
	src := `{"meta":{"format":1,"source":"m"},"events":[{"id":"e","temple_id":"t","title":"Lệ","kind":"KHAC",
	  "starts_at":"2026-11-07T06:00:00+07:00","ends_at":"2026-11-07T07:00:00+07:00","status":"SCHEDULED","recurrence":"NONE"}]}`
	c, rep, err := parseCatalogue([]byte(src))
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Normalized)
	assert.Equal(t, "Lệ", c.Events[0].Title)
}

func TestParseCatalogue_Rejects(t *testing.T) {
	ev := func(extra string) string {
		return `{"meta":{"format":1,"source":"m"},"events":[{"id":"e","temple_id":"t","title":"x","kind":"KHAC",
		"starts_at":"2026-11-07T06:00:00+07:00","ends_at":"2026-11-07T07:00:00+07:00","status":"SCHEDULED","recurrence":"NONE"` + extra + `}]}`
	}
	tests := []struct{ name, src, want string }{
		{"broken json", `{`, "decode"},
		{"unknown key", ev(`,"calendar_event_code":"x"`), "unknown field"},
		{"unknown meta key", `{"meta":{"format":1,"source":"m","x":1},"events":[]}`, "unknown field"},
		{"trailing data", goodFile + `{}`, "unexpected data"},
		{"wrong format", `{"meta":{"format":2,"source":"m"},"events":[]}`, "meta.format"},
		{"missing source", `{"meta":{"format":1},"events":[]}`, "meta.source"},
		{"starts_at without offset", `{"meta":{"format":1,"source":"m"},"events":[{"id":"e","starts_at":"2026-11-07T06:00:00","ends_at":"2026-11-07T07:00:00+07:00"}]}`, "starts_at"},
		{"ends_at not a time", `{"meta":{"format":1,"source":"m"},"events":[{"id":"e","starts_at":"2026-11-07T06:00:00+07:00","ends_at":"soon"}]}`, "ends_at"},
		{"until not a date", ev(`,"recurrence_until":"2027-13-01"`), "recurrence_until"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseCatalogue([]byte(tc.src))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestParseCatalogue_VerifyCatchesBadRecurrence(t *testing.T) {
	src := `{"meta":{"format":1,"source":"m"},"events":[{"id":"e","temple_id":"t","title":"x","kind":"KHAC",
	  "starts_at":"2026-11-07T06:00:00+07:00","ends_at":"2026-11-07T07:00:00+07:00","status":"SCHEDULED","recurrence":"WEEKLY"}]}`
	c, _, err := parseCatalogue([]byte(src))
	require.NoError(t, err)
	err = c.Verify(time.FixedZone("ICT", 7*3600))
	require.ErrorIs(t, err, domain.ErrInvalidTempleEvent)
	assert.Contains(t, err.Error(), `event "e"`)
}
