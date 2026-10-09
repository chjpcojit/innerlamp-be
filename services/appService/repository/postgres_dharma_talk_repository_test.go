package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func mkTalk(id string, order int) domain.DharmaTalk {
	return domain.DharmaTalk{
		ID: id, SortOrder: order, Title: "Tiêu đề " + id, Speaker: "Thích A", SpeakerInitials: "TA",
		Category: "Nhân quả", Summary: "s", KeyPoints: []string{"k1", "k2"}, Quote: "q", QuoteSource: "qs",
		Tags: []string{"t1"}, Language: "vi", MediaType: domain.MediaText,
		SourceName: "src", SourceURL: "https://x", License: "l",
	}
}

func newDharmaTalkRepo(t *testing.T) domain.DharmaTalkRepository {
	t.Helper()
	return NewDharmaTalkRepository(connectTestDB(t))
}

func talkIDs(ts []domain.DharmaTalk) []string {
	out := make([]string, 0, len(ts))
	for _, tk := range ts {
		out = append(out, tk.ID)
	}
	return out
}

func TestDharmaTalkRepository_InsertAndFind_RoundTripsEveryMediaType(t *testing.T) {
	repo := newDharmaTalkRepo(t)
	ctx := context.Background()

	date := time.Date(2024, 8, 27, 0, 0, 0, 0, time.UTC)
	yt := mkTalk("yt", 0)
	yt.MediaType, yt.YouTubeID = domain.MediaYouTube, "sudE6IR71FU"
	yt.PublishedDate, yt.DurationMinutes, yt.Season = &date, 46, "Vu Lan"
	yt.Chapters = []domain.DharmaTalkChapter{{StartSeconds: 0, Title: "Mở đầu"}, {StartSeconds: 680, Title: "Câu chuyện"}}
	yt.FollowUpMeditation = &domain.FollowUpMeditation{Title: "Quán niệm ân cha mẹ", Minutes: 10, MeditationID: "m1"}
	au := mkTalk("au", 1)
	au.MediaType, au.AudioURL = domain.MediaAudio, "https://example.org/a.mp3"
	tx := mkTalk("tx", 2)

	n, err := repo.InsertMissing(ctx, []domain.DharmaTalk{yt, au, tx})
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	got, err := repo.FindByID(ctx, "yt")
	require.NoError(t, err)
	require.NotNil(t, got.PublishedDate)
	assert.Equal(t, "2024-08-27", got.PublishedDate.Format("2006-01-02"))
	assert.Equal(t, domain.MediaYouTube, got.MediaType)
	assert.Equal(t, "sudE6IR71FU", got.YouTubeID)
	assert.Empty(t, got.AudioURL)
	assert.Equal(t, 46, got.DurationMinutes)
	assert.Equal(t, "Vu Lan", got.Season)
	assert.Equal(t, yt.Chapters, got.Chapters)
	assert.Equal(t, yt.FollowUpMeditation, got.FollowUpMeditation)
	assert.Equal(t, []string{"k1", "k2"}, got.KeyPoints)

	got, err = repo.FindByID(ctx, "au")
	require.NoError(t, err)
	assert.Equal(t, domain.MediaAudio, got.MediaType)
	assert.Equal(t, "https://example.org/a.mp3", got.AudioURL)
	assert.Empty(t, got.YouTubeID)

	got, err = repo.FindByID(ctx, "tx")
	require.NoError(t, err)
	assert.Equal(t, domain.MediaText, got.MediaType)
	assert.Nil(t, got.PublishedDate, "NULL date")
	assert.Zero(t, got.DurationMinutes, "NULL duration")
	assert.Empty(t, got.Season)
	assert.Empty(t, got.Chapters)
	assert.Nil(t, got.FollowUpMeditation)
}

func TestDharmaTalkRepository_FindByID_NotFound(t *testing.T) {
	repo := newDharmaTalkRepo(t)
	_, err := repo.FindByID(context.Background(), "nope")
	assert.ErrorIs(t, err, domain.ErrDharmaTalkNotFound)
}

