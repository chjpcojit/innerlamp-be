package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (e *tsEnv) seedTemple(id string) {
	e.dbExec(`INSERT INTO temples (id, name, type, region, sect, established_year, heritage_rank, heritage_priority,
		latitude, longitude, address) VALUES ($1, 'Chùa Test', 'CHUA', 'south', 'Bắc tông', 1990, 'DI_TICH_CAP_TINH', 1, 10.7, 106.6, 'x')`, id)
}

func TestTempleAnnouncement_E2E(t *testing.T) {
	e := newTempleSuggestionEnv(t)
	e.seedTemple("temple_south_900")
	e.seedTemple("temple_south_901")
	const temple = "/v1/admin/temples/temple_south_900"
	expires := time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)

	t.Run("no token is 401, viewer is 403 on every route and nothing is written", func(t *testing.T) {
		routes := []struct{ method, path string }{
			{http.MethodPost, temple + "/announcements"}, {http.MethodGet, temple + "/announcements"},
			{http.MethodPatch, "/v1/admin/temple-announcements/x"}, {http.MethodPost, "/v1/admin/temple-announcements/x/revoke"},
			{http.MethodPost, temple + "/issuing-bodies"}, {http.MethodGet, temple + "/issuing-bodies"},
		}
		for _, r := range routes {
			status, env := e.anon.do(t, r.method, r.path, `{}`)
			assert.Equal(t, http.StatusUnauthorized, status, "%s %s", r.method, r.path)
			assert.False(t, env.Success)
			status, env = e.viewer.do(t, r.method, r.path, `{}`)
			assert.Equal(t, http.StatusForbidden, status, "%s %s", r.method, r.path)
			assert.Equal(t, "FORBIDDEN", env.Error.Code)
		}
		status, env := e.admin.do(t, http.MethodGet, temple+"/announcements", nil)
		require.Equal(t, http.StatusOK, status)
		var p struct {
			Total int `json:"total"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &p))
		assert.Zero(t, p.Total)
	})

	// An issuing body per temple, created by content_editor (a reviewer role).
	status, env := e.editor.do(t, http.MethodPost, temple+"/issuing-bodies", map[string]string{"name": "Ban Trị sự"})
	require.Equal(t, http.StatusCreated, status, "%+v", env)
	var body struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &body))

	status, env = e.admin.do(t, http.MethodPost, temple+"/issuing-bodies", map[string]string{"name": "BAN TRỊ SỰ"})
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, "ISSUING_BODY_EXISTS", env.Error.Code)

	status, env = e.admin.do(t, http.MethodPost, "/v1/admin/temples/temple_south_901/issuing-bodies", map[string]string{"name": "Ban Khác"})
	require.Equal(t, http.StatusCreated, status)
	var otherBody struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &otherBody))

	status, env = e.admin.do(t, http.MethodPost, "/v1/admin/temples/temple_nope/issuing-bodies", map[string]string{"name": "x"})
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, "TEMPLE_NOT_FOUND", env.Error.Code)

	status, env = e.admin.do(t, http.MethodGet, temple+"/issuing-bodies", nil)
	require.Equal(t, http.StatusOK, status)
	var bodies struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &bodies))
	require.Len(t, bodies.Items, 1)

	// Publish: validation first.
	status, env = e.admin.do(t, http.MethodPost, temple+"/announcements",
		map[string]string{"title": "", "body": "b", "issuing_body_id": body.ID, "expires_at": expires})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "INVALID_ANNOUNCEMENT", env.Error.Code)
	assert.Equal(t, "title", env.Details["field"])

	status, env = e.admin.do(t, http.MethodPost, temple+"/announcements",
		map[string]string{"title": "t", "body": "b", "issuing_body_id": otherBody.ID, "expires_at": expires})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "INVALID_ISSUING_BODY", env.Error.Code)

	status, env = e.admin.do(t, http.MethodPost, temple+"/announcements",
		map[string]string{"title": "t", "body": "b", "issuing_body_id": body.ID,
			"expires_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "expires_at", env.Details["field"])

	status, env = e.admin.do(t, http.MethodPost, "/v1/admin/temples/temple_nope/announcements",
		map[string]string{"title": "t", "body": "b", "issuing_body_id": body.ID, "expires_at": expires})
	assert.Equal(t, http.StatusNotFound, status)

	// Publish -> edit -> revoke.
	status, env = e.editor.do(t, http.MethodPost, temple+"/announcements",
		map[string]string{"title": "Lễ Vu Lan", "body": "Nội dung", "issuing_body_id": body.ID, "expires_at": expires})
	require.Equal(t, http.StatusCreated, status, "%+v", env)
	var ann struct {
		ID          string  `json:"id"`
		Title       string  `json:"title"`
		State       string  `json:"state"`
		Source      string  `json:"source"`
		RevokedAt   *string `json:"revoked_at"`
		PublishedAt string  `json:"published_at"`
		IssuingBody struct {
			Name string `json:"name"`
		} `json:"issuing_body"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &ann))
	assert.Equal(t, "ACTIVE", ann.State)
	assert.Equal(t, "admin", ann.Source)
	assert.Equal(t, "Ban Trị sự", ann.IssuingBody.Name)

	status, env = e.admin.do(t, http.MethodPatch, "/v1/admin/temple-announcements/"+ann.ID, map[string]string{"title": "Lễ Vu Lan 2026"})
	require.Equal(t, http.StatusOK, status, "%+v", env)
	var edited struct {
		Title       string `json:"title"`
		PublishedAt string `json:"published_at"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &edited))
	assert.Equal(t, "Lễ Vu Lan 2026", edited.Title)
	assert.Equal(t, ann.PublishedAt, edited.PublishedAt, "published_at is kept")

	status, env = e.admin.do(t, http.MethodPatch, "/v1/admin/temple-announcements/"+ann.ID, `{"source":"system"}`)
	assert.Equal(t, http.StatusBadRequest, status, "unknown key is refused")
	assert.Equal(t, "VALIDATION_ERROR", env.Error.Code)

	status, env = e.admin.do(t, http.MethodPost, "/v1/admin/temple-announcements/"+ann.ID+"/revoke", nil)
	require.Equal(t, http.StatusOK, status)
	var first struct {
		State     string  `json:"state"`
		RevokedAt *string `json:"revoked_at"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &first))
	assert.Equal(t, "REVOKED", first.State)
	require.NotNil(t, first.RevokedAt)

	time.Sleep(10 * time.Millisecond)
	status, env = e.editor.do(t, http.MethodPost, "/v1/admin/temple-announcements/"+ann.ID+"/revoke", nil)
	require.Equal(t, http.StatusOK, status)
	var second struct {
		RevokedAt *string `json:"revoked_at"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &second))
	assert.Equal(t, *first.RevokedAt, *second.RevokedAt, "second revoke keeps the original revoked_at")

	status, env = e.admin.do(t, http.MethodPatch, "/v1/admin/temple-announcements/"+ann.ID, map[string]string{"title": "x"})
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, "ANNOUNCEMENT_REVOKED", env.Error.Code)

	status, env = e.admin.do(t, http.MethodPost, "/v1/admin/temple-announcements/tann_nope/revoke", nil)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, "ANNOUNCEMENT_NOT_FOUND", env.Error.Code)

	// The listing keeps the revoked one, with its derived state.
	status, env = e.admin.do(t, http.MethodGet, temple+"/announcements?page=1&page_size=10", nil)
	require.Equal(t, http.StatusOK, status)
	var list struct {
		Items []struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"items"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &list))
	require.Len(t, list.Items, 1)
	assert.Equal(t, ann.ID, list.Items[0].ID)
	assert.Equal(t, "REVOKED", list.Items[0].State)

	status, env = e.admin.do(t, http.MethodGet, temple+"/announcements?page=0", nil)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "INVALID_PAGE", env.Error.Code)
}
