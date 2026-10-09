package repository

import (
	"encoding/json"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/assets"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// doc is a mutable stand-in for one <id>.json, so each test breaks exactly one thing.
type doc = map[string]any

func validDoc(id string) doc {
	return doc{
		"id": id, "tradition": "MAHAYANA", "kind": "SUTRA", "collection": "Bát Nhã", "reference": "T0251",
		"originalTitle": "Bát Nhã Ba La Mật Đa Tâm Kinh", "author": "Au", "hanTranslator": "Huyền Trang", "hanTranslationYear": 649,
		"referenceNote": "rn", "summary": "Tóm tắt.",
		"titles":     doc{"vi": "Tâm Kinh", "en": "Heart Sutra"},
		"deities":    []doc{{"deity_id": "bodhisattva_avalokiteshvara", "role": "SPEAKER"}, {"deity_id": "buddha_shakyamuni", "role": "MENTIONED"}},
		"dailyChant": true, "purposes": []string{"HOC_GIAO_LY"}, "durationMinutes": 4, "reviewStatus": "UNREVIEWED",
		"translations": []doc{
			{
				"key": "vi-chant", "lang": "vi", "variant": "chant", "translator": "t", "sourceName": "sn",
				"sourceUrl": "https://x", "license": "CC0-1.0", "licenseUrl": "https://cc", "licenseNote": "as stated", "note": "n",
				"lines": []doc{{"n": 1, "text": "một"}, {"n": 2, "text": "hai", "section": "A"}, {"n": 3, "text": "ba", "section": "A"}},
			},
			{
				"key": "en-translation", "lang": "en", "variant": "translation", "license": "UNKNOWN",
				"lines": []doc{{"n": 1, "text": "one"}},
			},
		},
	}
}

// seedFS builds a data directory holding the given docs, indexed in the given order.
func seedFS(t *testing.T, ids []string, docs map[string]doc) fstest.MapFS {
	t.Helper()
	type entry struct {
		ID string `json:"id"`
	}
	idx := struct {
		Sutras []entry `json:"sutras"`
	}{}
	for _, id := range ids {
		idx.Sutras = append(idx.Sutras, entry{ID: id})
	}
	raw, err := json.Marshal(idx)
	require.NoError(t, err)
	fsys := fstest.MapFS{"index.json": {Data: raw}}
	for id, d := range docs {
		b, err := json.Marshal(d)
		require.NoError(t, err)
		fsys[id+".json"] = &fstest.MapFile{Data: b}
	}
	return fsys
}

func oneSutraFS(t *testing.T, mutate func(d doc)) fstest.MapFS {
	t.Helper()
	d := validDoc("tam-kinh")
	if mutate != nil {
		mutate(d)
	}
	return seedFS(t, []string{"tam-kinh"}, map[string]doc{"tam-kinh": d})
}

func TestParseSutraSeed_Valid(t *testing.T) {
	fsys := seedFS(t, []string{"b-sutra", "a-sutra"}, map[string]doc{"a-sutra": validDoc("a-sutra"), "b-sutra": validDoc("b-sutra")})

	got, err := ParseSutraSeed(fsys)

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "b-sutra", got[0].Sutra.ID, "order follows index.json, not the directory")
	assert.Equal(t, 0, got[0].Sutra.SortOrder)
	assert.Equal(t, 1, got[1].Sutra.SortOrder)

	s := got[0].Sutra
	assert.Equal(t, domain.TraditionMahayana, s.Tradition)
	assert.Equal(t, domain.KindSutra, s.Kind)
	assert.Equal(t, "Bát Nhã", s.Collection)
	assert.Equal(t, "T0251", s.Reference)
	assert.Equal(t, "Bát Nhã Ba La Mật Đa Tâm Kinh", s.OriginalTitle)
	assert.Equal(t, "Au", s.Author)
	assert.Equal(t, "Huyền Trang", s.HanTranslator)
	require.NotNil(t, s.HanTranslationYear)
	assert.Equal(t, 649, *s.HanTranslationYear)
	assert.Equal(t, "rn", s.ReferenceNote)
	assert.Equal(t, "Tóm tắt.", s.Summary)
	assert.Equal(t, map[string]string{"vi": "Tâm Kinh", "en": "Heart Sutra"}, s.Titles)
	assert.True(t, s.DailyChant)
	assert.Equal(t, []domain.Purpose{domain.PurposeHocGiaoLy}, s.Purposes)
	require.NotNil(t, s.DurationMinutes)
	assert.Equal(t, 4, *s.DurationMinutes)
	assert.Equal(t, "UNREVIEWED", s.ReviewStatus)
	assert.Equal(t, []domain.SutraDeity{
		{DeityID: "bodhisattva_avalokiteshvara", Role: domain.RoleSpeaker},
		{DeityID: "buddha_shakyamuni", Role: domain.RoleMentioned},
	}, s.Deities)

	require.Len(t, got[0].Translations, 2)
	tr := got[0].Translations[0]
	assert.Equal(t, "vi-chant", tr.Key)
	assert.Equal(t, domain.VariantChant, tr.Variant)
	assert.Equal(t, "https://x", tr.SourceURL)
	assert.Equal(t, domain.LicenseCC0, tr.License)
	assert.Equal(t, "https://cc", tr.LicenseURL)
	assert.Equal(t, "as stated", tr.LicenseNote)
	assert.Equal(t, "n", tr.Note)
	assert.Equal(t, 3, tr.LineCount)
	assert.Equal(t, []domain.SutraSection{{Name: "A", StartN: 2, LineCount: 2}}, tr.Sections)
	assert.Equal(t, []domain.SutraLine{{N: 1, Text: "một"}, {N: 2, Text: "hai", Section: "A"}, {N: 3, Text: "ba", Section: "A"}}, tr.Lines)
	assert.Empty(t, got[0].Translations[1].Sections, "no section lines, no table of contents")
}

