package e2e

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/assets"
)

// This file drives the real binary over HTTP against a real Postgres for the
// pháp thoại catalogue (docs/specs/features/dharmatalk). What it adds over the
// layer tests is what only the composition root can get wrong: that both routes
// are mounted, public and answering in the shared envelope; that the first-run
// seed really runs in the shipped binary and runs only once; and that a talk
// flagged needs_review is stored but never served.
//
// Expected values are NOT typed in here. They are computed from the shipped
// dharma_talks.json, read independently of the Go seed parser.

// ---- oracle: the shipped dharma_talks.json, read on its own -----------------

type talkOracle struct {
	ID          string  `json:"id"`
	MediaType   string  `json:"mediaType"`
	MediaURL    *string `json:"mediaUrl"`
	NeedsReview bool    `json:"needsReview"`
}

func loadTalkOracle(t *testing.T) (all, visible []talkOracle) {
	t.Helper()
	var doc struct {
		Talks []talkOracle `json:"talks"`
	}
	require.NoError(t, json.Unmarshal(assets.DharmaTalksJSON, &doc))
	require.NotEmpty(t, doc.Talks)
	for _, tk := range doc.Talks {
		if !tk.NeedsReview {
			visible = append(visible, tk)
		}
	}
	require.NotEmpty(t, visible, "the shipped data must have displayable talks")
	require.Less(t, len(visible), len(doc.Talks), "the shipped data should still contain flagged talks, or the hiding tests prove nothing")
	return doc.Talks, visible
}

var youtubeIDInURL = regexp.MustCompile(`(?:v=|youtu\.be/)([A-Za-z0-9_-]{11})`)

const talkPageSize = 10 // config.dev.yaml dharma_talk.page_size

