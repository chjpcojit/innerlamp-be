package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/assets"
)

// This file drives the real binary over HTTP against a real Postgres for the sutra
// library (docs/specs/features/sutra). What it adds over the layer tests is what only
// the composition root can get wrong: that the four routes are mounted, public and
// answering in the shared envelope; that `/v1/deities/{id}/sutras` does not disturb
// the deity routes; that the first-run seed really runs in the shipped binary, runs
// only once, and stops the service — leaving nothing behind — when it cannot finish.
//
// Expected values are NOT typed in here. They are computed from the shipped
// index.json, read independently of the Go seed parser, so a filter, an ordering or
// a total that drifts from the data fails the test instead of being copied into it.

// ---- oracle: the shipped index.json, read on its own ------------------------

type indexEntry struct {
	ID              string `json:"id"`
	Tradition       string `json:"tradition"`
	Kind            string `json:"kind"`
	Collection      string `json:"collection"`
	DailyChant      bool   `json:"dailyChant"`
	Purposes        []string
	DurationMinutes *int `json:"durationMinutes"`
	Deities         []struct {
		DeityID string `json:"deity_id"`
		Role    string `json:"role"`
	} `json:"deities"`
	Translations []struct {
		Key       string `json:"key"`
		Lang      string `json:"lang"`
		Variant   string `json:"variant"`
		LineCount int    `json:"lineCount"`
	} `json:"translations"`
}

