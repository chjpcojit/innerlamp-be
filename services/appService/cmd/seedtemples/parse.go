package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// sourceTemple mirrors one object of assets/data/temple.json.
type sourceTemple struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	ImageURL      string   `json:"image_url"`
	ThumbnailURL  string   `json:"thumbnail_url"`
	Aliases       []string `json:"aliases"`
	Sect          string   `json:"sect"`
	Founder       string   `json:"founder"`
	CurrentAbbot  string   `json:"current_abbot"`
	Established   int      `json:"established_year"`
	HeritageRank  string   `json:"heritage_rank"`
	OperatingHour string   `json:"operating_hours"`

	Introduction         string   `json:"introduction"`
	ArchitectureHeritage string   `json:"architecture_heritage"`
	VisitingNotes        []string `json:"visiting_notes"`
	Location             struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Address string  `json:"address"`
		// Province is explicit seed data (not parsed from Address); required.
		Province string `json:"province"`
	} `json:"location"`
	Amenities struct {
		ParkingCar        bool `json:"parking_car"`
		ParkingMotorcycle bool `json:"parking_motorcycle"`
		VegetarianMeals   bool `json:"vegetarian_meals"`
		SummerRetreat     bool `json:"summer_retreat"`
	} `json:"amenities"`
}

// stripBlockComments removes /* ... */ comments, which temple.json uses for its
// regional headers and encoding/json rejects. It tracks string literals so a
// "/*" inside a value is left alone.
func stripBlockComments(src []byte) ([]byte, error) {
	var out bytes.Buffer
	inString := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case inString:
			out.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
			} else if c == '"' {
				inString = false
			}
		case c == '"':
			inString = true
			out.WriteByte(c)
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := bytes.Index(src[i+2:], []byte("*/"))
			if end < 0 {
				return nil, fmt.Errorf("unterminated /* comment at byte %d", i)
			}
			i += 2 + end + 1
		default:
			out.WriteByte(c)
		}
	}
	return out.Bytes(), nil
}

// regionOf derives north/central/south from an id like "temple_north_001".
func regionOf(id string) (string, error) {
	parts := strings.Split(id, "_")
	if len(parts) != 3 || parts[0] != "temple" {
		return "", fmt.Errorf("id %q is not temple_<region>_<n>", id)
	}
	switch parts[1] {
	case "north", "central", "south":
		return parts[1], nil
	}
	return "", fmt.Errorf("id %q has unknown region %q", id, parts[1])
}

// parseTemples turns the JSONC source into domain temples, validating what the
// database would otherwise reject one row at a time: unique ids, a known
// heritage rank and a derivable region.
func parseTemples(src []byte) ([]domain.Temple, error) {
	clean, err := stripBlockComments(src)
	if err != nil {
		return nil, err
	}
	var raw []sourceTemple
	if err := json.Unmarshal(clean, &raw); err != nil {
		return nil, fmt.Errorf("decode temples: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("no temples in source")
	}

	seen := make(map[string]bool, len(raw))
	out := make([]domain.Temple, 0, len(raw))
	for _, s := range raw {
		if seen[s.ID] {
			return nil, fmt.Errorf("duplicate id %q", s.ID)
		}
		seen[s.ID] = true

		region, err := regionOf(s.ID)
		if err != nil {
			return nil, err
		}
		prio, ok := domain.HeritagePriority(s.HeritageRank)
		if !ok {
			return nil, fmt.Errorf("temple %q: unknown heritage_rank %q", s.ID, s.HeritageRank)
		}
		province := strings.TrimSpace(s.Location.Province)
		if province == "" {
			return nil, fmt.Errorf("temple %q: location.province is required", s.ID)
		}
		if !domain.IsProvince(province) {
			return nil, fmt.Errorf("temple %q: location.province %q is not one of the 34 province-level units (use the name without a \"Tỉnh\"/\"TP.\" prefix, e.g. \"Hồ Chí Minh\")", s.ID, province)
		}
		out = append(out, domain.Temple{
			ID: s.ID, Name: s.Name, Type: s.Type, Region: region,
			ImageURL: s.ImageURL, Thumbnail: s.ThumbnailURL, Aliases: s.Aliases,
			Sect: s.Sect, Founder: s.Founder, Abbot: s.CurrentAbbot,
			Established: s.Established, Heritage: s.HeritageRank, HeritagePriority: prio,
			Latitude: s.Location.Lat, Longitude: s.Location.Lng, Address: s.Location.Address, Province: province,
			OperatingHours: s.OperatingHour,
			Introduction:   s.Introduction, ArchitectureHeritage: s.ArchitectureHeritage, VisitingNotes: s.VisitingNotes,
			Amenities: domain.TempleAmenities{
				ParkingCar: s.Amenities.ParkingCar, ParkingMotorcycle: s.Amenities.ParkingMotorcycle,
				VegetarianMeals: s.Amenities.VegetarianMeals, SummerRetreat: s.Amenities.SummerRetreat,
			},
		})
	}
	return out, nil
}
