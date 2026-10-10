package usecase

import (
	"fmt"
	"sort"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Pure functions that compute what the member profile shows (AC64–AC69).
// No I/O: the whole tree (<= 5000 members) is loaded once and indexed.
//
// Rank (vai vế) follows the tree, never age: a person one generation above is
// senior whatever their birth year. birth_order is used only to say elder or
// younger between the two branches that diverge at the common ancestor.

// maxPathLen guards the walk up parent_id against a corrupt cycle. The spec
// caps depth at 60; the guard is generous.
const maxPathLen = 256

// TreeIndex is a read-only index over one genealogy's members.
type TreeIndex struct {
	byID          map[int64]*domain.GenealogyMember
	children      map[int64][]*domain.GenealogyMember // blood children by parent_id, (birth_order, id)
	spouses       map[int64][]*domain.GenealogyMember // in-laws by spouse_of_id, id
	byOtherParent map[int64][]*domain.GenealogyMember // children by other_parent_id, (birth_order, id)
}

// BuildIndex indexes members. The slice is not modified; the index points into
// it, so it must outlive the index.
func BuildIndex(members []domain.GenealogyMember) *TreeIndex {
	idx := &TreeIndex{
		byID:          make(map[int64]*domain.GenealogyMember, len(members)),
		children:      make(map[int64][]*domain.GenealogyMember),
		spouses:       make(map[int64][]*domain.GenealogyMember),
		byOtherParent: make(map[int64][]*domain.GenealogyMember),
	}
	for i := range members {
		m := &members[i]
		idx.byID[m.ID] = m
	}
	for i := range members {
		m := &members[i]
		if m.Kind == domain.MemberKindBlood && m.ParentID != nil {
			idx.children[*m.ParentID] = append(idx.children[*m.ParentID], m)
		}
		if m.Kind == domain.MemberKindInLaw && m.SpouseOfID != nil {
			idx.spouses[*m.SpouseOfID] = append(idx.spouses[*m.SpouseOfID], m)
		}
		if m.OtherParentID != nil {
			idx.byOtherParent[*m.OtherParentID] = append(idx.byOtherParent[*m.OtherParentID], m)
		}
	}
	for _, list := range idx.children {
		sortByOrder(list)
	}
	for _, list := range idx.byOtherParent {
		sortByOrder(list)
	}
	for _, list := range idx.spouses {
		sort.Slice(list, func(a, b int) bool { return list[a].ID < list[b].ID })
	}
	return idx
}

// Get returns the member with this id, if it is in the tree.
func (idx *TreeIndex) Get(id int64) (*domain.GenealogyMember, bool) {
	m, ok := idx.byID[id]
	return m, ok
}

func sortByOrder(list []*domain.GenealogyMember) {
	sort.SliceStable(list, func(a, b int) bool { return orderBefore(list[a], list[b]) })
}

// orderBefore is the sibling ordering: birth_order then id. A missing
// birth_order sorts last, ties break on id.
func orderBefore(a, b *domain.GenealogyMember) bool {
	ao, bo := orderValue(a), orderValue(b)
	if ao != bo {
		return ao < bo
	}
	return a.ID < b.ID
}

func orderValue(m *domain.GenealogyMember) int {
	if m.BirthOrder == nil {
		return int(^uint(0) >> 1)
	}
	return *m.BirthOrder
}

// anchor is the blood member a person is measured from: the member itself, or
// the blood member an in-law is attached to.
func (idx *TreeIndex) anchor(m *domain.GenealogyMember) *domain.GenealogyMember {
	if m.Kind != domain.MemberKindInLaw {
		return m
	}
	if m.SpouseOfID == nil {
		return nil
	}
	return idx.byID[*m.SpouseOfID]
}

// pathFromRoot lists the blood chain founder .. m following parent_id. It
// returns nil when the chain is broken or cyclic.
func (idx *TreeIndex) pathFromRoot(m *domain.GenealogyMember) []*domain.GenealogyMember {
	var rev []*domain.GenealogyMember
	for cur, n := m, 0; cur != nil; n++ {
		if n > maxPathLen {
			return nil
		}
		rev = append(rev, cur)
		if cur.ParentID == nil {
			break
		}
		next, ok := idx.byID[*cur.ParentID]
		if !ok {
			return nil
		}
		cur = next
	}
	out := make([]*domain.GenealogyMember, len(rev))
	for i, p := range rev {
		out[len(rev)-1-i] = p
	}
	return out
}

func memberRef(m *domain.GenealogyMember) domain.MemberRef {
	return domain.MemberRef{
		ID: m.ID, FullName: m.FullName, Gender: m.Gender,
		Generation: m.Generation, IsAlive: m.IsAlive,
	}
}

// InLawLabel is the derived DAU/RE label of an in-law, or nil for a blood
// member or an in-law whose attached member is missing (AC33).
func InLawLabel(idx *TreeIndex, m *domain.GenealogyMember) *domain.InLawLabel {
	if m.Kind != domain.MemberKindInLaw {
		return nil
	}
	a := idx.anchor(m)
	if a == nil {
		return nil
	}
	l := domain.InLawLabelDau
	if a.Gender == domain.GenderFemale {
		l = domain.InLawLabelRe
	}
	return &l
}

// Age is the age shown on a profile (AC65): the current year minus the birth
// year for a living member, the age at death for a deceased one, nil when the
// needed years are unknown. It is an approximation by solar year.
func Age(m *domain.GenealogyMember, currentYear int) *int {
	if m.BirthYear == nil {
		return nil
	}
	var age int
	switch {
	case m.IsAlive:
		age = currentYear - *m.BirthYear
	case m.DeathYear != nil:
		age = *m.DeathYear - *m.BirthYear
	default:
		return nil
	}
	if age < 0 {
		return nil
	}
	return &age
}

// Relatives is the set of direct relatives of one member (AC66).
type Relatives struct {
	Father   *domain.MemberRef
	Mother   *domain.MemberRef
	Spouses  []domain.MemberRef
	Siblings []domain.SiblingRef
	Children []domain.MemberRef
}

// RelativesOf computes the direct relatives of the member id. ok is false when
// the id is not in the tree. Lists are never nil.
func RelativesOf(idx *TreeIndex, id int64) (Relatives, bool) {
	m, ok := idx.byID[id]
	if !ok {
		return Relatives{}, false
	}
	r := Relatives{
		Spouses:  []domain.MemberRef{},
		Siblings: []domain.SiblingRef{},
		Children: []domain.MemberRef{},
	}

	if m.Kind == domain.MemberKindBlood {
		for _, pid := range []*int64{m.ParentID, m.OtherParentID} {
			if pid == nil {
				continue
			}
			p, ok := idx.byID[*pid]
			if !ok {
				continue
			}
			ref := memberRef(p)
			switch p.Gender {
			case domain.GenderMale:
				if r.Father == nil {
					r.Father = &ref
				}
			case domain.GenderFemale:
				if r.Mother == nil {
					r.Mother = &ref
				}
			}
		}
		for _, s := range idx.spouses[m.ID] {
			r.Spouses = append(r.Spouses, memberRef(s))
		}
		if m.ParentID != nil {
			for _, sib := range idx.children[*m.ParentID] {
				if sib.ID == m.ID {
					continue
				}
				half := sib.OtherParentID != nil && m.OtherParentID != nil && *sib.OtherParentID != *m.OtherParentID
				r.Siblings = append(r.Siblings, domain.SiblingRef{MemberRef: memberRef(sib), IsHalfSibling: half})
			}
		}
		for _, c := range idx.children[m.ID] {
			r.Children = append(r.Children, memberRef(c))
		}
		return r, true
	}

	// In-law: the spouse is the attached blood member; the children are those
	// naming this person as the other parent.
	if a := idx.anchor(m); a != nil {
		r.Spouses = append(r.Spouses, memberRef(a))
	}
	for _, c := range idx.byOtherParent[m.ID] {
		r.Children = append(r.Children, memberRef(c))
	}
	return r, true
}

// LineageOf is the chain from the founder down to the member, both included
// (AC67). For an in-law it is the lineage of the attached blood member. ok is
// false when the member is unknown or the chain is broken.
func LineageOf(idx *TreeIndex, id int64) ([]domain.LineageItem, bool) {
	m, ok := idx.byID[id]
	if !ok {
		return nil, false
	}
	a := idx.anchor(m)
	if a == nil {
		return nil, false
	}
	path := idx.pathFromRoot(a)
	if path == nil {
		return nil, false
	}
	out := make([]domain.LineageItem, 0, len(path))
	for _, p := range path {
		out = append(out, domain.LineageItem{ID: p.ID, FullName: p.FullName, Generation: p.Generation})
	}
	return out, true
}

// ChiOf computes the branch (chi) path of a member (AC68). A blood member N
// with parent P heads a branch when N is male and P has two or more male blood
// children. The path lists every branch head from the founder down to the
// member, outermost first; the current branch is its last element. Both are
// empty when the path has no branch head. ok is false when the member is
// unknown or the chain is broken.
func ChiOf(idx *TreeIndex, id int64) (current *domain.ChiItem, path []domain.ChiItem, ok bool) {
	m, found := idx.byID[id]
	if !found {
		return nil, nil, false
	}
	a := idx.anchor(m)
	if a == nil {
		return nil, nil, false
	}
	chain := idx.pathFromRoot(a)
	if chain == nil {
		return nil, nil, false
	}
	path = []domain.ChiItem{}
	for _, n := range chain {
		if n.ParentID == nil || n.Gender != domain.GenderMale {
			continue
		}
		var sons []*domain.GenealogyMember
		for _, c := range idx.children[*n.ParentID] {
			if c.Gender == domain.GenderMale {
				sons = append(sons, c)
			}
		}
		if len(sons) < 2 {
			continue
		}
		rank := 0
		for i, s := range sons {
			if s.ID == n.ID {
				rank = i + 1
				break
			}
		}
		path = append(path, domain.ChiItem{Ancestor: memberRef(n), Rank: rank, IsEldest: rank == 1})
	}
	if len(path) == 0 {
		return nil, path, true
	}
	last := path[len(path)-1]
	return &last, path, true
}

// Kinship codes are a stable enum (AC69). A "_HO" suffix marks a more distant
// line (min(dM, dT) >= 2), whose label gets the word "họ".
const (
	KinshipSpouse             domain.KinshipCode = "SPOUSE"
	KinshipFather             domain.KinshipCode = "FATHER"
	KinshipMother             domain.KinshipCode = "MOTHER"
	KinshipPaternalGrandpa    domain.KinshipCode = "PATERNAL_GRANDFATHER"
	KinshipPaternalGrandma    domain.KinshipCode = "PATERNAL_GRANDMOTHER"
	KinshipGreatGrandparent   domain.KinshipCode = "GREAT_GRANDPARENT"
	KinshipAncestor           domain.KinshipCode = "ANCESTOR"
	KinshipElderSibling       domain.KinshipCode = "ELDER_SIBLING"
	KinshipYoungerSibling     domain.KinshipCode = "YOUNGER_SIBLING"
	KinshipFatherElderBrother domain.KinshipCode = "FATHER_ELDER_BROTHER"
	KinshipFatherYoungBrother domain.KinshipCode = "FATHER_YOUNGER_BROTHER"
	KinshipFatherSister       domain.KinshipCode = "FATHER_SISTER"
	KinshipGrandUncleElder    domain.KinshipCode = "GRAND_UNCLE_ELDER"
	KinshipGrandUncleYounger  domain.KinshipCode = "GRAND_UNCLE_YOUNGER"
	KinshipGrandAunt          domain.KinshipCode = "GRAND_AUNT"
	KinshipChild              domain.KinshipCode = "CHILD"
	KinshipGrandchild         domain.KinshipCode = "GRANDCHILD"
	KinshipGreatGrandchild    domain.KinshipCode = "GREAT_GRANDCHILD"
	KinshipGreatGreatGrand    domain.KinshipCode = "GREAT_GREAT_GRANDCHILD"
	KinshipNephewNiece        domain.KinshipCode = "NEPHEW_NIECE"
	KinshipDaughterInLaw      domain.KinshipCode = "DAUGHTER_IN_LAW"
	KinshipSonInLaw           domain.KinshipCode = "SON_IN_LAW"
	KinshipGrandchildSpouse   domain.KinshipCode = "GRANDCHILD_SPOUSE"
	KinshipBrotherWifeElder   domain.KinshipCode = "ELDER_BROTHER_WIFE"
	KinshipBrotherWifeYounger domain.KinshipCode = "YOUNGER_BROTHER_WIFE"
	KinshipSisterHusbandElder domain.KinshipCode = "ELDER_SISTER_HUSBAND"
	KinshipSisterHusbandYoung domain.KinshipCode = "YOUNGER_SISTER_HUSBAND"
	KinshipUncleElderWife     domain.KinshipCode = "FATHER_ELDER_BROTHER_WIFE"
	KinshipUncleYoungerWife   domain.KinshipCode = "FATHER_YOUNGER_BROTHER_WIFE"
	KinshipAuntHusband        domain.KinshipCode = "FATHER_SISTER_HUSBAND"
	KinshipNephewSpouse       domain.KinshipCode = "NEPHEW_NIECE_SPOUSE"
)

func kin(code domain.KinshipCode, label string) domain.Kinship {
	return domain.Kinship{Code: code, Label: label}
}

var kinUnknown = domain.Kinship{Code: domain.KinshipUnknown}

// withHo marks a distant line: code gets "_HO", label gets " họ".
func withHo(k domain.Kinship, ho bool) domain.Kinship {
	if !ho || k.Code == domain.KinshipUnknown {
		return k
	}
	k.Code += "_HO"
	k.Label += " họ"
	return k
}

// rel is the structural relation of a blood member x to a blood member me.
type rel struct {
	same     bool
	ancestor bool // x is a direct ancestor of me
	descend  bool // x is a direct descendant of me
	dM, dT   int  // depth of me / x below the common ancestor
	xElder   bool // x's branch is elder (collateral lines only)
}

func relate(idx *TreeIndex, me, x *domain.GenealogyMember) (rel, bool) {
	if me.ID == x.ID {
		return rel{same: true}, true
	}
	pm := idx.pathFromRoot(me)
	px := idx.pathFromRoot(x)
	if pm == nil || px == nil {
		return rel{}, false
	}
	// Longest common prefix gives the lowest common ancestor.
	n := 0
	for n < len(pm) && n < len(px) && pm[n].ID == px[n].ID {
		n++
	}
	if n == 0 {
		return rel{}, false // different roots: should not happen in one tree
	}
	a := pm[n-1]
	r := rel{dM: me.Generation - a.Generation, dT: x.Generation - a.Generation}
	switch {
	case a.ID == x.ID:
		r.ancestor = true
	case a.ID == me.ID:
		r.descend = true
	default:
		// The two children of the common ancestor on each path decide elder.
		cm, cx := pm[n], px[n]
		r.xElder = orderBefore(cx, cm)
	}
	return r, true
}

func ordinalUp(n int, xGen int) domain.Kinship {
	switch n {
	case 1:
		return kin(KinshipFather, "Cha")
	case 2:
		return kin(KinshipPaternalGrandpa, "Ông nội")
	case 3:
		return kin(KinshipGreatGrandparent, "Cụ ông")
	case 4:
		return kin(KinshipAncestor, "Kỵ")
	default:
		return kin(KinshipAncestor, fmt.Sprintf("Tổ đời %d", xGen))
	}
}

func isHo(r rel) bool {
	m := r.dM
	if r.dT < m {
		m = r.dT
	}
	return m >= 2
}

// bloodTerm names a blood member x relative to me.
func bloodTerm(r rel, x *domain.GenealogyMember) domain.Kinship {
	male := x.Gender == domain.GenderMale
	switch {
	case r.same:
		return kin(domain.KinshipSelf, "Bản thân")
	case r.ancestor:
		return ordinalUp(r.dM, x.Generation)
	case r.descend:
		switch r.dT {
		case 1:
			return kin(KinshipChild, "Con")
		case 2:
			return kin(KinshipGrandchild, "Cháu nội")
		case 3:
			return kin(KinshipGreatGrandchild, "Chắt")
		case 4:
			return kin(KinshipGreatGreatGrand, "Chút")
		}
		return kinUnknown
	}

	ho := isHo(r)
	switch {
	case r.dT == r.dM: // peers
		base := kin(KinshipYoungerSibling, "Em")
		if r.xElder {
			if male {
				base = kin(KinshipElderSibling, "Anh")
			} else {
				base = kin(KinshipElderSibling, "Chị")
			}
		}
		return withHo(base, ho)
	case r.dT < r.dM: // x is senior by a line
		switch r.dM - r.dT {
		case 1:
			switch {
			case !male:
				return withHo(kin(KinshipFatherSister, "Cô"), ho)
			case r.xElder:
				return withHo(kin(KinshipFatherElderBrother, "Bác"), ho)
			default:
				return withHo(kin(KinshipFatherYoungBrother, "Chú"), ho)
			}
		case 2:
			switch {
			case !male:
				return withHo(kin(KinshipGrandAunt, "Bà cô"), ho)
			case r.xElder:
				return withHo(kin(KinshipGrandUncleElder, "Ông bác"), ho)
			default:
				return withHo(kin(KinshipGrandUncleYounger, "Ông chú"), ho)
			}
		}
		return kinUnknown
	default: // x is junior by a line
		if r.dT-r.dM <= 2 {
			return withHo(kin(KinshipNephewNiece, "Cháu"), ho)
		}
		return kinUnknown
	}
}

// inLawTerm names an in-law target whose attached blood member x stands in
// relation r to me (the "vợ/chồng của" rule of the table, AC69). maleTarget is
// the in-law's own gender.
func inLawTerm(r rel, x, target *domain.GenealogyMember) domain.Kinship {
	maleTarget := target.Gender == domain.GenderMale
	xMale := x.Gender == domain.GenderMale
	ho := isHo(r)

	switch {
	case r.same:
		if maleTarget {
			return kin(KinshipSpouse, "Chồng")
		}
		return kin(KinshipSpouse, "Vợ")
	case r.ancestor:
		switch r.dM {
		case 1:
			// Only the registered other parent is "Mẹ"; another wife of the
			// father has no approved term in v1.
			return kinUnknown
		case 2:
			return kin(KinshipPaternalGrandma, "Bà nội")
		case 3:
			return kin(KinshipGreatGrandparent, "Cụ bà")
		case 4:
			return kin(KinshipAncestor, "Kỵ")
		}
		return kinUnknown
	case r.descend:
		switch {
		case r.dT == 1 && !maleTarget:
			return kin(KinshipDaughterInLaw, "Con dâu")
		case r.dT == 1 && maleTarget:
			return kin(KinshipSonInLaw, "Con rể")
		case r.dT <= 3 && !maleTarget:
			return kin(KinshipGrandchildSpouse, "Cháu dâu")
		case r.dT <= 3 && maleTarget:
			return kin(KinshipGrandchildSpouse, "Cháu rể")
		}
		return kinUnknown
	case r.dT == r.dM: // spouse of a peer
		switch {
		case xMale && r.xElder:
			return withHo(kin(KinshipBrotherWifeElder, "Chị dâu"), ho)
		case xMale:
			return withHo(kin(KinshipBrotherWifeYounger, "Em dâu"), ho)
		case r.xElder:
			return withHo(kin(KinshipSisterHusbandElder, "Anh rể"), ho)
		default:
			return withHo(kin(KinshipSisterHusbandYoung, "Em rể"), ho)
		}
	case r.dT < r.dM && r.dM-r.dT == 1: // spouse of an uncle or aunt
		switch {
		case !xMale:
			return withHo(kin(KinshipAuntHusband, "Dượng"), ho)
		case r.xElder:
			return withHo(kin(KinshipUncleElderWife, "Bác gái"), ho)
		default:
			return withHo(kin(KinshipUncleYoungerWife, "Thím"), ho)
		}
	case r.dT > r.dM && r.dT-r.dM <= 2: // spouse of a nephew or niece
		if maleTarget {
			return withHo(kin(KinshipNephewSpouse, "Cháu rể"), ho)
		}
		return withHo(kin(KinshipNephewSpouse, "Cháu dâu"), ho)
	}
	return kinUnknown
}

// KinshipOf names what target is to "me" (AC69). It returns code UNKNOWN with
// an empty label when the relation is outside what v1 covers (the maternal
// side, very distant lines, or an in-law "me" other than their own family).
func KinshipOf(idx *TreeIndex, meID, targetID int64) domain.Kinship {
	me, ok := idx.byID[meID]
	if !ok {
		return kinUnknown
	}
	target, ok := idx.byID[targetID]
	if !ok {
		return kinUnknown
	}
	if me.ID == target.ID {
		return kin(domain.KinshipSelf, "Bản thân")
	}

	// "Me" is an in-law: only the spouse and our own children are named.
	if me.Kind == domain.MemberKindInLaw {
		if me.SpouseOfID != nil && target.ID == *me.SpouseOfID {
			if target.Gender == domain.GenderMale {
				return kin(KinshipSpouse, "Chồng")
			}
			return kin(KinshipSpouse, "Vợ")
		}
		if target.Kind == domain.MemberKindBlood && target.OtherParentID != nil && *target.OtherParentID == me.ID {
			return kin(KinshipChild, "Con")
		}
		return kinUnknown
	}

	// My registered mother (the other parent) is "Mẹ".
	if me.OtherParentID != nil && target.ID == *me.OtherParentID {
		return kin(KinshipMother, "Mẹ")
	}

	x := idx.anchor(target)
	if x == nil {
		return kinUnknown
	}
	r, ok := relate(idx, me, x)
	if !ok {
		return kinUnknown
	}
	if target.Kind == domain.MemberKindInLaw {
		return inLawTerm(r, x, target)
	}
	return bloodTerm(r, x)
}
