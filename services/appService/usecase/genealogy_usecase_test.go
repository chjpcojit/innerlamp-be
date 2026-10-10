package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func genCfg() config.GenealogyConfig {
	return config.GenealogyConfig{
		MaxPerUser: 10, MaxMembers: 5000, MaxGeneration: 60, MaxSpouses: 5,
		MaxViewers: 20, InvitationTTL: config.Duration(30 * 24 * time.Hour),
	}
}

type genFixture struct {
	u        *GenealogyUsecase
	genes    *domain.MockGenealogyRepository
	members  *domain.MockGenealogyMemberRepository
	phones   *domain.MockUserPhoneLookup
	cfg      config.GenealogyConfig
	userID   int64
	genealID int64
}

func newGenFixture(t *testing.T) *genFixture {
	t.Helper()
	f := &genFixture{
		genes:    new(domain.MockGenealogyRepository),
		members:  new(domain.MockGenealogyMemberRepository),
		phones:   new(domain.MockUserPhoneLookup),
		cfg:      genCfg(),
		userID:   7,
		genealID: 42,
	}
	f.u = NewGenealogyUsecase(f.genes, f.members, f.phones, f.cfg)
	f.u.now = func() time.Time { return time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC) }
	return f
}

func validCreate() domain.CreateGenealogyInput {
	return domain.CreateGenealogyInput{
		Name:          "Họ Nguyễn",
		AncestralHome: "Nam Định",
		Founder:       domain.MemberInfo{FullName: "Nguyễn Văn A", Gender: male(), IsAlive: trueP()},
	}
}

