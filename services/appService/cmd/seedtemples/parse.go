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
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
		// Address is the part below ward level (house number, street, hamlet); it
		// may be empty for a temple known only by its ward.
		Address *string `json:"address"`
		// ProvinceCode and WardCode are current-scheme codes from the
		// administrative-units dataset (spec temple v1.7). The unit NAMES are not
		// read from the file: resolveAddresses takes them from the dataset.
		ProvinceCode string `json:"province_code"`
		WardCode     string `json:"ward_code"`
		// Province is optional and legacy: when present it must equal the
		// dataset's name for ProvinceCode (a typo guard, checked in resolveAddresses).
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
// heritage rank, a derivable region and the presence of the structured address
// fields. It does not touch the database: whether the codes are real, and the
// unit names, come from resolveAddresses afterwards (flow Step 4).
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
		provinceCode := strings.TrimSpace(s.Location.ProvinceCode)
		if provinceCode == "" {
			return nil, fmt.Errorf("temple %q: location.province_code is required", s.ID)
		}
		wardCode := strings.TrimSpace(s.Location.WardCode)
		if wardCode == "" {
			return nil, fmt.Errorf("temple %q: location.ward_code is required", s.ID)
		}
		if s.Location.Address == nil {
			return nil, fmt.Errorf("temple %q: location.address is required (use \"\" when there is nothing below ward level)", s.ID)
		}
		out = append(out, domain.Temple{
			ID: s.ID, Name: s.Name, Type: s.Type, Region: region,
			ImageURL: s.ImageURL, Thumbnail: s.ThumbnailURL, Aliases: s.Aliases,
			Sect: s.Sect, Founder: s.Founder, Abbot: s.CurrentAbbot,
			Established: s.Established, Heritage: s.HeritageRank, HeritagePriority: prio,
			Latitude: s.Location.Lat, Longitude: s.Location.Lng, Address: strings.TrimSpace(*s.Location.Address),
			ProvinceCode: provinceCode, WardCode: wardCode, Province: strings.TrimSpace(s.Location.Province),
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
