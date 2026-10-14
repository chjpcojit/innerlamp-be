package usecase

import (
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Fixture tree (ids in brackets):
//
//	Tổ [1] gen1 ♂ ── Bà tổ [2] (in-law, mother of Lộc)
//	├─ Lộc [3] ♂ order1, other_parent [2] ── Vợ Lộc [6]
//	│   ├─ Bác Z [14] ♂ order1 ── Vợ Z [15]
//	│   │    └─ Anh họ F [19] ♂
//	│   ├─ A [7] ♂ order2, other_parent [6] ── Vợ A [10]
//	│   │    └─ C [11] ♂ other_parent [10] ── Vợ C [20]
//	│   ├─ B [8] ♂ order3 ── Vợ B [16]
//	│   │    └─ E [18] ♂
//	│   ├─ Mai [9] ♀ order4 ── Chồng Mai [17]
//	│   └─ Y [22] ♂ order5, other_parent [21] (half-brother; [21] = second wife of Lộc)
//	├─ Nghĩa [4] ♂ order2
//	│   └─ D [12] ♂
//	└─ Hoa [5] ♀ order3 ── Chồng Hoa [13]
func ip(v int) *int      { return &v }
func i64(v int64) *int64 { return &v }
func yr(v int) *int      { return &v }

func blood(id int64, name string, g domain.Gender, gen int, parent *int64, order *int, other *int64) domain.GenealogyMember {
	return domain.GenealogyMember{
		ID: id, FullName: name, Kind: domain.MemberKindBlood, Gender: g,
		IsAlive: true, Generation: gen, ParentID: parent, BirthOrder: order, OtherParentID: other,
	}
}

func inlaw(id int64, name string, g domain.Gender, gen int, spouseOf int64) domain.GenealogyMember {
	return domain.GenealogyMember{
		ID: id, FullName: name, Kind: domain.MemberKindInLaw, Gender: g,
		IsAlive: true, Generation: gen, SpouseOfID: i64(spouseOf),
	}
}

func fixture() []domain.GenealogyMember {
	m, f := domain.GenderMale, domain.GenderFemale
	return []domain.GenealogyMember{
		blood(1, "Tổ", m, 1, nil, nil, nil),
		inlaw(2, "Bà tổ", f, 1, 1),
		blood(3, "Lộc", m, 2, i64(1), ip(1), i64(2)),
		blood(4, "Nghĩa", m, 2, i64(1), ip(2), nil),
		blood(5, "Hoa", f, 2, i64(1), ip(3), nil),
		inlaw(6, "Vợ Lộc", f, 2, 3),
		blood(7, "A", m, 3, i64(3), ip(2), i64(6)),
		blood(8, "B", m, 3, i64(3), ip(3), nil),
		blood(9, "Mai", f, 3, i64(3), ip(4), nil),
		inlaw(10, "Vợ A", f, 3, 7),
		blood(11, "C", m, 4, i64(7), ip(1), i64(10)),
		blood(12, "D", m, 3, i64(4), ip(1), nil),
		inlaw(13, "Chồng Hoa", m, 2, 5),
		blood(14, "Z", m, 3, i64(3), ip(1), nil),
		inlaw(15, "Vợ Z", f, 3, 14),
		inlaw(16, "Vợ B", f, 3, 8),
		inlaw(17, "Chồng Mai", m, 3, 9),
		blood(18, "E", m, 4, i64(8), ip(1), nil),
		blood(19, "F", m, 4, i64(14), ip(1), nil),
		inlaw(20, "Vợ C", f, 4, 11),
		inlaw(21, "Vợ hai Lộc", f, 2, 3),
		blood(22, "Y", m, 3, i64(3), ip(5), i64(21)),
	}
}

func TestAge(t *testing.T) {
	now := 2026
	tests := []struct {
		name string
		m    domain.GenealogyMember
		want *int
	}{
		{"living with birth year", domain.GenealogyMember{IsAlive: true, BirthYear: yr(1980)}, ip(46)},
		{"deceased with both years", domain.GenealogyMember{IsAlive: false, BirthYear: yr(1930), DeathYear: yr(2005)}, ip(75)},
		{"deceased without death year", domain.GenealogyMember{IsAlive: false, BirthYear: yr(1930)}, nil},
		{"no birth year", domain.GenealogyMember{IsAlive: true}, nil},
		{"living with death year ignored", domain.GenealogyMember{IsAlive: true, BirthYear: yr(2000), DeathYear: yr(2010)}, ip(26)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Age(&tt.m, now)
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Fatalf("Age = %v, want %v", deref(got), deref(tt.want))
			}
		})
	}
}