func TestGenealogyUsecase_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates the tree with founder and spouses defaulting to the opposite gender", func(t *testing.T) {
		f := newGenFixture(t)
		in := validCreate()
		in.Spouses = []domain.MemberInfo{
			{FullName: "Trần Thị B", IsAlive: trueP()},
			{FullName: "Lê Thị C", Gender: female(), IsAlive: trueP()},
		}
		f.genes.On("Create", ctx, f.userID, mock.MatchedBy(func(p domain.CreateGenealogyParams) bool {
			return p.Name == "Họ Nguyễn" && p.AncestralHome == "Nam Định" &&
				p.Founder.Kind == domain.MemberKindBlood && p.Founder.Generation == 1 &&
				p.Founder.ParentID == nil && p.Founder.Gender == domain.GenderMale &&
				len(p.Spouses) == 2 &&
				p.Spouses[0].Kind == domain.MemberKindInLaw && p.Spouses[0].Generation == 1 &&
				p.Spouses[0].Gender == domain.GenderFemale
		}), f.cfg.MaxPerUser).Return(f.genealID, nil)
		f.genes.On("FindDetail", ctx, f.genealID).Return(&domain.GenealogyDetail{
			Genealogy: domain.Genealogy{ID: f.genealID, Name: "Họ Nguyễn"},
		}, nil)
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000001", nil)
		f.members.On("FindMemberIDByPhone", ctx, f.genealID, "+84900000001").Return(nil, nil)

		got, err := f.u.Create(ctx, f.userID, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.MyRole != domain.RoleOwner {
			t.Errorf("MyRole = %q, want owner", got.MyRole)
		}
		f.genes.AssertExpectations(t)
	})

	t.Run("limit reached is passed through", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("Create", ctx, f.userID, mock.Anything, f.cfg.MaxPerUser).Return(int64(0), domain.ErrGenealogyLimitReached)
		_, err := f.u.Create(ctx, f.userID, validCreate())
		if !errors.Is(err, domain.ErrGenealogyLimitReached) {
			t.Errorf("err = %v, want ErrGenealogyLimitReached", err)
		}
	})

	invalid := []struct {
		name      string
		mutate    func(in *domain.CreateGenealogyInput)
		wantField string
	}{
		{"blank name", func(in *domain.CreateGenealogyInput) { in.Name = "  " }, "name"},
		{"name too long", func(in *domain.CreateGenealogyInput) { in.Name = strings.Repeat("a", 101) }, "name"},
		{"blank hometown", func(in *domain.CreateGenealogyInput) { in.AncestralHome = "" }, "ancestral_home"},
		{"hometown too long", func(in *domain.CreateGenealogyInput) { in.AncestralHome = strings.Repeat("a", 201) }, "ancestral_home"},
		{"founder blank name", func(in *domain.CreateGenealogyInput) { in.Founder.FullName = "" }, "founder.full_name"},
		{"founder female", func(in *domain.CreateGenealogyInput) { in.Founder.Gender = female() }, "founder.gender"},
		{"founder is_alive missing", func(in *domain.CreateGenealogyInput) { in.Founder.IsAlive = nil }, "founder.is_alive"},
		{"founder deceased cannot be clan head", func(in *domain.CreateGenealogyInput) {
			in.Founder.IsAlive = falseP()
			in.Founder.IsClanHead = trueP()
		}, "founder.is_clan_head"},
		{"too many spouses", func(in *domain.CreateGenealogyInput) {
			for i := 0; i < 6; i++ {
				in.Spouses = append(in.Spouses, domain.MemberInfo{FullName: "V", IsAlive: trueP()})
			}
		}, "spouses"},
		{"spouse blank name", func(in *domain.CreateGenealogyInput) {
			in.Spouses = []domain.MemberInfo{{FullName: "ok", IsAlive: trueP()}, {FullName: "", IsAlive: trueP()}}
		}, "spouses[1].full_name"},
		{"spouse clan head", func(in *domain.CreateGenealogyInput) {
			in.Spouses = []domain.MemberInfo{{FullName: "ok", IsAlive: trueP(), IsClanHead: trueP()}}
		}, "spouses[0].is_clan_head"},
		{"same phone on founder and spouse", func(in *domain.CreateGenealogyInput) {
			in.Founder.PhoneNumber = strP("0912345678")
			in.Spouses = []domain.MemberInfo{{FullName: "V", IsAlive: trueP(), PhoneNumber: strP("0912345678")}}
		}, "spouses[0].phone_number"},
		{"same phone on two spouses", func(in *domain.CreateGenealogyInput) {
			in.Spouses = []domain.MemberInfo{
				{FullName: "V1", IsAlive: trueP(), PhoneNumber: strP("0912345678")},
				{FullName: "V2", IsAlive: trueP(), PhoneNumber: strP("0912345678")},
			}
		}, "spouses[1].phone_number"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			f := newGenFixture(t)
			in := validCreate()
			tt.mutate(&in)
			_, err := f.u.Create(ctx, f.userID, in)
			if err == nil {
				t.Fatalf("expected a validation error on %s", tt.wantField)
			}
			if got := fieldOf(t, err); got != tt.wantField {
				t.Errorf("field = %q, want %q", got, tt.wantField)
			}
			f.genes.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestGenealogyUsecase_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("viewer sees the tree, own position and the lunar chạp date", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.RoleViewer, nil)
		f.genes.On("FindDetail", ctx, f.genealID).Return(&domain.GenealogyDetail{
			Genealogy: domain.Genealogy{ID: f.genealID},
			Chronicle: domain.Chronicle{ChapYear: intP(2024), ChapMonth: intP(2), ChapDay: intP(10)},
		}, nil)
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000001", nil)
		mid := int64(11)
		f.members.On("FindMemberIDByPhone", ctx, f.genealID, "+84900000001").Return(&mid, nil)

		got, err := f.u.Get(ctx, f.userID, f.genealID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.MyRole != domain.RoleViewer {
			t.Errorf("MyRole = %q, want viewer", got.MyRole)
		}
		if got.MyMemberID == nil || *got.MyMemberID != 11 {
			t.Errorf("MyMemberID = %v, want 11", got.MyMemberID)
		}
		l := got.Chronicle.ChapLunar
		if l == nil || l.Day != 1 || l.Month != 1 || l.Leap {
			t.Errorf("ChapLunar = %+v, want 1/1 (Tết 2024 fell on 10 Feb 2024)", l)
		}
	})

	t.Run("no member carries the caller's number", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.RoleOwner, nil)
		f.genes.On("FindDetail", ctx, f.genealID).Return(&domain.GenealogyDetail{}, nil)
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000001", nil)
		f.members.On("FindMemberIDByPhone", ctx, f.genealID, "+84900000001").Return(nil, nil)

		got, err := f.u.Get(ctx, f.userID, f.genealID)
		if err != nil {
			t.Fatal(err)
		}
		if got.MyMemberID != nil {
			t.Errorf("MyMemberID = %v, want nil", *got.MyMemberID)
		}
		if got.Chronicle.ChapLunar != nil {
			t.Error("ChapLunar must be nil without a chạp date")
		}
	})

	t.Run("a stranger gets not found and nothing else is read", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.Role(""), domain.ErrGenealogyNotFound)
		_, err := f.u.Get(ctx, f.userID, f.genealID)
		if !errors.Is(err, domain.ErrGenealogyNotFound) {
			t.Errorf("err = %v, want ErrGenealogyNotFound", err)
		}
		f.genes.AssertNotCalled(t, "FindDetail", mock.Anything, mock.Anything)
	})

	t.Run("a non-positive id is not found without touching the repository", func(t *testing.T) {
		f := newGenFixture(t)
		for _, id := range []int64{0, -5} {
			if _, err := f.u.Get(ctx, f.userID, id); !errors.Is(err, domain.ErrGenealogyNotFound) {
				t.Errorf("id %d: err = %v, want ErrGenealogyNotFound", id, err)
			}
		}
		f.genes.AssertNotCalled(t, "ResolveRole", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestGenealogyUsecase_UpdateInfo(t *testing.T) {
	ctx := context.Background()

	t.Run("owner renames", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.RoleOwner, nil)
		f.genes.On("UpdateInfo", ctx, f.genealID, "Họ Trần", "Hà Nội").Return(nil)
		f.genes.On("FindDetail", ctx, f.genealID).Return(&domain.GenealogyDetail{}, nil)
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000001", nil)
		f.members.On("FindMemberIDByPhone", ctx, f.genealID, mock.Anything).Return(nil, nil)

		if _, err := f.u.UpdateInfo(ctx, f.userID, f.genealID, "  Họ Trần ", " Hà Nội "); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		f.genes.AssertCalled(t, "UpdateInfo", ctx, f.genealID, "Họ Trần", "Hà Nội")
	})

	t.Run("viewer is refused and nothing changes", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.RoleViewer, nil)
		_, err := f.u.UpdateInfo(ctx, f.userID, f.genealID, "x", "y")
		if !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
		f.genes.AssertNotCalled(t, "UpdateInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("invalid values", func(t *testing.T) {
		for _, tt := range []struct{ name, home, want string }{
			{"", "ok", "name"},
			{"ok", "", "ancestral_home"},
			{strings.Repeat("a", 101), "ok", "name"},
		} {
			f := newGenFixture(t)
			f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.RoleOwner, nil)
			_, err := f.u.UpdateInfo(ctx, f.userID, f.genealID, tt.name, tt.home)
			if err == nil || fieldOf(t, err) != tt.want {
				t.Errorf("name=%q home=%q: err = %v, want field %s", tt.name, tt.home, err, tt.want)
			}
		}
	})
}

