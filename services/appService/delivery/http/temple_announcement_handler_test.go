package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

const (
	annListPath     = "/v1/users/me/temple-announcements"
	annSettingsPath = "/v1/users/me/temple-announcement-settings"
)

func newAnnouncementMux(uc *mockTempleAnnouncementUsecase) *http.ServeMux {
	h := NewTempleAnnouncementHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+annListPath, h.List)
	mux.HandleFunc("GET "+annSettingsPath, h.GetSettings)
	mux.HandleFunc("PUT "+annSettingsPath, h.PutSettings)
	return mux
}

func TestTempleAnnouncementHandler_List(t *testing.T) {
	t.Run("maps the page to the response shape (TA1, TA9)", func(t *testing.T) {
		uc := new(mockTempleAnnouncementUsecase)
		uc.On("ListForUser", mock.Anything, int64(7), 2, 5).Return(domain.UserAnnouncements{
			Receive: true,
			Temple:  &domain.TempleRef{ID: "t1", Name: "Chùa A"},
			Page: domain.AnnouncementPage{
				Items: []domain.TempleAnnouncement{{
					ID: "tann_1", TempleID: "t1", Title: "Ghi danh", Body: "Còn 10 ngày",
					PublishedAt: time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC),
					ExpiresAt:   time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC),
					IssuingBody: domain.IssuingBodyRef{ID: "tib_1", Name: "Ban Trị sự"},
				}},
				Total: 6, Page: 2, PageSize: 5,
			},
		}, nil)

		rec := serve(newAnnouncementMux(uc), memorialReq(http.MethodGet, annListPath+"?page=2&page_size=5", "", "7"))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, true, data["receive"])
		assert.Equal(t, map[string]any{"id": "t1", "name": "Chùa A"}, data["temple"])
		assert.Equal(t, map[string]any{"page": float64(2), "page_size": float64(5), "total": float64(6)}, data["page"])
		items := data["items"].([]any)
		require.Len(t, items, 1)
		assert.Equal(t, map[string]any{
			"id": "tann_1", "title": "Ghi danh", "body": "Còn 10 ngày",
			"published_at": "2026-10-01T01:00:00Z", "expires_at": "2026-10-20T01:00:00Z",
			"issuing_body": map[string]any{"id": "tib_1", "name": "Ban Trị sự"},
		}, items[0])
		assert.NotContains(t, items[0], "temple_id", "no temple id on items")
		assert.NotContains(t, data, "unread_count", "TA11")
	})

	t.Run("no temple gives temple null and empty items (TA5)", func(t *testing.T) {
		uc := new(mockTempleAnnouncementUsecase)
		uc.On("ListForUser", mock.Anything, int64(7), 1, 0).Return(domain.UserAnnouncements{
			Receive: true, Page: domain.AnnouncementPage{Page: 1, PageSize: 20},
		}, nil)

		rec := serve(newAnnouncementMux(uc), memorialReq(http.MethodGet, annListPath, "", "7"))

		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.Nil(t, data["temple"])
		assert.Equal(t, []any{}, data["items"])
	})

	for name, q := range map[string]string{
		"page zero": "?page=0", "page negative": "?page=-1", "page text": "?page=abc",
		"page_size zero": "?page_size=0", "page_size text": "?page_size=x",
	} {
		t.Run(name+" is 400 INVALID_PAGE", func(t *testing.T) {
			uc := new(mockTempleAnnouncementUsecase)
			rec := serve(newAnnouncementMux(uc), memorialReq(http.MethodGet, annListPath+q, "", "7"))
			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "INVALID_PAGE", decode(t, rec)["error"].(map[string]any)["code"])
			uc.AssertNotCalled(t, "ListForUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}

	t.Run("a usecase error is mapped, not leaked", func(t *testing.T) {
		uc := new(mockTempleAnnouncementUsecase)
		uc.On("ListForUser", mock.Anything, int64(7), 1, 0).Return(domain.UserAnnouncements{}, assert.AnError)
		rec := serve(newAnnouncementMux(uc), memorialReq(http.MethodGet, annListPath, "", "7"))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestTempleAnnouncementHandler_Settings(t *testing.T) {
	t.Run("GET returns the stored value (TA16)", func(t *testing.T) {
		uc := new(mockTempleAnnouncementUsecase)
		uc.On("GetReceive", mock.Anything, int64(7)).Return(false, nil)
		rec := serve(newAnnouncementMux(uc), memorialReq(http.MethodGet, annSettingsPath, "", "7"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, map[string]any{"receive": false}, decode(t, rec)["data"])
	})

	t.Run("PUT stores and echoes (TA13)", func(t *testing.T) {
		uc := new(mockTempleAnnouncementUsecase)
		uc.On("SetReceive", mock.Anything, int64(7), false).Return(false, nil)
		rec := serve(newAnnouncementMux(uc), memorialReq(http.MethodPut, annSettingsPath, `{"receive":false}`, "7"))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, map[string]any{"receive": false}, decode(t, rec)["data"])
	})

	for name, body := range map[string]string{
		"missing receive": `{}`,
		"null receive":    `{"receive":null}`,
		"wrong type":      `{"receive":"yes"}`,
		"number":          `{"receive":1}`,
		"unknown key":     `{"receive":true,"temple_id":"x"}`,
		"not json":        `nope`,
		"empty body":      ``,
	} {
		t.Run(name+" is 400 and changes nothing (TA14)", func(t *testing.T) {
			uc := new(mockTempleAnnouncementUsecase)
			rec := serve(newAnnouncementMux(uc), memorialReq(http.MethodPut, annSettingsPath, body, "7"))
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.Equal(t, "VALIDATION_ERROR", decode(t, rec)["error"].(map[string]any)["code"])
			uc.AssertNotCalled(t, "SetReceive", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
