package usecase

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Field limits fixed by the spec and the genealogy schema (VARCHAR lengths and
// CHECK ranges). They are schema, not tuning knobs, so they are constants; the
// tunable limits (tree size, viewers, invitation lifetime) come from
// config.GenealogyConfig.
const (
	genNameMax          = 100
	genHomeMax          = 200
	genResidenceMax     = 200
	genDharmaNameMax    = 100
	genTabletMax        = 200
	genBurialMax        = 200
	genNoteMax          = 1000
	genMinYear          = 1000
	genBirthOrderMax    = 1000
	genChronicleSurname = 50
)

// genInvalid builds the validation error the handler turns into a 400 with
// details.field.
func genInvalid(field, message string) error {
	return &domain.GenealogyValidationError{Field: field, Message: message}
}

// memberCtx carries what differs between the places a member is submitted.
type memberCtx struct {
	// prefix is prepended to every reported field: "founder.", "spouses[0]." or "".
	prefix string
	// kind is the kind the member will have; in-laws cannot head the clan or
	// have a birth order.
	kind domain.MemberKind
	// isFounder is true for the progenitor, who has no birth order (AC56).
	isFounder bool
	// currentYear is the current year in Asia/Ho_Chi_Minh (AC7).
	currentYear int
}

func (c memberCtx) f(name string) string { return c.prefix + name }

func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }

// buildMember validates the submitted information of one person and turns it
// into an entity (without id, genealogy, parent, generation or timestamps).
// defaultGender is used when the request omits gender (spouses, AC9/AC20).
//
// Validation order follows the field order of the request so the first
// reported field is the first broken one.
func buildMember(info domain.MemberInfo, c memberCtx, defaultGender *domain.Gender) (*domain.GenealogyMember, error) {
	m := &domain.GenealogyMember{Kind: c.kind}

	name := strings.TrimSpace(info.FullName)
	if name == "" {
		return nil, genInvalid(c.f("full_name"), "full_name is required")
	}
	if tooLong(name, genNameMax) {
		return nil, genInvalid(c.f("full_name"), fmt.Sprintf("full_name must be at most %d characters", genNameMax))
	}
	m.FullName = name

	gender := info.Gender
	if gender == nil {
		gender = defaultGender
	}
	if gender == nil || !gender.Valid() {
		return nil, genInvalid(c.f("gender"), "gender must be male or female")
	}
	m.Gender = *gender

	if info.IsAlive == nil {
		return nil, genInvalid(c.f("is_alive"), "is_alive is required")
	}
	m.IsAlive = *info.IsAlive

	// Years (AC7).
	if info.BirthYear != nil {
		if *info.BirthYear < genMinYear || *info.BirthYear > c.currentYear {
			return nil, genInvalid(c.f("birth_year"), fmt.Sprintf("birth_year must be between %d and %d", genMinYear, c.currentYear))
		}
		m.BirthYear = info.BirthYear
	}
	if info.DeathYear != nil {
		if m.IsAlive {
			return nil, genInvalid(c.f("death_year"), "death_year is only allowed for a deceased member")
		}
		if *info.DeathYear < genMinYear || *info.DeathYear > c.currentYear {
			return nil, genInvalid(c.f("death_year"), fmt.Sprintf("death_year must be between %d and %d", genMinYear, c.currentYear))
		}
		if m.BirthYear != nil && *info.DeathYear < *m.BirthYear {
			return nil, genInvalid(c.f("death_year"), "death_year must not be before birth_year")
		}
		m.DeathYear = info.DeathYear
	}

	// Free text (AC8).
	var err error
	if m.Residence, err = optText(info.Residence, genResidenceMax, c.f("residence")); err != nil {
		return nil, err
	}
	if m.DharmaName, err = optText(info.DharmaName, genDharmaNameMax, c.f("dharma_name")); err != nil {
		return nil, err
	}
	if info.TabletLocation != nil {
		if m.IsAlive {
			return nil, genInvalid(c.f("tablet_location"), "tablet_location is only allowed for a deceased member")
		}
		if m.TabletLocation, err = optText(info.TabletLocation, genTabletMax, c.f("tablet_location")); err != nil {
			return nil, err
		}
	}
	if m.Note, err = optText(info.Note, genNoteMax, c.f("note")); err != nil {
		return nil, err
	}
	if info.PhoneNumber != nil {
		raw := strings.TrimSpace(*info.PhoneNumber)
		if raw != "" {
			n := validation.NormalizePhone(raw)
			if n == "" {
				return nil, genInvalid(c.f("phone_number"), "phone_number is not a valid phone number")
			}
			m.PhoneNumber = &n
		}
	}

	if err := applyAnniversary(m, info, c); err != nil {
		return nil, err
	}

	// Clan head flag (AC82): a living male blood member only.
	if info.IsClanHead != nil && *info.IsClanHead {
		if c.kind != domain.MemberKindBlood || m.Gender != domain.GenderMale || !m.IsAlive {
			return nil, genInvalid(c.f("is_clan_head"), "only a living male member of the clan can be clan head")
		}
		m.IsClanHead = true
	}

	// Birth order (AC56): only for blood members other than the founder.
	if info.BirthOrder != nil {
		if c.kind != domain.MemberKindBlood || c.isFounder {
			return nil, genInvalid(c.f("birth_order"), "birth_order does not apply to the founder or an in-law")
		}
		if *info.BirthOrder < 1 || *info.BirthOrder > genBirthOrderMax {
			return nil, genInvalid(c.f("birth_order"), fmt.Sprintf("birth_order must be between 1 and %d", genBirthOrderMax))
		}
		m.BirthOrder = info.BirthOrder
	}
	return m, nil
}