func TestDharmaTalkRepository_InsertMissing_DoesNotOverwrite(t *testing.T) {
	repo := newDharmaTalkRepo(t)
	ctx := context.Background()
	_, err := repo.InsertMissing(ctx, []domain.DharmaTalk{mkTalk("a", 0)})
	require.NoError(t, err)

	edited := mkTalk("a", 0)
	edited.Title = "ĐÃ SỬA"
	n, err := repo.InsertMissing(ctx, []domain.DharmaTalk{edited, mkTalk("b", 1)})
	require.NoError(t, err)
	assert.Equal(t, 1, n, "only the new id is inserted")

	got, err := repo.FindByID(ctx, "a")
	require.NoError(t, err)
	assert.Equal(t, "Tiêu đề a", got.Title)
}

func TestDharmaTalkRepository_InsertMissing_IsAtomic(t *testing.T) {
	repo := newDharmaTalkRepo(t)
	ctx := context.Background()

	bad := mkTalk("bad", 1)
	bad.MediaType = domain.MediaYouTube // no youtube_id: violates the media CHECK
	_, err := repo.InsertMissing(ctx, []domain.DharmaTalk{mkTalk("good", 0), bad})
	require.Error(t, err)

	n, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "the good row must roll back with the bad one")
}

func TestDharmaTalkRepository_CheckConstraintsRejectInconsistentMedia(t *testing.T) {
	repo := newDharmaTalkRepo(t)
	ctx := context.Background()

	tests := map[string]func(*domain.DharmaTalk){
		"youtube with audio url too": func(tk *domain.DharmaTalk) {
			tk.MediaType, tk.YouTubeID, tk.AudioURL = domain.MediaYouTube, "sudE6IR71FU", "https://x/a.mp3"
		},
		"audio without url":    func(tk *domain.DharmaTalk) { tk.MediaType = domain.MediaAudio },
		"text with youtube id": func(tk *domain.DharmaTalk) { tk.YouTubeID = "sudE6IR71FU" },
		"unknown media type":   func(tk *domain.DharmaTalk) { tk.MediaType = "podcast" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			tk := mkTalk("x", 0)
			mutate(&tk)
			_, err := repo.InsertMissing(ctx, []domain.DharmaTalk{tk})
			assert.Error(t, err)
		})
	}
}

func TestDharmaTalkRepository_List_OrderPagingAndTotal(t *testing.T) {
	repo := newDharmaTalkRepo(t)
	ctx := context.Background()

	var talks []domain.DharmaTalk
	for i := 0; i < 5; i++ {
		talks = append(talks, mkTalk(fmt.Sprintf("t%d", i), 4-i)) // file order is reversed vs ids
	}
	_, err := repo.InsertMissing(ctx, talks)
	require.NoError(t, err)

	page1, total, err := repo.List(ctx, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Equal(t, []string{"t4", "t3"}, talkIDs(page1), "ordered by sort_order ASC")

	page3, _, err := repo.List(ctx, 4, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"t0"}, talkIDs(page3))

	past, total, err := repo.List(ctx, 100, 2)
	require.NoError(t, err)
	assert.NotNil(t, past)
	assert.Empty(t, past)
	assert.Equal(t, 5, total)
}

func TestDharmaTalkRepository_List_TiebreaksOnID(t *testing.T) {
	repo := newDharmaTalkRepo(t)
	ctx := context.Background()
	_, err := repo.InsertMissing(ctx, []domain.DharmaTalk{mkTalk("b", 0), mkTalk("a", 0), mkTalk("c", 0)})
	require.NoError(t, err)

	got, _, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, talkIDs(got))
}

func TestDharmaTalkRepository_NeedsReviewIsHiddenFromReadsButCounted(t *testing.T) {
	repo := newDharmaTalkRepo(t)
	ctx := context.Background()

	hidden := mkTalk("hidden", 1)
	hidden.NeedsReview, hidden.ReviewNote = true, "yếu"
	_, err := repo.InsertMissing(ctx, []domain.DharmaTalk{mkTalk("shown", 0), hidden, mkTalk("shown2", 2)})
	require.NoError(t, err)

	items, total, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"shown", "shown2"}, talkIDs(items))
	assert.Equal(t, 2, total, "total counts displayable talks only")

	_, err = repo.FindByID(ctx, "hidden")
	assert.ErrorIs(t, err, domain.ErrDharmaTalkNotFound)

	all, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, all, "Count (seed emptiness) includes hidden rows")
}
