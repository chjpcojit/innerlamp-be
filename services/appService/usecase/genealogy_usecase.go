package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// GenealogyUsecase manages genealogies: create, list, open, rename, delete and
// the chronicle (US-GEN-01, 02, 05). The owner writes; a viewer only reads.
//
// Every method that takes a genealogy id resolves the caller's role first
// (design.md Step A) and answers ErrGenealogyNotFound for "no such tree" and
// "no role in it" alike (AC15).
type GenealogyUsecase struct {
	genealogies domain.GenealogyRepository
	members     domain.GenealogyMemberRepository
	phones      domain.UserPhoneLookup
	cfg         config.GenealogyConfig
	now         func() time.Time
}

// NewGenealogyUsecase wires the use case.
func NewGenealogyUsecase(
	genealogies domain.GenealogyRepository,
	members domain.GenealogyMemberRepository,
	phones domain.UserPhoneLookup,
	cfg config.GenealogyConfig,
) *GenealogyUsecase {
	return &GenealogyUsecase{genealogies: genealogies, members: members, phones: phones, cfg: cfg, now: time.Now}
}

// currentYear is "this year" in Asia/Ho_Chi_Minh (AC7). A fixed UTC+7 zone
// avoids depending on tzdata in the runtime image.
func currentYearVN(now time.Time) int {
	return now.In(vietnamZone).Year()
}

// resolveRole is Step A: the caller's role in the tree, or ErrGenealogyNotFound.
// A non-positive id cannot exist and never reaches the database.
func resolveRole(ctx context.Context, repo domain.GenealogyRepository, userID, genealogyID int64) (domain.Role, error) {
	if genealogyID <= 0 {
		return "", domain.ErrGenealogyNotFound
	}
	return repo.ResolveRole(ctx, genealogyID, userID)
}

// requireOwner resolves the role and rejects a viewer (AC81).
func requireOwner(ctx context.Context, repo domain.GenealogyRepository, userID, genealogyID int64) error {
	role, err := resolveRole(ctx, repo, userID, genealogyID)
	if err != nil {
		return err
	}
	if role != domain.RoleOwner {
		return domain.ErrGenealogyForbidden
	}
	return nil
}

// Create validates the input and creates the tree with its founder and the
// founder's spouses in one transaction, then returns the overview (AC1–AC12).
func (u *GenealogyUsecase) Create(ctx context.Context, userID int64, in domain.CreateGenealogyInput) (*domain.GenealogyDetail, error) {
	name, home, err := validateTreeInfo(in.Name, in.AncestralHome)
	if err != nil {
		return nil, err
	}
	year := currentYearVN(u.now())

	founder, err := buildMember(in.Founder, memberCtx{
		prefix: "founder.", kind: domain.MemberKindBlood, isFounder: true, currentYear: year,
	}, nil)
	if err != nil {
		return nil, err
	}
	if founder.Gender != domain.GenderMale {
		return nil, genInvalid("founder.gender", "the founder must be male")
	}
	founder.Generation = 1

	if len(in.Spouses) > u.cfg.MaxSpouses {
		return nil, genInvalid("spouses", fmt.Sprintf("at most %d spouses are allowed", u.cfg.MaxSpouses))
	}
	// A spouse without a gender takes the opposite of the founder's (AC9).
	opposite := domain.GenderFemale
	spouses := make([]domain.GenealogyMember, 0, len(in.Spouses))
	for i, s := range in.Spouses {
		m, err := buildMember(s, memberCtx{
			prefix: fmt.Sprintf("spouses[%d].", i), kind: domain.MemberKindInLaw, currentYear: year,
		}, &opposite)
		if err != nil {
			return nil, err
		}
		m.Generation = 1
		spouses = append(spouses, *m)
	}

	// A phone number belongs to one member per tree (AC60); catch a repeat
	// inside the request before the database does.
	seen := map[string]bool{}
	if founder.PhoneNumber != nil {
		seen[*founder.PhoneNumber] = true
	}
	for i := range spouses {
		p := spouses[i].PhoneNumber
		if p == nil {
			continue
		}
		if seen[*p] {
			return nil, genInvalid(fmt.Sprintf("spouses[%d].phone_number", i), "phone_number is repeated in this request")
		}
		seen[*p] = true
	}

	id, err := u.genealogies.Create(ctx, userID, domain.CreateGenealogyParams{
		Name: name, AncestralHome: home, Founder: *founder, Spouses: spouses,
	}, u.cfg.MaxPerUser)
	if err != nil {
		return nil, err
	}
	return u.detail(ctx, userID, id, domain.RoleOwner)
}

