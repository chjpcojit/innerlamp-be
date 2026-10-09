package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newDharmaTalkMux registers the routes the way cmd/main.go does.
func newDharmaTalkMux(uc *mockDharmaTalkUsecase) *http.ServeMux {
	h := NewDharmaTalkHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/dharma-talks", h.List)
	mux.HandleFunc("GET /v1/dharma-talks/{id}", h.Get)
	return mux
}

func sampleDharmaTalk() domain.DharmaTalk {
	d := time.Date(2024, 8, 27, 0, 0, 0, 0, time.UTC)
	return domain.DharmaTalk{
		ID: "hieu-la-dao", Title: "Hiếu là đạo", Speaker: "Thích A", SpeakerInitials: "TA", Category: "Đời sống",
		Season: "Vu Lan", PublishedDate: &d, DurationMinutes: 46, Summary: "Tóm tắt", Tags: []string{"hiếu"},
		KeyPoints: []string{"ý 1", "ý 2"}, Quote: "câu", QuoteSource: "TA", Language: "vi",
		MediaType: domain.MediaYouTube, YouTubeID: "sudE6IR71FU",
		Chapters:           []domain.DharmaTalkChapter{{StartSeconds: 0, Title: "Mở đầu"}, {StartSeconds: 680, Title: "Câu chuyện"}},
		FollowUpMeditation: &domain.FollowUpMeditation{Title: "Quán niệm ân cha mẹ", Minutes: 10, MeditationID: "m1"},
		SourceName:         "niemphat.vn", SourceURL: "https://example.org/x", License: "l",
	}
}

func TestDharmaTalkHandler_List(t *testing.T) {
	t.Run("defaults to page 1 and returns the summary shape only", func(t *testing.T) {
		uc := new(mockDharmaTalkUsecase)
		uc.On("List", mock.Anything, 1).Return(&domain.DharmaTalkPage{
			Items: []domain.DharmaTalk{sampleDharmaTalk()}, Page: 1, PageSize: 10, Total: 36, TotalPages: 4,
		}, nil)

		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 1, data["page"])
		assert.EqualValues(t, 10, data["page_size"])
		assert.EqualValues(t, 36, data["total"])
		assert.EqualValues(t, 4, data["total_pages"])
		items := data["items"].([]any)
		require.Len(t, items, 1)
		first := items[0].(map[string]any)
		assert.Equal(t, "hieu-la-dao", first["id"])
		assert.Equal(t, "2024-08-27", first["published_date"])
		assert.EqualValues(t, 46, first["duration_minutes"])
		assert.Equal(t, "Vu Lan", first["season"])
		assert.Equal(t, "youtube", first["media_type"])
		for _, heavy := range []string{"key_points", "quote", "chapters", "youtube_id", "audio_url", "follow_up_meditation", "source_url", "license", "needs_review"} {
			assert.NotContains(t, first, heavy, "summary must stay light")
		}
	})

	t.Run("unknown values are JSON null, never zero or empty", func(t *testing.T) {
		tk := sampleDharmaTalk()
		tk.PublishedDate, tk.DurationMinutes, tk.Season, tk.Tags = nil, 0, "", nil
		uc := new(mockDharmaTalkUsecase)
		uc.On("List", mock.Anything, 1).Return(&domain.DharmaTalkPage{Items: []domain.DharmaTalk{tk}, Page: 1, PageSize: 10, Total: 1, TotalPages: 1}, nil)

		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks")

		first := decode(t, rec)["data"].(map[string]any)["items"].([]any)[0].(map[string]any)
		for _, k := range []string{"published_date", "duration_minutes", "season"} {
			assert.Contains(t, first, k)
			assert.Nil(t, first[k], k)
		}
		assert.Equal(t, []any{}, first["tags"], "tags is [] not null")
	})

	t.Run("an empty page is items [] not null", func(t *testing.T) {
		uc := new(mockDharmaTalkUsecase)
		uc.On("List", mock.Anything, 9).Return(&domain.DharmaTalkPage{Items: nil, Page: 9, PageSize: 10, Total: 36, TotalPages: 4}, nil)
		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks?page=9")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, []any{}, decode(t, rec)["data"].(map[string]any)["items"])
	})

	t.Run("an empty page= is page 1", func(t *testing.T) {
		uc := new(mockDharmaTalkUsecase)
		uc.On("List", mock.Anything, 1).Return(&domain.DharmaTalkPage{Items: []domain.DharmaTalk{}, Page: 1, PageSize: 10}, nil)
		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks?page=")
		require.Equal(t, http.StatusOK, rec.Code)
		uc.AssertExpectations(t)
	})

	t.Run("non-numeric page is 400 INVALID_PAGE with details.field", func(t *testing.T) {
		uc := new(mockDharmaTalkUsecase)
		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks?page=abc")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
		assert.Contains(t, rec.Body.String(), `"page"`)
		uc.AssertNotCalled(t, "List")
	})

	t.Run("use case ErrInvalidPage maps to 400", func(t *testing.T) {
		uc := new(mockDharmaTalkUsecase)
		uc.On("List", mock.Anything, 0).Return(nil, domain.ErrInvalidPage)
		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks?page=0")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
	})

	t.Run("infrastructure failure is 500 without leaking the cause", func(t *testing.T) {
		uc := new(mockDharmaTalkUsecase)
		uc.On("List", mock.Anything, 1).Return(nil, errors.New("db down"))
		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "db down")
	})
}

