package main

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalDoc is a consistent one-of-everything data file as a generic map, so
// each test can break exactly one thing.
func minimalDoc() map[string]any {
	return map[string]any{
		"meta": map[string]any{
			"version": "2026-09-25",
			"format":  1,
			"source":  map[string]any{"repository": "https://example.test/repo", "commit": "abc123"},
			"counts": map[string]any{
				"new_provinces": 1, "new_wards": 1, "old_provinces": 1, "old_districts": 1,
				"old_wards": 1, "ward_mappings": 1, "district_mappings": 0,
			},
			"corrections": []any{map[string]any{"kind": "x"}},
			"unresolved":  []any{},
		},
		"new_provinces": []any{map[string]any{"code": "79", "name": "Hồ Chí Minh", "full_name": "Thành phố Hồ Chí Minh", "type": "city", "sort_order": 1}},
		"new_wards":     []any{map[string]any{"code": "26737", "province_code": "79", "name": "Tân Định", "full_name": "Phường Tân Định", "type": "ward", "sort_order": 1}},
		"old_provinces": []any{map[string]any{"code": "79", "name": "Hồ Chí Minh", "full_name": "Thành phố Hồ Chí Minh", "type": "city", "sort_order": 1}},
		"old_districts": []any{map[string]any{"code": "760", "province_code": "79", "name": "Quận 1", "full_name": "Quận 1", "type": "urban_district", "sort_order": 1}},
		"old_wards": []any{map[string]any{"code": "26737", "district_code": "760", "province_code": "79",
			"name": "Đa Kao", "full_name": "Phường Đa Kao", "type": "ward", "sort_order": 1}},
		"ward_mappings":     []any{map[string]any{"old_ward_code": "26737", "new_ward_code": "26737", "mapping_type": "ambiguous", "note": "Nhập một phần"}},
		"district_mappings": []any{},
	}
}

func encode(t *testing.T, doc map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	return b
}

func TestParseDataset_ReadsEveryPart(t *testing.T) {
	ds, rep, err := parseDataset(encode(t, minimalDoc()))
	require.NoError(t, err)

	assert.Equal(t, "2026-09-25", ds.Version)
	assert.Equal(t, "https://example.test/repo @ abc123", ds.SourceNote)
	require.Len(t, ds.CurrentProvinces, 1)
	assert.Equal(t, "Thành phố Hồ Chí Minh", ds.CurrentProvinces[0].FullName)
	assert.Equal(t, "city", ds.CurrentProvinces[0].Type)
	require.Len(t, ds.CurrentWards, 1)
	assert.Equal(t, "79", ds.CurrentWards[0].ProvinceCode)
	require.Len(t, ds.LegacyDistricts, 1)
	assert.Equal(t, "79", ds.LegacyDistricts[0].ProvinceCode)
	require.Len(t, ds.LegacyWards, 1)
	assert.Equal(t, "760", ds.LegacyWards[0].DistrictCode)
	assert.Equal(t, "79", ds.LegacyWards[0].ProvinceCode, "kept so Verify can compare it with the district's")
	require.Len(t, ds.WardMappings, 1)
	assert.Equal(t, "ambiguous", ds.WardMappings[0].MappingType)
	assert.Equal(t, "Nhập một phần", ds.WardMappings[0].Note)
	assert.Empty(t, ds.Verify(), "the minimal document is consistent")

	assert.Equal(t, 1, rep.Corrections)
	assert.Zero(t, rep.Unresolved)
	assert.Zero(t, rep.Normalized)
}

func TestParseDataset_DistrictMappingsAreReadForVerify(t *testing.T) {
	doc := minimalDoc()
	doc["district_mappings"] = []any{map[string]any{"old_district_code": "760", "new_ward_code": "26737", "mapping_type": "merged", "note": "island"}}
	doc["meta"].(map[string]any)["counts"].(map[string]any)["district_mappings"] = 1

	ds, _, err := parseDataset(encode(t, doc))

	require.NoError(t, err)
	require.Len(t, ds.DistrictMappings, 1)
	assert.Equal(t, "760", ds.DistrictMappings[0].LegacyDistrictCode)
	assert.Equal(t, "26737", ds.DistrictMappings[0].CurrentWardCode)
}

func TestParseDataset_NormalisesToNFCAndReportsIt(t *testing.T) {
	doc := minimalDoc()
	// "Tân Định" written with a combining circumflex (NFD) — what a copy from a
	// differently-encoded source looks like.
	nfd := "Tân Định"
	doc["new_wards"].([]any)[0].(map[string]any)["name"] = nfd

	ds, rep, err := parseDataset(encode(t, doc))

	require.NoError(t, err)
	assert.Equal(t, "Tân Định", ds.CurrentWards[0].Name, "stored form is the composed (NFC) one")
	assert.NotEqual(t, nfd, ds.CurrentWards[0].Name)
	assert.Equal(t, 1, rep.Normalized, "the change is reported, not silent")
}

