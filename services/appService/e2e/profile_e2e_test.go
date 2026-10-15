package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type profileData struct {
	Email            string `json:"email"`
	EmailVerified    bool   `json:"email_verified"`
	PhoneNumber      string `json:"phone_number"`
	DharmaName       string `json:"dharma_name"`
	OrdinationTemple *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"ordination_temple"`
	AvatarURL string `json:"avatar_url"`
}

// rawKeys returns the keys present in a response's data object, which profileData
// cannot tell apart from a zero value.
func (r *apiResponse) rawKeys() map[string]any {
	r.t.Helper()
	var m map[string]any
	r.data(&m)
	return m
}

// TestProfileFlow_E2E drives docs/specs/features/profile against the real binary and
// a fresh Postgres.
//
// What is here and not in the per-layer tests: that cmd/main.go puts all five
// /v1/users/me routes behind the access-token middleware and the per-account limit,
// that a PATCH tells an absent key from null with the real JSON decoder and real
// rows, and that a rejected request writes nothing.
//
// The avatar flow runs against a fake-gcs-server container with UNSIGNED URLs (the
// dev setup). It proves our flow and key handling; it proves nothing about signing,
// size/type enforcement by the store, or URL expiry — that is TD-001, to be checked
// on a real bucket before deploy.
func TestProfileFlow_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	emulator := sharedStorageEmulator(t)
	// Its own bucket: user ids restart at 1 in every Postgres container, so a shared bucket
	// would let another test's objects show up under the same avatars/<id>/ prefix.
	const bucket = "avatars-e2e-profile"
	baseURL, stop := startServer(t, host, port, configOverrides{"profile": {"bucket": bucket}})
	t.Cleanup(stop)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	db := openDB(t, host, port)

	execSQL(t, db, `INSERT INTO temples (id, name, type, region, sect, established_year, heritage_rank, heritage_priority, latitude, longitude, address)
		VALUES ('temple_e2e_p1', 'Chùa Bích Động', 'CHUA', 'north', 'Thiền', 1000, 'CHUA_XEP_HANG', 7, 21.0, 105.8, 'Ninh Bình')`)

	routes := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/v1/users/me", nil},
		{http.MethodPatch, "/v1/users/me", map[string]any{}},
		{http.MethodPost, "/v1/users/me/avatar/upload-url", map[string]any{"content_type": "image/png", "size_bytes": 10}},
		{http.MethodPost, "/v1/users/me/avatar/confirm", map[string]any{"upload_id": "0b9c4f3e-5a1d-4c7e-9b2a-6d8e1f0a3c11"}},
		{http.MethodDelete, "/v1/users/me/avatar", nil},
	}

	t.Run("AC7/AC17/AC27: all five routes are behind the access-token middleware", func(t *testing.T) {
		api.t = t
		pair := api.login(api.registerUser(), deviceA)

		for _, rt := range routes {
			none := api.do(rt.method, rt.path, rt.body, "")
			assert.Equal(t, http.StatusUnauthorized, none.status, "%s %s without a token: %s", rt.method, rt.path, none.raw)
			assert.Equal(t, "UNAUTHORIZED", none.errorCode())
			assert.NotContains(t, none.raw, "phone_number", "a stranger learns nothing")

			garbage := api.do(rt.method, rt.path, rt.body, "not-a-jwt")
			assert.Equal(t, http.StatusUnauthorized, garbage.status, "%s %s with garbage", rt.method, rt.path)

			wrongType := api.do(rt.method, rt.path, rt.body, pair.RefreshToken)
			assert.Equal(t, http.StatusUnauthorized, wrongType.status, "%s %s with a refresh token", rt.method, rt.path)
		}
	})

	t.Run("AC1/AC2/AC3/AC6: a new account reads back only what it has", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()
		token := api.login(phone, deviceA).AccessToken

		res := api.do(http.MethodGet, "/v1/users/me", nil, token)

		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, "no-store", res.header.Get("Cache-Control"), "PII must not be cached")
		var p profileData
		res.data(&p)
		assert.Equal(t, phone, p.PhoneNumber, "AC4: the owner sees the full number")
		assert.NotEmpty(t, p.Email)
		assert.True(t, p.EmailVerified, "onboarding v8.0 AC14: registration verifies the email")
		keys := res.rawKeys()
		for _, absent := range []string{"dharma_name", "ordination_temple", "avatar_url", "avatar_url_expires_at"} {
			assert.NotContains(t, keys, absent, "AC2")
		}
		for _, internal := range []string{"id", "user_id", "password_hash", "avatar_object", "created_at", "updated_at"} {
			assert.NotContains(t, keys, internal, "AC6")
		}
	})

	t.Run("full_name: read after registration, editable, never clearable", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken

		got := api.do(http.MethodGet, "/v1/users/me", nil, token)
		require.Equal(t, http.StatusOK, got.status, got.raw)
		assert.Equal(t, "Nguyen Van An", got.rawKeys()["full_name"], "captured at registration")

		edit := api.do(http.MethodPatch, "/v1/users/me", map[string]any{"full_name": "  Trần Thị Bình  "}, token)
		require.Equal(t, http.StatusOK, edit.status, edit.raw)
		assert.Equal(t, "Trần Thị Bình", edit.rawKeys()["full_name"])

		for _, bad := range []any{nil, "   ", strings.Repeat("a", 301)} {
			res := api.do(http.MethodPatch, "/v1/users/me", map[string]any{"full_name": bad}, token)
			require.Equal(t, http.StatusBadRequest, res.status, res.raw)
			assert.Equal(t, "INVALID_FULL_NAME", res.errorCode())
		}
		kept := api.do(http.MethodGet, "/v1/users/me", nil, token)
		assert.Equal(t, "Trần Thị Bình", kept.rawKeys()["full_name"], "a rejected PATCH changes nothing")
	})

	t.Run("AC8/AC10/AC12/AC13: PATCH tells absent from null from a value", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken
		uid := userIDOf(t, token)

		set := api.do(http.MethodPatch, "/v1/users/me",
			map[string]any{"dharma_name": "  Thanh Liên  ", "ordination_temple_id": "temple_e2e_p1"}, token)
		require.Equal(t, http.StatusOK, set.status, set.raw)
		var p profileData
		set.data(&p)
		assert.Equal(t, "Thanh Liên", p.DharmaName, "trimmed")
		require.NotNil(t, p.OrdinationTemple)
		assert.Equal(t, "Chùa Bích Động", p.OrdinationTemple.Name)

		// Only the temple, null: the dharma name key is absent and must be kept.
		clearTemple := api.do(http.MethodPatch, "/v1/users/me", map[string]any{"ordination_temple_id": nil}, token)
		require.Equal(t, http.StatusOK, clearTemple.status, clearTemple.raw)
		keys := clearTemple.rawKeys()
		assert.Equal(t, "Thanh Liên", keys["dharma_name"], "an absent key keeps the value")
		assert.NotContains(t, keys, "ordination_temple")

		// {} changes nothing.
		noop := api.do(http.MethodPatch, "/v1/users/me", map[string]any{}, token)
		require.Equal(t, http.StatusOK, noop.status, noop.raw)
		assert.Equal(t, "Thanh Liên", noop.rawKeys()["dharma_name"])

		// A whitespace-only name clears it, stored as NULL and not ''.
		blank := api.do(http.MethodPatch, "/v1/users/me", map[string]any{"dharma_name": "   "}, token)
		require.Equal(t, http.StatusOK, blank.status, blank.raw)
		assert.NotContains(t, blank.rawKeys(), "dharma_name")
		var stored *string
		require.NoError(t, db.QueryRow(t.Context(), `SELECT dharma_name FROM users WHERE id = $1`, uid).Scan(&stored))
		assert.Nil(t, stored, "SQL NULL, not an empty string")
	})

	t.Run("AC9: 255 characters pass and 256 are a 400 INVALID_DHARMA_NAME", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken

		ok := api.do(http.MethodPatch, "/v1/users/me", map[string]any{"dharma_name": strings.Repeat("ệ", 255)}, token)
		require.Equal(t, http.StatusOK, ok.status, ok.raw)

		tooLong := api.do(http.MethodPatch, "/v1/users/me", map[string]any{"dharma_name": strings.Repeat("ệ", 256)}, token)
		assert.Equal(t, http.StatusBadRequest, tooLong.status, tooLong.raw)
		assert.Equal(t, "INVALID_DHARMA_NAME", tooLong.errorCode())
		assert.Equal(t, "dharma_name", tooLong.details()["field"])
	})

	t.Run("AC11/AC14: an unknown temple is refused and nothing else in the request is written", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken

		res := api.do(http.MethodPatch, "/v1/users/me",
			map[string]any{"dharma_name": "Không được lưu", "ordination_temple_id": "temple_does_not_exist"}, token)

		assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "INVALID_TEMPLE", res.errorCode())
		assert.Equal(t, "ordination_temple_id", res.details()["field"])

		after := api.do(http.MethodGet, "/v1/users/me", nil, token)
		assert.NotContains(t, after.rawKeys(), "dharma_name", "the valid dharma name of the same request was not applied")
	})

	t.Run("AC15: email, phone_number and unknown keys reject the whole request", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken

		for _, body := range []map[string]any{
			{"email": "new@example.com"},
			{"phone_number": "+84900000000"},
			{"dharma_name": "Không được lưu", "email": "new@example.com"},
			{"id": 1},
			{"avatar_url": "https://example.com/x.png"},
		} {
			res := api.do(http.MethodPatch, "/v1/users/me", body, token)
			assert.Equal(t, http.StatusBadRequest, res.status, "%v: %s", body, res.raw)
			assert.Equal(t, "VALIDATION_ERROR", res.errorCode())
		}
		after := api.do(http.MethodGet, "/v1/users/me", nil, token)
		assert.NotContains(t, after.rawKeys(), "dharma_name")
	})

	t.Run("AC19/AC20: upload-url rejects a bad type and size before signing anything", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken

		badType := api.do(http.MethodPost, "/v1/users/me/avatar/upload-url",
			map[string]any{"content_type": "image/gif", "size_bytes": 10}, token)
		assert.Equal(t, http.StatusBadRequest, badType.status, badType.raw)
		assert.Equal(t, "content_type", badType.details()["field"])

		tooBig := api.do(http.MethodPost, "/v1/users/me/avatar/upload-url",
			map[string]any{"content_type": "image/png", "size_bytes": 5*1024*1024 + 1}, token)
		assert.Equal(t, http.StatusBadRequest, tooBig.status, tooBig.raw)
		assert.Equal(t, "size_bytes", tooBig.details()["field"])
	})

	// ticket asks for an upload slot and uploads body to it exactly as the ticket says,
	// the way a client would. It returns the upload id.
	ticket := func(api *apiClient, token, contentType string, body []byte, sendType string) string {
		api.t.Helper()
		res := api.do(http.MethodPost, "/v1/users/me/avatar/upload-url",
			map[string]any{"content_type": contentType, "size_bytes": len(body)}, token)
		require.Equal(api.t, http.StatusOK, res.status, res.raw)
		var tk struct {
			UploadID  string            `json:"upload_id"`
			UploadURL string            `json:"upload_url"`
			Method    string            `json:"method"`
			Headers   map[string]string `json:"headers"`
		}
		res.data(&tk)
		require.NotEmpty(api.t, tk.UploadID)
		require.NotEmpty(api.t, tk.Method, "the ticket must say which method to use")

		req, err := http.NewRequest(tk.Method, tk.UploadURL, bytes.NewReader(body))
		require.NoError(api.t, err)
		for k, v := range tk.Headers {
			req.Header.Set(k, v)
		}
		if sendType != "" {
			req.Header.Set("Content-Type", sendType) // a client lying about the type
		}
		resp, err := http.DefaultClient.Do(req)
		require.NoError(api.t, err)
		defer func() { _ = resp.Body.Close() }()
		out, _ := io.ReadAll(resp.Body)
		require.Less(api.t, resp.StatusCode, 300, string(out))
		return tk.UploadID
	}
	fetch := func(t *testing.T, url string) (int, []byte) {
		t.Helper()
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, b
	}
	objectCount := func(t *testing.T, prefix string) int {
		t.Helper()
		resp, err := http.Get("http://" + emulator + "/storage/v1/b/" + bucket + "/o?prefix=" + prefix)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		var l struct {
			Items []struct{ Name string } `json:"items"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&l))
		return len(l.Items)
	}

	t.Run("AC18/AC22/AC25/AC26: upload, confirm, replace (old image removed), remove", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken
		uid := userIDOf(t, token)
		first, second := []byte("first image bytes"), []byte("the second, longer image bytes")

		id1 := ticket(api, token, "image/png", first, "")
		res := api.do(http.MethodPost, "/v1/users/me/avatar/confirm", map[string]any{"upload_id": id1}, token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var p1 profileData
		res.data(&p1)
		require.NotEmpty(t, p1.AvatarURL, "AC22: the confirmed profile carries avatar_url")
		assert.Contains(t, res.rawKeys(), "avatar_url_expires_at")
		assert.NotContains(t, res.raw, "avatar_object", "AC6: the object key never leaves the server")

		status, got := fetch(t, p1.AvatarURL)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, first, got, "the avatar address serves what was uploaded")
		var key string
		require.NoError(t, db.QueryRow(t.Context(), `SELECT avatar_object FROM users WHERE id = $1`, uid).Scan(&key))
		assert.True(t, strings.HasPrefix(key, "avatars/"+uid+"/"), "permanent folder, owner-scoped: %s", key)
		assert.False(t, strings.HasPrefix(key, "avatars/tmp/"))

		// A later read re-signs and still shows the avatar.
		again := api.do(http.MethodGet, "/v1/users/me", nil, token)
		require.Equal(t, http.StatusOK, again.status)
		assert.Contains(t, again.rawKeys(), "avatar_url")

		// Replace: new image becomes the avatar, the previous object is deleted (AC25).
		id2 := ticket(api, token, "image/png", second, "")
		res = api.do(http.MethodPost, "/v1/users/me/avatar/confirm", map[string]any{"upload_id": id2}, token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var p2 profileData
		res.data(&p2)
		assert.NotEqual(t, p1.AvatarURL, p2.AvatarURL)
		status, got = fetch(t, p2.AvatarURL)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, second, got)
		status, _ = fetch(t, p1.AvatarURL)
		assert.Equal(t, http.StatusNotFound, status, "the replaced image is gone from storage")
		assert.Equal(t, 1, objectCount(t, "avatars/"+uid+"/"), "exactly one permanent image per user")

		// Remove: row cleared, object deleted, 200 without avatar_url (AC26).
		del := api.do(http.MethodDelete, "/v1/users/me/avatar", nil, token)
		require.Equal(t, http.StatusOK, del.status, del.raw)
		assert.NotContains(t, del.rawKeys(), "avatar_url")
		status, _ = fetch(t, p2.AvatarURL)
		assert.Equal(t, http.StatusNotFound, status)
		assert.Equal(t, 0, objectCount(t, "avatars/"+uid+"/"))
		var stored *string
		require.NoError(t, db.QueryRow(t.Context(), `SELECT avatar_object FROM users WHERE id = $1`, uid).Scan(&stored))
		assert.Nil(t, stored)
	})

	t.Run("AC23: an id nobody uploaded, or another user's id, is 404; a malformed id is 400", func(t *testing.T) {
		api.t = t
		owner := api.login(api.registerUser(), deviceA).AccessToken
		other := api.login(api.registerUser(), deviceA).AccessToken
		id := ticket(api, owner, "image/png", []byte("owners image"), "")

		never := api.do(http.MethodPost, "/v1/users/me/avatar/confirm",
			map[string]any{"upload_id": "11111111-2222-4333-8444-555555555555"}, owner)
		assert.Equal(t, http.StatusNotFound, never.status, never.raw)
		assert.Equal(t, "AVATAR_UPLOAD_NOT_FOUND", never.errorCode())

		// The other user presents the owner's id: it resolves inside THEIR temp folder.
		stolen := api.do(http.MethodPost, "/v1/users/me/avatar/confirm", map[string]any{"upload_id": id}, other)
		assert.Equal(t, http.StatusNotFound, stolen.status, stolen.raw)
		after := api.do(http.MethodGet, "/v1/users/me", nil, other)
		assert.NotContains(t, after.rawKeys(), "avatar_url", "the thief got no avatar")

		malformed := api.do(http.MethodPost, "/v1/users/me/avatar/confirm", map[string]any{"upload_id": "../43/x"}, owner)
		assert.Equal(t, http.StatusBadRequest, malformed.status, malformed.raw)
		assert.Equal(t, "upload_id", malformed.details()["field"])
	})

	t.Run("AC24/AC25: a stored object of a disallowed type is INVALID_AVATAR and the avatar is untouched", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken
		good := ticket(api, token, "image/png", []byte("a fine image"), "")
		ok := api.do(http.MethodPost, "/v1/users/me/avatar/confirm", map[string]any{"upload_id": good}, token)
		require.Equal(t, http.StatusOK, ok.status, ok.raw)
		var before profileData
		ok.data(&before)

		// The ticket says image/png, the client uploads HTML. This store does not enforce the
		// signed type, so only the recorded-type check on confirm can stop it (AC24).
		bad := ticket(api, token, "image/png", []byte("<html>not an image</html>"), "text/html")
		res := api.do(http.MethodPost, "/v1/users/me/avatar/confirm", map[string]any{"upload_id": bad}, token)

		assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "INVALID_AVATAR", res.errorCode())
		status, _ := fetch(t, before.AvatarURL)
		assert.Equal(t, http.StatusOK, status, "the previous image is still there")
		now := api.do(http.MethodGet, "/v1/users/me", nil, token)
		var after profileData
		now.data(&after)
		assert.NotEmpty(t, after.AvatarURL)
	})

	t.Run("AC26: removing an avatar when there is none is 200 and unchanged", func(t *testing.T) {
		api.t = t
		token := api.login(api.registerUser(), deviceA).AccessToken

		res := api.do(http.MethodDelete, "/v1/users/me/avatar", nil, token)

		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.NotContains(t, res.rawKeys(), "avatar_url")
	})

	t.Run("AC28/AC29: the 11th request in a minute is 429, and another account is unaffected", func(t *testing.T) {
		api.t = t
		busy := api.login(api.registerUser(), deviceA).AccessToken
		other := api.login(api.registerUser(), deviceA).AccessToken

		for i := 1; i <= 10; i++ {
			res := api.do(http.MethodGet, "/v1/users/me", nil, busy)
			require.Equal(t, http.StatusOK, res.status, "request %d: %s", i, res.raw)
		}
		limited := api.do(http.MethodGet, "/v1/users/me", nil, busy)

		assert.Equal(t, http.StatusTooManyRequests, limited.status, limited.raw)
		assert.Equal(t, "RATE_LIMITED", limited.errorCode())
		assert.NotEmpty(t, limited.header.Get("Retry-After"))

		// Reads and edits share the budget: the same account's PATCH is limited too.
		patch := api.do(http.MethodPatch, "/v1/users/me", map[string]any{}, busy)
		assert.Equal(t, http.StatusTooManyRequests, patch.status)

		fresh := api.do(http.MethodGet, "/v1/users/me", nil, other)
		assert.Equal(t, http.StatusOK, fresh.status, "AC29: another account has its own budget")
	})
}