func TestGenealogyUsecase_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("owner deletes", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.RoleOwner, nil)
		f.genes.On("Delete", ctx, f.genealID).Return(nil)
		if err := f.u.Delete(ctx, f.userID, f.genealID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		f.genes.AssertCalled(t, "Delete", ctx, f.genealID)
	})

	t.Run("viewer is refused", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.RoleViewer, nil)
		if err := f.u.Delete(ctx, f.userID, f.genealID); !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
		f.genes.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("stranger gets not found", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.Role(""), domain.ErrGenealogyNotFound)
		if err := f.u.Delete(ctx, f.userID, f.genealID); !errors.Is(err, domain.ErrGenealogyNotFound) {
			t.Errorf("err = %v, want ErrGenealogyNotFound", err)
		}
	})
}

func TestGenealogyUsecase_List(t *testing.T) {
	ctx := context.Background()
	f := newGenFixture(t)
	f.genes.On("ListByUser", ctx, f.userID).Return([]domain.GenealogySummary{{ID: 1, MyRole: domain.RoleOwner}, {ID: 2, MyRole: domain.RoleViewer}}, nil)
	got, err := f.u.List(ctx, f.userID)
	if err != nil || len(got) != 2 {
		t.Fatalf("List = %v, %v", got, err)
	}

	f2 := newGenFixture(t)
	f2.genes.On("ListByUser", ctx, f2.userID).Return(nil, nil)
	got, err = f2.u.List(ctx, f2.userID)
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("empty list must be a non-nil empty slice, got %#v, %v", got, err)
	}
}