func TestParseDataset_RefusesAMalformedFile(t *testing.T) {
	set := func(path string, v any) func(map[string]any) {
		return func(d map[string]any) {
			parts := strings.Split(path, ".")
			cur := d
			for _, p := range parts[:len(parts)-1] {
				if m, ok := cur[p].(map[string]any); ok {
					cur = m
					continue
				}
				cur = cur[p].([]any)[0].(map[string]any)
			}
			if v == nil {
				delete(cur, parts[len(parts)-1])
			} else {
				cur[parts[len(parts)-1]] = v
			}
		}
	}
	tests := []struct {
		name    string
		mutate  func(map[string]any)
		wantErr string
	}{
		{"unknown field on a unit", set("new_wards.zzz", "x"), "unknown field"},
		{"misspelt key becomes an unknown field", set("old_wards.distict_code", "760"), "unknown field"},
		{"unsupported format", set("meta.format", 2), "format"},
		{"missing format", set("meta.format", nil), "format"},
		{"empty version", set("meta.version", ""), "version"},
		{"missing meta", func(d map[string]any) { delete(d, "meta") }, "meta"},
		{"missing array", func(d map[string]any) { delete(d, "old_districts") }, "old_districts"},
		{"empty array", func(d map[string]any) { d["new_provinces"] = []any{} }, "new_provinces"},
		{"empty code", set("new_wards.code", ""), "code is empty"},
		{"empty name", set("old_wards.name", ""), "name is empty"},
		{"sort_order below 1", set("old_wards.sort_order", 0), "sort_order"},
		{"empty mapping type", set("ward_mappings.mapping_type", ""), "mapping_type is empty"},
		{"declared counts disagree with the rows", set("meta.counts.new_wards", 3321), "meta.counts"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := minimalDoc()
			tc.mutate(doc)

			_, _, err := parseDataset(encode(t, doc))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}

	t.Run("not JSON", func(t *testing.T) {
		_, _, err := parseDataset([]byte("not json"))
		assert.Error(t, err)
	})
	t.Run("data after the document", func(t *testing.T) {
		_, _, err := parseDataset(append(encode(t, minimalDoc()), []byte(` {"extra":1}`)...))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "after the JSON document")
	})
}

func TestParseDataset_FileWithoutCountsIsStillAccepted(t *testing.T) {
	doc := minimalDoc()
	delete(doc["meta"].(map[string]any), "counts")

	_, _, err := parseDataset(encode(t, doc))

	assert.NoError(t, err, "counts are a cross-check, not a requirement")
}

// The real data file must always be loadable and consistent; this is what
// catches a bad edit before it reaches a database. The file is untracked by
// default (.gitignore ignores *.json), so on a checkout without it the test
// skips instead of failing.
func TestParseDataset_RealDataFile(t *testing.T) {
	src, err := os.ReadFile("../../../../assets/data/administrative-units.json")
	if os.IsNotExist(err) {
		t.Skip("assets/data/administrative-units.json is not in this checkout (it is gitignored)")
	}
	require.NoError(t, err)

	ds, rep, err := parseDataset(src)
	require.NoError(t, err)

	assert.Empty(t, ds.Verify(), "the shipped dataset must pass every integrity rule")
	assert.Len(t, ds.CurrentProvinces, 34, "34 province-level units since 1 July 2025")
	assert.Len(t, ds.LegacyProvinces, 63)
	assert.Positive(t, rep.Corrections, "corrections are documented in meta, not silent")

	// The precomputed Vietnamese order compares digits as numbers: "Phường 9" comes
	// before "Phường 11", which plain string order would reverse.
	type named struct {
		name  string
		order int
	}
	var wards []named
	for _, w := range ds.LegacyWards {
		if w.DistrictCode == "774" { // a district with numbered wards
			wards = append(wards, named{w.Name, w.SortOrder})
		}
	}
	slices.SortFunc(wards, func(a, b named) int { return a.order - b.order })
	idx := func(name string) int {
		return slices.IndexFunc(wards, func(n named) bool { return n.name == name })
	}
	require.NotEqual(t, -1, idx("Phường 9"))
	require.NotEqual(t, -1, idx("Phường 11"))
	assert.Less(t, idx("Phường 9"), idx("Phường 11"))
}
