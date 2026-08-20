package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newMockSutraRepo(t *testing.T) (domain.SutraRepository, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return NewSutraRepository(&fakeDatabase{db: sqlDB}), mock
}

var sutraColumns = []string{"id", "slug", "title", "sect", "language", "body", "audio_url", "version", "created_at", "updated_at"}

// TestListSutras_Success covers AC1: the full list, no filters, is returned.
func TestListSutras_Success(t *testing.T) {
	repo, mock := newMockSutraRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at FROM sutras ORDER BY updated_at ASC`)).
		WillReturnRows(sqlmock.NewRows(sutraColumns).
			AddRow("sutra-1", "kinh-phap-hoa", "Kinh Pháp Hoa", "Bắc Tông", "vi", "body text", "https://cdn.example.com/a.mp3", 1, now, now).
			AddRow("sutra-2", "kinh-tu-niem-xu", "Kinh Tứ Niệm Xứ", "Nam Tông", "vi", "body text 2", nil, 1, now, now))

	got, err := repo.ListSutras(context.Background(), domain.ListSutrasInput{})
	if err != nil {
		t.Fatalf("ListSutras() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListSutras() returned %d sutras, want 2", len(got))
	}
	if got[0].ID != "sutra-1" || got[0].AudioURL != "https://cdn.example.com/a.mp3" {
		t.Errorf("ListSutras()[0] = %+v, unexpected fields", got[0])
	}
	// nullable audio_url must surface as "" rather than panicking on scan.
	if got[1].AudioURL != "" {
		t.Errorf("ListSutras()[1].AudioURL = %q, want empty", got[1].AudioURL)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListSutras_EmptyResultReturnsEmptySlice ensures no rows produces an
// empty (non-nil) slice, not nil — client cache-sync code shouldn't have to
// special-case nil vs empty.
func TestListSutras_EmptyResultReturnsEmptySlice(t *testing.T) {
	repo, mock := newMockSutraRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at FROM sutras ORDER BY updated_at ASC`)).
		WillReturnRows(sqlmock.NewRows(sutraColumns))

	got, err := repo.ListSutras(context.Background(), domain.ListSutrasInput{})
	if err != nil {
		t.Fatalf("ListSutras() unexpected error: %v", err)
	}
	if got == nil {
		t.Error("ListSutras() = nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Errorf("ListSutras() returned %d sutras, want 0", len(got))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListSutras_DeltaSync covers AC3/AC6: UpdatedSince filters to only
// sutras changed after that point.
func TestListSutras_DeltaSync(t *testing.T) {
	repo, mock := newMockSutraRepo(t)
	now := time.Now()
	since := now.Add(-24 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at FROM sutras WHERE updated_at > $1 ORDER BY updated_at ASC`)).
		WithArgs(since).
		WillReturnRows(sqlmock.NewRows(sutraColumns).
			AddRow("sutra-3", "kinh-kim-cang", "Kinh Kim Cang", "Kim Cương Thừa", "vi", "body", nil, 2, now, now))

	got, err := repo.ListSutras(context.Background(), domain.ListSutrasInput{UpdatedSince: since})
	if err != nil {
		t.Fatalf("ListSutras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "sutra-3" {
		t.Errorf("ListSutras() = %+v, want only sutra-3", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListSutras_SectFilter covers AC5.
func TestListSutras_SectFilter(t *testing.T) {
	repo, mock := newMockSutraRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at FROM sutras WHERE sect = $1 ORDER BY updated_at ASC`)).
		WithArgs("Nam Tông").
		WillReturnRows(sqlmock.NewRows(sutraColumns).
			AddRow("sutra-2", "kinh-tu-niem-xu", "Kinh Tứ Niệm Xứ", "Nam Tông", "vi", "body", nil, 1, now, now))

	got, err := repo.ListSutras(context.Background(), domain.ListSutrasInput{Sect: "Nam Tông"})
	if err != nil {
		t.Fatalf("ListSutras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Sect != "Nam Tông" {
		t.Errorf("ListSutras() = %+v, want only Nam Tông sutras", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListSutras_DeltaSyncAndSectFilter covers AC5+AC6 combined: both
// filters can be applied together.
func TestListSutras_DeltaSyncAndSectFilter(t *testing.T) {
	repo, mock := newMockSutraRepo(t)
	now := time.Now()
	since := now.Add(-24 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at FROM sutras WHERE updated_at > $1 AND sect = $2 ORDER BY updated_at ASC`)).
		WithArgs(since, "Bắc Tông").
		WillReturnRows(sqlmock.NewRows(sutraColumns))

	if _, err := repo.ListSutras(context.Background(), domain.ListSutrasInput{UpdatedSince: since, Sect: "Bắc Tông"}); err != nil {
		t.Fatalf("ListSutras() unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListSutras_QueryError propagates a query-level failure (e.g. DB
// unreachable) instead of swallowing it — needed for AC4's client-side
// offline fallback to actually see an error to fall back from.
func TestListSutras_QueryError(t *testing.T) {
	repo, mock := newMockSutraRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at FROM sutras ORDER BY updated_at ASC`)).
		WillReturnError(errors.New("connection reset"))

	_, err := repo.ListSutras(context.Background(), domain.ListSutrasInput{})
	if err == nil {
		t.Fatal("ListSutras() expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetSutraBySlug_Found covers US-006 AC1: full detail (including body)
// is returned for a known slug.
func TestGetSutraBySlug_Found(t *testing.T) {
	repo, mock := newMockSutraRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at`)).
		WithArgs("kinh-phap-hoa").
		WillReturnRows(sqlmock.NewRows(sutraColumns).
			AddRow("sutra-1", "kinh-phap-hoa", "Kinh Pháp Hoa", "Bắc Tông", "vi", "full body text", "https://cdn.example.com/a.mp3", 2, now, now))

	got, err := repo.GetSutraBySlug(context.Background(), "kinh-phap-hoa")
	if err != nil {
		t.Fatalf("GetSutraBySlug() unexpected error: %v", err)
	}
	if got.Slug != "kinh-phap-hoa" || got.Body != "full body text" || got.Version != 2 {
		t.Errorf("GetSutraBySlug() = %+v, unexpected fields", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetSutraBySlug_NullAudioURL covers AC2: a sutra with no audio_url
// scans cleanly to "" rather than erroring.
func TestGetSutraBySlug_NullAudioURL(t *testing.T) {
	repo, mock := newMockSutraRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at`)).
		WithArgs("kinh-tu-niem-xu").
		WillReturnRows(sqlmock.NewRows(sutraColumns).
			AddRow("sutra-2", "kinh-tu-niem-xu", "Kinh Tứ Niệm Xứ", "Nam Tông", "vi", "body", nil, 1, now, now))

	got, err := repo.GetSutraBySlug(context.Background(), "kinh-tu-niem-xu")
	if err != nil {
		t.Fatalf("GetSutraBySlug() unexpected error: %v", err)
	}
	if got.AudioURL != "" {
		t.Errorf("GetSutraBySlug().AudioURL = %q, want empty", got.AudioURL)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetSutraBySlug_NotFound covers an unknown slug mapping to
// domain.ErrSutraNotFound rather than a raw sql.ErrNoRows leaking upward.
func TestGetSutraBySlug_NotFound(t *testing.T) {
	repo, mock := newMockSutraRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, language, body, audio_url, version, created_at, updated_at`)).
		WithArgs("ghost-sutra").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetSutraBySlug(context.Background(), "ghost-sutra")
	if !errors.Is(err, domain.ErrSutraNotFound) {
		t.Fatalf("GetSutraBySlug() error = %v, want %v", err, domain.ErrSutraNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