func loadSutraIndex(t *testing.T) []indexEntry {
	t.Helper()
	raw, err := fs.ReadFile(assets.SutrasFS, "data/sutras/index.json")
	require.NoError(t, err)
	var doc struct {
		Sutras []indexEntry `json:"sutras"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Sutras)
	return doc.Sutras
}

func idsWhere(idx []indexEntry, keep func(indexEntry) bool) []string {
	var out []string
	for _, e := range idx {
		if keep(e) {
			out = append(out, e.ID)
		}
	}
	return out
}

func roleRank(role string) int {
	switch role {
	case "SUBJECT":
		return 0
	case "SPEAKER":
		return 1
	}
	return 2
}

// byDeityOracle is the sutras linked to a deity (optionally with one role), ordered
// by the deity's role (SUBJECT, SPEAKER, MENTIONED) and then by curated order.
func byDeityOracle(idx []indexEntry, deity, role string) (ids, roles []string) {
	type hit struct {
		id, role string
		pos      int
	}
	var hits []hit
	for pos, e := range idx {
		for _, d := range e.Deities {
			if d.DeityID == deity && (role == "" || d.Role == role) {
				hits = append(hits, hit{e.ID, d.Role, pos})
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if ri, rj := roleRank(hits[i].role), roleRank(hits[j].role); ri != rj {
			return ri < rj
		}
		return hits[i].pos < hits[j].pos
	})
	for _, h := range hits {
		ids, roles = append(ids, h.id), append(roles, h.role)
	}
	return ids, roles
}

// ---- HTTP helpers -----------------------------------------------------------

// request is apiClient.post's sibling for bodiless requests; it also returns the
// response headers (the Allow header of a 405).
func (c *apiClient) request(method, path string) (*apiResponse, http.Header) {
	c.t.Helper()
	req, err := http.NewRequest(method, c.baseURL+path, nil)
	require.NoError(c.t, err)
	resp, err := c.http.Do(req)
	require.NoError(c.t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)

	res := &apiResponse{t: c.t, status: resp.StatusCode, contentType: resp.Header.Get("Content-Type"), raw: string(raw)}
	require.NoErrorf(c.t, json.Unmarshal(raw, &res.envelope), "%s %s %s: body is not the JSON envelope: %q", method, path, resp.Status, raw)
	return res, resp.Header
}

func (c *apiClient) get(path string) *apiResponse {
	c.t.Helper()
	res, _ := c.request(http.MethodGet, path)
	return res
}

// pagedItems is the part of a paged list response the walkers read.
type pagedItems struct {
	Items []struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	} `json:"items"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func withPage(path string, page int) string {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%spage=%d", path, sep, page)
}

// walkPages reads a paged list from page 1 to its last page, checking the paging
// metadata on the way, and returns every id (and role, for the by-deity list) in the
// order served plus the reported total. It also asks for the page after the last and
// requires an empty page, not an error.
func walkPages(t *testing.T, api *apiClient, path string, wantPageSize int) (ids, roles []string, total int) {
	t.Helper()
	for page := 1; ; page++ {
		require.LessOrEqual(t, page, 100, "runaway paging on %s", path)
		res := api.get(withPage(path, page))
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var p pagedItems
		res.data(&p)

		require.Equal(t, page, p.Page)
		require.Equal(t, wantPageSize, p.PageSize)
		require.Equal(t, (p.Total+wantPageSize-1)/wantPageSize, p.TotalPages, "total_pages must be ceil(total/page_size) on %s", path)
		total = p.Total
		if page > p.TotalPages {
			assert.Empty(t, p.Items, "a page past the end is empty, not an error (%s page %d)", path, page)
			require.NotNil(t, p.Items, "items must be [] not null")
			break
		}
		if page < p.TotalPages {
			require.Len(t, p.Items, wantPageSize, "every page but the last is full (%s page %d)", path, page)
		}
		for _, it := range p.Items {
			ids, roles = append(ids, it.ID), append(roles, it.Role)
		}
	}
	require.Len(t, ids, total, "the pages must add up to the reported total on %s", path)
	return ids, roles, total
}

func sutraAPI(t *testing.T) *apiClient {
	baseURL, client := newE2EServerSeeded(t)
	return &apiClient{t: t, baseURL: baseURL, http: client}
}

// ---- the read API -----------------------------------------------------------

func TestSutraAPI_E2E(t *testing.T) {
	idx := loadSutraIndex(t)
	api := sutraAPI(t)
	const pageSize = 20 // sutra.page_size in config.dev.yaml

	t.Run("AC1/AC5 the list is public, complete, in curated order, and pages add up", func(t *testing.T) {
		api.t = t
		ids, _, total := walkPages(t, api, "/v1/sutras", pageSize)
		assert.Equal(t, len(idx), total)
		assert.Equal(t, idsWhere(idx, func(indexEntry) bool { return true }), ids, "sort_order ASC across every page, no repeat, no skip")

		// the first-run seed really ran in the shipped binary: nothing else puts rows there
		last := api.get(withPage("/v1/sutras", (total+pageSize-1)/pageSize))
		var p pagedItems
		last.data(&p)
		assert.Len(t, p.Items, total-pageSize*((total+pageSize-1)/pageSize-1))
	})

	t.Run("AC2/AC7/AC25/AC26/AC27 summary shape", func(t *testing.T) {
		api.t = t
		res := api.get("/v1/sutras")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Contains(t, res.contentType, "application/json")
		var data struct {
			Items []map[string]any `json:"items"`
		}
		res.data(&data)
		require.NotEmpty(t, data.Items)

		wantKeys := []string{"audio_url", "collection", "daily_chant", "deities", "duration_minutes", "id", "kind", "purposes", "reference", "summary", "titles", "tradition", "translations"}
		forbidden := []string{"review_status", "sections", "source_url", "source_name", "note", "lines"}
		for _, item := range data.Items {
			var keys []string
			for k := range item {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			assert.Equal(t, wantKeys, keys, "summary of %v", item["id"])
			for _, k := range forbidden {
				assert.NotContains(t, item, k)
			}
			titles, _ := item["titles"].(map[string]any)
			assert.NotEmpty(t, titles["vi"], "titles always holds vi (%v)", item["id"])
			for _, arr := range []string{"purposes", "deities", "translations"} {
				assert.NotNil(t, item[arr], "%s must be [] not null (%v)", arr, item["id"])
			}
			for _, tr := range item["translations"].([]any) {
				for _, k := range forbidden {
					assert.NotContains(t, tr, k, "a translation in the list carries no sources or sections")
				}
			}
		}

		// every text carries a duration estimate (migrations/0019 backfills the old
		// NULLs) and audio_url is "" — a string, never null, never omitted — until a
		// recording exists
		for _, item := range data.Items {
			assert.NotNil(t, item["duration_minutes"], "no null duration (%v)", item["id"])
			assert.Equal(t, "", item["audio_url"], "no recording yet (%v)", item["id"])
		}
		for _, e := range idx {
			assert.NotNil(t, e.DurationMinutes, "seed %s has an estimate", e.ID)
		}
		d := api.get("/v1/sutras/kinh-hanh-phuc")
		var one map[string]any
		d.data(&one)
		assert.Equal(t, "", one["audio_url"], "the detail carries audio_url too")
		assert.NotNil(t, one["duration_minutes"])
	})

	t.Run("search q ignores case and Vietnamese diacritics, combines with filters", func(t *testing.T) {
		api.t = t
		type page struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
			Total int `json:"total"`
		}
		ids := func(p page) []string {
			var out []string
			for _, i := range p.Items {
				out = append(out, i.ID)
			}
			return out
		}

		var hit page
		api.get("/v1/sutras?q=" + url.QueryEscape("KINH HANH PHUC")).data(&hit)
		assert.Contains(t, ids(hit), "kinh-hanh-phuc", "upper case, no accents")
		assert.Equal(t, hit.Total, len(hit.Items), "total counts the filtered result")

		var accented page
		api.get("/v1/sutras?q=" + url.QueryEscape("Hạnh Phúc")).data(&accented)
		assert.Equal(t, ids(hit), ids(accented), "accents or not, same result")

		var byEnglish page
		api.get("/v1/sutras?q=true+blessings").data(&byEnglish)
		assert.Contains(t, ids(byEnglish), "kinh-hanh-phuc", "a title in any language is searched")

		var none page
		api.get("/v1/sutras?q=zzzkhongco").data(&none)
		assert.Equal(t, 0, none.Total)
		assert.Empty(t, none.Items)

		var excluded page
		api.get("/v1/sutras?q=" + url.QueryEscape("hạnh phúc") + "&tradition=VAJRAYANA").data(&excluded)
		assert.NotContains(t, ids(excluded), "kinh-hanh-phuc", "q is ANDed with the other filters")

		for _, bad := range []string{"q=", "q=+++"} {
			res := api.get("/v1/sutras?" + bad)
			assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
			assert.Contains(t, res.raw, "INVALID_FILTER")
		}
	})

	t.Run("AC3 every filter, and their combination, against the data", func(t *testing.T) {
		api.t = t
		tests := []struct {
			name  string
			query string
			keep  func(indexEntry) bool
		}{
			{"tradition", "tradition=VAJRAYANA", func(e indexEntry) bool { return e.Tradition == "VAJRAYANA" }},
			{"kind", "kind=DHARANI", func(e indexEntry) bool { return e.Kind == "DHARANI" }},
			{"collection is an exact match", "collection=" + url.QueryEscape("Thập Chú"), func(e indexEntry) bool { return e.Collection == "Thập Chú" }},
			{"purpose", "purpose=TRI_CHU", func(e indexEntry) bool { return slices.Contains(e.Purposes, "TRI_CHU") }},
			{"daily true", "daily=true", func(e indexEntry) bool { return e.DailyChant }},
			{"daily false", "daily=false", func(e indexEntry) bool { return !e.DailyChant }},
			{"max_minutes drops a text with no estimate", "max_minutes=5", func(e indexEntry) bool { return e.DurationMinutes != nil && *e.DurationMinutes <= 5 }},
			{"lang", "lang=bo", func(e indexEntry) bool {
				for _, tr := range e.Translations {
					if tr.Lang == "bo" {
						return true
					}
				}
				return false
			}},
			{"all filters at once", "tradition=MAHAYANA&kind=SUTRA&daily=true&max_minutes=60&lang=vi&purpose=HOC_GIAO_LY", func(e indexEntry) bool {
				if e.Tradition != "MAHAYANA" || e.Kind != "SUTRA" || !e.DailyChant || e.DurationMinutes == nil || *e.DurationMinutes > 60 || !slices.Contains(e.Purposes, "HOC_GIAO_LY") {
					return false
				}
				for _, tr := range e.Translations {
					if tr.Lang == "vi" {
						return true
					}
				}
				return false
			}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				api.t = t
				want := idsWhere(idx, tt.keep)
				ids, _, total := walkPages(t, api, "/v1/sutras?"+tt.query, pageSize)
				assert.Equal(t, len(want), total, "total counts only what matches")
				assert.Equal(t, want, ids)
			})
		}

		t.Run("filters that match nothing give an empty page, not an error", func(t *testing.T) {
			api.t = t
			res := api.get("/v1/sutras?tradition=VAJRAYANA&daily=true")
			require.Equal(t, http.StatusOK, res.status, res.raw)
			var p pagedItems
			res.data(&p)
			assert.Empty(t, p.Items)
			assert.NotNil(t, p.Items)
			assert.Equal(t, 0, p.Total)
			assert.Equal(t, 0, p.TotalPages)
		})
	})

	t.Run("AC4/AC6 a bad filter or page is a 400 that names the parameter", func(t *testing.T) {
		api.t = t
		tests := []struct {
			query     string
			wantCode  string
			wantField string
		}{
			{"tradition=ZEN", "INVALID_FILTER", "tradition"},
			{"tradition=", "INVALID_FILTER", "tradition"},
			{"kind=POEM", "INVALID_FILTER", "kind"},
			{"purpose=LUCK", "INVALID_FILTER", "purpose"},
			{"daily=yes", "INVALID_FILTER", "daily"},
			{"max_minutes=0", "INVALID_FILTER", "max_minutes"},
			{"max_minutes=abc", "INVALID_FILTER", "max_minutes"},
			{"lang=", "INVALID_FILTER", "lang"},
			{"collection=", "INVALID_FILTER", "collection"},
			{"page=0", "INVALID_PAGE", "page"},
			{"page=-1", "INVALID_PAGE", "page"},
			{"page=abc", "INVALID_PAGE", "page"},
			{"page=2147483648", "INVALID_PAGE", "page"},
		}
		for _, tt := range tests {
			res := api.get("/v1/sutras?" + tt.query)
			assert.Equal(t, http.StatusBadRequest, res.status, "%s: %s", tt.query, res.raw)
			assert.False(t, res.envelope.Success)
			assert.Equal(t, tt.wantCode, res.errorCode(), tt.query)
			assert.Equal(t, tt.wantField, res.details()["field"], tt.query)
		}
	})

	t.Run("AC8/AC9/AC11 detail: translations picked by key, sources, review status", func(t *testing.T) {
		api.t = t
		res := api.get("/v1/sutras/tam-kinh")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var d struct {
			ID           string   `json:"id"`
			ReviewStatus string   `json:"review_status"`
			Duration     *int     `json:"duration_minutes"`
			Purposes     []string `json:"purposes"`
			Deities      []struct {
				DeityID string `json:"deity_id"`
				Role    string `json:"role"`
				NameVI  string `json:"name_vi"`
			} `json:"deities"`
			Translations []struct {
				Key        string           `json:"key"`
				Lang       string           `json:"lang"`
				Variant    string           `json:"variant"`
				LineCount  int              `json:"line_count"`
				SourceName string           `json:"source_name"`
				SourceURL  string           `json:"source_url"`
				Sections   []map[string]any `json:"sections"`
			} `json:"translations"`
		}
		res.data(&d)
		assert.Equal(t, "tam-kinh", d.ID)
		assert.Equal(t, "UNREVIEWED", d.ReviewStatus)

		var want indexEntry
		for _, e := range idx {
			if e.ID == "tam-kinh" {
				want = e
			}
		}
		require.NotNil(t, want.DurationMinutes)
		require.NotNil(t, d.Duration)
		assert.Equal(t, *want.DurationMinutes, *d.Duration, "the stored estimate is returned unchanged")
		assert.Equal(t, want.Purposes, d.Purposes)

		var keys, wantKeys []string
		for i, tr := range d.Translations {
			keys = append(keys, tr.Key)
			assert.NotEmpty(t, tr.SourceName, "detail carries the attribution (%s)", tr.Key)
			assert.NotEmpty(t, tr.SourceURL)
			assert.NotNil(t, tr.Sections, "sections is [] not null (%s)", tr.Key)
			assert.Equal(t, want.Translations[i].LineCount, tr.LineCount, tr.Key)
		}
		for _, tr := range want.Translations {
			wantKeys = append(wantKeys, tr.Key)
		}
		assert.Equal(t, wantKeys, keys, "translations come back in file order")
		assert.Contains(t, keys, "vi-meaning")
		assert.Contains(t, keys, "vi-meaning-2", "two vi texts of the same variant are told apart by key")

		require.Len(t, d.Deities, 1)
		assert.Equal(t, "bodhisattva_avalokiteshvara", d.Deities[0].DeityID)
		assert.Equal(t, "SPEAKER", d.Deities[0].Role)
		assert.NotEmpty(t, d.Deities[0].NameVI, "the card needs no second request to name the deity")
	})

	t.Run("AC36-AC38 detail returns the split fields; the list does not", func(t *testing.T) {
		api.t = t
		res := api.get("/v1/sutras/a-di-da")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var d struct {
			Reference          string `json:"reference"`
			OriginalTitle      string `json:"original_title"`
			Author             string `json:"author"`
			HanTranslator      string `json:"han_translator"`
			HanTranslationYear *int   `json:"han_translation_year"`
			ReferenceNote      string `json:"reference_note"`
			Summary            string `json:"summary"`
			Translations       []struct {
				License     string `json:"license"`
				LicenseURL  string `json:"license_url"`
				LicenseNote string `json:"license_note"`
			} `json:"translations"`
		}
		res.data(&d)
		assert.Equal(t, "Taishō T0366", d.Reference, "reference is only the code")
		assert.Equal(t, "Cưu Ma La Thập", d.HanTranslator)
		require.NotNil(t, d.HanTranslationYear)
		assert.Equal(t, 402, *d.HanTranslationYear)
		assert.Empty(t, d.OriginalTitle)
		assert.Empty(t, d.Author)
		assert.Empty(t, d.ReferenceNote)
		assert.NotEmpty(t, d.Summary)
		require.NotEmpty(t, d.Translations)
		for _, tr := range d.Translations {
			assert.Contains(t, []string{"CC0-1.0", "CC-BY-NC-4.0", "PUBLIC_DOMAIN", "COPY_ALLOWED", "UNKNOWN"}, tr.License)
			if tr.License == "UNKNOWN" {
				assert.NotEmpty(t, tr.LicenseNote, "the source's own words stay in license_note")
			}
		}

		// A text with no Hán year returns null, not 0.
		res = api.get("/v1/sutras/kinh-hanh-phuc")
		var raw map[string]any
		res.data(&raw)
		assert.Contains(t, raw, "han_translation_year")
		assert.Nil(t, raw["han_translation_year"])

		// The list keeps its card shape: none of the six detail-only fields.
		res = api.get("/v1/sutras?page=1")
		var list struct {
			Items []map[string]any `json:"items"`
		}
		res.data(&list)
		require.NotEmpty(t, list.Items)
		assert.NotEmpty(t, list.Items[0]["summary"], "the card carries the summary")
		for _, k := range []string{"original_title", "author", "han_translator", "han_translation_year", "reference_note"} {
			assert.NotContains(t, list.Items[0], k)
		}
	})

	t.Run("AC10/AC28 an unknown sutra is 404 in the shared envelope", func(t *testing.T) {
		api.t = t
		res := api.get("/v1/sutras/khong-co-kinh-nay")
		assert.Equal(t, http.StatusNotFound, res.status)
		assert.False(t, res.envelope.Success)
		assert.Equal(t, "SUTRA_NOT_FOUND", res.errorCode())
		assert.Contains(t, res.contentType, "application/json")
	})

	t.Run("AC9/AC12/AC13/AC14/AC19 a long text is read page by page, and a phẩm is found from start_n", func(t *testing.T) {
		api.t = t
		var detail struct {
			Translations []struct {
				Key       string `json:"key"`
				Lang      string `json:"lang"`
				LineCount int    `json:"line_count"`
				Sections  []struct {
					Name      string `json:"name"`
					StartN    int    `json:"start_n"`
					LineCount int    `json:"line_count"`
				} `json:"sections"`
			} `json:"translations"`
		}
		api.get("/v1/sutras/phap-hoa").data(&detail)
		var key string
		var total int
		var sections []struct {
			Name      string `json:"name"`
			StartN    int    `json:"start_n"`
			LineCount int    `json:"line_count"`
		}
		for _, tr := range detail.Translations {
			if tr.Lang == "vi" {
				key, total, sections = tr.Key, tr.LineCount, tr.Sections
			}
		}
		require.NotEmpty(t, key)
		require.Greater(t, total, 6000, "the longest text in the library")
		require.GreaterOrEqual(t, len(sections), 3)

		type linesPage struct {
			Translation string `json:"translation"`
			Items       []struct {
				N       int     `json:"n"`
				Text    string  `json:"text"`
				Section *string `json:"section"`
			} `json:"items"`
			Page       int `json:"page"`
			PageSize   int `json:"page_size"`
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		}
		lines := func(query string) linesPage {
			res := api.get("/v1/sutras/phap-hoa/lines?translation=" + key + query)
			require.Equal(t, http.StatusOK, res.status, res.raw)
			var p linesPage
			res.data(&p)
			return p
		}

		// default page size 200: 35 pages, the last one short
		first := lines("")
		assert.Equal(t, key, first.Translation)
		assert.Equal(t, 1, first.Page)
		assert.Equal(t, 200, first.PageSize)
		assert.Equal(t, total, first.Total)
		assert.Equal(t, (total+199)/200, first.TotalPages)
		require.Len(t, first.Items, 200)
		assert.Equal(t, 1, first.Items[0].N)
		assert.Equal(t, 200, first.Items[199].N)

		last := lines(fmt.Sprintf("&page=%d", first.TotalPages))
		require.Len(t, last.Items, total-200*(first.TotalPages-1))
		assert.Equal(t, total, last.Items[len(last.Items)-1].N)

		past := lines(fmt.Sprintf("&page=%d", first.TotalPages+1))
		assert.NotNil(t, past.Items)
		assert.Empty(t, past.Items, "a page past the end is empty, not an error")
		assert.Equal(t, first.TotalPages, past.TotalPages)

		// a page_size above the cap is served at the cap, and says so
		big := lines("&page_size=100000")
		assert.Equal(t, 500, big.PageSize)
		assert.Len(t, big.Items, 500)

		// a phẩm is found from its start_n: that line is on page floor((n-1)/size)+1 and carries the phẩm's name
		sec := sections[2]
		page := (sec.StartN-1)/200 + 1
		p := lines(fmt.Sprintf("&page=%d", page))
		var at *struct {
			N       int     `json:"n"`
			Text    string  `json:"text"`
			Section *string `json:"section"`
		}
		for i := range p.Items {
			if p.Items[i].N == sec.StartN {
				at = &p.Items[i]
			}
		}
		require.NotNil(t, at, "line %d must be on page %d", sec.StartN, page)
		require.NotNil(t, at.Section)
		assert.Equal(t, sec.Name, *at.Section)
		assert.NotEmpty(t, at.Text)
	})

	t.Run("AC19 n is contiguous 1..total and text is returned as stored", func(t *testing.T) {
		api.t = t
		// walk a whole short text with a small page and check there is no gap, repeat or reorder
		var entry indexEntry
		for _, e := range idx {
			if e.ID == "kinh-tu-bi" {
				entry = e
			}
		}
		var key string
		var count int
		for _, tr := range entry.Translations {
			if tr.Lang == "vi" {
				key, count = tr.Key, tr.LineCount
			}
		}
		require.NotEmpty(t, key)
		var got []int
		var texts []string
		for page := 1; ; page++ {
			res := api.get(fmt.Sprintf("/v1/sutras/kinh-tu-bi/lines?translation=%s&page=%d&page_size=10", key, page))
			require.Equal(t, http.StatusOK, res.status, res.raw)
			var p struct {
				Items []struct {
					N    int    `json:"n"`
					Text string `json:"text"`
				} `json:"items"`
				TotalPages int `json:"total_pages"`
			}
			res.data(&p)
			if len(p.Items) == 0 {
				assert.Greater(t, page, p.TotalPages)
				break
			}
			for _, it := range p.Items {
				got, texts = append(got, it.N), append(texts, it.Text)
			}
		}
		require.Len(t, got, count)
		for i, n := range got {
			require.Equal(t, i+1, n, "line numbers must run 1..N with no gap or repeat")
		}
		assert.NotContains(t, texts, "", "no line is blank")
		for _, s := range texts {
			assert.Equal(t, strings.TrimSpace(s), s, "text is served as stored, not re-trimmed or re-wrapped: %q", s)
		}
	})

	t.Run("AC15-AC18 lines errors", func(t *testing.T) {
		api.t = t
		tests := []struct {
			name      string
			path      string
			wantCode  int
			wantErr   string
			wantField string
		}{
			{"translation missing", "/v1/sutras/tam-kinh/lines", http.StatusBadRequest, "INVALID_TRANSLATION", "translation"},
			{"translation empty", "/v1/sutras/tam-kinh/lines?translation=", http.StatusBadRequest, "INVALID_TRANSLATION", "translation"},
			{"page 0", "/v1/sutras/tam-kinh/lines?translation=vi-chant&page=0", http.StatusBadRequest, "INVALID_PAGE", "page"},
			{"page not a number", "/v1/sutras/tam-kinh/lines?translation=vi-chant&page=x", http.StatusBadRequest, "INVALID_PAGE", "page"},
			{"page_size 0", "/v1/sutras/tam-kinh/lines?translation=vi-chant&page_size=0", http.StatusBadRequest, "INVALID_PAGE_SIZE", "page_size"},
			{"page_size negative", "/v1/sutras/tam-kinh/lines?translation=vi-chant&page_size=-5", http.StatusBadRequest, "INVALID_PAGE_SIZE", "page_size"},
			{"page_size not a number", "/v1/sutras/tam-kinh/lines?translation=vi-chant&page_size=x", http.StatusBadRequest, "INVALID_PAGE_SIZE", "page_size"},
			{"unknown translation of a known sutra", "/v1/sutras/tam-kinh/lines?translation=zz-nothing", http.StatusNotFound, "TRANSLATION_NOT_FOUND", ""},
			{"unknown sutra", "/v1/sutras/khong-co/lines?translation=vi-chant", http.StatusNotFound, "SUTRA_NOT_FOUND", ""},
			{"unknown sutra wins over an unknown key", "/v1/sutras/khong-co/lines?translation=zz-nothing", http.StatusNotFound, "SUTRA_NOT_FOUND", ""},
		}
		for _, tt := range tests {
			res := api.get(tt.path)
			assert.Equal(t, tt.wantCode, res.status, "%s: %s", tt.name, res.raw)
			assert.Equal(t, tt.wantErr, res.errorCode(), tt.name)
			if tt.wantField != "" {
				assert.Equal(t, tt.wantField, res.details()["field"], tt.name)
			}
		}
	})

	t.Run("AC20/AC21/AC23 the sutras of a deity, by role, in role order", func(t *testing.T) {
		api.t = t
		for _, deity := range []string{"buddha_shakyamuni", "bodhisattva_avalokiteshvara", "buddha_amitabha"} {
			for _, role := range []string{"", "SUBJECT", "SPEAKER", "MENTIONED"} {
				name := deity + "/" + role
				path := "/v1/deities/" + deity + "/sutras"
				if role != "" {
					path += "?role=" + role
				}
				wantIDs, wantRoles := byDeityOracle(idx, deity, role)
				ids, roles, total := walkPages(t, api, path, pageSize)
				assert.Equal(t, len(wantIDs), total, name)
				assert.Equal(t, wantIDs, ids, name)
				assert.Equal(t, wantRoles, roles, "%s: each item carries the deity's role in it", name)
			}
		}

		// a deity that exists but is linked to no sutra is an empty page, not an error
		var linked = map[string]bool{}
		for _, e := range idx {
			for _, d := range e.Deities {
				linked[d.DeityID] = true
			}
		}
		for _, id := range []string{"buddha_vairocana", "bodhisattva_tara_white"} {
			if linked[id] {
				continue
			}
			res := api.get("/v1/deities/" + id + "/sutras")
			require.Equal(t, http.StatusOK, res.status, res.raw)
			var p pagedItems
			res.data(&p)
			assert.Empty(t, p.Items)
			assert.NotNil(t, p.Items)
			assert.Equal(t, 0, p.Total)
			assert.Equal(t, 0, p.TotalPages)
		}
	})

	t.Run("AC21/AC22 by-deity errors", func(t *testing.T) {
		api.t = t
		res := api.get("/v1/deities/khong-co-vi-nay/sutras")
		assert.Equal(t, http.StatusNotFound, res.status)
		assert.Equal(t, "DEITY_NOT_FOUND", res.errorCode())

		for _, role := range []string{"", "speaker", "AUTHOR"} {
			res = api.get("/v1/deities/buddha_amitabha/sutras?role=" + role)
			assert.Equal(t, http.StatusBadRequest, res.status, "role=%q: %s", role, res.raw)
			assert.Equal(t, "INVALID_FILTER", res.errorCode())
			assert.Equal(t, "role", res.details()["field"])
		}
		res = api.get("/v1/deities/buddha_amitabha/sutras?page=abc")
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "INVALID_PAGE", res.errorCode())
	})

	t.Run("AC24 the deity routes answer exactly as before", func(t *testing.T) {
		api.t = t
		res := api.get("/v1/deities/buddha_amitabha")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var d struct {
			ID string `json:"id"`
		}
		res.data(&d)
		assert.Equal(t, "buddha_amitabha", d.ID)

		res = api.get("/v1/deities/top")
		require.Equal(t, http.StatusOK, res.status, res.raw)
		res = api.get("/v1/deities")
		require.Equal(t, http.StatusOK, res.status, res.raw)
	})

	t.Run("AC28 every route is GET-only: anything else is the shared 405 envelope", func(t *testing.T) {
		api.t = t
		for _, path := range []string{"/v1/sutras", "/v1/sutras/tam-kinh", "/v1/sutras/tam-kinh/lines?translation=vi-chant", "/v1/deities/buddha_amitabha/sutras"} {
			for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
				res, header := api.request(method, path)
				assert.Equal(t, http.StatusMethodNotAllowed, res.status, "%s %s", method, path)
				assert.Equal(t, "METHOD_NOT_ALLOWED", res.errorCode(), "%s %s", method, path)
				assert.Equal(t, "GET, HEAD", header.Get("Allow"), "%s %s", method, path)
			}
		}
	})
}

