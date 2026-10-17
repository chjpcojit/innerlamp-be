package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func goodDoc() map[string]any {
	return map[string]any{
		"meta":   map[string]any{"format": 1, "lang": "vi", "anything": "ignored"},
		"groups": []any{map[string]any{"code": "folk", "name": "Dân gian", "sort_order": 1}},
		"events": []any{
			event("tet", "lunar", map[string]any{"lunar_month": 1, "lunar_day": 1}),
			event("xuan-phan", "solar_term", map[string]any{"solar_term_longitude": 0}),
		},
	}
}

func event(code, kind string, params map[string]any) map[string]any {
	e := map[string]any{
		"code": code, "kind": kind, "is_public_holiday": false, "sort_order": 1,
		"groups": []any{"folk"}, "name": "N", "short_description": "s", "long_description": "l",
		"guidance": "g", "image_url": nil, "source": "test",
	}
	for k, v := range params {
		e[k] = v
	}
	return e
}

func encode(t *testing.T, doc map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	return b
}

func TestParseCatalogueReadsEverything(t *testing.T) {
	c, _, err := parseCatalogue(encode(t, goodDoc()))
	require.NoError(t, err)
	assert.Equal(t, "vi", c.Lang)
	require.Len(t, c.Groups, 1)
	require.Len(t, c.Events, 2)
	assert.Equal(t, domain.RecurrenceRule{Kind: domain.KindLunar, LunarMonth: 1, LunarDay: 1}, c.Events[0].Rule)
	assert.Equal(t, domain.RecurrenceRule{Kind: domain.KindSolarTerm, SolarTermLongitude: 0}, c.Events[1].Rule)
	assert.Nil(t, c.Events[0].ImageURL)
	assert.Empty(t, c.Verify())
}

func TestParseCatalogueIsMajor(t *testing.T) {
	d := goodDoc()
	d["events"].([]any)[0].(map[string]any)["is_major"] = true
	c, _, err := parseCatalogue(encode(t, d))
	require.NoError(t, err)
	assert.True(t, c.Events[0].IsMajor)
	assert.False(t, c.Events[1].IsMajor, "absent is_major is false")

	d = goodDoc()
	d["events"].([]any)[0].(map[string]any)["is_major"] = "yes"
	_, _, err = parseCatalogue(encode(t, d))
	require.Error(t, err, "is_major must be a boolean")
}

func TestParseCatalogueRefuses(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(d map[string]any)
		want   string
	}{
		{"unknown key", func(d map[string]any) {
			d["events"].([]any)[0].(map[string]any)["lunar_dayy"] = 3
		}, "unknown field"},
		{"no meta", func(d map[string]any) { delete(d, "meta") }, `missing "meta"`},
		{"wrong format", func(d map[string]any) { d["meta"].(map[string]any)["format"] = 2 }, "meta.format"},
		{"no lang", func(d map[string]any) { delete(d["meta"].(map[string]any), "lang") }, "meta.lang"},
		{"unknown kind", func(d map[string]any) {
			d["events"].([]any)[0].(map[string]any)["kind"] = "weekly"
		}, "unknown kind"},
		{"missing param", func(d map[string]any) {
			delete(d["events"].([]any)[0].(map[string]any), "lunar_day")
		}, "requires lunar_day"},
		{"missing longitude", func(d map[string]any) {
			delete(d["events"].([]any)[1].(map[string]any), "solar_term_longitude")
		}, "requires solar_term_longitude"},
		{"stray param", func(d map[string]any) {
			d["events"].([]any)[0].(map[string]any)["solar_month"] = 4
		}, "does not take solar_month"},
		{"no source", func(d map[string]any) {
			d["events"].([]any)[0].(map[string]any)["source"] = ""
		}, "source is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := goodDoc()
			tc.mutate(d)
			_, _, err := parseCatalogue(encode(t, d))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestParseCatalogueRejectsTrailingData(t *testing.T) {
	_, _, err := parseCatalogue(append(encode(t, goodDoc()), []byte(`{"x":1}`)...))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected data")
}

func TestParseCatalogueNormalisesToNFC(t *testing.T) {
	d := goodDoc()
	d["events"].([]any)[0].(map[string]any)["name"] = "Te\u0302\u0301t" // decomposed
	c, rep, err := parseCatalogue(encode(t, d))
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Normalized)
	assert.Equal(t, "Tết", c.Events[0].Name)
}

// The real data file must always load: this is the gate that keeps a bad edit
// of assets/data/calendar-events.json from reaching the seed command.
func TestRealDataFileIsValid(t *testing.T) {
	src, err := os.ReadFile("../../../../assets/data/calendar-events.json")
	require.NoError(t, err)
	c, _, err := parseCatalogue(src)
	require.NoError(t, err)
	issues := c.Verify()
	assert.Empty(t, issues, strings.Join(issues, "\n"))
	assert.GreaterOrEqual(t, len(c.Events), 40)
	groups := map[string]bool{}
	for _, g := range c.Groups {
		groups[g.Code] = true
	}
	for _, want := range []string{"buddhist", "vietnam", "folk"} {
		assert.True(t, groups[want], want)
	}
}
