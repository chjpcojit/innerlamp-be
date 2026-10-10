package domain

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Administrative units of Việt Nam under two schemes.
//
// The "current" scheme is the two-level system in force since 1 July 2025
// (province → ward/commune/special zone). The "legacy" scheme is the three-level
// system before it (province → district → ward/commune/town). The two live in
// separate types and tables because the same numeric code can denote different
// units in each (code 26737 is the legacy ward Đa Kao and the current ward
// Tân Định) — a code is only meaningful together with its scheme.
//
// All types here are plain data: no json/db tag. The wire shape is delivery's,
// the SQL shape is repository's.

// CurrentProvince is a province-level unit of the current scheme (34 of them).
type CurrentProvince struct {
	Code      string
	Name      string // short, no type prefix: "Hồ Chí Minh"
	FullName  string // with type prefix: "Thành phố Hồ Chí Minh"
	Type      string // "province" | "city"
	SortOrder int    // 1-based Vietnamese alphabetical rank among all provinces
}

// CurrentWard is a ward-level unit of the current scheme (3,321 of them).
type CurrentWard struct {
	Code         string
	ProvinceCode string
	Name         string
	FullName     string
	Type         string // "ward" | "commune" | "special_zone"
	SortOrder    int    // rank within its province
}

// LegacyProvince is a province-level unit of the legacy scheme (63 of them).
type LegacyProvince struct {
	Code      string
	Name      string
	FullName  string
	Type      string
	SortOrder int
}

// LegacyDistrict is a district-level unit of the legacy scheme. The current
// scheme has no such level.
type LegacyDistrict struct {
	Code         string
	ProvinceCode string
	Name         string
	FullName     string
	Type         string
	SortOrder    int // rank within its province
}

// LegacyWard is a ward-level unit of the legacy scheme.
//
// ProvinceCode is carried only so Dataset.Verify can check that a ward belongs
// to the same province as its district; storage derives the province through
// DistrictCode and does not keep it.
type LegacyWard struct {
	Code         string
	DistrictCode string
	ProvinceCode string
	Name         string
	FullName     string
	Type         string
	SortOrder    int // rank within its district
}

// WardMapping links one legacy ward to one current ward. It is many-to-many:
// a legacy ward may be spread over several current wards and a current ward may
// gather several legacy wards. MappingType and Note are the source dataset's
// own labels (merged, province_changed, ambiguous, unchanged, split_population,
// renamed) and are passed through, not interpreted here.
type WardMapping struct {
	LegacyWardCode  string
	CurrentWardCode string
	MappingType     string
	Note            string
}

// DistrictMapping links a legacy district that has no wards (the island
// districts) to the current ward it became. It travels with the dataset but is
// not stored yet — district-level conversion is deferred.
type DistrictMapping struct {
	LegacyDistrictCode string
	CurrentWardCode    string
	MappingType        string
	Note               string
}

// CurrentWardCandidate is one possible result of converting a legacy ward.
type CurrentWardCandidate struct {
	Ward        CurrentWard
	Province    CurrentProvince
	MappingType string
	Note        string
}

// LegacyWardCandidate is one possible result of converting a current ward.
type LegacyWardCandidate struct {
	Ward        LegacyWard
	District    LegacyDistrict
	Province    LegacyProvince
	MappingType string
	Note        string
}

// Conversion is the answer to "what is this unit in the other scheme". It lists
// every candidate: the system never chooses one for the caller.
type Conversion[T any] struct {
	Candidates []T
}

// Unique reports whether the conversion has exactly one candidate.
func (c Conversion[T]) Unique() bool {
	return len(c.Candidates) == 1
}

// ErrAdministrativeUnitNotFound — the named unit (or its parent) does not exist
// in the named scheme.
var ErrAdministrativeUnitNotFound = errors.New("administrative unit not found")