// ---- the first-run seed -----------------------------------------------------

// seededTotals are the rows the shipped library must produce, derived from index.json
// and the per-text counts it carries, not typed in.
func seededTotals(idx []indexEntry) (sutras, translations, lines, links int) {
	for _, e := range idx {
		sutras++
		links += len(e.Deities)
		for _, tr := range e.Translations {
			translations++
			lines += tr.LineCount
		}
	}
	return sutras, translations, lines, links
}

// appService seeds nothing when it starts: on an empty database it comes up and serves
// empty lists. The library arrives only through cmd/seedsutras (TestSeedSutrasCommand_E2E).
func TestSutraService_StartsOnAnEmptyLibraryAndSeedsNothing_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	url, stop := startServer(t, host, port, nil)
	t.Cleanup(stop)
	api := &apiClient{t: t, baseURL: url, http: &http.Client{Timeout: 5 * time.Second}}

	res := api.get("/v1/sutras")
	require.Equal(t, http.StatusOK, res.status, res.raw)
	var p pagedItems
	res.data(&p)
	assert.Zero(t, p.Total)
	assert.Empty(t, p.Items)

	db := openDB(t, host, port)
	for _, table := range []string{"sutras", "sutra_translations", "sutra_lines", "sutra_deities"} {
		assert.Zero(t, countRows(t, db, table), "starting the service must not seed %s", table)
	}
}

