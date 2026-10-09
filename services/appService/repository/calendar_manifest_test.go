package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/assets"
)

func entry(file string) string {
	return `{"file":"` + file + `","author":"A","license":"CC0","license_url":"u","source_url":"s"}`
}

func TestParseCalendarManifest(t *testing.T) {
	t.Run("sorts by file name and derives the id", func(t *testing.T) {
		got, err := ParseCalendarManifest([]byte("[" + entry("b_2.jpg") + "," + entry("a_1.jpg") + "]"))

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "a_1", got[0].ID)
		assert.Equal(t, "a_1.jpg", got[0].File)
		assert.Equal(t, "b_2", got[1].ID)
		assert.Equal(t, "A", got[0].Author)
		assert.Equal(t, "CC0", got[0].License)
	})

	bad := map[string]string{
		"not json":          `{`,
		"empty list":        `[]`,
		"path traversal":    "[" + entry("../x.jpg") + "]",
		"subdirectory":      "[" + entry("a/b.jpg") + "]",
		"wrong extension":   "[" + entry("a.png") + "]",
		"empty file":        "[" + entry("") + "]",
		"duplicate":         "[" + entry("a.jpg") + "," + entry("a.jpg") + "]",
		"missing author":    `[{"file":"a.jpg","license":"CC0","source_url":"s"}]`,
		"missing license":   `[{"file":"a.jpg","author":"A","source_url":"s"}]`,
		"missing sourceurl": `[{"file":"a.jpg","author":"A","license":"CC0"}]`,
	}
	for name, raw := range bad {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := ParseCalendarManifest([]byte(raw))
			assert.Error(t, err)
		})
	}
}

// The embedded manifest is what the server really parses at startup: it must be
// valid and every file it lists must exist on disk, or image_url would 404.
func TestParseCalendarManifest_EmbeddedMatchesDisk(t *testing.T) {
	got, err := ParseCalendarManifest(assets.CalendarManifestJSON)
	require.NoError(t, err)
	require.Len(t, got, 100)

	dir := filepath.Join("..", "..", "..", "assets", "images", "home-calendar")
	for _, img := range got {
		_, err := os.Stat(filepath.Join(dir, img.File))
		assert.NoError(t, err, img.File)
	}
}