func newTalkAPI(t *testing.T, baseURL string) *apiClient {
	t.Helper()
	return &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

// ---- the routes -------------------------------------------------------------

func TestDharmaTalkList_ServesOnlyDisplayableTalksInFileOrder_E2E(t *testing.T) {
	_, visible := loadTalkOracle(t)
	baseURL, _ := newE2EServer(t)
	api := newTalkAPI(t, baseURL)

	ids, _, total := walkPages(t, api, "/v1/dharma-talks", talkPageSize)

	want := make([]string, 0, len(visible))
	for _, tk := range visible {
		want = append(want, tk.ID)
	}
	assert.Equal(t, want, ids, "needs_review talks are hidden and the rest keep file order")
	assert.Equal(t, len(visible), total)
}

func TestDharmaTalkList_SummaryShapeAndDefaultPage_E2E(t *testing.T) {
	baseURL, _ := newE2EServer(t)
	api := newTalkAPI(t, baseURL)

	res := api.get("/v1/dharma-talks") // no Authorization: public
	require.Equal(t, http.StatusOK, res.status, res.raw)
	var p struct {
		Items []map[string]any `json:"items"`
		Page  int              `json:"page"`
	}
	res.data(&p)
	assert.Equal(t, 1, p.Page)
	require.Len(t, p.Items, talkPageSize)
	for _, it := range p.Items {
		for _, k := range []string{"id", "title", "speaker", "speaker_initials", "category", "season", "published_date", "duration_minutes", "summary", "tags", "media_type"} {
			assert.Contains(t, it, k)
		}
		for _, k := range []string{"key_points", "quote", "chapters", "youtube_id", "audio_url", "needs_review", "review_note"} {
			assert.NotContains(t, it, k, "list items stay light")
		}
		assert.Contains(t, []any{"youtube", "audio", "text"}, it["media_type"])
	}

	empty := api.get("/v1/dharma-talks?page=")
	assert.Equal(t, http.StatusOK, empty.status, "an empty page= is page 1")
}

func TestDharmaTalkDetail_MediaMatchesTheShippedData_E2E(t *testing.T) {
	_, visible := loadTalkOracle(t)
	baseURL, _ := newE2EServer(t)
	api := newTalkAPI(t, baseURL)

	sawType := map[string]bool{}
	for _, tk := range visible {
		res := api.get("/v1/dharma-talks/" + tk.ID)
		require.Equal(t, http.StatusOK, res.status, "%s: %s", tk.ID, res.raw)
		var d struct {
			MediaType string           `json:"media_type"`
			YouTubeID *string          `json:"youtube_id"`
			AudioURL  *string          `json:"audio_url"`
			Chapters  []map[string]any `json:"chapters"`
			Key       []string         `json:"key_points"`
		}
		res.data(&d)

		sawType[d.MediaType] = true
		assert.NotNil(t, d.Chapters, "%s: chapters is [] not null", tk.ID)
		switch tk.MediaType {
		case "video":
			require.Equal(t, "youtube", d.MediaType, tk.ID)
			require.NotNil(t, d.YouTubeID, tk.ID)
			require.NotNil(t, tk.MediaURL, tk.ID)
			m := youtubeIDInURL.FindStringSubmatch(*tk.MediaURL)
			require.NotNil(t, m, tk.ID)
			assert.Equal(t, m[1], *d.YouTubeID, "%s: id only, never the URL", tk.ID)
			assert.Nil(t, d.AudioURL, tk.ID)
		case "audio":
			require.Equal(t, "audio", d.MediaType, tk.ID)
			require.NotNil(t, d.AudioURL, tk.ID)
			assert.True(t, strings.HasPrefix(*d.AudioURL, "http"), tk.ID)
			assert.Nil(t, d.YouTubeID, tk.ID)
		case "text":
			require.Equal(t, "text", d.MediaType, tk.ID)
			assert.Nil(t, d.YouTubeID, tk.ID)
			assert.Nil(t, d.AudioURL, tk.ID)
		}
	}
	assert.NotEmpty(t, sawType, "at least one media type was exercised")
}

func TestDharmaTalkDetail_FlaggedAndUnknownAreNotFound_E2E(t *testing.T) {
	all, _ := loadTalkOracle(t)
	baseURL, _ := newE2EServer(t)
	api := newTalkAPI(t, baseURL)

	var flagged string
	for _, tk := range all {
		if tk.NeedsReview {
			flagged = tk.ID
			break
		}
	}
	require.NotEmpty(t, flagged)

	for _, id := range []string{flagged, "khong-ton-tai"} {
		res := api.get("/v1/dharma-talks/" + id)
		assert.Equal(t, http.StatusNotFound, res.status, id)
		assert.Equal(t, "DHARMA_TALK_NOT_FOUND", res.errorCode(), id)
	}
}

func TestDharmaTalkList_InvalidPageAndNonGet_E2E(t *testing.T) {
	baseURL, _ := newE2EServer(t)
	api := newTalkAPI(t, baseURL)

	for _, page := range []string{"0", "-1", "abc", "2147483648", "99999999999999999999"} {
		res := api.get("/v1/dharma-talks?page=" + page)
		assert.Equal(t, http.StatusBadRequest, res.status, page)
		assert.Equal(t, "INVALID_PAGE", res.errorCode(), page)
		assert.Equal(t, "page", res.details()["field"], page)
	}

	past := api.get("/v1/dharma-talks?page=9999")
	require.Equal(t, http.StatusOK, past.status, "a page past the end is empty, not an error")
	var p struct {
		Items []any `json:"items"`
	}
	past.data(&p)
	assert.Empty(t, p.Items)

	for _, path := range []string{"/v1/dharma-talks", "/v1/dharma-talks/x"} {
		res, hdr := api.request(http.MethodPost, path)
		assert.Equal(t, http.StatusMethodNotAllowed, res.status, path)
		assert.Equal(t, "METHOD_NOT_ALLOWED", res.errorCode(), path)
		assert.Equal(t, "GET, HEAD", hdr.Get("Allow"), path)
	}
}

// ---- the first-run seed -----------------------------------------------------

func TestDharmaTalkSeed_FirstRunStoresEveryTalkIncludingFlagged_E2E(t *testing.T) {
	requireE2E(t)
	all, visible := loadTalkOracle(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	_, stop := startServer(t, host, port, nil)
	t.Cleanup(stop)

	db := openDB(t, host, port)
	assert.Equal(t, len(all), countRows(t, db, "dharma_talks"), "flagged talks are stored, just not served")
	assert.Equal(t, len(visible), countRows(t, db, "dharma_talks WHERE needs_review = false"))
}

// A restart never overwrites an edit and never brings back a deleted talk.
func TestDharmaTalkSeed_SecondStartLeavesTheDatabaseAlone_E2E(t *testing.T) {
	requireE2E(t)
	_, visible := loadTalkOracle(t)
	require.GreaterOrEqual(t, len(visible), 2)
	edit, del := visible[0].ID, visible[1].ID
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)

	_, stop1 := startServer(t, host, port, nil)
	t.Cleanup(stop1)
	db := openDB(t, host, port)
	execSQL(t, db, `UPDATE dharma_talks SET title = $2 WHERE id = $1`, edit, "Đã sửa tay")
	execSQL(t, db, `DELETE FROM dharma_talks WHERE id = $1`, del)
	before := countRows(t, db, "dharma_talks")
	stop1()

	url2, stop2 := startServer(t, host, port, nil)
	t.Cleanup(stop2)
	api := newTalkAPI(t, url2)

	var got struct {
		Title string `json:"title"`
	}
	api.get("/v1/dharma-talks/" + edit).data(&got)
	assert.Equal(t, "Đã sửa tay", got.Title, "the seed must not overwrite an edit")
	assert.Equal(t, http.StatusNotFound, api.get("/v1/dharma-talks/"+del).status, "the seed must not bring a deleted talk back")
	assert.Equal(t, before, countRows(t, db, "dharma_talks"), "a second start writes no row")
}

// If every talk is under review the table is not empty: a restart must not
// reseed over the reviewers' flags.
func TestDharmaTalkSeed_AllFlaggedIsNotEmpty_E2E(t *testing.T) {
	requireE2E(t)
	all, _ := loadTalkOracle(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)

	_, stop1 := startServer(t, host, port, nil)
	t.Cleanup(stop1)
	db := openDB(t, host, port)
	execSQL(t, db, `UPDATE dharma_talks SET needs_review = true`)
	stop1()

	url2, stop2 := startServer(t, host, port, nil)
	t.Cleanup(stop2)
	assert.Equal(t, len(all), countRows(t, db, "dharma_talks"))
	assert.Zero(t, countRows(t, db, "dharma_talks WHERE needs_review = false"), "no flag was reset by a reseed")

	var p struct {
		Total int   `json:"total"`
		Items []any `json:"items"`
	}
	newTalkAPI(t, url2).get("/v1/dharma-talks").data(&p)
	assert.Zero(t, p.Total)
	assert.Empty(t, p.Items)
}