func TestTraditionalAge(t *testing.T) {
	now := 2026
	tests := []struct {
		name string
		m    domain.GenealogyMember
		want *int
	}{
		{"living: Western age + 1", domain.GenealogyMember{IsAlive: true, BirthYear: yr(1990)}, ip(37)},
		{"deceased: age at death + 1", domain.GenealogyMember{IsAlive: false, BirthYear: yr(1930), DeathYear: yr(2005)}, ip(76)},
		{"born this year is already 1", domain.GenealogyMember{IsAlive: true, BirthYear: yr(2026)}, ip(1)},
		{"no birth year", domain.GenealogyMember{IsAlive: true}, nil},
		{"deceased without death year", domain.GenealogyMember{IsAlive: false, BirthYear: yr(1930)}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TraditionalAge(&tt.m, now)
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Fatalf("TraditionalAge = %v, want %v", deref(got), deref(tt.want))
			}
			// The two ages never disagree about when one is unknown, and differ by exactly one.
			if w := Age(&tt.m, now); (w == nil) != (got == nil) || (w != nil && *got != *w+1) {
				t.Fatalf("Age = %v, TraditionalAge = %v: must be Age+1 or both nil", deref(w), deref(got))
			}
		})
	}
}

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestInLawLabel(t *testing.T) {
	idx := BuildIndex(fixture())
	get := func(id int64) *domain.GenealogyMember { m, _ := idx.Get(id); return m }

	if l := InLawLabel(idx, get(10)); l == nil || *l != domain.InLawLabelDau {
		t.Errorf("wife of a male blood member = %v, want DAU", l)
	}
	if l := InLawLabel(idx, get(13)); l == nil || *l != domain.InLawLabelRe {
		t.Errorf("husband of a female blood member = %v, want RE", l)
	}
	if l := InLawLabel(idx, get(7)); l != nil {
		t.Errorf("blood member label = %v, want nil", *l)
	}
}

func TestRelativesOf(t *testing.T) {
	idx := BuildIndex(fixture())

	t.Run("blood member with both parents, spouse, siblings", func(t *testing.T) {
		r, ok := RelativesOf(idx, 7) // A
		if !ok {
			t.Fatal("A not found")
		}
		if r.Father == nil || r.Father.ID != 3 {
			t.Errorf("father = %+v, want Lộc", r.Father)
		}
		if r.Mother == nil || r.Mother.ID != 6 {
			t.Errorf("mother = %+v, want Vợ Lộc", r.Mother)
		}
		if len(r.Spouses) != 1 || r.Spouses[0].ID != 10 {
			t.Errorf("spouses = %+v", r.Spouses)
		}
		var sibs []int64
		half := map[int64]bool{}
		for _, s := range r.Siblings {
			sibs = append(sibs, s.ID)
			half[s.ID] = s.IsHalfSibling
		}
		want := []int64{14, 8, 9, 22} // Z(1), B(3), Mai(4), Y(5) by birth_order
		if len(sibs) != len(want) {
			t.Fatalf("siblings = %v, want %v", sibs, want)
		}
		for i := range want {
			if sibs[i] != want[i] {
				t.Fatalf("siblings = %v, want %v", sibs, want)
			}
		}
		if !half[22] {
			t.Error("Y has another other_parent_id, must be a half-sibling")
		}
		if half[14] {
			t.Error("Z has no other parent recorded, must not be flagged half-sibling")
		}
		if len(r.Children) != 1 || r.Children[0].ID != 11 {
			t.Errorf("children = %+v", r.Children)
		}
	})

	t.Run("in-law has spouse and the children naming them", func(t *testing.T) {
		r, ok := RelativesOf(idx, 10) // Vợ A
		if !ok {
			t.Fatal("not found")
		}
		if r.Father != nil || r.Mother != nil {
			t.Error("an in-law has no parents in the tree")
		}
		if len(r.Spouses) != 1 || r.Spouses[0].ID != 7 {
			t.Errorf("spouses = %+v", r.Spouses)
		}
		if len(r.Children) != 1 || r.Children[0].ID != 11 {
			t.Errorf("children = %+v", r.Children)
		}
		if len(r.Siblings) != 0 {
			t.Errorf("an in-law has no siblings in the tree, got %+v", r.Siblings)
		}
	})

	t.Run("child without other parent is not listed under any wife", func(t *testing.T) {
		r, _ := RelativesOf(idx, 21) // second wife of Lộc: only Y names her
		if len(r.Children) != 1 || r.Children[0].ID != 22 {
			t.Errorf("children = %+v", r.Children)
		}
		r2, _ := RelativesOf(idx, 6) // Vợ Lộc: only A names her
		if len(r2.Children) != 1 || r2.Children[0].ID != 7 {
			t.Errorf("children = %+v (Z, B, Mai have no other parent and must not appear)", r2.Children)
		}
	})

	t.Run("lists are never nil", func(t *testing.T) {
		r, _ := RelativesOf(idx, 12) // D, leaf
		if r.Children == nil || r.Spouses == nil || r.Siblings == nil {
			t.Error("lists must be empty slices, not nil")
		}
	})

	t.Run("unknown member", func(t *testing.T) {
		if _, ok := RelativesOf(idx, 999); ok {
			t.Error("ok = true for unknown id")
		}
	})
}

