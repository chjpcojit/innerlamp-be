package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func TestParseDeities_RealDataFile(t *testing.T) {
	src, err := os.ReadFile("../../../../assets/data/phat_botat.json")
	require.NoError(t, err)

	got, err := parseDeities(src)

	require.NoError(t, err)
	require.Len(t, got, 20)
	assert.Equal(t, "buddha_shakyamuni", got[0].ID)
	for i, d := range got {
		assert.Equal(t, i, d.SortOrder, "sort_order is file position")
	}
	assert.Len(t, got[0].LunarMemorialDays, 4)
	assert.NotEmpty(t, got[0].Mantras.ShortSanskrit)
}

func TestParseDeities_Validation(t *testing.T) {
	tests := []struct {
		name, src, wantErr string
	}{
		{"malformed", `[{`, "decode deities"},
		{"empty", `[]`, "no deities"},
		{"empty id", `[{"id":"","type":"BUDDHA","name_vi":"x"}]`, "empty id"},
		{"duplicate id", `[{"id":"a","type":"BUDDHA","name_vi":"x"},{"id":"a","type":"BUDDHA","name_vi":"y"}]`, "duplicate id"},
		{"unknown type", `[{"id":"a","type":"ASURA","name_vi":"x"}]`, "unknown type"},
		{"empty name", `[{"id":"a","type":"BUDDHA","name_vi":""}]`, "empty name_vi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDeities([]byte(tt.src))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestParseDeities_OrderFollowsFile(t *testing.T) {
	got, err := parseDeities([]byte(`[
		{"id":"z","type":"BODHISATTVA","name_vi":"Z"},
		{"id":"a","type":"BUDDHA","name_vi":"A"}]`))
	require.NoError(t, err)
	assert.Equal(t, "z", got[0].ID)
	assert.Equal(t, domain.DeityBodhisattva, got[0].Type)
	assert.Equal(t, 1, got[1].SortOrder)
}

func TestParseDeities_AcceptsDeva(t *testing.T) {
	got, err := parseDeities([]byte(`[{"id":"d","type":"DEVA","name_vi":"D"}]`))
	require.NoError(t, err)
	assert.Equal(t, domain.DeityDeva, got[0].Type)
}