func TestParseSutraSeed_OptionalFieldsDefaultToEmpty(t *testing.T) {
	for _, drop := range []string{"originalTitle", "author", "hanTranslator", "hanTranslationYear", "referenceNote", "summary"} {
		got, err := ParseSutraSeed(oneSutraFS(t, func(d doc) { delete(d, drop) }))
		require.NoError(t, err, drop)
		assert.NotEmpty(t, got)
	}

	got, err := ParseSutraSeed(oneSutraFS(t, func(d doc) {
		for _, k := range []string{"originalTitle", "author", "hanTranslator", "hanTranslationYear", "referenceNote", "summary"} {
			delete(d, k)
		}
		tr := d["translations"].([]doc)[0]
		delete(tr, "licenseUrl")
		delete(tr, "licenseNote")
	}))
	require.NoError(t, err)
	s := got[0].Sutra
	assert.Empty(t, s.OriginalTitle)
	assert.Empty(t, s.Author)
	assert.Empty(t, s.HanTranslator)
	assert.Nil(t, s.HanTranslationYear, "a missing year is unknown, not 0")
	assert.Empty(t, s.ReferenceNote)
	assert.Empty(t, s.Summary)
	assert.Empty(t, got[0].Translations[0].LicenseURL)
	assert.Empty(t, got[0].Translations[0].LicenseNote)

	got, err = ParseSutraSeed(oneSutraFS(t, func(d doc) { d["hanTranslationYear"] = nil }))
	require.NoError(t, err)
	assert.Nil(t, got[0].Sutra.HanTranslationYear, "null year is unknown")
}

func TestParseSutraSeed_NullDurationAndEmptyPurposes(t *testing.T) {
	got, err := ParseSutraSeed(oneSutraFS(t, func(d doc) {
		d["durationMinutes"] = nil
		d["purposes"] = []string{}
		d["deities"] = []doc{}
		delete(d, "reviewStatus")
	}))

	require.NoError(t, err)
	s := got[0].Sutra
	assert.Nil(t, s.DurationMinutes, "null stays nil, never 0")
	assert.NotNil(t, s.Purposes)
	assert.Empty(t, s.Purposes)
	assert.Empty(t, s.Deities)
	assert.Equal(t, "UNREVIEWED", s.ReviewStatus, "a missing reviewStatus takes the schema default")
}

