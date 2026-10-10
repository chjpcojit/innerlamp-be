package domain

// heritagePriority orders heritage ranks from most to least prominent. It is a
// deliberate COPY of appService/domain/temple.go's table: backoffice writes the
// heritage_priority column of temples, and appService sorts the catalogue by it,
// so the two must agree. heritage_priority_sync_test.go fails if they drift.
// A new rank is a migration (chk_temples_heritage_rank) plus an entry in BOTH maps.
var heritagePriority = map[string]int{
	"DI_SAN_THE_GIOI":           1,
	"DI_TICH_QUOC_GIA_DAC_BIET": 2,
	"KHU_DU_LICH_QUOC_GIA":      3,
	"DI_TICH_QUOC_GIA":          4,
	"DANH_LAM_THANG_CANH":       5,
	"DI_TICH_CAP_TINH":          6,
	"DI_TICH_CAP_THANH_PHO":     7,
	"CHUA_XEP_HANG":             8,
}

// HeritagePriority returns the ranking weight of a heritage_rank value and
// whether the value is known.
func HeritagePriority(rank string) (int, bool) {
	p, ok := heritagePriority[rank]
	return p, ok
}

// HeritageRanks lists every known rank (order unspecified).
func HeritageRanks() []string {
	out := make([]string, 0, len(heritagePriority))
	for r := range heritagePriority {
		out = append(out, r)
	}
	return out
}
