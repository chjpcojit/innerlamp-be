package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newDeityMux registers the routes the way cmd/main.go does, so path-value
// extraction and the /top vs /{id} precedence are exercised for real.
func newDeityMux(uc *mockDeityUsecase) *http.ServeMux {
	h := NewDeityHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/deities/top", h.Top)
	mux.HandleFunc("GET /v1/deities/{id}", h.Get)
	mux.HandleFunc("GET /v1/deities", h.ListDeities)
	return mux
}

func sampleDeity() domain.Deity {
	return domain.Deity{
		ID: "bodhisattva_avalokiteshvara", Type: domain.DeityBodhisattva, NameVI: "Quán Thế Âm Bồ Tát",
		NameSanskrit: "Avalokiteśvara", NameChinese: "觀世音菩薩", TitleHonorific: "Đại Từ Đại Bi",
		ThumbnailURL: "thumb.webp", ImageURL: "full.webp", CoreVirtue: "Từ bi", PureLand: "Phổ Đà",
		CanonicalSource:   "Kinh Pháp Hoa",
		LunarMemorialDays: []domain.MemorialDay{{LunarDay: 19, LunarMonth: 2, EventName: "Đản sanh"}},
		Iconography:       domain.Iconography{Posture: "Đứng", SacredArtifacts: []string{"Tịnh bình"}},
		GreatVows:         []string{"Cứu khổ"},
		Mantras:           domain.Mantras{ShortSanskrit: "Om Mani Padme Hum"},
	}
}

func TestDeityHandler_Top(t *testing.T) {
	uc := new(mockDeityUsecase)
	uc.On("Top", mock.Anything).Return([]domain.Deity{sampleDeity()}, nil)

	rec := doGet(newDeityMux(uc), "/v1/deities/top")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := decode(t, rec)
	assert.Equal(t, true, body["success"])
	items := body["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, "bodhisattva_avalokiteshvara", first["id"])
	assert.Equal(t, "BODHISATTVA", first["type"])
	assert.Equal(t, "Quán Thế Âm Bồ Tát", first["name_vi"])
	assert.NotContains(t, first, "biography_summary", "summary must stay light")
}

func TestDeityHandler_Top_ServerError(t *testing.T) {
	uc := new(mockDeityUsecase)
	uc.On("Top", mock.Anything).Return(nil, errors.New("db down"))
	rec := doGet(newDeityMux(uc), "/v1/deities/top")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "db down")
}

func TestDeityHandler_ListDeities(t *testing.T) {
	t.Run("defaults to page 1", func(t *testing.T) {
		uc := new(mockDeityUsecase)
		uc.On("ListDeities", mock.Anything, 1).Return(&domain.DeityPage{
			Items: []domain.Deity{sampleDeity()}, Page: 1, PageSize: 20, Total: 12, TotalPages: 1,
		}, nil)

		rec := doGet(newDeityMux(uc), "/v1/deities")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 1, data["page"])
		assert.EqualValues(t, 20, data["page_size"])
		assert.EqualValues(t, 12, data["total"])
		assert.EqualValues(t, 1, data["total_pages"])
		assert.Len(t, data["items"], 1)
	})

	t.Run("passes page through and empty page is []", func(t *testing.T) {
		uc := new(mockDeityUsecase)
		uc.On("ListDeities", mock.Anything, 5).Return(&domain.DeityPage{
			Items: []domain.Deity{}, Page: 5, PageSize: 20, Total: 12, TotalPages: 1,
		}, nil)
		rec := doGet(newDeityMux(uc), "/v1/deities?page=5")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, []any{}, decode(t, rec)["data"].(map[string]any)["items"])
	})

	t.Run("non-numeric page is 400 INVALID_PAGE", func(t *testing.T) {
		uc := new(mockDeityUsecase)
		rec := doGet(newDeityMux(uc), "/v1/deities?page=abc")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
		uc.AssertNotCalled(t, "ListDeities")
	})

	t.Run("usecase ErrInvalidPage is 400", func(t *testing.T) {
		uc := new(mockDeityUsecase)
		uc.On("ListDeities", mock.Anything, 0).Return(nil, domain.ErrInvalidPage)
		rec := doGet(newDeityMux(uc), "/v1/deities?page=0")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "INVALID_PAGE")
	})
}

func TestDeityHandler_Get(t *testing.T) {
	t.Run("returns the full detail", func(t *testing.T) {
		uc := new(mockDeityUsecase)
		d := sampleDeity()
		uc.On("Get", mock.Anything, d.ID).Return(&d, nil)

		rec := doGet(newDeityMux(uc), "/v1/deities/"+d.ID)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, d.ID, data["id"])
		assert.Equal(t, "Kinh Pháp Hoa", data["canonical_source"])
		days := data["lunar_memorial_days"].([]any)
		require.Len(t, days, 1)
		assert.Equal(t, "Đản sanh", days[0].(map[string]any)["event_name"])
		assert.Equal(t, []any{"Tịnh bình"}, data["iconography"].(map[string]any)["sacred_artifacts"])
		assert.Equal(t, []any{"Cứu khổ"}, data["great_vows"])
		assert.Equal(t, "Om Mani Padme Hum", data["mantras"].(map[string]any)["short_mantra_sanskrit"])
	})

	t.Run("nil slices marshal as []", func(t *testing.T) {
		uc := new(mockDeityUsecase)
		d := domain.Deity{ID: "x", Type: domain.DeityBuddha}
		uc.On("Get", mock.Anything, "x").Return(&d, nil)
		data := decode(t, doGet(newDeityMux(uc), "/v1/deities/x"))["data"].(map[string]any)
		assert.Equal(t, []any{}, data["great_vows"])
		assert.Equal(t, []any{}, data["lunar_memorial_days"])
		assert.Equal(t, []any{}, data["iconography"].(map[string]any)["sacred_artifacts"])
	})

	t.Run("unknown id is 404 DEITY_NOT_FOUND", func(t *testing.T) {
		uc := new(mockDeityUsecase)
		uc.On("Get", mock.Anything, "nope").Return(nil, domain.ErrDeityNotFound)
		rec := doGet(newDeityMux(uc), "/v1/deities/nope")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "DEITY_NOT_FOUND")
	})

	t.Run("top is not captured by {id}", func(t *testing.T) {
		uc := new(mockDeityUsecase)
		uc.On("Top", mock.Anything).Return([]domain.Deity{}, nil)
		rec := doGet(newDeityMux(uc), "/v1/deities/top")
		assert.Equal(t, http.StatusOK, rec.Code)
		uc.AssertNotCalled(t, "Get")
	})
}
