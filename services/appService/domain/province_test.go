package domain

import "testing"

func TestProvinces_AreThe34UnitsOf2025(t *testing.T) {
	if got := len(provinces); got != 34 {
		t.Fatalf("len(provinces) = %d, want 34 (28 provinces + 6 cities, Nghị quyết 202/2025/QH15)", got)
	}
	seen := map[string]bool{}
	for _, p := range provinces {
		if p == "" || p != trimmed(p) {
			t.Errorf("province %q is empty or has surrounding spaces", p)
		}
		if seen[p] {
			t.Errorf("duplicate province %q", p)
		}
		seen[p] = true
	}
}

func trimmed(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func TestIsProvince(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"city without prefix", "Hà Nội", true},
		{"city Hồ Chí Minh", "Hồ Chí Minh", true},
		{"merged province", "Ninh Bình", true},
		{"unchanged province", "Hà Tĩnh", true},
		{"new unit Huế", "Huế", true},
		{"old name merged away", "Hà Nam", false},
		{"old name Thừa Thiên Huế", "Thừa Thiên Huế", false},
		{"old name Bình Dương", "Bình Dương", false},
		{"with prefix", "Tỉnh Ninh Bình", false},
		{"abbreviation", "TP.HCM", false},
		{"wrong case", "hà nội", false},
		{"surrounding space", " Hà Nội", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsProvince(tt.in); got != tt.want {
				t.Errorf("IsProvince(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
