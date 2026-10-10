package usecase

import (
	"errors"
	"strings"
	"testing"

	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func male() *domain.Gender   { g := domain.GenderMale; return &g }
func female() *domain.Gender { g := domain.GenderFemale; return &g }
func trueP() *bool           { b := true; return &b }
func falseP() *bool          { b := false; return &b }
func strP(s string) *string  { return &s }
func intP(n int) *int        { return &n }

func okInfo() domain.MemberInfo {
	return domain.MemberInfo{FullName: "Nguyễn Văn A", Gender: male(), IsAlive: trueP()}
}

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	var ve *domain.GenealogyValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error %v is not a GenealogyValidationError", err)
	}
	if !errors.Is(err, domain.ErrGenealogyInvalid) {
		t.Fatalf("error %v does not unwrap to ErrGenealogyInvalid", err)
	}
	return ve.Field
}

func TestBuildMember_Valid(t *testing.T) {
	info := okInfo()
	info.FullName = "  Nguyễn Văn A  "
	info.BirthYear = intP(1980)
	info.PhoneNumber = strP("0912345678")
	m, err := buildMember(info, memberCtx{kind: domain.MemberKindBlood, currentYear: 2026}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.FullName != "Nguyễn Văn A" {
		t.Errorf("FullName = %q, want trimmed", m.FullName)
	}
	if m.Kind != domain.MemberKindBlood || m.Gender != domain.GenderMale || !m.IsAlive {
		t.Errorf("unexpected member %+v", m)
	}
	want := validation.NormalizePhone("0912345678")
	if m.PhoneNumber == nil || *m.PhoneNumber != want {
		t.Errorf("PhoneNumber = %v, want normalised %q", m.PhoneNumber, want)
	}
}

func TestBuildMember_Invalid(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }
	tests := []struct {
		name      string
		mutate    func(i *domain.MemberInfo)
		ctx       memberCtx
		defGender *domain.Gender
		wantField string
	}{
		{"blank name", func(i *domain.MemberInfo) { i.FullName = "   " }, memberCtx{}, nil, "full_name"},
		{"name too long", func(i *domain.MemberInfo) { i.FullName = long(101) }, memberCtx{}, nil, "full_name"},
		{"gender missing without default", func(i *domain.MemberInfo) { i.Gender = nil }, memberCtx{}, nil, "gender"},
		{"gender not male or female", func(i *domain.MemberInfo) { g := domain.Gender("x"); i.Gender = &g }, memberCtx{}, nil, "gender"},
		{"is_alive missing", func(i *domain.MemberInfo) { i.IsAlive = nil }, memberCtx{}, nil, "is_alive"},
		{"living with death year", func(i *domain.MemberInfo) { i.DeathYear = intP(2000) }, memberCtx{}, nil, "death_year"},
		{"living with tablet location", func(i *domain.MemberInfo) { i.TabletLocation = strP("nhà thờ") }, memberCtx{}, nil, "tablet_location"},
		{"birth year before 1000", func(i *domain.MemberInfo) { i.BirthYear = intP(999) }, memberCtx{}, nil, "birth_year"},
		{"birth year in the future", func(i *domain.MemberInfo) { i.BirthYear = intP(2027) }, memberCtx{currentYear: 2026}, nil, "birth_year"},
		{"death before birth", func(i *domain.MemberInfo) {
			i.IsAlive = falseP()
			i.BirthYear = intP(1950)
			i.DeathYear = intP(1940)
		}, memberCtx{currentYear: 2026}, nil, "death_year"},
		{"death year before 1000", func(i *domain.MemberInfo) { i.IsAlive = falseP(); i.DeathYear = intP(999) }, memberCtx{currentYear: 2026}, nil, "death_year"},
		{"residence too long", func(i *domain.MemberInfo) { i.Residence = strP(long(201)) }, memberCtx{}, nil, "residence"},
		{"dharma name too long", func(i *domain.MemberInfo) { i.DharmaName = strP(long(101)) }, memberCtx{}, nil, "dharma_name"},
		{"tablet location too long", func(i *domain.MemberInfo) { i.IsAlive = falseP(); i.TabletLocation = strP(long(201)) }, memberCtx{}, nil, "tablet_location"},
		{"note too long", func(i *domain.MemberInfo) { i.Note = strP(long(1001)) }, memberCtx{}, nil, "note"},
		{"phone not a number", func(i *domain.MemberInfo) { i.PhoneNumber = strP("abc") }, memberCtx{}, nil, "phone_number"},
		{"burial place too long", func(i *domain.MemberInfo) { i.IsAlive = falseP(); i.BurialPlace = strP(long(201)) }, memberCtx{}, nil, "burial_place"},
		{"living with burial place", func(i *domain.MemberInfo) { i.BurialPlace = strP("nghĩa trang") }, memberCtx{}, nil, "burial_place"},
		{"month without day", func(i *domain.MemberInfo) {
			i.IsAlive = falseP()
			i.DeathYear = intP(2000)
			i.DeathMonth = intP(3)
		}, memberCtx{currentYear: 2026}, nil, "death_day"},
		{"day without month", func(i *domain.MemberInfo) {
			i.IsAlive = falseP()
			i.DeathYear = intP(2000)
			i.DeathDay = intP(3)
		}, memberCtx{currentYear: 2026}, nil, "death_month"},
		{"anniversary without death year", func(i *domain.MemberInfo) {
			i.IsAlive = falseP()
			i.DeathMonth = intP(3)
			i.DeathDay = intP(3)
		}, memberCtx{currentYear: 2026}, nil, "death_year"},
		{"month 13", func(i *domain.MemberInfo) {
			i.IsAlive = falseP()
			i.DeathYear = intP(2000)
			i.DeathMonth = intP(13)
			i.DeathDay = intP(1)
		}, memberCtx{currentYear: 2026}, nil, "death_month"},
		{"30 February", func(i *domain.MemberInfo) {
			i.IsAlive = falseP()
			i.DeathYear = intP(2000)
			i.DeathMonth = intP(2)
			i.DeathDay = intP(30)
		}, memberCtx{currentYear: 2026}, nil, "death_day"},
		{"lunar flag without a date", func(i *domain.MemberInfo) {
			i.IsAlive = falseP()
			i.DeathYear = intP(2000)
			i.DeathEnteredAsLunar = trueP()
		}, memberCtx{currentYear: 2026}, nil, "death_entered_as_lunar"},
		{"living with anniversary day", func(i *domain.MemberInfo) { i.DeathMonth = intP(3); i.DeathDay = intP(3) }, memberCtx{currentYear: 2026}, nil, "death_day"},
		{"living with lunar flag set", func(i *domain.MemberInfo) { i.DeathEnteredAsLunar = trueP() }, memberCtx{currentYear: 2026}, nil, "death_entered_as_lunar"},
		{"clan head female", func(i *domain.MemberInfo) { i.Gender = female(); i.IsClanHead = trueP() }, memberCtx{kind: domain.MemberKindBlood}, nil, "is_clan_head"},
		{"clan head deceased", func(i *domain.MemberInfo) { i.IsAlive = falseP(); i.IsClanHead = trueP() }, memberCtx{kind: domain.MemberKindBlood, currentYear: 2026}, nil, "is_clan_head"},
		{"clan head in-law", func(i *domain.MemberInfo) { i.IsClanHead = trueP() }, memberCtx{kind: domain.MemberKindInLaw}, nil, "is_clan_head"},
		{"birth order below 1", func(i *domain.MemberInfo) { i.BirthOrder = intP(0) }, memberCtx{kind: domain.MemberKindBlood}, nil, "birth_order"},
		{"birth order above 1000", func(i *domain.MemberInfo) { i.BirthOrder = intP(1001) }, memberCtx{kind: domain.MemberKindBlood}, nil, "birth_order"},
		{"birth order on in-law", func(i *domain.MemberInfo) { i.BirthOrder = intP(1) }, memberCtx{kind: domain.MemberKindInLaw}, nil, "birth_order"},
		{"birth order on founder", func(i *domain.MemberInfo) { i.BirthOrder = intP(1) }, memberCtx{kind: domain.MemberKindBlood, isFounder: true}, nil, "birth_order"},
		{"prefix is applied", func(i *domain.MemberInfo) { i.FullName = "" }, memberCtx{prefix: "founder."}, nil, "founder.full_name"},
		{"spouse prefix", func(i *domain.MemberInfo) { i.FullName = "" }, memberCtx{prefix: "spouses[1]."}, nil, "spouses[1].full_name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := okInfo()
			tt.mutate(&info)
			if tt.ctx.kind == "" {
				tt.ctx.kind = domain.MemberKindBlood
			}
			if tt.ctx.currentYear == 0 {
				tt.ctx.currentYear = 2026
			}
			_, err := buildMember(info, tt.ctx, tt.defGender)
			if err == nil {
				t.Fatalf("expected a validation error on %s", tt.wantField)
			}
			if got := fieldOf(t, err); got != tt.wantField {
				t.Errorf("field = %q, want %q", got, tt.wantField)
			}
		})
	}
}

