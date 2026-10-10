package domain

import (
	"strings"
	"testing"
)

// validDataset is a small consistent dataset that exercises every shape the
// real one has: a merge (W1, W2 → C1), a partial move (W2 also → C2), a province
// change (W3 → C3 in another province) and a ward-less district (D3 → C4).
func validDataset() Dataset {
	return Dataset{
		Version:    "test",
		SourceNote: "unit test",
		CurrentProvinces: []CurrentProvince{
			{Code: "01", Name: "Một", FullName: "Tỉnh Một", Type: "province", SortOrder: 1},
			{Code: "02", Name: "Hai", FullName: "Tỉnh Hai", Type: "province", SortOrder: 2},
		},
		CurrentWards: []CurrentWard{
			{Code: "C1", ProvinceCode: "01", Name: "C1", FullName: "Phường C1", Type: "ward", SortOrder: 1},
			{Code: "C2", ProvinceCode: "01", Name: "C2", FullName: "Phường C2", Type: "ward", SortOrder: 2},
			{Code: "C3", ProvinceCode: "02", Name: "C3", FullName: "Xã C3", Type: "commune", SortOrder: 1},
			{Code: "C4", ProvinceCode: "02", Name: "C4", FullName: "Đặc khu C4", Type: "special_zone", SortOrder: 2},
		},
		LegacyProvinces: []LegacyProvince{
			{Code: "L1", Name: "L1", FullName: "Tỉnh L1", Type: "province", SortOrder: 1},
			{Code: "L2", Name: "L2", FullName: "Tỉnh L2", Type: "province", SortOrder: 2},
		},
		LegacyDistricts: []LegacyDistrict{
			{Code: "D1", ProvinceCode: "L1", Name: "D1", FullName: "Quận D1", Type: "district", SortOrder: 1},
			{Code: "D2", ProvinceCode: "L2", Name: "D2", FullName: "Huyện D2", Type: "district", SortOrder: 1},
			{Code: "D3", ProvinceCode: "L2", Name: "D3", FullName: "Huyện D3", Type: "district", SortOrder: 2},
		},
		LegacyWards: []LegacyWard{
			{Code: "W1", DistrictCode: "D1", ProvinceCode: "L1", Name: "W1", FullName: "Phường W1", Type: "ward", SortOrder: 1},
			{Code: "W2", DistrictCode: "D1", ProvinceCode: "L1", Name: "W2", FullName: "Phường W2", Type: "ward", SortOrder: 2},
			{Code: "W3", DistrictCode: "D2", ProvinceCode: "L2", Name: "W3", FullName: "Xã W3", Type: "commune", SortOrder: 1},
		},
		WardMappings: []WardMapping{
			{LegacyWardCode: "W1", CurrentWardCode: "C1", MappingType: "merged", Note: "whole"},
			{LegacyWardCode: "W2", CurrentWardCode: "C1", MappingType: "ambiguous", Note: "part"},
			{LegacyWardCode: "W2", CurrentWardCode: "C2", MappingType: "ambiguous", Note: "part"},
			{LegacyWardCode: "W3", CurrentWardCode: "C3", MappingType: "province_changed", Note: "whole"},
		},
		DistrictMappings: []DistrictMapping{
			{LegacyDistrictCode: "D3", CurrentWardCode: "C4", MappingType: "merged", Note: "island"},
		},
	}
}

func TestDatasetVerify_ValidDatasetHasNoIssues(t *testing.T) {
	if got := validDataset().Verify(); got != nil {
		t.Fatalf("Verify() = %v, want nil", got)
	}
}

