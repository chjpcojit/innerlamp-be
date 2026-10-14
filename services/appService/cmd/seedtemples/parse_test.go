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
   "location":{"lat":21.0478,"lng":105.8368,"address":"Đường Thanh Niên","province_code":"01","ward_code":"00103"},"operating_hours":"07:30 - 18:00",
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
	assert.Equal(t, "Đường Thanh Niên", tp.Address, "address is the part below ward level")
	assert.Equal(t, "01", tp.ProvinceCode)
	assert.Equal(t, "00103", tp.WardCode)
	assert.Empty(t, tp.Province, "unit names come from the dataset in resolveAddresses, not from the file")
	assert.Empty(t, tp.Ward)
	assert.Equal(t, "I", tp.Introduction)
	assert.Equal(t, "A", tp.ArchitectureHeritage)
	assert.Equal(t, []string{"N1", "N2"}, tp.VisitingNotes)
}

// withLocation returns oneTemple with its whole location object replaced by loc.
func withLocation(loc string) string {
	const old = `{"lat":21.0478,"lng":105.8368,"address":"Đường Thanh Niên","province_code":"01","ward_code":"00103"}`
	return strings.Replace(oneTemple, old, loc, 1)
}

// TestParseTemples_AddressFields pins AC 34 of spec temple v1.7: province_code and
// ward_code are required, address must be present but may be empty, values are
// trimmed. Whether the codes are real is resolveAddresses' job (resolve_test.go).
func TestParseTemples_AddressFields(t *testing.T) {
	t.Run("values are trimmed", func(t *testing.T) {
		got, err := parseTemples([]byte(withLocation(`{"lat":1,"lng":2,"address":"  73 Mai Thị Lựu ","province_code":" 79 ","ward_code":" 26737 "}`)))
		require.NoError(t, err)
		assert.Equal(t, "73 Mai Thị Lựu", got[0].Address)
		assert.Equal(t, "79", got[0].ProvinceCode)
		assert.Equal(t, "26737", got[0].WardCode)
	})

	t.Run("an empty address is accepted (A-004)", func(t *testing.T) {
		got, err := parseTemples([]byte(withLocation(`{"lat":1,"lng":2,"address":"","province_code":"01","ward_code":"00103"}`)))
		require.NoError(t, err)
		assert.Empty(t, got[0].Address)
	})

	t.Run("a legacy location.province is kept for the typo guard, not used as a name", func(t *testing.T) {
		got, err := parseTemples([]byte(withLocation(`{"lat":1,"lng":2,"address":"x","province":" Hà Nội ","province_code":"01","ward_code":"00103"}`)))
		require.NoError(t, err)
		assert.Equal(t, "Hà Nội", got[0].Province)
	})

	for name, loc := range map[string]string{
		"missing province_code": `{"lat":1,"lng":2,"address":"x","ward_code":"00103"}`,
		"blank province_code":   `{"lat":1,"lng":2,"address":"x","province_code":"  ","ward_code":"00103"}`,
		"missing ward_code":     `{"lat":1,"lng":2,"address":"x","province_code":"01"}`,
		"blank ward_code":       `{"lat":1,"lng":2,"address":"x","province_code":"01","ward_code":""}`,
		"missing address":       `{"lat":1,"lng":2,"province_code":"01","ward_code":"00103"}`,
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := parseTemples([]byte(withLocation(loc)))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "temple_north_001")
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
	for _, tp := range got {
		assert.NotEmpty(t, tp.ProvinceCode, tp.ID)
		assert.NotEmpty(t, tp.WardCode, tp.ID)
	}
}