// AC32: running the command again never overwrites an edited sutra and never brings
// back a deleted one.
func TestSutraSeed_SecondRunLeavesTheDatabaseAlone_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	cfg := writeTempConfig(t, host, port, "0", nil)
	bin := seedSutrasBinaryPath(t)
	db := openDB(t, host, port)

	res := runUntilExit(t, bin, cfg, 90*time.Second)
	require.Equal(t, 0, res.exitCode, res.stderr)
	execSQL(t, db, `UPDATE sutras SET titles = jsonb_set(titles, '{vi}', to_jsonb($1::text)) WHERE id = 'tam-kinh'`, "Đã sửa tay")
	linesBefore := countRows(t, db, "sutra_lines")
	execSQL(t, db, `DELETE FROM sutras WHERE id = 'kinh-tu-bi'`) // cascades to its translations, lines and links
	linesAfterDelete := countRows(t, db, "sutra_lines")
	require.Less(t, linesAfterDelete, linesBefore)

	res = runUntilExit(t, bin, cfg, 90*time.Second)
	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stderr, "already seeded")

	var title string
	require.NoError(t, db.QueryRow(t.Context(), `SELECT titles->>'vi' FROM sutras WHERE id = 'tam-kinh'`).Scan(&title))
	assert.Equal(t, "Đã sửa tay", title, "the seed must not overwrite an edit")
	assert.Zero(t, countRows(t, db, "sutras WHERE id = 'kinh-tu-bi'"), "the seed must not bring a deleted sutra back")
	assert.Equal(t, linesAfterDelete, countRows(t, db, "sutra_lines"), "a second run writes no row")
}

