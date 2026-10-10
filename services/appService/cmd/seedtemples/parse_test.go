package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const oneTemple = `[
  /* MIỀN BẮC */
  {"id":"temple_north_001","name":"Chùa /* not a comment */","type":"CHUA","image_url":"","thumbnail_url":"",
   "aliases":["A"],"sect":"BẮC_TÔNG","founder":"F","current_abbot":"B","established_year":541,
   "heritage_rank":"DI_TICH_QUOC_GIA_DAC_BIET",
   "location":{"lat":21.0478,"lng":105.8368,"address":"Quận Tây Hồ, Hà Nội","province":"Hà Nội"},"operating_hours":"07:30 - 18:00",
   "introduction":"I","architecture_heritage":"A","visiting_notes":["N1","N2"],
   "amenities":{"parking_car":false,"parking_motorcycle":true,"vegetarian_meals":false,"summer_retreat":true}}
]`

func TestParseTemples(t *testing.T) {
	got, err := parseTemples([]byte(oneTemple))
	require.NoError(t, err)
	require.Len(t, got, 1)
	tp := got[0]
	assert.Equal(t, "Chùa /* not a comment */", tp.Name, "comment markers inside strings survive")
	assert.Equal(t, "north", tp.Region)
	assert.Equal(t, 2, tp.HeritagePriority)
	assert.True(t, tp.Amenities.ParkingMotorcycle)
	assert.True(t, tp.Amenities.SummerRetreat)
	assert.Equal(t, 21.0478, tp.Latitude)
	assert.Equal(t, "Quận Tây Hồ, Hà Nội", tp.Address)
	assert.Equal(t, "Hà Nội", tp.Province, "province comes from the file verbatim, not parsed from address")
	assert.Equal(t, "I", tp.Introduction)
	assert.Equal(t, "A", tp.ArchitectureHeritage)
	assert.Equal(t, []string{"N1", "N2"}, tp.VisitingNotes)
}

// withProvince returns oneTemple with its province replaced by raw (a JSON
// fragment: `"X"` for a value, or "" to drop the key).
func withProvince(raw string) string {
	const old = `,"province":"Hà Nội"`
	if raw == "" {
		return strings.Replace(oneTemple, old, "", 1)
	}
	return strings.Replace(oneTemple, old, `,"province":`+raw, 1)
}

func TestParseTemples_Province(t *testing.T) {
	t.Run("is trimmed", func(t *testing.T) {
		got, err := parseTemples([]byte(withProvince(`"  Hồ Chí Minh "`)))
		require.NoError(t, err)
		assert.Equal(t, "Hồ Chí Minh", got[0].Province)
	})

	for name, src := range map[string]string{
		"missing":                  withProvince(""),
		"blank":                    withProvince(`"   "`),
		"old name merged away":     withProvince(`"Thừa Thiên Huế"`),
		"old name Hà Nam":          withProvince(`"Hà Nam"`),
		"with prefix":              withProvince(`"Tỉnh Ninh Bình"`),
		"abbreviation":             withProvince(`"TP.HCM"`),
		"not a province at all":    withProvince(`"Atlantis"`),
		"wrong case":               withProvince(`"hà nội"`),
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := parseTemples([]byte(src))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "province")
		})
	}
}

func TestParseTemples_Rejects(t *testing.T) {
	tests := map[string]string{
		"unterminated comment": `[ /* oops {"id":"x"} ]`,
		"unknown rank":         `[{"id":"temple_north_001","heritage_rank":"NOPE"}]`,
		"bad id":               `[{"id":"weird","heritage_rank":"CHUA_XEP_HANG"}]`,
		"unknown region":       `[{"id":"temple_moon_001","heritage_rank":"CHUA_XEP_HANG"}]`,
		"duplicate id":         `[{"id":"temple_north_001","heritage_rank":"CHUA_XEP_HANG"},{"id":"temple_north_001","heritage_rank":"CHUA_XEP_HANG"}]`,
		"empty":                `[]`,
	}
	for name, src := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseTemples([]byte(src))
			assert.Error(t, err)
		})
	}
}

// The real data file must always be seedable; this is what catches a bad edit
// to temple.json before it reaches a database.
func TestParseTemples_RealDataFile(t *testing.T) {
	src, err := os.ReadFile("../../../../assets/data/temple.json")
	require.NoError(t, err)
	got, err := parseTemples(src)
	require.NoError(t, err)
	assert.Len(t, got, 93)
}