func TestLineageOf(t *testing.T) {
	idx := BuildIndex(fixture())
	ids := func(items []domain.LineageItem) []int64 {
		var out []int64
		for _, it := range items {
			out = append(out, it.ID)
		}
		return out
	}
	eq := func(a, b []int64) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	got, ok := LineageOf(idx, 11) // C
	if !ok || !eq(ids(got), []int64{1, 3, 7, 11}) {
		t.Errorf("lineage(C) = %v, want [1 3 7 11]", ids(got))
	}
	got, _ = LineageOf(idx, 1)
	if !eq(ids(got), []int64{1}) {
		t.Errorf("lineage(founder) = %v, want [1]", ids(got))
	}
	// An in-law's lineage is the attached member's, without the in-law.
	got, _ = LineageOf(idx, 10)
	if !eq(ids(got), []int64{1, 3, 7}) {
		t.Errorf("lineage(in-law) = %v, want [1 3 7]", ids(got))
	}
	if _, ok := LineageOf(idx, 999); ok {
		t.Error("ok = true for unknown id")
	}
}

func TestChiOf(t *testing.T) {
	idx := BuildIndex(fixture())
	names := func(path []domain.ChiItem) []string {
		var out []string
		for _, p := range path {
			out = append(out, p.Ancestor.FullName)
		}
		return out
	}
	tests := []struct {
		name        string
		id          int64
		wantPath    []string
		wantCurrent string // "" = nil
		wantRank    int
		wantEldest  bool
	}{
		{"founder has no branch", 1, nil, "", 0, false},
		{"eldest son heads a branch", 3, []string{"Lộc"}, "Lộc", 1, true},
		{"younger son heads a branch", 4, []string{"Nghĩa"}, "Nghĩa", 2, false},
		{"nested branch", 7, []string{"Lộc", "A"}, "A", 2, false},
		{"eldest nested branch", 14, []string{"Lộc", "Z"}, "Z", 1, true},
		{"only son does not start a branch", 11, []string{"Lộc", "A"}, "A", 2, false},
		{"only son of Nghĩa stays in Nghĩa's branch", 12, []string{"Nghĩa"}, "Nghĩa", 2, false},
		{"daughter belongs to father's branch", 9, []string{"Lộc"}, "Lộc", 1, true},
		{"daughter of founder has no branch", 5, nil, "", 0, false},
		{"in-law follows the attached member", 10, []string{"Lộc", "A"}, "A", 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cur, path, ok := ChiOf(idx, tt.id)
			if !ok {
				t.Fatal("ok = false")
			}
			got := names(path)
			if len(got) != len(tt.wantPath) {
				t.Fatalf("path = %v, want %v", got, tt.wantPath)
			}
			for i := range got {
				if got[i] != tt.wantPath[i] {
					t.Fatalf("path = %v, want %v", got, tt.wantPath)
				}
			}
			if tt.wantCurrent == "" {
				if cur != nil {
					t.Fatalf("current = %+v, want nil", cur)
				}
				if path == nil {
					t.Error("path must be an empty slice, not nil")
				}
				return
			}
			if cur == nil || cur.Ancestor.FullName != tt.wantCurrent {
				t.Fatalf("current = %+v, want %s", cur, tt.wantCurrent)
			}
			if cur.Rank != tt.wantRank || cur.IsEldest != tt.wantEldest {
				t.Errorf("rank = %d eldest = %v, want %d %v", cur.Rank, cur.IsEldest, tt.wantRank, tt.wantEldest)
			}
		})
	}
}

