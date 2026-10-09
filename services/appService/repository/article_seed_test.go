package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/assets"
)

func TestParseArticleSeed_EmbeddedData(t *testing.T) {
	got, err := ParseArticleSeed(assets.BuddhistArticlesJSON)

	require.NoError(t, err)
	require.Len(t, got, 60)
	assert.Equal(t, "vipassana-nghe-thuat-song", got[0].ID)
	assert.Nil(t, got[0].PublishedDate, "null in source stays nil")
	assert.NotEmpty(t, got[0].ImageURL)
	assert.NotEmpty(t, got[0].ThumbnailURL)
	assert.NotNil(t, got[2].PublishedDate)
	assert.Equal(t, "2013-01-01", got[2].PublishedDate.Format("2006-01-02"))
	for i, a := range got {
		assert.Equal(t, i, a.SortOrder, "sort_order is file position")
		assert.NotEmpty(t, a.Title)
		assert.True(t, strings.HasPrefix(a.ImageURL, "https://"), a.ID)
		// The thumbnail is hosted by this service: its URL ends in the file name below,
		// and that file must exist, or the client would get a 404 for it.
		assert.True(t, strings.HasSuffix(a.ThumbnailURL, "/static/images/articles/thumb/"+a.ID+".webp"), a.ID)
		_, err := os.Stat(filepath.Join("..", "..", "..", "assets", "images", "articles", "thumb", a.ID+".webp"))
		assert.NoError(t, err, "thumbnail file for %s", a.ID)
		assert.NotEmpty(t, a.BodyParagraphs, a.ID)
	}
}

func TestParseArticleSeed_Validation(t *testing.T) {
	tests := []struct{ name, src, wantErr string }{
		{"not json", `nope`, "decode"},
		{"empty", `{"articles":[]}`, "no articles"},
		{"empty id", `{"articles":[{"id":"","title":"t"}]}`, "empty id"},
		{"duplicate id", `{"articles":[{"id":"a","title":"t"},{"id":"a","title":"t"}]}`, "duplicate id"},
		{"empty title", `{"articles":[{"id":"a","title":" "}]}`, "empty title"},
		{"bad date", `{"articles":[{"id":"a","title":"t","publishedDate":"11/04/2019"}]}`, "publishedDate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseArticleSeed([]byte(tt.src))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