// List returns the trees the user owns or views, newest first (AC13).
func (u *GenealogyUsecase) List(ctx context.Context, userID int64) ([]domain.GenealogySummary, error) {
	items, err := u.genealogies.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.GenealogySummary{}
	}
	return items, nil
}

// Get returns the overview for the owner or a viewer (AC14).
func (u *GenealogyUsecase) Get(ctx context.Context, userID, genealogyID int64) (*domain.GenealogyDetail, error) {
	role, err := resolveRole(ctx, u.genealogies, userID, genealogyID)
	if err != nil {
		return nil, err
	}
	return u.detail(ctx, userID, genealogyID, role)
}

// UpdateInfo renames the tree (AC16). Owner only.
func (u *GenealogyUsecase) UpdateInfo(ctx context.Context, userID, genealogyID int64, name, ancestralHome string) (*domain.GenealogyDetail, error) {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	n, h, err := validateTreeInfo(name, ancestralHome)
	if err != nil {
		return nil, err
	}
	if err := u.genealogies.UpdateInfo(ctx, genealogyID, n, h); err != nil {
		return nil, err
	}
	return u.detail(ctx, userID, genealogyID, domain.RoleOwner)
}

// Delete removes the tree and everything under it (AC17). Owner only.
func (u *GenealogyUsecase) Delete(ctx context.Context, userID, genealogyID int64) error {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return err
	}
	return u.genealogies.Delete(ctx, genealogyID)
}

// UpdateChronicle replaces every chronicle field (AC38–AC40). Owner only.
func (u *GenealogyUsecase) UpdateChronicle(ctx context.Context, userID, genealogyID int64, in domain.Chronicle) (*domain.GenealogyDetail, error) {
	if err := requireOwner(ctx, u.genealogies, userID, genealogyID); err != nil {
		return nil, err
	}
	c, err := buildChronicle(in, currentYearVN(u.now()))
	if err != nil {
		return nil, err
	}
	c.GenealogyID = genealogyID
	if err := u.genealogies.UpsertChronicle(ctx, c); err != nil {
		return nil, err
	}
	return u.detail(ctx, userID, genealogyID, domain.RoleOwner)
}

// detail loads the overview and fills in what depends on the caller: their
// role, their position (AC59) and the derived lunar date (AC40).
func (u *GenealogyUsecase) detail(ctx context.Context, userID, genealogyID int64, role domain.Role) (*domain.GenealogyDetail, error) {
	d, err := u.genealogies.FindDetail(ctx, genealogyID)
	if err != nil {
		return nil, err
	}
	d.MyRole = role

	phone, err := u.phones.FindPhoneByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if phone != "" {
		id, err := u.members.FindMemberIDByPhone(ctx, genealogyID, phone)
		if err != nil {
			return nil, err
		}
		d.MyMemberID = id
	}
	attachChapLunar(&d.Chronicle)
	return d, nil
}

// attachChapLunar derives the lunar chạp date from the stored solar one.
func attachChapLunar(c *domain.Chronicle) {
	c.ChapLunar = nil
	if c.ChapYear == nil || c.ChapMonth == nil || c.ChapDay == nil {
		return
	}
	if l, ok := SolarToLunar(*c.ChapYear, *c.ChapMonth, *c.ChapDay); ok {
		c.ChapLunar = &l
	}
}

