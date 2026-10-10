package usecase

import "testing"

func TestSolarToLunar(t *testing.T) {
	tests := []struct {
		name             string
		y, m, d          int
		wantD, wantM     int
		wantLeap, wantOK bool
	}{
		// Mùng 1 Tết các năm (Vietnamese lunar calendar, UTC+7).
		{"Tết 2023", 2023, 1, 22, 1, 1, false, true},
		{"Tết 2024", 2024, 2, 10, 1, 1, false, true},
		{"Tết 2025", 2025, 1, 29, 1, 1, false, true},
		{"Tết 2026", 2026, 2, 17, 1, 1, false, true},
		// Ngày liền trước Tết là 30 (hoặc 29) tháng Chạp.
		{"giao thừa 2024", 2024, 2, 9, 30, 12, false, true},
		// Trung thu.
		{"Trung thu 2024", 2024, 9, 17, 15, 8, false, true},
		{"Trung thu 2025", 2025, 10, 6, 15, 8, false, true},
		// Tháng nhuận: 2023 nhuận tháng 2 (22/3 → 19/4/2023), 2025 nhuận tháng 6
		// (bắt đầu 25/7/2025), 2020 nhuận tháng 4 (bắt đầu 23/5/2020).
		{"đầu tháng 2 thường 2023", 2023, 2, 20, 1, 2, false, true},
		{"đầu tháng 2 nhuận 2023", 2023, 3, 22, 1, 2, true, true},
		{"cuối tháng 2 nhuận 2023", 2023, 4, 19, 29, 2, true, true},
		{"đầu tháng 3 sau nhuận 2023", 2023, 4, 20, 1, 3, false, true},
		{"đầu tháng 6 nhuận 2025", 2025, 7, 25, 1, 6, true, true},
		{"đầu tháng 4 nhuận 2020", 2020, 5, 23, 1, 4, true, true},
		// Ranh giới năm dương lịch và các năm xa hơn.
		{"1/1/2000", 2000, 1, 1, 25, 11, false, true},
		{"Tết 2000", 2000, 2, 5, 1, 1, false, true},
		{"Tết 1990", 1990, 1, 27, 1, 1, false, true},

		// Ngoài khoảng hỗ trợ (AC40/AC55): không quy đổi.
		{"trước 1900", 1899, 12, 31, 0, 0, false, false},
		{"sau 2199", 2200, 1, 1, 0, 0, false, false},
		// Ngày không tồn tại.
		{"30 tháng 2", 2024, 2, 30, 0, 0, false, false},
		{"tháng 13", 2024, 13, 1, 0, 0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SolarToLunar(tt.y, tt.m, tt.d)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.Day != tt.wantD || got.Month != tt.wantM || got.Leap != tt.wantLeap {
				t.Errorf("SolarToLunar(%d-%02d-%02d) = %d/%d leap=%v, want %d/%d leap=%v",
					tt.y, tt.m, tt.d, got.Day, got.Month, got.Leap, tt.wantD, tt.wantM, tt.wantLeap)
			}
		})
	}
}

func TestDaysInSolarMonth(t *testing.T) {
	tests := []struct{ m, want int }{
		{1, 31}, {2, 29}, {3, 31}, {4, 30}, {5, 31}, {6, 30},
		{7, 31}, {8, 31}, {9, 30}, {10, 31}, {11, 30}, {12, 31},
	}
	for _, tt := range tests {
		if got := DaysInSolarMonth(tt.m); got != tt.want {
			t.Errorf("DaysInSolarMonth(%d) = %d, want %d", tt.m, got, tt.want)
		}
	}
	if DaysInSolarMonth(0) != 0 || DaysInSolarMonth(13) != 0 {
		t.Error("DaysInSolarMonth out of range must be 0")
	}
}
