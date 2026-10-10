package domain

// provinces are the 34 province-level administrative units of Việt Nam in force
// since 1 July 2025 (Nghị quyết 202/2025/QH15 of 12 June 2025): 28 provinces and
// 6 centrally-governed cities, written without the "Tỉnh" / "Thành phố" prefix —
// the form stored in temples.province.
//
// This is the single list the seeder validates against. It is data fixed by law,
// not deployment configuration, and it is deliberately not a database CHECK: a
// future administrative change is one edit here plus a data migration, not a
// schema change.
var provinces = []string{
	// 6 cities
	"Hà Nội", "Huế", "Hải Phòng", "Đà Nẵng", "Hồ Chí Minh", "Cần Thơ",
	// 28 provinces
	"Tuyên Quang", "Lào Cai", "Thái Nguyên", "Phú Thọ", "Bắc Ninh", "Hưng Yên",
	"Ninh Bình", "Quảng Trị", "Quảng Ngãi", "Gia Lai", "Khánh Hòa", "Lâm Đồng",
	"Đắk Lắk", "Đồng Nai", "Tây Ninh", "Vĩnh Long", "Đồng Tháp", "Cà Mau", "An Giang",
	"Lai Châu", "Điện Biên", "Sơn La", "Lạng Sơn", "Quảng Ninh", "Thanh Hóa",
	"Nghệ An", "Hà Tĩnh", "Cao Bằng",
}

// IsProvince reports whether name is exactly one of the 34 province-level units
// (case-sensitive, no surrounding space, no prefix).
func IsProvince(name string) bool {
	for _, p := range provinces {
		if p == name {
			return true
		}
	}
	return false
}