func TestKinshipOf(t *testing.T) {
	idx := BuildIndex(fixture())
	tests := []struct {
		name       string
		me, target int64
		wantCode   domain.KinshipCode
		wantLabel  string
	}{
		{"self", 11, 11, domain.KinshipSelf, "Bản thân"},
		{"my father", 11, 7, KinshipFather, "Cha"},
		{"my registered mother", 11, 10, KinshipMother, "Mẹ"},
		{"my grandfather", 11, 3, KinshipPaternalGrandpa, "Ông nội"},
		{"my grandmother (wife of grandfather)", 11, 6, KinshipPaternalGrandma, "Bà nội"},
		{"my great-grandfather", 11, 1, KinshipGreatGrandparent, "Cụ ông"},
		{"my great-grandmother", 11, 2, KinshipGreatGrandparent, "Cụ bà"},
		{"father's elder brother", 11, 14, KinshipFatherElderBrother, "Bác"},
		{"father's younger brother", 11, 8, KinshipFatherYoungBrother, "Chú"},
		{"father's sister", 11, 9, KinshipFatherSister, "Cô"},
		{"father's elder brother's wife", 11, 15, KinshipUncleElderWife, "Bác gái"},
		{"father's younger brother's wife", 11, 16, KinshipUncleYoungerWife, "Thím"},
		{"father's sister's husband", 11, 17, KinshipAuntHusband, "Dượng"},
		{"son of father's elder brother", 11, 19, KinshipElderSibling + "_HO", "Anh họ"},
		{"son of father's younger brother", 11, 18, KinshipYoungerSibling + "_HO", "Em họ"},
		{"my elder brother's wife (I am B)", 8, 10, KinshipBrotherWifeElder, "Chị dâu"},
		{"my younger brother's wife (I am Z)", 14, 10, KinshipBrotherWifeYounger, "Em dâu"},
		{"my elder brother", 8, 7, KinshipElderSibling, "Anh"},
		{"my younger sister", 7, 9, KinshipYoungerSibling, "Em"},
		{"my younger sister's husband", 7, 17, KinshipSisterHusbandYoung, "Em rể"},
		{"my son", 7, 11, KinshipChild, "Con"},
		{"my son's wife", 7, 20, KinshipDaughterInLaw, "Con dâu"},
		{"my wife", 7, 10, KinshipSpouse, "Vợ"},
		{"my nephew", 14, 11, KinshipNephewNiece, "Cháu"},
		{"my grandson", 3, 11, KinshipGrandchild, "Cháu nội"},
		{"my paternal cousin's elder-by-rank wife", 11, 15, KinshipUncleElderWife, "Bác gái"},
		{"father's cousin (distant younger line)", 11, 12, KinshipFatherYoungBrother + "_HO", "Chú họ"},
		{"in-law of a distant senior is not named", 11, 13, domain.KinshipUnknown, ""},
		{"unknown target", 11, 999, domain.KinshipUnknown, ""},
		// "Me" is an in-law: only own family is named.
		{"in-law me: my husband", 10, 7, KinshipSpouse, "Chồng"},
		{"in-law me: my child", 10, 11, KinshipChild, "Con"},
		{"in-law me: anyone else", 10, 8, domain.KinshipUnknown, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KinshipOf(idx, tt.me, tt.target)
			if got.Code != tt.wantCode || got.Label != tt.wantLabel {
				t.Errorf("KinshipOf(%d → %d) = %s %q, want %s %q",
					tt.me, tt.target, got.Code, got.Label, tt.wantCode, tt.wantLabel)
			}
		})
	}
}

// Rank follows the tree, not age: a younger uncle is still the uncle, and the
// elder-by-rank cousin may be the younger in years.
func TestKinshipIgnoresBirthYears(t *testing.T) {
	members := fixture()
	for i := range members {
		switch members[i].ID {
		case 14: // Z, my father's elder brother, born far later than me
			members[i].BirthYear = yr(2010)
		case 11: // C, me
			members[i].BirthYear = yr(1980)
		case 19: // F, cousin through the elder branch, much younger
			members[i].BirthYear = yr(2015)
		case 18: // E, cousin through the younger branch, much older
			members[i].BirthYear = yr(1950)
		}
	}
	idx := BuildIndex(members)

	if got := KinshipOf(idx, 11, 14); got.Label != "Bác" {
		t.Errorf("younger-in-years uncle = %q, want Bác", got.Label)
	}
	if got := KinshipOf(idx, 11, 19); got.Label != "Anh họ" {
		t.Errorf("cousin from elder branch, younger in years = %q, want Anh họ", got.Label)
	}
	if got := KinshipOf(idx, 11, 18); got.Label != "Em họ" {
		t.Errorf("cousin from younger branch, older in years = %q, want Em họ", got.Label)
	}
}

func TestKinshipTieBreaksOnID(t *testing.T) {
	// Two sons with the same birth_order: the lower id is the elder.
	members := []domain.GenealogyMember{
		blood(1, "Cha", domain.GenderMale, 1, nil, nil, nil),
		blood(2, "Anh", domain.GenderMale, 2, i64(1), ip(1), nil),
		blood(3, "Em", domain.GenderMale, 2, i64(1), ip(1), nil),
	}
	idx := BuildIndex(members)
	if got := KinshipOf(idx, 3, 2); got.Label != "Anh" {
		t.Errorf("equal birth_order, lower id = %q, want Anh", got.Label)
	}
	if got := KinshipOf(idx, 2, 3); got.Label != "Em" {
		t.Errorf("equal birth_order, higher id = %q, want Em", got.Label)
	}
}