func TestParseSutraSeed_InvalidSutra(t *testing.T) {
	setTrans := func(f func(tr doc)) func(d doc) {
		return func(d doc) { f(d["translations"].([]doc)[0]) }
	}
	tests := []struct {
		name    string
		mutate  func(d doc)
		wantErr string
	}{
		{"id differs from the file name", func(d doc) { d["id"] = "other" }, "does not match the file name"},
		{"unknown tradition", func(d doc) { d["tradition"] = "ZEN" }, "unknown tradition"},
		{"unknown kind", func(d doc) { d["kind"] = "POEM" }, "unknown kind"},
		{"titles.vi empty", func(d doc) { d["titles"] = doc{"vi": "  ", "en": "x"} }, "titles.vi"},
		{"titles.vi missing", func(d doc) { d["titles"] = doc{"en": "x"} }, "titles.vi"},
		{"dailyChant missing", func(d doc) { delete(d, "dailyChant") }, "dailyChant"},
		{"dailyChant not a boolean", func(d doc) { d["dailyChant"] = "yes" }, "dailyChant"},
		{"purposes missing", func(d doc) { delete(d, "purposes") }, "purposes is missing"},
		{"unknown purpose", func(d doc) { d["purposes"] = []string{"LUCK"} }, "unknown purpose"},
		{"durationMinutes missing", func(d doc) { delete(d, "durationMinutes") }, "durationMinutes is missing"},
		{"durationMinutes zero", func(d doc) { d["durationMinutes"] = 0 }, "durationMinutes must be >= 1"},
		{"durationMinutes negative", func(d doc) { d["durationMinutes"] = -3 }, "durationMinutes must be >= 1"},
		{"durationMinutes fractional", func(d doc) { d["durationMinutes"] = 1.5 }, "durationMinutes"},
		{"durationMinutes a string", func(d doc) { d["durationMinutes"] = "30" }, "durationMinutes"},
		{"unknown top-level field is rejected, not ignored", func(d doc) { d["dailyChants"] = true }, "unknown field"},
		{"deity listed twice", func(d doc) {
			d["deities"] = []doc{{"deity_id": "x", "role": "SPEAKER"}, {"deity_id": "x", "role": "SUBJECT"}}
		}, "listed twice"},
		{"deity with empty id", func(d doc) { d["deities"] = []doc{{"deity_id": "", "role": "SPEAKER"}} }, "empty deity_id"},
		{"unknown deity role", func(d doc) { d["deities"] = []doc{{"deity_id": "x", "role": "AUTHOR"}} }, "unknown role"},
		{"no translations", func(d doc) { d["translations"] = []doc{} }, "no translations"},
		{"translation without key", setTrans(func(tr doc) { delete(tr, "key") }), "has no key"},
		{"duplicate translation key", func(d doc) { d["translations"].([]doc)[1]["key"] = "vi-chant" }, "used twice"},
		{"translation without lang", setTrans(func(tr doc) { tr["lang"] = "" }), "empty lang"},
		{"unknown variant", setTrans(func(tr doc) { tr["variant"] = "gloss" }), "unknown variant"},
		{"license missing", setTrans(func(tr doc) { delete(tr, "license") }), "unknown license"},
		{"license outside the closed set", setTrans(func(tr doc) { tr["license"] = "unknown" }), "unknown license"},
		{"hanTranslationYear zero", func(d doc) { d["hanTranslationYear"] = 0 }, "hanTranslationYear must be >= 1"},
		{"hanTranslationYear negative", func(d doc) { d["hanTranslationYear"] = -5 }, "hanTranslationYear must be >= 1"},
		{"hanTranslationYear a string", func(d doc) { d["hanTranslationYear"] = "649" }, "hanTranslationYear"},
		{"hanTranslationYear fractional", func(d doc) { d["hanTranslationYear"] = 6.5 }, "hanTranslationYear"},
		{"translation without lines", setTrans(func(tr doc) { tr["lines"] = []doc{} }), "no lines"},
		{"line numbers with a gap", setTrans(func(tr doc) {
			tr["lines"] = []doc{{"n": 1, "text": "a"}, {"n": 3, "text": "b"}}
		}), "without gaps"},
		{"line numbers not starting at 1", setTrans(func(tr doc) {
			tr["lines"] = []doc{{"n": 0, "text": "a"}, {"n": 1, "text": "b"}}
		}), "without gaps"},
		{"line numbers repeated", setTrans(func(tr doc) {
			tr["lines"] = []doc{{"n": 1, "text": "a"}, {"n": 1, "text": "b"}}
		}), "without gaps"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSutraSeed(oneSutraFS(t, tt.mutate))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Contains(t, err.Error(), `"tam-kinh"`, "the error must name the sutra")
		})
	}
}