func TestGenealogyUsecase_UpdateChronicle(t *testing.T) {
	ctx := context.Background()
	long := func(n int) *string { s := strings.Repeat("a", n); return &s }

	ownerFixture := func(t *testing.T) *genFixture {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.RoleOwner, nil)
		return f
	}

	t.Run("owner replaces the chronicle", func(t *testing.T) {
		f := ownerFixture(t)
		in := domain.Chronicle{
			Surname: strP("  Nguyễn "), CoupletUpper: strP("上聯"), CoupletUpperReading: strP("thượng liên"),
			CoupletUpperMeaning: strP("vế trên"),
			ChapYear:            intP(2026), ChapMonth: intP(1), ChapDay: intP(31), ChapEnteredAsLunar: true,
		}
		f.genes.On("UpsertChronicle", ctx, mock.MatchedBy(func(c *domain.Chronicle) bool {
			return c.GenealogyID == f.genealID && c.Surname != nil && *c.Surname == "Nguyễn" &&
				c.ChapEnteredAsLunar && c.CoupletUpperReading != nil
		})).Return(nil)
		f.genes.On("FindDetail", ctx, f.genealID).Return(&domain.GenealogyDetail{}, nil)
		f.phones.On("FindPhoneByID", ctx, f.userID).Return("+84900000001", nil)
		f.members.On("FindMemberIDByPhone", ctx, f.genealID, mock.Anything).Return(nil, nil)

		if _, err := f.u.UpdateChronicle(ctx, f.userID, f.genealID, in); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		f.genes.AssertExpectations(t)
	})

	t.Run("viewer is refused", func(t *testing.T) {
		f := newGenFixture(t)
		f.genes.On("ResolveRole", ctx, f.genealID, f.userID).Return(domain.RoleViewer, nil)
		_, err := f.u.UpdateChronicle(ctx, f.userID, f.genealID, domain.Chronicle{})
		if !errors.Is(err, domain.ErrGenealogyForbidden) {
			t.Errorf("err = %v, want ErrGenealogyForbidden", err)
		}
		f.genes.AssertNotCalled(t, "UpsertChronicle", mock.Anything, mock.Anything)
	})

	invalid := []struct {
		name      string
		in        domain.Chronicle
		wantField string
	}{
		{"surname too long", domain.Chronicle{Surname: long(51)}, "surname"},
		{"location too long", domain.Chronicle{CurrentLocation: long(201)}, "current_location"},
		{"since_year before 1000", domain.Chronicle{SinceYear: intP(999)}, "since_year"},
		{"since_year in the future", domain.Chronicle{SinceYear: intP(2027)}, "since_year"},
		{"origin too long", domain.Chronicle{Origin: long(5001)}, "origin"},
		{"history too long", domain.Chronicle{History: long(20001)}, "history"},
		{"hall name too long", domain.Chronicle{HallName: long(201)}, "hall_name"},
		{"hall address too long", domain.Chronicle{HallAddress: long(301)}, "hall_address"},
		{"plaque reading too long", domain.Chronicle{HoanhPhiReading: long(201)}, "hoanh_phi_reading"},
		{"plaque meaning too long", domain.Chronicle{HoanhPhiMeaning: long(1001)}, "hoanh_phi_meaning"},
		{"couplet too long", domain.Chronicle{CoupletLower: long(501)}, "couplet_lower"},
		{"couplet reading too long", domain.Chronicle{CoupletLowerReading: long(501)}, "couplet_lower_reading"},
		{"couplet meaning too long", domain.Chronicle{CoupletUpperMeaning: long(1001)}, "couplet_upper_meaning"},
		{"teaching too long", domain.Chronicle{Teaching: long(5001)}, "teaching"},
		{"chạp year without month and day", domain.Chronicle{ChapYear: intP(2026)}, "chap_month"},
		{"chạp month without day", domain.Chronicle{ChapYear: intP(2026), ChapMonth: intP(1)}, "chap_day"},
		{"chạp month 13", domain.Chronicle{ChapYear: intP(2026), ChapMonth: intP(13), ChapDay: intP(1)}, "chap_month"},
		{"chạp 30 February", domain.Chronicle{ChapYear: intP(2026), ChapMonth: intP(2), ChapDay: intP(30)}, "chap_day"},
		{"chạp year in the future", domain.Chronicle{ChapYear: intP(2027), ChapMonth: intP(1), ChapDay: intP(1)}, "chap_year"},
		{"lunar flag without a date", domain.Chronicle{ChapEnteredAsLunar: true}, "chap_entered_as_lunar"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			f := ownerFixture(t)
			_, err := f.u.UpdateChronicle(ctx, f.userID, f.genealID, tt.in)
			if err == nil {
				t.Fatalf("expected a validation error on %s", tt.wantField)
			}
			if got := fieldOf(t, err); got != tt.wantField {
				t.Errorf("field = %q, want %q", got, tt.wantField)
			}
			f.genes.AssertNotCalled(t, "UpsertChronicle", mock.Anything, mock.Anything)
		})
	}
}