// AdministrativeUnitRepository is the port the administrative-units use case
// reads through and the seeder writes through.
//
// Ordering is part of the contract: every list returns units by SortOrder ASC,
// Code ASC (Vietnamese alphabetical order of the short name, precomputed in the
// dataset so it does not depend on the database's collation).
type AdministrativeUnitRepository interface {
	// ListCurrentProvinces returns every current-scheme province.
	ListCurrentProvinces(ctx context.Context) ([]CurrentProvince, error)

	// ListCurrentWards returns every ward of a current-scheme province, or
	// ErrAdministrativeUnitNotFound when the province does not exist.
	ListCurrentWards(ctx context.Context, provinceCode string) ([]CurrentWard, error)

	// ListLegacyProvinces returns every legacy-scheme province.
	ListLegacyProvinces(ctx context.Context) ([]LegacyProvince, error)

	// ListLegacyDistricts returns every district of a legacy-scheme province, or
	// ErrAdministrativeUnitNotFound when the province does not exist.
	ListLegacyDistricts(ctx context.Context, provinceCode string) ([]LegacyDistrict, error)

	// ListLegacyWards returns every ward of a legacy-scheme district. A district
	// that exists but has no wards returns an empty slice, not an error;
	// ErrAdministrativeUnitNotFound means the district does not exist.
	ListLegacyWards(ctx context.Context, districtCode string) ([]LegacyWard, error)

	// ConvertLegacyWard returns the current wards a legacy ward maps to, or
	// ErrAdministrativeUnitNotFound when the legacy ward does not exist.
	ConvertLegacyWard(ctx context.Context, wardCode string) ([]CurrentWardCandidate, error)

	// ConvertCurrentWard returns the legacy wards that map to a current ward, or
	// ErrAdministrativeUnitNotFound when the current ward does not exist. The
	// slice is empty for a current ward that is the target of no ward mapping
	// (the special zones that replaced whole island districts).
	ConvertCurrentWard(ctx context.Context, wardCode string) ([]LegacyWardCandidate, error)

	// ConvertLegacyProvince returns the current provinces the wards of a legacy
	// province map into (exactly one for a verified dataset), or
	// ErrAdministrativeUnitNotFound.
	ConvertLegacyProvince(ctx context.Context, provinceCode string) ([]CurrentProvince, error)

	// ConvertCurrentProvince returns the legacy provinces whose wards map into a
	// current province, or ErrAdministrativeUnitNotFound.
	ConvertCurrentProvince(ctx context.Context, provinceCode string) ([]LegacyProvince, error)

	// ReplaceDataset upserts the whole dataset in one transaction, idempotently.
	// It never deletes: a unit absent from ds is left as it was.
	ReplaceDataset(ctx context.Context, ds Dataset) error
}

// Dataset is one complete, versioned snapshot of both schemes and the mappings
// between them — the unit the seeder loads.
type Dataset struct {
	// Version identifies the snapshot (the dataset's own date, "2026-09-25").
	Version string
	// SourceNote says where it came from (repository and commit).
	SourceNote string

	CurrentProvinces []CurrentProvince
	CurrentWards     []CurrentWard
	LegacyProvinces  []LegacyProvince
	LegacyDistricts  []LegacyDistrict
	LegacyWards      []LegacyWard

	WardMappings     []WardMapping
	DistrictMappings []DistrictMapping
}

// DatasetCounts is how many rows of each kind a Dataset holds, for the seeder's
// report and for comparing with the dataset's own declared counts.
type DatasetCounts struct {
	CurrentProvinces int
	CurrentWards     int
	LegacyProvinces  int
	LegacyDistricts  int
	LegacyWards      int
	WardMappings     int
	DistrictMappings int
}

// Counts returns the row count of every part of the dataset.
func (d Dataset) Counts() DatasetCounts {
	return DatasetCounts{
		CurrentProvinces: len(d.CurrentProvinces),
		CurrentWards:     len(d.CurrentWards),
		LegacyProvinces:  len(d.LegacyProvinces),
		LegacyDistricts:  len(d.LegacyDistricts),
		LegacyWards:      len(d.LegacyWards),
		WardMappings:     len(d.WardMappings),
		DistrictMappings: len(d.DistrictMappings),
	}
}

