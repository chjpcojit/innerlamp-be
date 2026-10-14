package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file drives the real binary for what only the composition root can get
// wrong about images: that /static/images/ is mounted from server.static_images_dir,
// that it serves files but never a folder listing or a dotfile, and that the
// thumbnail_url the articles API returns is a URL this very server can serve.

func get(t *testing.T, client *http.Client, rawURL string) (status int, contentType string, body []byte) {
	t.Helper()
	resp, err := client.Get(rawURL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, resp.Header.Get("Content-Type"), body
}

func TestStaticImages_ServesFilesButNotFoldersOrDotfiles_E2E(t *testing.T) {
	baseURL, client := newE2EServer(t)

	status, ct, body := get(t, client, baseURL+"/static/images/temples/full/temple_north_001.jpg")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "image/jpeg", ct)
	assert.NotEmpty(t, body)

	for name, path := range map[string]string{
		"root listing":   "/static/images/",
		"folder listing": "/static/images/temples/",
		"dotfile":        "/static/images/home-calendar/.gitignore",
		"missing file":   "/static/images/temples/full/nope.jpg",
	} {
		status, _, body := get(t, client, baseURL+path)
		assert.Equalf(t, http.StatusNotFound, status, "%s: %s", name, path)
		assert.NotContainsf(t, string(body), "temple_north_001", "%s must not list files", name)
	}
}

func TestArticleThumbnails_AreServedByTheSameServer_E2E(t *testing.T) {
	baseURL, client := newE2EServerSeeded(t)

	status, _, raw := get(t, client, baseURL+"/v1/articles/top")
	require.Equal(t, http.StatusOK, status)
	var top struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &top))
	require.NotEmpty(t, top.Data.Items)

	for _, item := range top.Data.Items {
		id := item["id"].(string)
		require.Contains(t, item, "thumbnail_url", "key is always present (%s)", id)
		thumb, ok := item["thumbnail_url"].(string)
		require.Truef(t, ok, "first-run seed gives every article a thumbnail_url, got %v (%s)", item["thumbnail_url"], id)
		assert.Truef(t, strings.HasSuffix(thumb, "/static/images/articles/thumb/"+id+".webp"), "%s: %s", id, thumb)

		// The seeded host is the dev machine's LAN IP, not this test server, so keep
		// only the path and ask the server under test.
		u, err := url.Parse(thumb)
		require.NoError(t, err)
		status, ct, body := get(t, client, baseURL+u.Path)
		assert.Equalf(t, http.StatusOK, status, "%s", id)
		assert.Equal(t, "image/webp", ct, id)
		assert.NotEmpty(t, body, id)
	}

	// The detail carries the same thumbnail as the summary.
	id := top.Data.Items[0]["id"].(string)
	status, _, raw = get(t, client, baseURL+"/v1/articles/"+id)
	require.Equal(t, http.StatusOK, status)
	var detail struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &detail))
	assert.Equal(t, top.Data.Items[0]["thumbnail_url"], detail.Data["thumbnail_url"])
}
