package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
	"chjpcoj.com.innerlamp/services/appService/usecase"
)

func newUsecase(repo *domain.MockSutraRepository) *usecase.SutraUsecase {
	return usecase.NewSutraUsecase(repo, nil, config.SutraConfig{PageSize: 20, LinesPageSize: 200, LinesMaxPageSize: 500})
}

func TestRun_SeedsTheEmbeddedDataWhenEmpty(t *testing.T) {
	repo := new(domain.MockSutraRepository)
	repo.On("Count", mock.Anything, domain.SutraFilter{}).Return(0, nil)
	repo.On("MissingDeities", mock.Anything, mock.Anything).Return(nil, nil)
	var got []domain.SutraSeed
	repo.On("InsertAll", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		got = args.Get(1).([]domain.SutraSeed)
	}).Return(domain.SutraSeedResult{Sutras: 153}, nil)

	res, err := run(context.Background(), newUsecase(repo), "")

	require.NoError(t, err)
	assert.Equal(t, 153, res.Sutras)
	assert.Len(t, got, 153, "the whole embedded library is handed to the repository")
}

func TestRun_DoesNothingWhenAlreadySeeded(t *testing.T) {
	repo := new(domain.MockSutraRepository)
	repo.On("Count", mock.Anything, domain.SutraFilter{}).Return(153, nil)

	// A directory that does not exist proves the data is not even read.
	res, err := run(context.Background(), newUsecase(repo), filepath.Join(t.TempDir(), "missing"))

	require.NoError(t, err, "an already-seeded database is success, not an error")
	assert.Equal(t, domain.SutraSeedResult{}, res)
	repo.AssertNotCalled(t, "InsertAll", mock.Anything, mock.Anything)
	assert.Contains(t, report(res, ""), "already seeded")
}

func TestRun_ReadsADirectoryWhenGiven(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "index.json"), map[string]any{"sutras": []map[string]any{{"id": "only-one"}}})
	writeJSON(t, filepath.Join(dir, "only-one.json"), map[string]any{
		"id": "only-one", "tradition": "THERAVADA", "kind": "SUTRA", "collection": "c", "reference": "",
		"titles": map[string]any{"vi": "Chỉ một"}, "deities": []any{}, "dailyChant": false, "purposes": []string{},
		"durationMinutes": nil, "reviewStatus": "UNREVIEWED",
		"translations": []map[string]any{{"key": "vi-translation", "lang": "vi", "variant": "translation", "license": "UNKNOWN", "lines": []map[string]any{{"n": 1, "text": "x"}}}},
	})
	repo := new(domain.MockSutraRepository)
	repo.On("Count", mock.Anything, domain.SutraFilter{}).Return(0, nil)
	var got []domain.SutraSeed
	repo.On("InsertAll", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		got = args.Get(1).([]domain.SutraSeed)
	}).Return(domain.SutraSeedResult{Sutras: 1, Translations: 1, Lines: 1}, nil)

	res, err := run(context.Background(), newUsecase(repo), dir)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "only-one", got[0].Sutra.ID)
	assert.Contains(t, report(res, dir), dir, "the report names where the data came from")
}

func TestRun_FailsOnABadDirectoryWithoutWriting(t *testing.T) {
	for name, dir := range map[string]string{
		"missing":      filepath.Join(t.TempDir(), "missing"),
		"not a folder": writeFile(t, "notes.txt"),
		"no index":     t.TempDir(),
	} {
		t.Run(name, func(t *testing.T) {
			repo := new(domain.MockSutraRepository)
			repo.On("Count", mock.Anything, domain.SutraFilter{}).Return(0, nil)

			_, err := run(context.Background(), newUsecase(repo), dir)

			require.Error(t, err)
			repo.AssertNotCalled(t, "InsertAll", mock.Anything, mock.Anything)
		})
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0o600))
}

func writeFile(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, []byte("x"), 0o600))
	return p
}
