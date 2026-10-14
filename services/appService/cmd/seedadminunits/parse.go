package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/text/unicode/norm"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// supportedFormat is the only layout of assets/data/administrative-units.json
// this seeder understands (meta.format). A different layout is refused rather
// than half-read.
const supportedFormat = 1

// Shapes of the data file. Every unit array is decoded strictly: an unknown
// field is an error, so a renamed or misspelt key cannot silently load as an
// empty string. meta is kept raw because it is documentation-heavy and the
// seeder needs only a few of its fields.
type fileJSON struct {
	Meta             json.RawMessage       `json:"meta"`
	NewProvinces     []provinceJSON        `json:"new_provinces"`
	NewWards         []newWardJSON         `json:"new_wards"`
	OldProvinces     []provinceJSON        `json:"old_provinces"`
	OldDistricts     []oldDistrictJSON     `json:"old_districts"`
	OldWards         []oldWardJSON         `json:"old_wards"`
	WardMappings     []wardMappingJSON     `json:"ward_mappings"`
	DistrictMappings []districtMappingJSON `json:"district_mappings"`
}

type provinceJSON struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	Type      string `json:"type"`
	SortOrder int    `json:"sort_order"`
}

type newWardJSON struct {
	Code         string `json:"code"`
	ProvinceCode string `json:"province_code"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Type         string `json:"type"`
	SortOrder    int    `json:"sort_order"`
}

type oldDistrictJSON struct {
	Code         string `json:"code"`
	ProvinceCode string `json:"province_code"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Type         string `json:"type"`
	SortOrder    int    `json:"sort_order"`
}