// Rules Dataset.Verify enforces. Each IntegrityIssue names the rule it broke.
const (
	// RuleDuplicateCode — a code appears twice in one table (for the legacy
	// districts: claimed by two provinces).
	RuleDuplicateCode = "duplicate_code"
	// RuleUnknownParent — a unit names a parent that is not in the dataset.
	RuleUnknownParent = "unknown_parent"
	// RuleWardProvinceMismatch — a legacy ward's province is not the province
	// of its own district.
	RuleWardProvinceMismatch = "ward_province_mismatch"
	// RuleLegacyWardUnmapped — a legacy ward has no ward mapping.
	RuleLegacyWardUnmapped = "legacy_ward_unmapped"
	// RuleCurrentWardUntargeted — a current ward is the target of no mapping.
	RuleCurrentWardUntargeted = "current_ward_untargeted"
	// RuleMappingUnknownUnit — a mapping points at a unit that is not in the
	// dataset.
	RuleMappingUnknownUnit = "mapping_unknown_unit"
	// RuleLegacyProvinceSplit — the wards of one legacy province map into more
	// than one current province, which contradicts the merge of whole provinces.
	RuleLegacyProvinceSplit = "legacy_province_split"
)

// IntegrityIssue is one violation found by Dataset.Verify.
type IntegrityIssue struct {
	Rule    string
	Subject string // what is wrong, e.g. "legacy ward 27691"
	Detail  string // why, in one line
}

// String renders the issue for the seeder's report.
func (i IntegrityIssue) String() string {
	return fmt.Sprintf("%s: %s — %s", i.Rule, i.Subject, i.Detail)
}

