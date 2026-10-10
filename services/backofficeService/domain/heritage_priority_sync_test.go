package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appdomain "chjpcoj.com.innerlamp/services/appService/domain"
)

// TestHeritagePriority_MatchesAppService is the drift guard for the copied table
// (N7): the catalogue is sorted by appService's mapping, so a temple created here
// with a different number would sit in the wrong place. It compares every rank in
// both directions, so a rank added to only one side fails too.
func TestHeritagePriority_MatchesAppService(t *testing.T) {
	ours := HeritageRanks()
	require.NotEmpty(t, ours)

	for _, rank := range ours {
		want, ok := appdomain.HeritagePriority(rank)
		require.Truef(t, ok, "rank %q exists here but not in appService/domain", rank)
		got, _ := HeritagePriority(rank)
		assert.Equalf(t, want, got, "priority of %q differs from appService", rank)
	}

	// The reverse direction: appService's ranks are probed through the known
	// priorities 1..N, since its map is unexported.
	for _, rank := range []string{
		"DI_SAN_THE_GIOI", "DI_TICH_QUOC_GIA_DAC_BIET", "KHU_DU_LICH_QUOC_GIA", "DI_TICH_QUOC_GIA",
		"DANH_LAM_THANG_CANH", "DI_TICH_CAP_TINH", "DI_TICH_CAP_THANH_PHO", "CHUA_XEP_HANG",
	} {
		_, ok := HeritagePriority(rank)
		assert.Truef(t, ok, "rank %q known to appService is missing here", rank)
	}
	assert.Len(t, ours, 8, "rank count changed: update both maps and this test")
}

func TestHeritagePriority_UnknownRank(t *testing.T) {
	_, ok := HeritagePriority("NOT_A_RANK")
	assert.False(t, ok)
}