func TestParseSutraSeed_InvalidIndex(t *testing.T) {
	t.Run("index.json missing", func(t *testing.T) {
		_, err := ParseSutraSeed(fstest.MapFS{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "index.json")
	})
	t.Run("index.json is not JSON", func(t *testing.T) {
		_, err := ParseSutraSeed(fstest.MapFS{"index.json": {Data: []byte("nope")}})
		require.Error(t, err)
	})
	t.Run("empty index", func(t *testing.T) {
		_, err := ParseSutraSeed(seedFS(t, nil, nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no sutras")
	})
	t.Run("a file listed in the index is missing", func(t *testing.T) {
		_, err := ParseSutraSeed(seedFS(t, []string{"ghost"}, nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"ghost"`)
	})
	t.Run("empty id", func(t *testing.T) {
		_, err := ParseSutraSeed(seedFS(t, []string{""}, nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty id")
	})
	t.Run("duplicate id", func(t *testing.T) {
		_, err := ParseSutraSeed(seedFS(t, []string{"tam-kinh", "tam-kinh"}, map[string]doc{"tam-kinh": validDoc("tam-kinh")}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate id")
	})
	t.Run("an id that is a path is refused", func(t *testing.T) {
		for _, id := range []string{"../index", "a/b", `a\b`, ".."} {
			_, err := ParseSutraSeed(seedFS(t, []string{id}, nil))
			require.Error(t, err, id)
			assert.Contains(t, err.Error(), "plain file name", id)
		}
	})
}

func TestSectionsOf(t *testing.T) {
	line := func(n int, sec string) domain.SutraLine { return domain.SutraLine{N: n, Section: sec} }
	tests := []struct {
		name  string
		lines []domain.SutraLine
		want  []domain.SutraSection
	}{
		{"no sections", []domain.SutraLine{line(1, ""), line(2, "")}, nil},
		{"one run", []domain.SutraLine{line(1, "A"), line(2, "A"), line(3, "A")}, []domain.SutraSection{{Name: "A", StartN: 1, LineCount: 3}}},
		{"consecutive sections", []domain.SutraLine{line(1, "A"), line(2, "B"), line(3, "B")},
			[]domain.SutraSection{{Name: "A", StartN: 1, LineCount: 1}, {Name: "B", StartN: 2, LineCount: 2}}},
		{"unsectioned lines are in no section", []domain.SutraLine{line(1, ""), line(2, "A"), line(3, ""), line(4, "B")},
			[]domain.SutraSection{{Name: "A", StartN: 2, LineCount: 1}, {Name: "B", StartN: 4, LineCount: 1}}},
		{"a name that comes back starts a new entry", []domain.SutraLine{line(1, "A"), line(2, "B"), line(3, "A")},
			[]domain.SutraSection{{Name: "A", StartN: 1, LineCount: 1}, {Name: "B", StartN: 2, LineCount: 1}, {Name: "A", StartN: 3, LineCount: 1}}},
		{"the same name split by an unsectioned line is two entries", []domain.SutraLine{line(1, "A"), line(2, ""), line(3, "A")},
			[]domain.SutraSection{{Name: "A", StartN: 1, LineCount: 1}, {Name: "A", StartN: 3, LineCount: 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sectionsOf(tt.lines))
		})
	}
}

// TestParseSutraSeed_ShippedData parses the data that is actually embedded in the
// binary. It is the check that the first-run seed will not abort on real input, and
// it pins the numbers the spec quotes (docs/specs/features/sutra).
func TestParseSutraSeed_ShippedData(t *testing.T) {
	sub, err := fs.Sub(assets.SutrasFS, "data/sutras")
	require.NoError(t, err)

	seeds, err := ParseSutraSeed(sub)
	require.NoError(t, err)

	var translations, lines, links, withSections, nullDuration, daily int
	ids := map[string]bool{}
	for i, s := range seeds {
		assert.Equal(t, i, s.Sutra.SortOrder)
		assert.False(t, ids[s.Sutra.ID], "duplicate %s", s.Sutra.ID)
		ids[s.Sutra.ID] = true
		links += len(s.Sutra.Deities)
		if s.Sutra.DurationMinutes == nil {
			nullDuration++
		}
		if s.Sutra.DailyChant {
			daily++
		}
		for _, tr := range s.Translations {
			translations++
			lines += len(tr.Lines)
			assert.Equal(t, len(tr.Lines), tr.LineCount)
			if len(tr.Sections) > 0 {
				withSections++
			}
			for _, sec := range tr.Sections {
				assert.GreaterOrEqual(t, sec.StartN, 1)
				assert.LessOrEqual(t, sec.StartN+sec.LineCount-1, tr.LineCount, "%s/%s section %q runs past the text", s.Sutra.ID, tr.Key, sec.Name)
			}
		}
	}
	assert.Len(t, seeds, 153)
	assert.Equal(t, 391, translations)
	assert.Equal(t, 64009, lines)
	assert.Equal(t, 191, links)
	assert.Equal(t, 51, withSections)
	assert.Equal(t, 0, nullDuration, "every text carries an estimate (the ten Vajrayana ones are line-count based, see migrations/0019)")
	assert.Equal(t, 34, daily)
}

// The shipped data must have been split: `reference` holds only a citation code, every
// text has a summary, every licence is a code, and the packed forms are gone.
func TestParseSutraSeed_ShippedDataIsSplit(t *testing.T) {
	sub, err := fs.Sub(assets.SutrasFS, "data/sutras")
	require.NoError(t, err)
	seeds, err := ParseSutraSeed(sub)
	require.NoError(t, err)

	var withYear int
	for _, s := range seeds {
		assert.NotEmpty(t, strings.TrimSpace(s.Sutra.Summary), "%s has no summary", s.Sutra.ID)
		assert.NotContains(t, s.Sutra.Reference, " dịch Hán", "%s still packs the Hán translator into reference", s.Sutra.ID)
		assert.NotContains(t, s.Sutra.Reference, ";", "%s still packs several things into reference", s.Sutra.ID)
		if s.Sutra.HanTranslationYear != nil {
			withYear++
			assert.NotEmpty(t, s.Sutra.HanTranslator, "%s has a Hán year without a translator", s.Sutra.ID)
		}
		for _, tr := range s.Translations {
			assert.True(t, domain.ValidLicense(tr.License), "%s/%s license %q", s.Sutra.ID, tr.Key, tr.License)
			if tr.LicenseURL != "" {
				assert.Equal(t, domain.LicenseCC0, tr.License, "%s/%s: a URL is only given for CC0 in the data", s.Sutra.ID, tr.Key)
			}
		}
	}
	assert.Equal(t, 4, withYear)
}

// Every deity the shipped sutras link to must be one the deities seed provides,
// otherwise the first run would abort with ErrSutraSeedDeitiesMissing.
func TestParseSutraSeed_ShippedDeitiesExistInDeitySeed(t *testing.T) {
	raw, err := os.ReadFile("../../../assets/data/phat_botat.json")
	require.NoError(t, err)
	var deities []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(raw, &deities))
	known := map[string]bool{}
	for _, d := range deities {
		known[d.ID] = true
	}
	require.NotEmpty(t, known, "phat_botat.json: no deities decoded")

	sub, err := fs.Sub(assets.SutrasFS, "data/sutras")
	require.NoError(t, err)
	seeds, err := ParseSutraSeed(sub)
	require.NoError(t, err)
	for _, s := range seeds {
		for _, d := range s.Sutra.Deities {
			assert.True(t, known[d.DeityID], "%s links to %s, which phat_botat.json does not define", s.Sutra.ID, d.DeityID)
		}
	}
}
