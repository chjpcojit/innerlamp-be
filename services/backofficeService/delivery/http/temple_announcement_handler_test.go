package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/jwt/claims"
	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

type mockAnnUsecase struct{ mock.Mock }

func (m *mockAnnUsecase) Publish(ctx context.Context, templeID, adminID string, in domain.NewAnnouncement) (*domain.Announcement, error) {
	a := m.Called(ctx, templeID, adminID, in)
	if r := a.Get(0); r != nil {
		return r.(*domain.Announcement), a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockAnnUsecase) Edit(ctx context.Context, id string, p domain.AnnouncementPatch) (*domain.Announcement, error) {
	a := m.Called(ctx, id, p)
	if r := a.Get(0); r != nil {
		return r.(*domain.Announcement), a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockAnnUsecase) Revoke(ctx context.Context, id string) (*domain.Announcement, error) {
	a := m.Called(ctx, id)
	if r := a.Get(0); r != nil {
		return r.(*domain.Announcement), a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockAnnUsecase) List(ctx context.Context, templeID string, page, pageSize int) (*domain.AnnouncementPage, error) {
	a := m.Called(ctx, templeID, page, pageSize)
	if r := a.Get(0); r != nil {
		return r.(*domain.AnnouncementPage), a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockAnnUsecase) CreateIssuingBody(ctx context.Context, templeID, name string) (*domain.IssuingBody, error) {
	a := m.Called(ctx, templeID, name)
	if r := a.Get(0); r != nil {
		return r.(*domain.IssuingBody), a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockAnnUsecase) ListIssuingBodies(ctx context.Context, templeID string) ([]domain.IssuingBody, error) {
	a := m.Called(ctx, templeID)
	if r := a.Get(0); r != nil {
		return r.([]domain.IssuingBody), a.Error(1)
	}
	return nil, a.Error(1)
}

func serveAnn(uc *mockAnnUsecase, pattern, method, target, body string, pick func(*TempleAnnouncementHandler) http.HandlerFunc) *httptest.ResponseRecorder {
	h := NewTempleAnnouncementHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, pick(h))
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req = req.WithContext(jwtcontext.WithClaims(req.Context(), &claims.CustomClaims{UserID: "7", Role: "admin"}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

type annEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   struct {
		Code string `json:"code"`
	} `json:"error"`
	Details map[string]any `json:"details"`
}

func annDecode(t *testing.T, rec *httptest.ResponseRecorder) annEnvelope {
	t.Helper()
	var e annEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e), rec.Body.String())
	return e
}

var annT = time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)

func sampleAnn() *domain.Announcement {
	by := "7"
	return &domain.Announcement{
		ID: "tann_1", TempleID: "t", IssuingBodyID: "tib_1", IssuingBodyName: "Ban Trị sự", Title: "T", Body: "B",
		Source: "admin", PublishedAt: annT, ExpiresAt: annT.Add(time.Hour), CreatedBy: &by, State: domain.AnnouncementActive,
	}
}

func TestAnnouncementHandler_Publish(t *testing.T) {
	pub := func(h *TempleAnnouncementHandler) http.HandlerFunc { return h.Publish }
	const pattern = "POST /v1/admin/temples/{id}/announcements"
	const target = "/v1/admin/temples/t/announcements"

	t.Run("201 with admin id from the token", func(t *testing.T) {
		uc := &mockAnnUsecase{}
		exp := annT.Add(time.Hour)
		uc.On("Publish", mock.Anything, "t", "7", domain.NewAnnouncement{IssuingBodyID: "tib_1", Title: "T", Body: "B", ExpiresAt: exp}).Return(sampleAnn(), nil)
		rec := serveAnn(uc, pattern, http.MethodPost, target,
			`{"title":"T","body":"B","issuing_body_id":"tib_1","expires_at":"2026-10-14T10:00:00Z"}`, pub)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var got map[string]any
		require.NoError(t, json.Unmarshal(annDecode(t, rec).Data, &got))
		assert.Equal(t, "ACTIVE", got["state"])
		assert.Equal(t, "Ban Trị sự", got["issuing_body"].(map[string]any)["name"])
	})

	t.Run("unknown key is 400 VALIDATION_ERROR", func(t *testing.T) {
		rec := serveAnn(&mockAnnUsecase{}, pattern, http.MethodPost, target, `{"title":"T","source":"system"}`, pub)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "VALIDATION_ERROR", annDecode(t, rec).Error.Code)
	})
	t.Run("wrong type is 400", func(t *testing.T) {
		rec := serveAnn(&mockAnnUsecase{}, pattern, http.MethodPost, target, `{"title":5}`, pub)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("bad timestamp is INVALID_ANNOUNCEMENT with field", func(t *testing.T) {
		rec := serveAnn(&mockAnnUsecase{}, pattern, http.MethodPost, target, `{"expires_at":"tomorrow"}`, pub)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		e := annDecode(t, rec)
		assert.Equal(t, "INVALID_ANNOUNCEMENT", e.Error.Code)
		assert.Equal(t, "expires_at", e.Details["field"])
	})

	for name, tc := range map[string]struct {
		err    error
		status int
		code   string
	}{
		"invalid field":    {&domain.AnnouncementInputError{Field: "title", Message: "title is required"}, 400, "INVALID_ANNOUNCEMENT"},
		"invalid issuing":  {domain.ErrInvalidIssuingBody, 400, "INVALID_ISSUING_BODY"},
		"temple not found": {domain.ErrTempleNotFound, 404, "TEMPLE_NOT_FOUND"},
		"internal":         {errors.New("boom"), 500, "INTERNAL_ERROR"},
	} {
		t.Run(name, func(t *testing.T) {
			uc := &mockAnnUsecase{}
			uc.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, tc.err)
			rec := serveAnn(uc, pattern, http.MethodPost, target, `{}`, pub)
			assert.Equal(t, tc.status, rec.Code)
			e := annDecode(t, rec)
			assert.Equal(t, tc.code, e.Error.Code)
			if tc.code == "INVALID_ANNOUNCEMENT" {
				assert.Equal(t, "title", e.Details["field"])
			}
		})
	}
}

func TestAnnouncementHandler_Edit(t *testing.T) {
	edit := func(h *TempleAnnouncementHandler) http.HandlerFunc { return h.Edit }
	const pattern = "PATCH /v1/admin/temple-announcements/{id}"
	const target = "/v1/admin/temple-announcements/tann_1"
	title := "Mới"

	t.Run("200, only given fields", func(t *testing.T) {
		uc := &mockAnnUsecase{}
		uc.On("Edit", mock.Anything, "tann_1", domain.AnnouncementPatch{Title: &title}).Return(sampleAnn(), nil)
		rec := serveAnn(uc, pattern, http.MethodPatch, target, `{"title":"Mới"}`, edit)
		assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})
	t.Run("unknown key 400", func(t *testing.T) {
		rec := serveAnn(&mockAnnUsecase{}, pattern, http.MethodPatch, target, `{"published_at":"2026-01-01T00:00:00Z"}`, edit)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("revoked 409, not found 404", func(t *testing.T) {
		uc := &mockAnnUsecase{}
		uc.On("Edit", mock.Anything, "tann_1", mock.Anything).Return(nil, domain.ErrAnnouncementRevoked)
		uc.On("Edit", mock.Anything, "nope", mock.Anything).Return(nil, domain.ErrAnnouncementNotFound)
		rec := serveAnn(uc, pattern, http.MethodPatch, target, `{"title":"x"}`, edit)
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "ANNOUNCEMENT_REVOKED", annDecode(t, rec).Error.Code)
		rec = serveAnn(uc, pattern, http.MethodPatch, "/v1/admin/temple-announcements/nope", `{"title":"x"}`, edit)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "ANNOUNCEMENT_NOT_FOUND", annDecode(t, rec).Error.Code)
	})
}

func TestAnnouncementHandler_Revoke(t *testing.T) {
	rev := func(h *TempleAnnouncementHandler) http.HandlerFunc { return h.Revoke }
	const pattern = "POST /v1/admin/temple-announcements/{id}/revoke"
	uc := &mockAnnUsecase{}
	a := sampleAnn()
	a.RevokedAt, a.State = &annT, domain.AnnouncementRevoked
	uc.On("Revoke", mock.Anything, "tann_1").Return(a, nil)
	uc.On("Revoke", mock.Anything, "nope").Return(nil, domain.ErrAnnouncementNotFound)

	rec := serveAnn(uc, pattern, http.MethodPost, "/v1/admin/temple-announcements/tann_1/revoke", "", rev)
	require.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(annDecode(t, rec).Data, &got))
	assert.Equal(t, "REVOKED", got["state"])
	rec = serveAnn(uc, pattern, http.MethodPost, "/v1/admin/temple-announcements/nope/revoke", "", rev)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAnnouncementHandler_List(t *testing.T) {
	list := func(h *TempleAnnouncementHandler) http.HandlerFunc { return h.List }
	const pattern = "GET /v1/admin/temples/{id}/announcements"

	t.Run("passes page and page_size", func(t *testing.T) {
		uc := &mockAnnUsecase{}
		uc.On("List", mock.Anything, "t", 2, 5).Return(&domain.AnnouncementPage{
			Items: []domain.Announcement{*sampleAnn()}, Page: 2, PageSize: 5, Total: 6, TotalPages: 2,
		}, nil)
		rec := serveAnn(uc, pattern, http.MethodGet, "/v1/admin/temples/t/announcements?page=2&page_size=5", "", list)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var p struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		}
		require.NoError(t, json.Unmarshal(annDecode(t, rec).Data, &p))
		assert.Len(t, p.Items, 1)
		assert.Equal(t, 6, p.Total)
	})
	t.Run("defaults", func(t *testing.T) {
		uc := &mockAnnUsecase{}
		uc.On("List", mock.Anything, "t", 1, 0).Return(&domain.AnnouncementPage{Items: []domain.Announcement{}}, nil)
		rec := serveAnn(uc, pattern, http.MethodGet, "/v1/admin/temples/t/announcements", "", list)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("non-numeric page is 400 INVALID_PAGE", func(t *testing.T) {
		rec := serveAnn(&mockAnnUsecase{}, pattern, http.MethodGet, "/v1/admin/temples/t/announcements?page_size=abc", "", list)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "INVALID_PAGE", annDecode(t, rec).Error.Code)
	})
}

func TestAnnouncementHandler_IssuingBodies(t *testing.T) {
	create := func(h *TempleAnnouncementHandler) http.HandlerFunc { return h.CreateIssuingBody }
	list := func(h *TempleAnnouncementHandler) http.HandlerFunc { return h.ListIssuingBodies }

	t.Run("create 201", func(t *testing.T) {
		uc := &mockAnnUsecase{}
		uc.On("CreateIssuingBody", mock.Anything, "t", "Ban").Return(&domain.IssuingBody{ID: "tib_1", Name: "Ban", CreatedAt: annT}, nil)
		rec := serveAnn(uc, "POST /v1/admin/temples/{id}/issuing-bodies", http.MethodPost, "/v1/admin/temples/t/issuing-bodies", `{"name":"Ban"}`, create)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
	t.Run("create strict decode", func(t *testing.T) {
		rec := serveAnn(&mockAnnUsecase{}, "POST /v1/admin/temples/{id}/issuing-bodies", http.MethodPost, "/v1/admin/temples/t/issuing-bodies", `{"name":"Ban","x":1}`, create)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("create duplicate 409, blank 400 with field", func(t *testing.T) {
		uc := &mockAnnUsecase{}
		uc.On("CreateIssuingBody", mock.Anything, "t", "dup").Return(nil, domain.ErrIssuingBodyExists)
		uc.On("CreateIssuingBody", mock.Anything, "t", "").Return(nil,
			&domain.AnnouncementInputError{Field: "name", Message: "name is required", Kind: domain.ErrInvalidIssuingBodyName})
		rec := serveAnn(uc, "POST /v1/admin/temples/{id}/issuing-bodies", http.MethodPost, "/v1/admin/temples/t/issuing-bodies", `{"name":"dup"}`, create)
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "ISSUING_BODY_EXISTS", annDecode(t, rec).Error.Code)
		rec = serveAnn(uc, "POST /v1/admin/temples/{id}/issuing-bodies", http.MethodPost, "/v1/admin/temples/t/issuing-bodies", `{"name":""}`, create)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "name", annDecode(t, rec).Details["field"])
	})
	t.Run("list 200 and 404", func(t *testing.T) {
		uc := &mockAnnUsecase{}
		uc.On("ListIssuingBodies", mock.Anything, "t").Return([]domain.IssuingBody{{ID: "tib_1", Name: "Ban", CreatedAt: annT}}, nil)
		uc.On("ListIssuingBodies", mock.Anything, "gone").Return(nil, domain.ErrTempleNotFound)
		rec := serveAnn(uc, "GET /v1/admin/temples/{id}/issuing-bodies", http.MethodGet, "/v1/admin/temples/t/issuing-bodies", "", list)
		assert.Equal(t, http.StatusOK, rec.Code)
		rec = serveAnn(uc, "GET /v1/admin/temples/{id}/issuing-bodies", http.MethodGet, "/v1/admin/temples/gone/issuing-bodies", "", list)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "TEMPLE_NOT_FOUND", annDecode(t, rec).Error.Code)
	})
}