// AC34: when the database rejects the seed part-way, the command fails and the
// transaction leaves nothing behind. sutra_deities is dropped so the insert fails after
// the sutra, its translations and its lines have already been written inside the
// transaction.
func TestSutraSeed_FailedInsertRollsBackEverything_E2E(t *testing.T) {
	requireE2E(t)
	host, port, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	db := openDB(t, host, port)
	execSQL(t, db, `DROP TABLE sutra_deities`)

	res := runUntilExit(t, seedSutrasBinaryPath(t), writeTempConfig(t, host, port, "0", nil), 90*time.Second)

	require.False(t, res.timedOut, "the command must exit, not hang:\n%s", res.stderr)
	assert.NotEqual(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stderr, "seed sutras")
	for _, table := range []string{"sutras", "sutra_translations", "sutra_lines"} {
		assert.Equal(t, 0, countRows(t, db, table), "%s must be rolled back", table)
	}
}

// A sutra config that would make paging meaningless stops the service before it
// touches the database (cmd/main.go calls Validate right after Load). No database is
// started: the process must exit on its own without reaching one.
func TestSutraConfig_InvalidValuesRefuseToStart_E2E(t *testing.T) {
	requireE2E(t)
	tests := []struct {
		name      string
		overrides configOverrides
		want      string
	}{
		{"page size 0", configOverrides{"sutra": {"page_size": 0}}, "sutra.page_size"},
		{"lines page size 0", configOverrides{"sutra": {"lines_page_size": 0}}, "sutra.lines_page_size"},
		{"lines max 0", configOverrides{"sutra": {"lines_max_page_size": 0}}, "sutra.lines_max_page_size"},
		{"default above the cap", configOverrides{"sutra": {"lines_page_size": 900}}, "must not exceed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := writeTempConfig(t, "127.0.0.1", "1", freePort(t), tt.overrides)
			res := runUntilExit(t, binaryPath(t), cfg, 30*time.Second)
			require.False(t, res.timedOut, res.stderr)
			assert.NotEqual(t, 0, res.exitCode, res.stderr)
			assert.Contains(t, res.stderr, "invalid sutra config")
			assert.Contains(t, res.stderr, tt.want)
		})
	}
}