// Verify checks the dataset's internal consistency without any I/O and returns
// every violation, ordered by rule then subject so a report is stable. A nil
// result means the dataset is safe to load.
//
// It deliberately repairs nothing: a dataset that fails is the seeder's to
// refuse, never to quietly fix.
func (d Dataset) Verify() []IntegrityIssue {
	var issues []IntegrityIssue
	add := func(rule, subject, detail string) {
		issues = append(issues, IntegrityIssue{Rule: rule, Subject: subject, Detail: detail})
	}

	// 1. duplicate codes, per table.
	dupCodes := func(table string, n int, code func(i int) string) {
		seen := make(map[string]int, n)
		for i := range n {
			seen[code(i)]++
		}
		for c, k := range seen {
			if k > 1 {
				add(RuleDuplicateCode, fmt.Sprintf("%s %s", table, c), fmt.Sprintf("appears %d times", k))
			}
		}
	}
	dupCodes("current province", len(d.CurrentProvinces), func(i int) string { return d.CurrentProvinces[i].Code })
	dupCodes("current ward", len(d.CurrentWards), func(i int) string { return d.CurrentWards[i].Code })
	dupCodes("legacy province", len(d.LegacyProvinces), func(i int) string { return d.LegacyProvinces[i].Code })
	dupCodes("legacy district", len(d.LegacyDistricts), func(i int) string { return d.LegacyDistricts[i].Code })
	dupCodes("legacy ward", len(d.LegacyWards), func(i int) string { return d.LegacyWards[i].Code })

	// Lookup sets.
	currentProvince := make(map[string]bool, len(d.CurrentProvinces))
	for _, p := range d.CurrentProvinces {
		currentProvince[p.Code] = true
	}
	currentWardProvince := make(map[string]string, len(d.CurrentWards))
	for _, w := range d.CurrentWards {
		currentWardProvince[w.Code] = w.ProvinceCode
	}
	legacyProvince := make(map[string]bool, len(d.LegacyProvinces))
	for _, p := range d.LegacyProvinces {
		legacyProvince[p.Code] = true
	}
	// A district code may (wrongly) be claimed by several provinces; keep them
	// all so the ward check is not fooled by whichever row came first.
	districtProvinces := make(map[string]map[string]bool, len(d.LegacyDistricts))
	for _, x := range d.LegacyDistricts {
		if districtProvinces[x.Code] == nil {
			districtProvinces[x.Code] = map[string]bool{}
		}
		districtProvinces[x.Code][x.ProvinceCode] = true
	}
	legacyWardProvince := make(map[string]string, len(d.LegacyWards))
	for _, w := range d.LegacyWards {
		legacyWardProvince[w.Code] = w.ProvinceCode
	}

	// 2. parents exist.
	for _, w := range d.CurrentWards {
		if !currentProvince[w.ProvinceCode] {
			add(RuleUnknownParent, "current ward "+w.Code, "province "+w.ProvinceCode+" is not in the dataset")
		}
	}
	for _, x := range d.LegacyDistricts {
		if !legacyProvince[x.ProvinceCode] {
			add(RuleUnknownParent, "legacy district "+x.Code, "province "+x.ProvinceCode+" is not in the dataset")
		}
	}
	for _, w := range d.LegacyWards {
		provs, ok := districtProvinces[w.DistrictCode]
		if !ok {
			add(RuleUnknownParent, "legacy ward "+w.Code, "district "+w.DistrictCode+" is not in the dataset")
			continue
		}
		// 3. the ward's province must be its district's province.
		if !provs[w.ProvinceCode] {
			add(RuleWardProvinceMismatch, "legacy ward "+w.Code,
				fmt.Sprintf("labelled province %s but district %s belongs to %s", w.ProvinceCode, w.DistrictCode, joinKeys(provs)))
		}
	}

	// 4. mappings point at known units; remember what is mapped / targeted.
	mappedLegacy := make(map[string]bool, len(d.WardMappings))
	targeted := make(map[string]bool, len(d.WardMappings)+len(d.DistrictMappings))
	for _, m := range d.WardMappings {
		_, legacyKnown := legacyWardProvince[m.LegacyWardCode]
		_, currentKnown := currentWardProvince[m.CurrentWardCode]
		if !legacyKnown {
			add(RuleMappingUnknownUnit, fmt.Sprintf("ward mapping %s→%s", m.LegacyWardCode, m.CurrentWardCode),
				"legacy ward "+m.LegacyWardCode+" is not in the dataset")
		}
		if !currentKnown {
			add(RuleMappingUnknownUnit, fmt.Sprintf("ward mapping %s→%s", m.LegacyWardCode, m.CurrentWardCode),
				"current ward "+m.CurrentWardCode+" is not in the dataset")
		}
		mappedLegacy[m.LegacyWardCode] = true
		targeted[m.CurrentWardCode] = true
	}
	for _, m := range d.DistrictMappings {
		if _, ok := districtProvinces[m.LegacyDistrictCode]; !ok {
			add(RuleMappingUnknownUnit, fmt.Sprintf("district mapping %s→%s", m.LegacyDistrictCode, m.CurrentWardCode),
				"legacy district "+m.LegacyDistrictCode+" is not in the dataset")
		}
		if _, ok := currentWardProvince[m.CurrentWardCode]; !ok {
			add(RuleMappingUnknownUnit, fmt.Sprintf("district mapping %s→%s", m.LegacyDistrictCode, m.CurrentWardCode),
				"current ward "+m.CurrentWardCode+" is not in the dataset")
		}
		targeted[m.CurrentWardCode] = true
	}

	// 5. nothing left out on either side.
	for _, w := range d.LegacyWards {
		if !mappedLegacy[w.Code] {
			add(RuleLegacyWardUnmapped, "legacy ward "+w.Code, "has no ward mapping")
		}
	}
	for _, w := range d.CurrentWards {
		if !targeted[w.Code] {
			add(RuleCurrentWardUntargeted, "current ward "+w.Code, "is the target of no mapping")
		}
	}

	// 6. a legacy province merges whole into exactly one current province.
	into := make(map[string]map[string]bool)
	for _, m := range d.WardMappings {
		lp, lok := legacyWardProvince[m.LegacyWardCode]
		cp, cok := currentWardProvince[m.CurrentWardCode]
		if !lok || !cok {
			continue // already reported as an unknown unit
		}
		if into[lp] == nil {
			into[lp] = map[string]bool{}
		}
		into[lp][cp] = true
	}
	for lp, set := range into {
		if len(set) > 1 {
			add(RuleLegacyProvinceSplit, "legacy province "+lp, "wards map into current provinces "+joinKeys(set))
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Rule != issues[j].Rule {
			return issues[i].Rule < issues[j].Rule
		}
		if issues[i].Subject != issues[j].Subject {
			return issues[i].Subject < issues[j].Subject
		}
		return issues[i].Detail < issues[j].Detail
	})
	return issues
}

// joinKeys renders a set's members, sorted, comma-separated.
func joinKeys(set map[string]bool) string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
