package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func TestParseGallery_RealDataFile(t *testing.T) {
	src, err := os.ReadFile("../../../../assets/data/buddhist_gallery.json")
	require.NoError(t, err)

	got, err := parseGallery(src)

	require.NoError(t, err)
	require.Len(t, got, 86)
	counts := map[string]int{}
	for i, g := range got {
		assert.Equal(t, i, g.SortOrder, "sort_order is the position after interleaving")
		assert.NotEmpty(t, g.License, g.ID)
		counts[g.Type]++
	}
	assert.Equal(t, map[string]int{"CHUA": 41, "TUONG_PHAT": 21, "THIEN": 12, "LE_HOI": 12}, counts)

	// Top must not be all one type even though the file groups by type.
	for i, want := range domain.GalleryTypes {
		assert.Equal(t, want, got[i].Type, "first round interleaves every type")
	}
}

func TestParseGallery_Validation(t *testing.T) {
	const ok = `{"id":"g1","type":"CHUA","title":"T","thumbnail_url":"t","image_url":"i"}`
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"not json", `{`, "decode gallery"},
		{"empty list", `[]`, "no gallery images"},
		{"empty id", `[{"type":"CHUA","title":"T","thumbnail_url":"t","image_url":"i"}]`, "empty id"},
		{"duplicate id", `[` + ok + `,` + ok + `]`, "duplicate id"},
		{"unknown type", `[{"id":"g1","type":"X","title":"T","thumbnail_url":"t","image_url":"i"}]`, "unknown type"},
		{"empty title", `[{"id":"g1","type":"CHUA","thumbnail_url":"t","image_url":"i"}]`, "empty title"},
		{"empty image url", `[{"id":"g1","type":"CHUA","title":"T","thumbnail_url":"t"}]`, "empty image_url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseGallery([]byte(tt.src))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestParseGallery_InterleavesAndKeepsFileOrderWithinType(t *testing.T) {
	src := `[
	 {"id":"c1","type":"CHUA","title":"T","thumbnail_url":"t","image_url":"i"},
	 {"id":"c2","type":"CHUA","title":"T","thumbnail_url":"t","image_url":"i"},
	 {"id":"c3","type":"CHUA","title":"T","thumbnail_url":"t","image_url":"i"},
	 {"id":"l1","type":"LE_HOI","title":"T","thumbnail_url":"t","image_url":"i"}]`

	got, err := parseGallery([]byte(src))

	require.NoError(t, err)
	ids := make([]string, 0, len(got))
	for _, g := range got {
		ids = append(ids, g.ID)
	}
	assert.Equal(t, []string{"c1", "l1", "c2", "c3"}, ids)
}