func validateTreeInfo(name, home string) (string, string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", "", genInvalid("name", "name is required")
	}
	if tooLong(n, genNameMax) {
		return "", "", genInvalid("name", fmt.Sprintf("name must be at most %d characters", genNameMax))
	}
	h := strings.TrimSpace(home)
	if h == "" {
		return "", "", genInvalid("ancestral_home", "ancestral_home is required")
	}
	if tooLong(h, genHomeMax) {
		return "", "", genInvalid("ancestral_home", fmt.Sprintf("ancestral_home must be at most %d characters", genHomeMax))
	}
	return n, h, nil
}

// buildChronicle validates every chronicle field and returns the normalised
// copy (AC39, AC40). Text is trimmed; blank becomes nil.
func buildChronicle(in domain.Chronicle, currentYear int) (*domain.Chronicle, error) {
	c := &domain.Chronicle{}
	text := []struct {
		field string
		src   *string
		dst   **string
		max   int
	}{
		{"surname", in.Surname, &c.Surname, genChronicleSurname},
		{"current_location", in.CurrentLocation, &c.CurrentLocation, 200},
		{"origin", in.Origin, &c.Origin, 5000},
		{"history", in.History, &c.History, 20000},
		{"hall_name", in.HallName, &c.HallName, 200},
		{"hall_address", in.HallAddress, &c.HallAddress, 300},
		{"hoanh_phi", in.HoanhPhi, &c.HoanhPhi, 200},
		{"hoanh_phi_reading", in.HoanhPhiReading, &c.HoanhPhiReading, 200},
		{"hoanh_phi_meaning", in.HoanhPhiMeaning, &c.HoanhPhiMeaning, 1000},
		{"couplet_upper", in.CoupletUpper, &c.CoupletUpper, 500},
		{"couplet_upper_reading", in.CoupletUpperReading, &c.CoupletUpperReading, 500},
		{"couplet_upper_meaning", in.CoupletUpperMeaning, &c.CoupletUpperMeaning, 1000},
		{"couplet_lower", in.CoupletLower, &c.CoupletLower, 500},
		{"couplet_lower_reading", in.CoupletLowerReading, &c.CoupletLowerReading, 500},
		{"couplet_lower_meaning", in.CoupletLowerMeaning, &c.CoupletLowerMeaning, 1000},
		{"teaching", in.Teaching, &c.Teaching, 5000},
	}
	for _, t := range text {
		v, err := optText(t.src, t.max, t.field)
		if err != nil {
			return nil, err
		}
		*t.dst = v
	}

	if in.SinceYear != nil {
		if *in.SinceYear < genMinYear || *in.SinceYear > currentYear {
			return nil, genInvalid("since_year", fmt.Sprintf("since_year must be between %d and %d", genMinYear, currentYear))
		}
		c.SinceYear = in.SinceYear
	}

	// Chạp họ: a solar date, all three parts or none (AC40).
	hasY, hasM, hasD := in.ChapYear != nil, in.ChapMonth != nil, in.ChapDay != nil
	if hasY || hasM || hasD {
		switch {
		case !hasY:
			return nil, genInvalid("chap_year", "chap_year is required with chap_month and chap_day")
		case !hasM:
			return nil, genInvalid("chap_month", "chap_month is required with chap_year and chap_day")
		case !hasD:
			return nil, genInvalid("chap_day", "chap_day is required with chap_year and chap_month")
		}
		if *in.ChapYear < genMinYear || *in.ChapYear > currentYear {
			return nil, genInvalid("chap_year", fmt.Sprintf("chap_year must be between %d and %d", genMinYear, currentYear))
		}
		if *in.ChapMonth < 1 || *in.ChapMonth > 12 {
			return nil, genInvalid("chap_month", "chap_month must be between 1 and 12")
		}
		if *in.ChapDay < 1 || *in.ChapDay > DaysInSolarMonth(*in.ChapMonth) {
			return nil, genInvalid("chap_day", "chap_day is not a valid day of that month")
		}
		c.ChapYear, c.ChapMonth, c.ChapDay = in.ChapYear, in.ChapMonth, in.ChapDay
	} else if in.ChapEnteredAsLunar {
		return nil, genInvalid("chap_entered_as_lunar", "chap_entered_as_lunar needs chap_year, chap_month and chap_day")
	}
	c.ChapEnteredAsLunar = in.ChapEnteredAsLunar
	return c, nil
}