func TestBuildMember_Accepts(t *testing.T) {
	t.Run("gender defaults when absent", func(t *testing.T) {
		info := okInfo()
		info.Gender = nil
		m, err := buildMember(info, memberCtx{kind: domain.MemberKindInLaw, currentYear: 2026}, female())
		if err != nil {
			t.Fatal(err)
		}
		if m.Gender != domain.GenderFemale {
			t.Errorf("Gender = %q, want female (default)", m.Gender)
		}
	})
	t.Run("given gender beats the default", func(t *testing.T) {
		info := okInfo()
		info.Gender = male()
		m, _ := buildMember(info, memberCtx{kind: domain.MemberKindInLaw, currentYear: 2026}, female())
		if m.Gender != domain.GenderMale {
			t.Errorf("Gender = %q, want male (as given)", m.Gender)
		}
	})
	t.Run("deceased with a full anniversary", func(t *testing.T) {
		info := okInfo()
		info.IsAlive = falseP()
		info.BirthYear = intP(1930)
		info.DeathYear = intP(2005)
		info.DeathMonth = intP(2)
		info.DeathDay = intP(29) // February is allowed 29: the year is validated elsewhere
		info.DeathEnteredAsLunar = trueP()
		info.TabletLocation = strP("nhà thờ họ")
		info.BurialPlace = strP("nghĩa trang quê")
		m, err := buildMember(info, memberCtx{kind: domain.MemberKindBlood, currentYear: 2026}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.DeathDay == nil || *m.DeathDay != 29 || m.DeathEnteredAsLunar == nil || !*m.DeathEnteredAsLunar {
			t.Errorf("anniversary not carried: %+v", m)
		}
	})
	t.Run("equal birth and death year is allowed", func(t *testing.T) {
		info := okInfo()
		info.IsAlive = falseP()
		info.BirthYear = intP(1990)
		info.DeathYear = intP(1990)
		if _, err := buildMember(info, memberCtx{kind: domain.MemberKindBlood, currentYear: 2026}, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("living clan head male blood", func(t *testing.T) {
		info := okInfo()
		info.IsClanHead = trueP()
		m, err := buildMember(info, memberCtx{kind: domain.MemberKindBlood, currentYear: 2026}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !m.IsClanHead {
			t.Error("IsClanHead not set")
		}
	})
	t.Run("birth order on a blood non-founder", func(t *testing.T) {
		info := okInfo()
		info.BirthOrder = intP(3)
		m, err := buildMember(info, memberCtx{kind: domain.MemberKindBlood, currentYear: 2026}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if m.BirthOrder == nil || *m.BirthOrder != 3 {
			t.Errorf("BirthOrder = %v, want 3", m.BirthOrder)
		}
	})
}
