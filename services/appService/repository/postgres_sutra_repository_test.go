package repository

import (
	"context"
	"fmt"
	"io/fs"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/assets"
	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newSutraRepo(t *testing.T) (domain.SutraRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	return NewSutraRepository(db), db
}

// insertTestDeity adds the minimum deities row a sutra link needs.
func insertTestDeity(t *testing.T, db database.Database, id string, order int, nameVI, thumb string) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO deities (id, type, sort_order, name_vi, thumbnail_url) VALUES ($1, 'BODHISATTVA', $2, $3, $4)`,
		id, order, nameVI, thumb)
	require.NoError(t, err)
}

type seedOpt func(*domain.SutraSeed)

// mkSutraSeed builds a seed with two translations (a sectioned Vietnamese chant and
// an English text) and no deities.
func mkSutraSeed(id string, order int, opts ...seedOpt) domain.SutraSeed {
	dur, year := 10, 649
	s := domain.SutraSeed{
		Sutra: domain.Sutra{
			ID: id, SortOrder: order, Tradition: domain.TraditionMahayana, Kind: domain.KindSutra,
			Collection: "Bát Nhã", Reference: "R-" + id, Titles: map[string]string{"vi": "Kinh " + id, "en": "Sutra " + id},
			DailyChant: false, Purposes: []domain.Purpose{domain.PurposeHocGiaoLy}, DurationMinutes: &dur,
			ReviewStatus:  "UNREVIEWED",
			OriginalTitle: "Gốc " + id, Author: "Tác giả " + id, HanTranslator: "Huyền Trang", HanTranslationYear: &year,
			ReferenceNote: "Ghi chú " + id, Summary: "Tóm tắt " + id + ".",
		},
		Translations: []domain.SutraTranslationSeed{
			{
				SutraTranslation: domain.SutraTranslation{
					Key: "vi-chant", Lang: "vi", Variant: domain.VariantChant, Translator: "T", SourceName: "SN",
					SourceURL: "https://x/" + id, License: domain.LicenseUnknown, LicenseURL: "https://l/" + id, LicenseNote: "không có giấy phép", Note: "ghi chú", LineCount: 4,
					Sections: []domain.SutraSection{{Name: "Phẩm 1", StartN: 2, LineCount: 2}},
				},
				Lines: []domain.SutraLine{
					{N: 1, Text: "một"}, {N: 2, Text: "hai", Section: "Phẩm 1"}, {N: 3, Text: "ba", Section: "Phẩm 1"}, {N: 4, Text: "bốn"},
				},
			},
			{
				SutraTranslation: domain.SutraTranslation{Key: "en-translation", Lang: "en", Variant: domain.VariantTranslation, License: domain.LicenseCC0, LineCount: 1},
				Lines:            []domain.SutraLine{{N: 1, Text: "one"}},
			},
		},
	}
	for _, o := range opts {
		o(&s)
	}
	return s
}

func withDeities(links ...domain.SutraDeity) seedOpt {
	return func(s *domain.SutraSeed) { s.Sutra.Deities = links }
}

func seedIDs(items []domain.Sutra) []string {
	out := make([]string, 0, len(items))
	for _, s := range items {
		out = append(out, s.ID)
	}
	return out
}

func TestSutraRepository_InsertAllAndFindByID(t *testing.T) {
	repo, db := newSutraRepo(t)
	ctx := context.Background()
	insertTestDeity(t, db, "bodhisattva_a", 1, "Quán Thế Âm", "thumb-a.jpg")
	insertTestDeity(t, db, "buddha_b", 2, "Thích Ca", "thumb-b.jpg")

	seed := mkSutraSeed("tam-kinh", 0, withDeities(
		domain.SutraDeity{DeityID: "buddha_b", Role: domain.RoleMentioned},
		domain.SutraDeity{DeityID: "bodhisattva_a", Role: domain.RoleSpeaker},
	))
	res, err := repo.InsertAll(ctx, []domain.SutraSeed{seed})
	require.NoError(t, err)
	assert.Equal(t, domain.SutraSeedResult{Sutras: 1, Translations: 2, Lines: 5, DeityLinks: 2}, res)

	got, err := repo.FindByID(ctx, "tam-kinh")
	require.NoError(t, err)
	assert.Equal(t, "tam-kinh", got.ID)
	assert.Equal(t, domain.TraditionMahayana, got.Tradition)
	assert.Equal(t, domain.KindSutra, got.Kind)
	assert.Equal(t, "Bát Nhã", got.Collection)
	assert.Equal(t, "R-tam-kinh", got.Reference)
	assert.Equal(t, map[string]string{"vi": "Kinh tam-kinh", "en": "Sutra tam-kinh"}, got.Titles)
	assert.False(t, got.DailyChant)
	assert.Equal(t, []domain.Purpose{domain.PurposeHocGiaoLy}, got.Purposes)
	require.NotNil(t, got.DurationMinutes)
	assert.Equal(t, 10, *got.DurationMinutes)
	assert.Equal(t, "UNREVIEWED", got.ReviewStatus)
	assert.Equal(t, "Gốc tam-kinh", got.OriginalTitle)
	assert.Equal(t, "Tác giả tam-kinh", got.Author)
	assert.Equal(t, "Huyền Trang", got.HanTranslator)
	require.NotNil(t, got.HanTranslationYear)
	assert.Equal(t, 649, *got.HanTranslationYear)
	assert.Equal(t, "Ghi chú tam-kinh", got.ReferenceNote)
	assert.Equal(t, "Tóm tắt tam-kinh.", got.Summary)

	require.Len(t, got.Translations, 2)
	assert.Equal(t, []string{"vi-chant", "en-translation"}, []string{got.Translations[0].Key, got.Translations[1].Key}, "file order is kept")
	vi := got.Translations[0]
	assert.Equal(t, "vi", vi.Lang)
	assert.Equal(t, domain.VariantChant, vi.Variant)
	assert.Equal(t, "T", vi.Translator)
	assert.Equal(t, "SN", vi.SourceName)
	assert.Equal(t, "https://x/tam-kinh", vi.SourceURL)
	assert.Equal(t, domain.LicenseUnknown, vi.License)
	assert.Equal(t, "https://l/tam-kinh", vi.LicenseURL)
	assert.Equal(t, "không có giấy phép", vi.LicenseNote)
	assert.Equal(t, "ghi chú", vi.Note)
	assert.Equal(t, 4, vi.LineCount)
	assert.Equal(t, []domain.SutraSection{{Name: "Phẩm 1", StartN: 2, LineCount: 2}}, vi.Sections)
	assert.Empty(t, got.Translations[1].Sections)

	assert.Equal(t, []domain.SutraDeity{
		{DeityID: "bodhisattva_a", Role: domain.RoleSpeaker, NameVI: "Quán Thế Âm", ThumbnailURL: "thumb-a.jpg"},
		{DeityID: "buddha_b", Role: domain.RoleMentioned, NameVI: "Thích Ca", ThumbnailURL: "thumb-b.jpg"},
	}, got.Deities, "ordered by role (SPEAKER before MENTIONED), with names from deities")

	_, err = repo.FindByID(ctx, "nope")
	assert.ErrorIs(t, err, domain.ErrSutraNotFound)
}

func TestSutraRepository_InsertAll_NullDurationAndEmptyArrays(t *testing.T) {
	repo, _ := newSutraRepo(t)
	ctx := context.Background()
	seed := mkSutraSeed("no-estimate", 0)
	seed.Sutra.DurationMinutes = nil
	seed.Sutra.Purposes = nil

	_, err := repo.InsertAll(ctx, []domain.SutraSeed{seed})
	require.NoError(t, err)

	got, err := repo.FindByID(ctx, "no-estimate")
	require.NoError(t, err)
	assert.Nil(t, got.DurationMinutes, "NULL, not 0")
	assert.Empty(t, got.Purposes)
	assert.Empty(t, got.Deities)
}

func TestSutraRepository_InsertAll_LeavesExistingSutrasAlone(t *testing.T) {
	repo, _ := newSutraRepo(t)
	ctx := context.Background()

	_, err := repo.InsertAll(ctx, []domain.SutraSeed{mkSutraSeed("old", 0)})
	require.NoError(t, err)

	changed := mkSutraSeed("old", 5)
	changed.Sutra.Titles = map[string]string{"vi": "ĐÃ ĐỔI"}
	changed.Translations = changed.Translations[:1]
	res, err := repo.InsertAll(ctx, []domain.SutraSeed{changed, mkSutraSeed("new", 1)})

	require.NoError(t, err)
	assert.Equal(t, domain.SutraSeedResult{Sutras: 1, Translations: 2, Lines: 5}, res, "only the new sutra is counted")

	old, err := repo.FindByID(ctx, "old")
	require.NoError(t, err)
	assert.Equal(t, "Kinh old", old.Titles["vi"], "an existing sutra is never overwritten")
	assert.Equal(t, 0, old.SortOrder)
	assert.Len(t, old.Translations, 2, "and its translations are not touched or duplicated")
}

func TestSutraRepository_InsertAll_IsAllOrNothing(t *testing.T) {
	repo, _ := newSutraRepo(t)
	ctx := context.Background()

	good := mkSutraSeed("good", 0)
	bad := mkSutraSeed("bad", 1, withDeities(domain.SutraDeity{DeityID: "not-in-deities", Role: domain.RoleSubject}))

	_, err := repo.InsertAll(ctx, []domain.SutraSeed{good, bad})

	require.Error(t, err, "a link to a missing deity violates the foreign key")
	n, err := repo.Count(ctx, domain.SutraFilter{})
	require.NoError(t, err)
	assert.Equal(t, 0, n, "the good sutra must roll back with the bad one")
}

func TestSutraRepository_InsertAll_ManyLinesAcrossChunks(t *testing.T) {
	repo, _ := newSutraRepo(t)
	ctx := context.Background()
	total := insertLinesChunk*2 + 37 // spans three INSERT statements
	seed := mkSutraSeed("long", 0)
	lines := make([]domain.SutraLine, total)
	for i := range lines {
		lines[i] = domain.SutraLine{N: i + 1, Text: fmt.Sprintf("dòng %d", i+1)}
	}
	seed.Translations = []domain.SutraTranslationSeed{{
		SutraTranslation: domain.SutraTranslation{Key: "vi-translation", Lang: "vi", Variant: domain.VariantTranslation, LineCount: total},
		Lines:            lines,
	}}

	res, err := repo.InsertAll(ctx, []domain.SutraSeed{seed})
	require.NoError(t, err)
	assert.Equal(t, total, res.Lines)

	tr, err := repo.FindTranslation(ctx, "long", "vi-translation")
	require.NoError(t, err)
	for _, edge := range []int{1, insertLinesChunk, insertLinesChunk + 1, 2 * insertLinesChunk, 2*insertLinesChunk + 1, total} {
		got, err := repo.ListLines(ctx, tr.ID, int64(edge-1), int64(edge))
		require.NoError(t, err)
		require.Len(t, got, 1, "line %d must exist", edge)
		assert.Equal(t, fmt.Sprintf("dòng %d", edge), got[0].Text)
	}
}

func TestSutraRepository_FindTranslation(t *testing.T) {
	repo, _ := newSutraRepo(t)
	ctx := context.Background()
	_, err := repo.InsertAll(ctx, []domain.SutraSeed{mkSutraSeed("tam-kinh", 0)})
	require.NoError(t, err)

	tr, err := repo.FindTranslation(ctx, "tam-kinh", "vi-chant")
	require.NoError(t, err)
	assert.NotZero(t, tr.ID)
	assert.Equal(t, "vi-chant", tr.Key)
	assert.Equal(t, "vi", tr.Lang)
	assert.Equal(t, domain.VariantChant, tr.Variant)
	assert.Equal(t, 4, tr.LineCount)

	_, err = repo.FindTranslation(ctx, "tam-kinh", "zh-translation")
	assert.ErrorIs(t, err, domain.ErrSutraTranslationNotFound, "known sutra, unknown key")

	_, err = repo.FindTranslation(ctx, "ghost", "vi-chant")
	assert.ErrorIs(t, err, domain.ErrSutraNotFound, "an unknown sutra is checked first")
	_, err = repo.FindTranslation(ctx, "ghost", "no-such-key")
	assert.ErrorIs(t, err, domain.ErrSutraNotFound)
}

func TestSutraRepository_ListLines(t *testing.T) {
	repo, _ := newSutraRepo(t)
	ctx := context.Background()
	_, err := repo.InsertAll(ctx, []domain.SutraSeed{mkSutraSeed("tam-kinh", 0)})
	require.NoError(t, err)
	tr, err := repo.FindTranslation(ctx, "tam-kinh", "vi-chant")
	require.NoError(t, err)

	t.Run("a range is a page: from < n <= to", func(t *testing.T) {
		got, err := repo.ListLines(ctx, tr.ID, 1, 3)
		require.NoError(t, err)
		assert.Equal(t, []domain.SutraLine{{N: 2, Text: "hai", Section: "Phẩm 1"}, {N: 3, Text: "ba", Section: "Phẩm 1"}}, got)
	})
	t.Run("a line outside any section has an empty Section", func(t *testing.T) {
		got, err := repo.ListLines(ctx, tr.ID, 0, 1)
		require.NoError(t, err)
		assert.Equal(t, []domain.SutraLine{{N: 1, Text: "một"}}, got)
	})
	t.Run("a partial last page", func(t *testing.T) {
		got, err := repo.ListLines(ctx, tr.ID, 3, 100)
		require.NoError(t, err)
		assert.Equal(t, []domain.SutraLine{{N: 4, Text: "bốn"}}, got)
	})
	t.Run("past the end is empty, not an error, and not nil", func(t *testing.T) {
		got, err := repo.ListLines(ctx, tr.ID, 100, 200)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
	t.Run("a range beyond int32 does not overflow", func(t *testing.T) {
		from := int64(math.MaxInt32-1) * 500
		got, err := repo.ListLines(ctx, tr.ID, from, from+500)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("another translation's lines are never mixed in", func(t *testing.T) {
		en, err := repo.FindTranslation(ctx, "tam-kinh", "en-translation")
		require.NoError(t, err)
		got, err := repo.ListLines(ctx, en.ID, 0, 100)
		require.NoError(t, err)
		assert.Equal(t, []domain.SutraLine{{N: 1, Text: "one"}}, got)
	})
}

// listFixture seeds six sutras whose attributes differ in every filterable way.
func listFixture(t *testing.T, repo domain.SutraRepository) {
	t.Helper()
	min := func(n int) *int { return &n }
	mk := func(id string, order int, trad domain.Tradition, kind domain.Kind, coll string, daily bool, purposes []domain.Purpose, dur *int, langs ...string) domain.SutraSeed {
		s := mkSutraSeed(id, order)
		s.Sutra.Tradition, s.Sutra.Kind, s.Sutra.Collection, s.Sutra.DailyChant = trad, kind, coll, daily
		s.Sutra.Purposes, s.Sutra.DurationMinutes = purposes, dur
		s.Translations = nil
		for _, l := range langs {
			s.Translations = append(s.Translations, domain.SutraTranslationSeed{
				SutraTranslation: domain.SutraTranslation{Key: l + "-translation", Lang: l, Variant: domain.VariantTranslation, LineCount: 1},
				Lines:            []domain.SutraLine{{N: 1, Text: "x"}},
			})
		}
		return s
	}
	T, M, V := domain.TraditionTheravada, domain.TraditionMahayana, domain.TraditionVajrayana
	seeds := []domain.SutraSeed{
		mk("a", 0, T, domain.KindSutra, "Trung Bộ", false, []domain.Purpose{domain.PurposeHocGiaoLy}, min(40), "vi", "pli", "en"),
		mk("b", 1, T, domain.KindVerse, "Tiểu Bộ", true, []domain.Purpose{domain.PurposeCauAn}, min(5), "vi", "en"),
		mk("c", 2, M, domain.KindSutra, "Bát Nhã", true, []domain.Purpose{domain.PurposeHocGiaoLy, domain.PurposeCauAn}, min(15), "vi", "zh"),
		mk("d", 3, M, domain.KindDharani, "Chú", true, []domain.Purpose{domain.PurposeTriChu}, min(1), "vi"),
		mk("e", 4, V, domain.KindDharani, "Mật tông", false, nil, nil, "en", "bo"),
		mk("f", 5, M, domain.KindVerse, "Sám hối", true, []domain.Purpose{domain.PurposeSamHoi}, min(15), "vi"),
	}
	_, err := repo.InsertAll(context.Background(), seeds)
	require.NoError(t, err)
}

func TestSutraRepository_ListAndCount_Filters(t *testing.T) {
	repo, _ := newSutraRepo(t)
	ctx := context.Background()
	listFixture(t, repo)

	trad := func(v domain.Tradition) *domain.Tradition { return &v }
	kind := func(v domain.Kind) *domain.Kind { return &v }
	purpose := func(v domain.Purpose) *domain.Purpose { return &v }
	str := func(v string) *string { return &v }
	boolp := func(v bool) *bool { return &v }
	intp := func(v int) *int { return &v }

	tests := []struct {
		name   string
		filter domain.SutraFilter
		want   []string
	}{
		{"no filter lists everything in curated order", domain.SutraFilter{}, []string{"a", "b", "c", "d", "e", "f"}},
		{"tradition", domain.SutraFilter{Tradition: trad(domain.TraditionMahayana)}, []string{"c", "d", "f"}},
		{"kind", domain.SutraFilter{Kind: kind(domain.KindDharani)}, []string{"d", "e"}},
		{"collection is an exact match", domain.SutraFilter{Collection: str("Bát Nhã")}, []string{"c"}},
		{"collection with no match", domain.SutraFilter{Collection: str("Bát")}, nil},
		{"purpose matches one of several", domain.SutraFilter{Purpose: purpose(domain.PurposeCauAn)}, []string{"b", "c"}},
		{"daily true", domain.SutraFilter{Daily: boolp(true)}, []string{"b", "c", "d", "f"}},
		{"daily false", domain.SutraFilter{Daily: boolp(false)}, []string{"a", "e"}},
		{"max_minutes keeps <= and drops a NULL estimate", domain.SutraFilter{MaxMinutes: intp(15)}, []string{"b", "c", "d", "f"}},
		{"max_minutes tighter", domain.SutraFilter{MaxMinutes: intp(5)}, []string{"b", "d"}},
		{"lang", domain.SutraFilter{Lang: str("vi")}, []string{"a", "b", "c", "d", "f"}},
		{"lang held by one sutra only", domain.SutraFilter{Lang: str("bo")}, []string{"e"}},
		{"lang nobody has", domain.SutraFilter{Lang: str("fr")}, nil},
		{"filters combine with AND", domain.SutraFilter{Tradition: trad(domain.TraditionMahayana), Daily: boolp(true), MaxMinutes: intp(15), Lang: str("zh")}, []string{"c"}},
		{"filters that exclude each other", domain.SutraFilter{Tradition: trad(domain.TraditionVajrayana), Daily: boolp(true)}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := repo.List(ctx, tt.filter, 0, 100)
			require.NoError(t, err)
			assert.NotNil(t, items)
			assert.Equal(t, tt.want, nonEmptyIDs(seedIDs(items)))

			n, err := repo.Count(ctx, tt.filter)
			require.NoError(t, err)
			assert.Equal(t, len(tt.want), n, "Count must agree with List for the same filter")
		})
	}
}

func TestSutraRepository_Search(t *testing.T) {
	repo, _ := newSutraRepo(t)
	ctx := context.Background()

	mk := func(id string, order int, coll, ref string, titles map[string]string) domain.SutraSeed {
		s := mkSutraSeed(id, order)
		s.Sutra.Collection, s.Sutra.Reference, s.Sutra.Titles = coll, ref, titles
		return s
	}
	_, err := repo.InsertAll(ctx, []domain.SutraSeed{
		mk("tam-kinh", 0, "Bát Nhã", "T0251", map[string]string{"vi": "Bát Nhã Ba La Mật Đa Tâm Kinh", "en": "Heart Sutra"}),
		mk("kinh-kalama", 1, "Tăng Chi Bộ", "AN 3.65", map[string]string{"vi": "Kinh Kālāma", "pli": "Kālāmasutta"}),
		mk("duoc-su", 2, "Dược Sư", "T0450", map[string]string{"vi": "Kinh Dược Sư"}),
		mk("percent", 3, "", "50%_off", map[string]string{"vi": "Kinh Đặc Biệt"}),
	})
	require.NoError(t, err)

	str := func(v string) *string { return &v }
	trad := domain.TraditionMahayana
	tests := []struct {
		name   string
		filter domain.SutraFilter
		want   []string
	}{
		{"case-insensitive", domain.SutraFilter{Query: str("HEART")}, []string{"tam-kinh"}},
		{"no accents matches accented title", domain.SutraFilter{Query: str("bat nha ba la")}, []string{"tam-kinh"}},
		{"accents match unaccented stored text", domain.SutraFilter{Query: str("dược sư")}, []string{"duoc-su"}},
		{"đ folds to d", domain.SutraFilter{Query: str("dac biet")}, []string{"percent"}},
		{"any title language", domain.SutraFilter{Query: str("kalamasutta")}, []string{"kinh-kalama"}},
		{"collection", domain.SutraFilter{Query: str("tang chi")}, []string{"kinh-kalama"}},
		{"reference", domain.SutraFilter{Query: str("an 3.65")}, []string{"kinh-kalama"}},
		{"substring across several rows keeps curated order", domain.SutraFilter{Query: str("kinh")}, []string{"tam-kinh", "kinh-kalama", "duoc-su", "percent"}},
		{"json key is not searched", domain.SutraFilter{Query: str("pli")}, nil},
		{"LIKE wildcards are literal", domain.SutraFilter{Query: str("%")}, []string{"percent"}},
		{"underscore is literal", domain.SutraFilter{Query: str("a_b")}, nil},
		{"no match", domain.SutraFilter{Query: str("zzz")}, nil},
		{"combines with another filter", domain.SutraFilter{Query: str("kinh"), Tradition: &trad, Collection: str("Dược Sư")}, []string{"duoc-su"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := repo.List(ctx, tt.filter, 0, 100)
			require.NoError(t, err)
			assert.Equal(t, tt.want, nonEmptyIDs(seedIDs(items)))
			n, err := repo.Count(ctx, tt.filter)
			require.NoError(t, err)
			assert.Equal(t, len(tt.want), n, "total must count the filtered result")
		})
	}
}

func TestSutraRepository_AudioURLDefaultsToEmptyString(t *testing.T) {
	repo, db := newSutraRepo(t)
	ctx := context.Background()
	_, err := repo.InsertAll(ctx, []domain.SutraSeed{mkSutraSeed("a", 0), mkSutraSeed("b", 1)})
	require.NoError(t, err)
	_, err = db.Exec(ctx, `UPDATE sutras SET audio_url = 'https://cdn/a.mp3' WHERE id = 'a'`)
	require.NoError(t, err)

	items, err := repo.List(ctx, domain.SutraFilter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "https://cdn/a.mp3", items[0].AudioURL)
	assert.Equal(t, "", items[1].AudioURL, "no recording is '' (never NULL)")

	one, err := repo.FindByID(ctx, "a")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn/a.mp3", one.AudioURL)
}

// nonEmptyIDs maps an empty slice to nil so a table can say "no match" with nil.
func nonEmptyIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func TestSutraRepository_List_PagingAndShape(t *testing.T) {
	repo, db := newSutraRepo(t)
	ctx := context.Background()
	insertTestDeity(t, db, "bodhisattva_a", 1, "Quán Thế Âm", "t.jpg")
	_, err := repo.InsertAll(ctx, []domain.SutraSeed{
		mkSutraSeed("s0", 0, withDeities(domain.SutraDeity{DeityID: "bodhisattva_a", Role: domain.RoleSubject})),
		mkSutraSeed("s1", 1), mkSutraSeed("s2", 2), mkSutraSeed("s3", 3), mkSutraSeed("s4", 4),
	})
	require.NoError(t, err)

	t.Run("windows do not repeat or skip", func(t *testing.T) {
		p1, err := repo.List(ctx, domain.SutraFilter{}, 0, 2)
		require.NoError(t, err)
		p2, err := repo.List(ctx, domain.SutraFilter{}, 2, 2)
		require.NoError(t, err)
		p3, err := repo.List(ctx, domain.SutraFilter{}, 4, 2)
		require.NoError(t, err)
		assert.Equal(t, []string{"s0", "s1"}, seedIDs(p1))
		assert.Equal(t, []string{"s2", "s3"}, seedIDs(p2))
		assert.Equal(t, []string{"s4"}, seedIDs(p3))
	})
	t.Run("an offset past the end is empty, not an error", func(t *testing.T) {
		got, err := repo.List(ctx, domain.SutraFilter{}, 999, 20)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
	t.Run("list shape: translations without sources or sections, deities with names", func(t *testing.T) {
		got, err := repo.List(ctx, domain.SutraFilter{}, 0, 1)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Len(t, got[0].Translations, 2)
		vi := got[0].Translations[0]
		assert.Equal(t, "vi-chant", vi.Key)
		assert.Equal(t, "T", vi.Translator)
		assert.Equal(t, domain.LicenseUnknown, vi.License)
		assert.Equal(t, 4, vi.LineCount)
		assert.Empty(t, vi.LicenseURL, "licence URL and note are detail-only")
		assert.Empty(t, vi.LicenseNote)
		assert.Equal(t, "Tóm tắt "+got[0].ID+".", got[0].Summary, "the card carries the summary")
		assert.Empty(t, got[0].OriginalTitle)
		assert.Nil(t, got[0].HanTranslationYear)
		assert.Empty(t, vi.SourceURL, "sources are detail-only")
		assert.Empty(t, vi.Note)
		assert.Empty(t, vi.Sections)
		assert.Equal(t, []domain.SutraDeity{{DeityID: "bodhisattva_a", Role: domain.RoleSubject, NameVI: "Quán Thế Âm", ThumbnailURL: "t.jpg"}}, got[0].Deities)
	})
	t.Run("each sutra gets only its own translations and deities", func(t *testing.T) {
		got, err := repo.List(ctx, domain.SutraFilter{}, 0, 5)
		require.NoError(t, err)
		for _, s := range got {
			assert.Len(t, s.Translations, 2, s.ID)
			if s.ID == "s0" {
				assert.Len(t, s.Deities, 1)
			} else {
				assert.Empty(t, s.Deities, s.ID)
			}
		}
	})
}

func TestSutraRepository_ByDeity(t *testing.T) {
	repo, db := newSutraRepo(t)
	ctx := context.Background()
	insertTestDeity(t, db, "bodhisattva_a", 1, "Quán Thế Âm", "a.jpg")
	insertTestDeity(t, db, "buddha_b", 2, "Thích Ca", "b.jpg")
	insertTestDeity(t, db, "lonely", 3, "Chưa có kinh", "")

	link := func(d string, r domain.DeityRole) domain.SutraDeity { return domain.SutraDeity{DeityID: d, Role: r} }
	_, err := repo.InsertAll(ctx, []domain.SutraSeed{
		mkSutraSeed("s0", 0, withDeities(link("bodhisattva_a", domain.RoleMentioned))),
		mkSutraSeed("s1", 1, withDeities(link("bodhisattva_a", domain.RoleSpeaker), link("buddha_b", domain.RoleSubject))),
		mkSutraSeed("s2", 2, withDeities(link("bodhisattva_a", domain.RoleSubject))),
		mkSutraSeed("s3", 3, withDeities(link("bodhisattva_a", domain.RoleSubject))),
		mkSutraSeed("s4", 4, withDeities(link("buddha_b", domain.RoleSpeaker))),
	})
	require.NoError(t, err)

	role := func(r domain.DeityRole) *domain.DeityRole { return &r }

	t.Run("ordered by role, then curated order", func(t *testing.T) {
		got, err := repo.ListByDeity(ctx, "bodhisattva_a", nil, 0, 10)
		require.NoError(t, err)
		require.Len(t, got, 4)
		var ids []string
		var roles []domain.DeityRole
		for _, g := range got {
			ids, roles = append(ids, g.ID), append(roles, g.Role)
		}
		assert.Equal(t, []string{"s2", "s3", "s1", "s0"}, ids)
		assert.Equal(t, []domain.DeityRole{domain.RoleSubject, domain.RoleSubject, domain.RoleSpeaker, domain.RoleMentioned}, roles)
	})
	t.Run("count agrees with the list", func(t *testing.T) {
		n, err := repo.CountByDeity(ctx, "bodhisattva_a", nil)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
	})
	t.Run("role filter", func(t *testing.T) {
		n, err := repo.CountByDeity(ctx, "bodhisattva_a", role(domain.RoleSubject))
		require.NoError(t, err)
		assert.Equal(t, 2, n)
		got, err := repo.ListByDeity(ctx, "bodhisattva_a", role(domain.RoleSubject), 0, 10)
		require.NoError(t, err)
		assert.Equal(t, []string{"s2", "s3"}, []string{got[0].ID, got[1].ID})
	})
	t.Run("windows", func(t *testing.T) {
		p1, err := repo.ListByDeity(ctx, "bodhisattva_a", nil, 0, 3)
		require.NoError(t, err)
		p2, err := repo.ListByDeity(ctx, "bodhisattva_a", nil, 3, 3)
		require.NoError(t, err)
		assert.Len(t, p1, 3)
		require.Len(t, p2, 1)
		assert.Equal(t, "s0", p2[0].ID)
	})
	t.Run("items are in list shape and carry every deity of the sutra, not just the one asked about", func(t *testing.T) {
		got, err := repo.ListByDeity(ctx, "buddha_b", role(domain.RoleSubject), 0, 10)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "s1", got[0].ID)
		assert.Equal(t, domain.RoleSubject, got[0].Role)
		require.Len(t, got[0].Deities, 2)
		assert.Equal(t, []string{"buddha_b", "bodhisattva_a"}, []string{got[0].Deities[0].DeityID, got[0].Deities[1].DeityID}, "SUBJECT before SPEAKER")
		assert.Len(t, got[0].Translations, 2)
		assert.Empty(t, got[0].Translations[0].SourceURL)
	})
	t.Run("a deity with no sutras", func(t *testing.T) {
		n, err := repo.CountByDeity(ctx, "lonely", nil)
		require.NoError(t, err)
		assert.Equal(t, 0, n)
		got, err := repo.ListByDeity(ctx, "lonely", nil, 0, 10)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
	t.Run("an offset past the end is empty", func(t *testing.T) {
		got, err := repo.ListByDeity(ctx, "bodhisattva_a", nil, 50, 10)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestSutraRepository_MissingDeities(t *testing.T) {
	repo, db := newSutraRepo(t)
	ctx := context.Background()
	insertTestDeity(t, db, "have-1", 1, "Một", "")
	insertTestDeity(t, db, "have-2", 2, "Hai", "")

	got, err := repo.MissingDeities(ctx, []string{"have-1", "gone-1", "have-2", "gone-2"})
	require.NoError(t, err)
	assert.Equal(t, []string{"gone-1", "gone-2"}, got, "in the order asked")

	got, err = repo.MissingDeities(ctx, []string{"have-1", "have-2"})
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestSutraRepository_ShippedData seeds the real embedded library into a real
// Postgres and checks what the API will read back. It also reports how long the
// first-run seed takes (tasks.md Phase 5).
func TestSutraRepository_ShippedData(t *testing.T) {
	repo, db := newSutraRepo(t)
	ctx := context.Background()

	sub, err := fs.Sub(assets.SutrasFS, "data/sutras")
	require.NoError(t, err)
	seeds, err := ParseSutraSeed(sub)
	require.NoError(t, err)

	// The seed's foreign keys need the 18 deities the shipped data links to.
	for i, id := range seedDeityIDsForTest(seeds) {
		insertTestDeity(t, db, id, i, "Tên "+id, "")
	}

	start := time.Now()
	res, err := repo.InsertAll(ctx, seeds)
	require.NoError(t, err)
	t.Logf("seeded %d sutras, %d translations, %d lines, %d deity links in %s",
		res.Sutras, res.Translations, res.Lines, res.DeityLinks, time.Since(start).Round(time.Millisecond))
	assert.Equal(t, domain.SutraSeedResult{Sutras: 153, Translations: 391, Lines: 64009, DeityLinks: 191}, res)

	n, err := repo.Count(ctx, domain.SutraFilter{})
	require.NoError(t, err)
	assert.Equal(t, 153, n)

	// The split fields and the summary survive the round trip through the real library.
	adida, err := repo.FindByID(ctx, "a-di-da")
	require.NoError(t, err)
	assert.Equal(t, "Taishō T0366", adida.Reference)
	assert.Equal(t, "Cưu Ma La Thập", adida.HanTranslator)
	require.NotNil(t, adida.HanTranslationYear)
	assert.Equal(t, 402, *adida.HanTranslationYear)
	assert.NotEmpty(t, adida.Summary)
	var emptySummaries int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM sutras WHERE summary = ''`).Scan(&emptySummaries))
	assert.Equal(t, 0, emptySummaries, "every shipped text has a summary")
	var badLicenses int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM sutra_translations WHERE license NOT IN ('CC0-1.0','CC-BY-NC-4.0','PUBLIC_DOMAIN','COPY_ALLOWED','UNKNOWN')`).Scan(&badLicenses))
	assert.Equal(t, 0, badLicenses)

	// Every translation's stored line_count equals its rows, and n runs 1..count.
	var broken int
	require.NoError(t, db.QueryRow(ctx, `
		SELECT count(*) FROM sutra_translations t
		WHERE t.line_count <> (SELECT count(*) FROM sutra_lines l WHERE l.translation_id = t.id)
		   OR t.line_count <> (SELECT max(n) FROM sutra_lines l WHERE l.translation_id = t.id)`).Scan(&broken))
	assert.Equal(t, 0, broken, "line_count must equal the number of rows and the highest n")

	// The longest text is paged by range exactly as the spec describes.
	long, err := repo.FindByID(ctx, "phap-hoa")
	require.NoError(t, err)
	var key string
	for _, tr := range long.Translations {
		if tr.Lang == "vi" {
			key = tr.Key
			assert.Equal(t, 6967, tr.LineCount)
			assert.NotEmpty(t, tr.Sections, "Pháp Hoa has a table of contents")
		}
	}
	tr, err := repo.FindTranslation(ctx, "phap-hoa", key)
	require.NoError(t, err)
	page35, err := repo.ListLines(ctx, tr.ID, 34*200, 35*200)
	require.NoError(t, err)
	assert.Len(t, page35, 6967-34*200, "page 35 of 200 is the short last page")
	assert.Equal(t, 6967, page35[len(page35)-1].N)

	// Reverse lookup over the real links: Shakyamuni is in 81 texts.
	cnt, err := repo.CountByDeity(ctx, "buddha_shakyamuni", nil)
	require.NoError(t, err)
	assert.Equal(t, 113, cnt)

	// Seeding twice must write nothing the second time (ON CONFLICT DO NOTHING).
	again, err := repo.InsertAll(ctx, seeds)
	require.NoError(t, err)
	assert.Equal(t, domain.SutraSeedResult{}, again)
}

func seedDeityIDsForTest(seeds []domain.SutraSeed) []string {
	seen := map[string]bool{}
	var ids []string
	for _, s := range seeds {
		for _, d := range s.Sutra.Deities {
			if !seen[d.DeityID] {
				seen[d.DeityID] = true
				ids = append(ids, d.DeityID)
			}
		}
	}
	return ids
}