func TestDharmaTalkHandler_Get(t *testing.T) {
	t.Run("returns full detail for a youtube talk", func(t *testing.T) {
		uc := new(mockDharmaTalkUsecase)
		tk := sampleDharmaTalk()
		uc.On("Get", mock.Anything, "hieu-la-dao").Return(&tk, nil)

		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks/hieu-la-dao")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, "hieu-la-dao", data["id"])
		assert.Equal(t, "youtube", data["media_type"])
		assert.Equal(t, "sudE6IR71FU", data["youtube_id"], "id only, never the full URL")
		assert.Nil(t, data["audio_url"])
		assert.Contains(t, data, "audio_url")
		assert.Equal(t, []any{"ý 1", "ý 2"}, data["key_points"])
		assert.Equal(t, "câu", data["quote"])
		assert.Equal(t, "vi", data["language"])
		assert.Equal(t, "niemphat.vn", data["source_name"])
		assert.Equal(t, "https://example.org/x", data["source_url"])
		assert.Equal(t, "l", data["license"])
		assert.Equal(t, []any{
			map[string]any{"start_seconds": float64(0), "title": "Mở đầu"},
			map[string]any{"start_seconds": float64(680), "title": "Câu chuyện"},
		}, data["chapters"])
		assert.Equal(t, map[string]any{"title": "Quán niệm ân cha mẹ", "minutes": float64(10), "meditation_id": "m1"}, data["follow_up_meditation"])
		assert.NotContains(t, data, "needs_review")
		assert.NotContains(t, data, "review_note")
	})

	t.Run("audio talk has audio_url and no youtube_id", func(t *testing.T) {
		tk := sampleDharmaTalk()
		tk.MediaType, tk.YouTubeID, tk.AudioURL = domain.MediaAudio, "", "https://example.org/a.mp3"
		uc := new(mockDharmaTalkUsecase)
		uc.On("Get", mock.Anything, "a").Return(&tk, nil)
		data := decode(t, doGet(newDharmaTalkMux(uc), "/v1/dharma-talks/a"))["data"].(map[string]any)
		assert.Equal(t, "audio", data["media_type"])
		assert.Equal(t, "https://example.org/a.mp3", data["audio_url"])
		assert.Nil(t, data["youtube_id"])
		assert.Contains(t, data, "youtube_id")
	})

	t.Run("text talk has neither, and empty chapters and null follow-up", func(t *testing.T) {
		tk := sampleDharmaTalk()
		tk.MediaType, tk.YouTubeID = domain.MediaText, ""
		tk.Chapters, tk.FollowUpMeditation, tk.KeyPoints = nil, nil, nil
		uc := new(mockDharmaTalkUsecase)
		uc.On("Get", mock.Anything, "t").Return(&tk, nil)
		data := decode(t, doGet(newDharmaTalkMux(uc), "/v1/dharma-talks/t"))["data"].(map[string]any)
		assert.Equal(t, "text", data["media_type"])
		assert.Nil(t, data["youtube_id"])
		assert.Nil(t, data["audio_url"])
		assert.Equal(t, []any{}, data["chapters"], "chapters is [] not null so the client hides the section")
		assert.Equal(t, []any{}, data["key_points"])
		assert.Contains(t, data, "follow_up_meditation")
		assert.Nil(t, data["follow_up_meditation"])
	})

	t.Run("follow-up without a linked meditation has meditation_id null", func(t *testing.T) {
		tk := sampleDharmaTalk()
		tk.FollowUpMeditation = &domain.FollowUpMeditation{Title: "Ngồi thiền", Minutes: 10}
		uc := new(mockDharmaTalkUsecase)
		uc.On("Get", mock.Anything, "x").Return(&tk, nil)
		data := decode(t, doGet(newDharmaTalkMux(uc), "/v1/dharma-talks/x"))["data"].(map[string]any)
		f := data["follow_up_meditation"].(map[string]any)
		assert.Contains(t, f, "meditation_id")
		assert.Nil(t, f["meditation_id"])
	})

	t.Run("not found is 404 DHARMA_TALK_NOT_FOUND", func(t *testing.T) {
		uc := new(mockDharmaTalkUsecase)
		uc.On("Get", mock.Anything, "nope").Return(nil, domain.ErrDharmaTalkNotFound)
		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks/nope")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "DHARMA_TALK_NOT_FOUND")
	})

	t.Run("infrastructure failure is 500 without leaking the cause", func(t *testing.T) {
		uc := new(mockDharmaTalkUsecase)
		uc.On("Get", mock.Anything, "x").Return(nil, errors.New("db down"))
		rec := doGet(newDharmaTalkMux(uc), "/v1/dharma-talks/x")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "db down")
	})
}

// Both routes are GET-only: through the same fallback cmd/main.go wraps the mux in,
// any other method is the shared 405 envelope with Allow: GET, HEAD.
func TestDharmaTalkHandler_NonGetIsMethodNotAllowed(t *testing.T) {
	h := WithAPIFallback(newDharmaTalkMux(new(mockDharmaTalkUsecase)))
	for _, path := range []string{"/v1/dharma-talks", "/v1/dharma-talks/hieu-la-dao"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code, path)
		assert.Contains(t, rec.Body.String(), "METHOD_NOT_ALLOWED", path)
		assert.Equal(t, "GET, HEAD", rec.Header().Get("Allow"), path)
	}
}