// AC30/AC35: cmd/seedsutras refuses to run without the deities, seeds an empty table in
// one go and leaves a filled one alone.
func TestSeedSutrasCommand_E2E(t *testing.T) {
	requireE2E(t)
	idx := loadSutraIndex(t)
	sutras, translations, lines, links := seededTotals(idx)
	host, port, cleanupDB := startPostgresWithoutDeities(t)
	t.Cleanup(cleanupDB)
	cfg := writeTempConfig(t, host, port, "0", nil)
	bin := seedSutrasBinaryPath(t)
	db := openDB(t, host, port)

	t.Run("without the deities it exits 1, names them and writes nothing", func(t *testing.T) {
		res := runUntilExit(t, bin, cfg, 90*time.Second)
		require.False(t, res.timedOut, res.stderr)
		assert.Equal(t, 1, res.exitCode, res.stderr)
		assert.Contains(t, res.stderr, "cmd/seeddeities")
		assert.Equal(t, 0, countRows(t, db, "sutras"))
	})

	seedDeities(t, host, port)

	t.Run("on an empty table it seeds the whole library and exits 0", func(t *testing.T) {
		res := runUntilExit(t, bin, cfg, 90*time.Second)
		require.False(t, res.timedOut, res.stderr)
		assert.Equal(t, 0, res.exitCode, res.stderr)
		assert.Contains(t, res.stderr, fmt.Sprintf("seeded %d sutras, %d translations, %d lines, %d deity links", sutras, translations, lines, links))
		assert.Equal(t, sutras, countRows(t, db, "sutras"))
		assert.Equal(t, lines, countRows(t, db, "sutra_lines"))
	})

	t.Run("on a table that already has sutras it exits 0 and writes nothing", func(t *testing.T) {
		execSQL(t, db, `UPDATE sutras SET titles = jsonb_set(titles, '{vi}', to_jsonb($1::text)) WHERE id = 'tam-kinh'`, "Đã sửa tay")
		res := runUntilExit(t, bin, cfg, 90*time.Second)
		require.False(t, res.timedOut, res.stderr)
		assert.Equal(t, 0, res.exitCode, res.stderr)
		assert.Contains(t, res.stderr, "already seeded")
		assert.Equal(t, sutras, countRows(t, db, "sutras"))
		assert.Equal(t, translations, countRows(t, db, "sutra_translations"))
		assert.Equal(t, lines, countRows(t, db, "sutra_lines"))
		var title string
		require.NoError(t, db.QueryRow(t.Context(), `SELECT titles->>'vi' FROM sutras WHERE id = 'tam-kinh'`).Scan(&title))
		assert.Equal(t, "Đã sửa tay", title, "it must not overwrite what is there")
	})

	t.Run("the service then starts on the seeded database and serves it", func(t *testing.T) {
		baseURL, stop := startServer(t, host, port, nil)
		t.Cleanup(stop)
		api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
		var p pagedItems
		api.get("/v1/sutras").data(&p)
		assert.Equal(t, sutras, p.Total)
		assert.Equal(t, sutras, countRows(t, db, "sutras"), "the service seeds nothing")
		assert.Equal(t, lines, countRows(t, db, "sutra_lines"))
	})
}