type oldWardJSON struct {
	Code         string `json:"code"`
	DistrictCode string `json:"district_code"`
	ProvinceCode string `json:"province_code"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Type         string `json:"type"`
	SortOrder    int    `json:"sort_order"`
}

type wardMappingJSON struct {
	OldWardCode string `json:"old_ward_code"`
	NewWardCode string `json:"new_ward_code"`
	MappingType string `json:"mapping_type"`
	Note        string `json:"note"`
}

type districtMappingJSON struct {
	OldDistrictCode string `json:"old_district_code"`
	NewWardCode     string `json:"new_ward_code"`
	MappingType     string `json:"mapping_type"`
	Note            string `json:"note"`
}

// metaJSON is the part of meta the seeder reads.
type metaJSON struct {
	Version string `json:"version"`
	Format  int    `json:"format"`
	Source  struct {
		Repository string `json:"repository"`
		Commit     string `json:"commit"`
	} `json:"source"`
	Counts *struct {
		NewProvinces     int `json:"new_provinces"`
		NewWards         int `json:"new_wards"`
		OldProvinces     int `json:"old_provinces"`
		OldDistricts     int `json:"old_districts"`
		OldWards         int `json:"old_wards"`
		WardMappings     int `json:"ward_mappings"`
		DistrictMappings int `json:"district_mappings"`
	} `json:"counts"`
	// Corrections and Unresolved are only counted, to be printed in the report:
	// a loader that hides them would be repairing data silently.
	Corrections []json.RawMessage `json:"corrections"`
	Unresolved  []json.RawMessage `json:"unresolved"`
}

// parseReport is what parsing learned beyond the dataset itself.
type parseReport struct {
	// Normalized is how many strings were changed by NFC normalisation. It is
	// lossless (canonically equivalent text) but reported, never silent.
	Normalized int
	// Corrections / Unresolved are the sizes of meta.corrections and
	// meta.unresolved: edits already applied to the file, and rows left out.
	Corrections int
	Unresolved  int
}

// parseDataset decodes the data file into a domain.Dataset. It checks the
// file's own structure — layout version, required keys, non-empty codes and
// names, declared counts — but not the dataset's consistency; that is
// domain.Dataset.Verify, run by the caller on the result. It repairs nothing.
func parseDataset(src []byte) (domain.Dataset, parseReport, error) {
	var f fileJSON
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return domain.Dataset{}, parseReport{}, fmt.Errorf("decode: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return domain.Dataset{}, parseReport{}, errors.New("decode: unexpected data after the JSON document")
	}

	if len(f.Meta) == 0 {
		return domain.Dataset{}, parseReport{}, errors.New(`missing "meta"`)
	}
	var meta metaJSON
	if err := json.Unmarshal(f.Meta, &meta); err != nil {
		return domain.Dataset{}, parseReport{}, fmt.Errorf("decode meta: %w", err)
	}
	if meta.Format != supportedFormat {
		return domain.Dataset{}, parseReport{}, fmt.Errorf("meta.format is %d, this seeder reads format %d", meta.Format, supportedFormat)
	}
	if meta.Version == "" {
		return domain.Dataset{}, parseReport{}, errors.New("meta.version is empty")
	}

	required := []struct {
		key string
		n   int
	}{
		{"new_provinces", len(f.NewProvinces)}, {"new_wards", len(f.NewWards)},
		{"old_provinces", len(f.OldProvinces)}, {"old_districts", len(f.OldDistricts)},
		{"old_wards", len(f.OldWards)}, {"ward_mappings", len(f.WardMappings)},
	}
	for _, r := range required {
		if r.n == 0 {
			return domain.Dataset{}, parseReport{}, fmt.Errorf("%q is missing or empty", r.key)
		}
	}

	var rep parseReport
	s := func(v string) string {
		out := norm.NFC.String(v)
		if out != v {
			rep.Normalized++
		}
		return out
	}

	ds := domain.Dataset{
		Version:    meta.Version,
		SourceNote: s(meta.Source.Repository + " @ " + meta.Source.Commit),
	}
	var problems []error
	need := func(what string, i int, fields map[string]string, order int) {
		for name, v := range fields {
			if v == "" {
				problems = append(problems, fmt.Errorf("%s[%d]: %s is empty", what, i, name))
			}
		}
		if order < 1 {
			problems = append(problems, fmt.Errorf("%s[%d]: sort_order must be >= 1, got %d", what, i, order))
		}
	}

	for i, p := range f.NewProvinces {
		need("new_provinces", i, map[string]string{"code": p.Code, "name": p.Name, "full_name": p.FullName, "type": p.Type}, p.SortOrder)
		ds.CurrentProvinces = append(ds.CurrentProvinces, domain.CurrentProvince{
			Code: p.Code, Name: s(p.Name), FullName: s(p.FullName), Type: p.Type, SortOrder: p.SortOrder})
	}
	for i, w := range f.NewWards {
		need("new_wards", i, map[string]string{"code": w.Code, "province_code": w.ProvinceCode, "name": w.Name, "full_name": w.FullName, "type": w.Type}, w.SortOrder)
		ds.CurrentWards = append(ds.CurrentWards, domain.CurrentWard{
			Code: w.Code, ProvinceCode: w.ProvinceCode, Name: s(w.Name), FullName: s(w.FullName), Type: w.Type, SortOrder: w.SortOrder})
	}
	for i, p := range f.OldProvinces {
		need("old_provinces", i, map[string]string{"code": p.Code, "name": p.Name, "full_name": p.FullName, "type": p.Type}, p.SortOrder)
		ds.LegacyProvinces = append(ds.LegacyProvinces, domain.LegacyProvince{
			Code: p.Code, Name: s(p.Name), FullName: s(p.FullName), Type: p.Type, SortOrder: p.SortOrder})
	}
	for i, d := range f.OldDistricts {
		need("old_districts", i, map[string]string{"code": d.Code, "province_code": d.ProvinceCode, "name": d.Name, "full_name": d.FullName, "type": d.Type}, d.SortOrder)
		ds.LegacyDistricts = append(ds.LegacyDistricts, domain.LegacyDistrict{
			Code: d.Code, ProvinceCode: d.ProvinceCode, Name: s(d.Name), FullName: s(d.FullName), Type: d.Type, SortOrder: d.SortOrder})
	}
	for i, w := range f.OldWards {
		need("old_wards", i, map[string]string{"code": w.Code, "district_code": w.DistrictCode, "province_code": w.ProvinceCode, "name": w.Name, "full_name": w.FullName, "type": w.Type}, w.SortOrder)
		ds.LegacyWards = append(ds.LegacyWards, domain.LegacyWard{
			Code: w.Code, DistrictCode: w.DistrictCode, ProvinceCode: w.ProvinceCode,
			Name: s(w.Name), FullName: s(w.FullName), Type: w.Type, SortOrder: w.SortOrder})
	}
	for i, m := range f.WardMappings {
		need("ward_mappings", i, map[string]string{"old_ward_code": m.OldWardCode, "new_ward_code": m.NewWardCode, "mapping_type": m.MappingType}, 1)
		ds.WardMappings = append(ds.WardMappings, domain.WardMapping{
			LegacyWardCode: m.OldWardCode, CurrentWardCode: m.NewWardCode, MappingType: m.MappingType, Note: s(m.Note)})
	}
	// District mappings are read and kept in the Dataset so Verify can count
	// their targets, but the repository does not store them (district-level
	// conversion is deferred).
	for i, m := range f.DistrictMappings {
		need("district_mappings", i, map[string]string{"old_district_code": m.OldDistrictCode, "new_ward_code": m.NewWardCode, "mapping_type": m.MappingType}, 1)
		ds.DistrictMappings = append(ds.DistrictMappings, domain.DistrictMapping{
			LegacyDistrictCode: m.OldDistrictCode, CurrentWardCode: m.NewWardCode, MappingType: m.MappingType, Note: s(m.Note)})
	}
	if len(problems) > 0 {
		return domain.Dataset{}, parseReport{}, fmt.Errorf("%d malformed row(s), first: %w", len(problems), problems[0])
	}

	// The file declares how many rows it should hold; a truncated or hand-edited
	// file that no longer matches is refused before it reaches Verify.
	if meta.Counts != nil {
		got := ds.Counts()
		want := domain.DatasetCounts{
			CurrentProvinces: meta.Counts.NewProvinces, CurrentWards: meta.Counts.NewWards,
			LegacyProvinces: meta.Counts.OldProvinces, LegacyDistricts: meta.Counts.OldDistricts,
			LegacyWards: meta.Counts.OldWards, WardMappings: meta.Counts.WardMappings,
			DistrictMappings: meta.Counts.DistrictMappings,
		}
		if got != want {
			return domain.Dataset{}, parseReport{}, fmt.Errorf("meta.counts does not match the rows: file holds %+v, meta declares %+v", got, want)
		}
	}

	rep.Corrections = len(meta.Corrections)
	rep.Unresolved = len(meta.Unresolved)
	return ds, rep, nil
}
