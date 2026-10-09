package main

import (
	"encoding/json"
	"fmt"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// sourceDeity mirrors one object of assets/data/phat_botat.json.
type sourceDeity struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	NameVI          string `json:"name_vi"`
	NameSanskrit    string `json:"name_sanskrit"`
	NameChinese     string `json:"name_chinese"`
	TitleHonorific  string `json:"title_honorific"`
	ThumbnailURL    string `json:"thumbnail_url"`
	ImageURL        string `json:"image_url"`
	CanonicalSource string `json:"canonical_source"`
	CoreVirtue      string `json:"core_virtue"`
	PureLand        string `json:"pure_land"`
	MemorialDays    []struct {
		LunarDay   int    `json:"lunar_day"`
		LunarMonth int    `json:"lunar_month"`
		EventName  string `json:"event_name"`
	} `json:"lunar_memorial_days"`
	Iconography struct {
		Posture         string   `json:"posture"`
		MountOrThrone   string   `json:"mount_or_throne"`
		SacredArtifacts []string `json:"sacred_artifacts"`
		SymbolicMeaning string   `json:"symbolic_meaning"`
	} `json:"iconography"`
	BiographySummary string   `json:"biography_summary"`
	GreatVows        []string `json:"great_vows"`
	Mantras          struct {
		ShortSanskrit          string `json:"short_mantra_sanskrit"`
		ShortSinoVietnamese    string `json:"short_mantra_sino_vietnamese"`
		LongDharaniName        string `json:"long_dharani_name"`
		DharaniPhoneticSnippet string `json:"dharani_phonetic_snippet"`
	} `json:"mantras"`
}

// parseDeities turns the source JSON into domain deities, validating what the
// database would otherwise reject one row at a time: unique ids, a known type,
// a name. SortOrder is the position in the file (see migrations/0007).
func parseDeities(src []byte) ([]domain.Deity, error) {
	var raw []sourceDeity
	if err := json.Unmarshal(src, &raw); err != nil {
		return nil, fmt.Errorf("decode deities: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("no deities in source")
	}

	seen := make(map[string]bool, len(raw))
	out := make([]domain.Deity, 0, len(raw))
	for i, s := range raw {
		if s.ID == "" {
			return nil, fmt.Errorf("entry %d: empty id", i)
		}
		if seen[s.ID] {
			return nil, fmt.Errorf("duplicate id %q", s.ID)
		}
		seen[s.ID] = true
		if s.Type != domain.DeityBuddha && s.Type != domain.DeityBodhisattva && s.Type != domain.DeityDeva {
			return nil, fmt.Errorf("deity %q: unknown type %q", s.ID, s.Type)
		}
		if s.NameVI == "" {
			return nil, fmt.Errorf("deity %q: empty name_vi", s.ID)
		}

		days := make([]domain.MemorialDay, 0, len(s.MemorialDays))
		for _, m := range s.MemorialDays {
			days = append(days, domain.MemorialDay{LunarDay: m.LunarDay, LunarMonth: m.LunarMonth, EventName: m.EventName})
		}
		out = append(out, domain.Deity{
			ID: s.ID, Type: s.Type, SortOrder: i,
			NameVI: s.NameVI, NameSanskrit: s.NameSanskrit, NameChinese: s.NameChinese,
			TitleHonorific: s.TitleHonorific, ThumbnailURL: s.ThumbnailURL, ImageURL: s.ImageURL,
			CanonicalSource: s.CanonicalSource, CoreVirtue: s.CoreVirtue, PureLand: s.PureLand,
			LunarMemorialDays: days,
			Iconography: domain.Iconography{
				Posture: s.Iconography.Posture, MountOrThrone: s.Iconography.MountOrThrone,
				SacredArtifacts: s.Iconography.SacredArtifacts, SymbolicMeaning: s.Iconography.SymbolicMeaning,
			},
			BiographySummary: s.BiographySummary, GreatVows: s.GreatVows,
			Mantras: domain.Mantras{
				ShortSanskrit: s.Mantras.ShortSanskrit, ShortSinoVietnamese: s.Mantras.ShortSinoVietnamese,
				LongDharaniName: s.Mantras.LongDharaniName, DharaniPhoneticSnippet: s.Mantras.DharaniPhoneticSnippet,
			},
		})
	}
	return out, nil
}
