package lunarcal

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

// memorialVectorPath is the product rule file for the memorial book. It is read
// from the repository, never copied, so the test fails when the product owner
// adds or changes a case (AC46, NFR-MEM-03).
const memorialVectorPath = "../../../../docs/product/memorial/rules/death-date-conversion.json"

type vectorLunarDate struct {
	Day       int  `json:"day"`
	Month     int  `json:"month"`
	Year      int  `json:"year"`
	LeapMonth bool `json:"leap_month"`
}

type memorialVectors struct {
	Rule  []string `json:"rule"`
	Cases []struct {
		Name  string `json:"name"`
		Input struct {
			Solar string           `json:"solar"`
			Lunar *vectorLunarDate `json:"lunar"`
		} `json:"input"`
		Expected struct {
			Lunar *vectorLunarDate `json:"lunar"`
			Solar string           `json:"solar"`
			Valid *bool            `json:"valid"`
		} `json:"expected"`
	} `json:"cases"`
}

// TestMemorialDeathDateVectors drives every case of the product's
// death-date-conversion rules through the converter the memorial book uses.
func TestMemorialDeathDateVectors(t *testing.T) {
	raw, err := os.ReadFile(memorialVectorPath)
	if err != nil {
		t.Fatalf("read product vectors %s: %v", memorialVectorPath, err)
	}
	var file memorialVectors
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode product vectors: %v", err)
	}
	if len(file.Cases) == 0 {
		t.Fatal("product vectors contain no cases")
	}

	for _, tc := range file.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			switch {
			case tc.Input.Solar != "":
				solar, err := time.Parse("2006-01-02", tc.Input.Solar)
				if err != nil {
					t.Fatalf("case input solar %q: %v", tc.Input.Solar, err)
				}
				if tc.Expected.Lunar == nil {
					t.Fatalf("case has a solar input but no expected lunar date")
				}
				want := Date{
					Day:         tc.Expected.Lunar.Day,
					Month:       tc.Expected.Lunar.Month,
					Year:        tc.Expected.Lunar.Year,
					IsLeapMonth: tc.Expected.Lunar.LeapMonth,
				}
				if got := SolarToLunar(solar); got != want {
					t.Errorf("SolarToLunar(%s) = %+v, want %+v", tc.Input.Solar, got, want)
				}

			case tc.Input.Lunar != nil:
				in := tc.Input.Lunar
				got, err := LunarToSolar(Date{
					Day:         in.Day,
					Month:       in.Month,
					Year:        in.Year,
					IsLeapMonth: in.LeapMonth,
				})
				if tc.Expected.Valid != nil && !*tc.Expected.Valid {
					if !errors.Is(err, ErrInvalidLunarDate) {
						t.Errorf("LunarToSolar(%+v) error = %v, want ErrInvalidLunarDate", *in, err)
					}
					return
				}
				if err != nil {
					t.Fatalf("LunarToSolar(%+v): %v", *in, err)
				}
				if tc.Expected.Solar == "" {
					t.Fatalf("case has a lunar input but no expected solar date")
				}
				if got.Format("2006-01-02") != tc.Expected.Solar {
					t.Errorf("LunarToSolar(%+v) = %s, want %s", *in, got.Format("2006-01-02"), tc.Expected.Solar)
				}

			default:
				t.Fatalf("case has neither a solar nor a lunar input")
			}
		})
	}
}