func TestDatasetVerify_ReportsEachRule(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*Dataset)
		wantRule    string
		wantSubject string // substring of IntegrityIssue.Subject
		wantExactly int    // number of issues expected overall
	}{
		{
			name: "legacy ward without a mapping",
			mutate: func(d *Dataset) {
				d.WardMappings = d.WardMappings[:2] // drops W2→C2 and W3→C3
				d.WardMappings = append(d.WardMappings, WardMapping{LegacyWardCode: "W2", CurrentWardCode: "C2"})
			},
			wantRule: RuleLegacyWardUnmapped, wantSubject: "legacy ward W3",
			wantExactly: 2, // W3 unmapped and C3 now targeted by nothing
		},
		{
			name: "current ward that nothing maps to",
			mutate: func(d *Dataset) {
				d.CurrentWards = append(d.CurrentWards,
					CurrentWard{Code: "C9", ProvinceCode: "01", Name: "C9", FullName: "Phường C9", Type: "ward", SortOrder: 3})
			},
			wantRule: RuleCurrentWardUntargeted, wantSubject: "current ward C9", wantExactly: 1,
		},
		{
			name: "a ward-less district's target counts as targeted",
			mutate: func(d *Dataset) {
				// C4 is targeted only by the district mapping; removing the
				// mapping must expose it, keeping it must not.
				d.DistrictMappings = nil
			},
			wantRule: RuleCurrentWardUntargeted, wantSubject: "current ward C4", wantExactly: 1,
		},
		{
			name: "ward mapping to an unknown current ward",
			mutate: func(d *Dataset) {
				d.WardMappings = append(d.WardMappings, WardMapping{LegacyWardCode: "W1", CurrentWardCode: "NOPE"})
			},
			wantRule: RuleMappingUnknownUnit, wantSubject: "ward mapping W1→NOPE", wantExactly: 1,
		},
		{
			name: "ward mapping from an unknown legacy ward",
			mutate: func(d *Dataset) {
				d.WardMappings = append(d.WardMappings, WardMapping{LegacyWardCode: "GHOST", CurrentWardCode: "C1"})
			},
			wantRule: RuleMappingUnknownUnit, wantSubject: "ward mapping GHOST→C1", wantExactly: 1,
		},
		{
			name: "district mapping to an unknown district",
			mutate: func(d *Dataset) {
				d.DistrictMappings = append(d.DistrictMappings, DistrictMapping{LegacyDistrictCode: "GHOST", CurrentWardCode: "C4"})
			},
			wantRule: RuleMappingUnknownUnit, wantSubject: "district mapping GHOST→C4", wantExactly: 1,
		},
		{
			name: "legacy ward labelled with another province than its district",
			mutate: func(d *Dataset) {
				d.LegacyWards[0].ProvinceCode = "L2" // W1 sits in D1 which is in L1
			},
			// W1 is now also mapped into current province 01 while labelled L2, so
			// L2 appears to split across 01 and 02: both rules fire, both are real.
			wantRule: RuleWardProvinceMismatch, wantSubject: "legacy ward W1", wantExactly: 2,
		},
		{
			name: "district code claimed by two provinces",
			mutate: func(d *Dataset) {
				d.LegacyDistricts = append(d.LegacyDistricts,
					LegacyDistrict{Code: "D1", ProvinceCode: "L2", Name: "D1", FullName: "Quận D1", Type: "district", SortOrder: 3})
			},
			wantRule: RuleDuplicateCode, wantSubject: "legacy district D1", wantExactly: 1,
		},
		{
			name: "duplicate legacy ward code",
			mutate: func(d *Dataset) {
				d.LegacyWards = append(d.LegacyWards, d.LegacyWards[0])
			},
			wantRule: RuleDuplicateCode, wantSubject: "legacy ward W1", wantExactly: 1,
		},
		{
			name: "current ward in an unknown province",
			mutate: func(d *Dataset) {
				d.CurrentWards[2].ProvinceCode = "99" // C3, mapped only from W3
			},
			wantRule: RuleUnknownParent, wantSubject: "current ward C3", wantExactly: 1,
		},
		{
			name: "legacy ward in an unknown district",
			mutate: func(d *Dataset) {
				d.LegacyWards[0].DistrictCode = "D404"
			},
			wantRule: RuleUnknownParent, wantSubject: "legacy ward W1", wantExactly: 1,
		},
		{
			name: "legacy province whose wards map into two current provinces",
			mutate: func(d *Dataset) {
				// W1 (province L1) also maps to C3, which is in current province 02.
				d.WardMappings = append(d.WardMappings, WardMapping{LegacyWardCode: "W1", CurrentWardCode: "C3"})
			},
			wantRule: RuleLegacyProvinceSplit, wantSubject: "legacy province L1", wantExactly: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := validDataset()
			tc.mutate(&d)
			got := d.Verify()
			if len(got) != tc.wantExactly {
				t.Fatalf("Verify() returned %d issues, want %d: %v", len(got), tc.wantExactly, got)
			}
			for _, is := range got {
				if is.Rule == tc.wantRule && strings.Contains(is.Subject, tc.wantSubject) {
					return
				}
			}
			t.Fatalf("no issue with rule %q and subject containing %q in %v", tc.wantRule, tc.wantSubject, got)
		})
	}
}