// applyAnniversary validates the burial place and the solar death anniversary
// (AC55): month and day together, a death year to anchor them, a valid day for
// the month (February counts 29 — the year is checked separately), and none of
// it for a living member.
func applyAnniversary(m *domain.GenealogyMember, info domain.MemberInfo, c memberCtx) error {
	if info.BurialPlace != nil {
		if m.IsAlive {
			return genInvalid(c.f("burial_place"), "burial_place is only allowed for a deceased member")
		}
		v, err := optText(info.BurialPlace, genBurialMax, c.f("burial_place"))
		if err != nil {
			return err
		}
		m.BurialPlace = v
	}

	flag := info.DeathEnteredAsLunar != nil && *info.DeathEnteredAsLunar
	hasDay, hasMonth := info.DeathDay != nil, info.DeathMonth != nil

	if m.IsAlive {
		switch {
		case hasDay:
			return genInvalid(c.f("death_day"), "death_day is only allowed for a deceased member")
		case hasMonth:
			return genInvalid(c.f("death_month"), "death_month is only allowed for a deceased member")
		case flag:
			return genInvalid(c.f("death_entered_as_lunar"), "death_entered_as_lunar is only allowed for a deceased member")
		}
		return nil
	}

	switch {
	case hasMonth && !hasDay:
		return genInvalid(c.f("death_day"), "death_day is required together with death_month")
	case hasDay && !hasMonth:
		return genInvalid(c.f("death_month"), "death_month is required together with death_day")
	case !hasDay && flag:
		return genInvalid(c.f("death_entered_as_lunar"), "death_entered_as_lunar needs death_month and death_day")
	}
	if hasDay {
		if m.DeathYear == nil {
			return genInvalid(c.f("death_year"), "death_year is required with death_month and death_day")
		}
		if *info.DeathMonth < 1 || *info.DeathMonth > 12 {
			return genInvalid(c.f("death_month"), "death_month must be between 1 and 12")
		}
		if *info.DeathDay < 1 || *info.DeathDay > DaysInSolarMonth(*info.DeathMonth) {
			return genInvalid(c.f("death_day"), "death_day is not a valid day of that month")
		}
		m.DeathMonth = info.DeathMonth
		m.DeathDay = info.DeathDay
		v := flag
		m.DeathEnteredAsLunar = &v
	}
	return nil
}

// optText validates an optional free-text field: nil stays nil, an
// all-whitespace value becomes nil, otherwise it is trimmed and length-checked.
func optText(p *string, max int, field string) (*string, error) {
	if p == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil, nil
	}
	if tooLong(v, max) {
		return nil, genInvalid(field, fmt.Sprintf("%s must be at most %d characters", field, max))
	}
	return &v, nil
}
