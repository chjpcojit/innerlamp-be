package repository

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/assets"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// talkJSON builds a one-talk seed document from a raw object body.
func talkJSON(fields string) []byte {
	return []byte(`{"talks":[{"id":"t1","title":"Tiêu đề","speaker":"Thích A",` + fields + `}]}`)
}

func TestParseDharmaTalkSeed_EmbeddedDataIsValid(t *testing.T) {
	talks, err := ParseDharmaTalkSeed(assets.DharmaTalksJSON)
	require.NoError(t, err)
	require.NotEmpty(t, talks)

	for i, tk := range talks {
		assert.Equal(t, i, tk.SortOrder, "sort_order is the position in the file (%s)", tk.ID)
		switch tk.MediaType {
		case domain.MediaYouTube:
			assert.Len(t, tk.YouTubeID, 11, tk.ID)
			assert.Empty(t, tk.AudioURL, tk.ID)
		case domain.MediaAudio:
			assert.True(t, strings.HasPrefix(tk.AudioURL, "http"), tk.ID)
			assert.Empty(t, tk.YouTubeID, tk.ID)
		case domain.MediaText:
			assert.Empty(t, tk.YouTubeID+tk.AudioURL, tk.ID)
		default:
			t.Errorf("%s: unexpected media type %q", tk.ID, tk.MediaType)
		}
	}
}

func TestParseDharmaTalkSeed_MapsMedia(t *testing.T) {
	tests := []struct {
		name      string
		fields    string
		wantType  domain.MediaType
		wantYT    string
		wantAudio string
	}{
		{"youtube watch url", `"mediaType":"video","mediaUrl":"https://www.youtube.com/watch?v=sudE6IR71FU"`, domain.MediaYouTube, "sudE6IR71FU", ""},
		{"youtube watch url with extra params", `"mediaType":"video","mediaUrl":"https://www.youtube.com/watch?v=Ue9ZSOTr7R8&t=30s"`, domain.MediaYouTube, "Ue9ZSOTr7R8", ""},
		{"youtu.be short url", `"mediaType":"video","mediaUrl":"https://youtu.be/gkKS6-5Nz8U"`, domain.MediaYouTube, "gkKS6-5Nz8U", ""},
		{"audio", `"mediaType":"audio","mediaUrl":"https://example.org/a.mp3"`, domain.MediaAudio, "", "https://example.org/a.mp3"},
		{"text ignores mediaUrl", `"mediaType":"text","mediaUrl":"https://example.org/page"`, domain.MediaText, "", ""},
		{"text without mediaUrl", `"mediaType":"text","mediaUrl":null`, domain.MediaText, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDharmaTalkSeed(talkJSON(tt.fields))
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, tt.wantType, got[0].MediaType)
			assert.Equal(t, tt.wantYT, got[0].YouTubeID)
			assert.Equal(t, tt.wantAudio, got[0].AudioURL)
		})
	}
}

func TestParseDharmaTalkSeed_MapsFields(t *testing.T) {
	got, err := ParseDharmaTalkSeed(talkJSON(`
		"speakerInitials":"TA","category":"Nhân quả","season":"Vu Lan","publishedDate":"2024-08-27",
		"durationMinutes":46,"summary":"s","keyPoints":["k1","k2"],"quote":"q","quoteSource":"qs",
		"tags":["hiếu"],"language":"vi","mediaType":"text","sourceName":"src","sourceUrl":"https://x","license":"l",
		"needsReview":true,"reviewNote":"yếu",
		"chapters":[{"startSeconds":0,"title":"Mở đầu"},{"startSeconds":680,"title":"Câu chuyện"}]`))
	require.NoError(t, err)
	tk := got[0]
	assert.Equal(t, "Vu Lan", tk.Season)
	require.NotNil(t, tk.PublishedDate)
	assert.Equal(t, "2024-08-27", tk.PublishedDate.Format("2006-01-02"))
	assert.Equal(t, 46, tk.DurationMinutes)
	assert.Equal(t, []string{"k1", "k2"}, tk.KeyPoints)
	assert.True(t, tk.NeedsReview)
	assert.Equal(t, "yếu", tk.ReviewNote)
	assert.Equal(t, []domain.DharmaTalkChapter{{StartSeconds: 0, Title: "Mở đầu"}, {StartSeconds: 680, Title: "Câu chuyện"}}, tk.Chapters)
	assert.Nil(t, tk.FollowUpMeditation, "the seed carries no follow-up meditation yet")
}

func TestParseDharmaTalkSeed_OptionalFieldsMayBeAbsent(t *testing.T) {
	got, err := ParseDharmaTalkSeed(talkJSON(`"mediaType":"text"`))
	require.NoError(t, err)
	tk := got[0]
	assert.Empty(t, tk.Season)
	assert.Nil(t, tk.PublishedDate)
	assert.Zero(t, tk.DurationMinutes)
	assert.Empty(t, tk.Chapters)
}

func TestParseDharmaTalkSeed_Rejects(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"not json", `{`},
		{"no talks", `{"talks":[]}`},
		{"empty id", `{"talks":[{"id":"","title":"t","speaker":"s","mediaType":"text"}]}`},
		{"duplicate id", `{"talks":[{"id":"a","title":"t","speaker":"s","mediaType":"text"},{"id":"a","title":"t","speaker":"s","mediaType":"text"}]}`},
		{"empty title", `{"talks":[{"id":"a","title":"  ","speaker":"s","mediaType":"text"}]}`},
		{"empty speaker", `{"talks":[{"id":"a","title":"t","speaker":"","mediaType":"text"}]}`},
		{"unknown media type", string(talkJSON(`"mediaType":"podcast"`))},
		{"missing media type", string(talkJSON(`"summary":"s"`))},
		{"video without url", string(talkJSON(`"mediaType":"video","mediaUrl":null`))},
		{"video with non-youtube url", string(talkJSON(`"mediaType":"video","mediaUrl":"https://vimeo.com/123"`))},
		{"youtube id wrong length", string(talkJSON(`"mediaType":"video","mediaUrl":"https://www.youtube.com/watch?v=short"`))},
		{"audio without url", string(talkJSON(`"mediaType":"audio","mediaUrl":null`))},
		{"audio with non-http url", string(talkJSON(`"mediaType":"audio","mediaUrl":"ftp://x/a.mp3"`))},
		{"zero duration", string(talkJSON(`"mediaType":"text","durationMinutes":0`))},
		{"negative duration", string(talkJSON(`"mediaType":"text","durationMinutes":-5`))},
		{"bad date", string(talkJSON(`"mediaType":"text","publishedDate":"27/08/2024"`))},
		{"negative chapter start", string(talkJSON(`"mediaType":"text","chapters":[{"startSeconds":-1,"title":"a"}]`))},
		{"empty chapter title", string(talkJSON(`"mediaType":"text","chapters":[{"startSeconds":0,"title":" "}]`))},
		{"chapters out of order", string(talkJSON(`"mediaType":"text","chapters":[{"startSeconds":60,"title":"a"},{"startSeconds":10,"title":"b"}]`))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDharmaTalkSeed([]byte(tt.src))
			assert.Error(t, err, fmt.Sprintf("got %+v", got))
		})
	}
}