func TestDatasetVerify_DoesNotMutateTheDataset(t *testing.T) {
	d := validDataset()
	d.WardMappings = append(d.WardMappings, WardMapping{LegacyWardCode: "GHOST", CurrentWardCode: "C1"})
	before := d.Counts()
	_ = d.Verify()
	if after := d.Counts(); after != before {
		t.Fatalf("Verify changed the dataset: %+v → %+v", before, after)
	}
}

func TestDatasetVerify_IssuesAreOrderedByRuleThenSubject(t *testing.T) {
	d := validDataset()
	d.WardMappings = append(d.WardMappings,
		WardMapping{LegacyWardCode: "GHOST2", CurrentWardCode: "C1"},
		WardMapping{LegacyWardCode: "GHOST1", CurrentWardCode: "C1"},
	)
	d.CurrentWards = append(d.CurrentWards,
		CurrentWard{Code: "C9", ProvinceCode: "01", Name: "C9", FullName: "Phường C9", Type: "ward", SortOrder: 3})
	got := d.Verify()
	if len(got) != 3 {
		t.Fatalf("want 3 issues, got %v", got)
	}
	want := []string{RuleCurrentWardUntargeted, RuleMappingUnknownUnit, RuleMappingUnknownUnit}
	for i, w := range want {
		if got[i].Rule != w {
			t.Errorf("issue %d rule = %q, want %q (%v)", i, got[i].Rule, w, got)
		}
	}
	if !strings.Contains(got[1].Subject, "GHOST1") || !strings.Contains(got[2].Subject, "GHOST2") {
		t.Errorf("unknown-unit issues not sorted by subject: %v", got)
	}
}

func TestDataset_Counts(t *testing.T) {
	want := DatasetCounts{
		CurrentProvinces: 2, CurrentWards: 4, LegacyProvinces: 2,
		LegacyDistricts: 3, LegacyWards: 3, WardMappings: 4, DistrictMappings: 1,
	}
	if got := validDataset().Counts(); got != want {
		t.Fatalf("Counts() = %+v, want %+v", got, want)
	}
}

func TestConversion_Unique(t *testing.T) {
	tests := []struct {
		name  string
		cands []string
		want  bool
	}{
		{"no candidates", nil, false},
		{"one candidate", []string{"a"}, true},
		{"several candidates", []string{"a", "b"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := (Conversion[string]{Candidates: tc.cands}).Unique(); got != tc.want {
				t.Fatalf("Unique() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIntegrityIssue_String(t *testing.T) {
	got := IntegrityIssue{Rule: RuleLegacyWardUnmapped, Subject: "legacy ward W3", Detail: "has no ward mapping"}.String()
	want := "legacy_ward_unmapped: legacy ward W3 — has no ward mapping"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
