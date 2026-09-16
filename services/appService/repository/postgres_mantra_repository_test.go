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

func newMockMantraRepo(t *testing.T) (domain.MantraRepository, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return NewMantraRepository(&fakeDatabase{db: sqlDB}), mock
}

var mantraColumns = []string{"id", "slug", "title", "sect", "body", "audio_url", "bodhisattva_id", "version", "created_at", "updated_at"}

// TestListMantras_Success covers AC1: the full list, no filters, is returned.
func TestListMantras_Success(t *testing.T) {
	repo, mock := newMockMantraRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, body, audio_url, bodhisattva_id, version, created_at, updated_at FROM mantras ORDER BY updated_at ASC`)).
		WillReturnRows(sqlmock.NewRows(mantraColumns).
			AddRow("mantra-1", "chu-dai-bi", "Chú Đại Bi", "bac_tong", "body text", "https://cdn.example.com/a.mp3", "bodhisattva-1", 1, now, now).
			AddRow("mantra-2", "om-mani-padme-hum", "Om Mani Padme Hum", nil, "body text 2", nil, nil, 1, now, now))

	got, err := repo.ListMantras(context.Background(), domain.ListMantrasInput{})
	if err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListMantras() returned %d mantras, want 2", len(got))
	}
	if got[0].ID != "mantra-1" || got[0].BodhisattvaID != "bodhisattva-1" || got[0].Sect != "bac_tong" {
		t.Errorf("ListMantras()[0] = %+v, unexpected fields", got[0])
	}
	// nullable sect/audio_url/bodhisattva_id must surface as "" rather than
	// panicking on scan — AC2.
	if got[1].Sect != "" || got[1].AudioURL != "" || got[1].BodhisattvaID != "" {
		t.Errorf("ListMantras()[1] = %+v, want Sect, AudioURL and BodhisattvaID empty", got[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListMantras_EmptyResultReturnsEmptySlice ensures no rows produces an
// empty (non-nil) slice, not nil.
func TestListMantras_EmptyResultReturnsEmptySlice(t *testing.T) {
	repo, mock := newMockMantraRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, body, audio_url, bodhisattva_id, version, created_at, updated_at FROM mantras ORDER BY updated_at ASC`)).
		WillReturnRows(sqlmock.NewRows(mantraColumns))

	got, err := repo.ListMantras(context.Background(), domain.ListMantrasInput{})
	if err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if got == nil {
		t.Error("ListMantras() = nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Errorf("ListMantras() returned %d mantras, want 0", len(got))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListMantras_BodhisattvaFilter covers AC3.
func TestListMantras_BodhisattvaFilter(t *testing.T) {
	repo, mock := newMockMantraRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, body, audio_url, bodhisattva_id, version, created_at, updated_at FROM mantras WHERE bodhisattva_id = $1 ORDER BY updated_at ASC`)).
		WithArgs("bodhisattva-1").
		WillReturnRows(sqlmock.NewRows(mantraColumns).
			AddRow("mantra-1", "chu-dai-bi", "Chú Đại Bi", "bac_tong", "body", nil, "bodhisattva-1", 1, now, now))

	got, err := repo.ListMantras(context.Background(), domain.ListMantrasInput{BodhisattvaID: "bodhisattva-1"})
	if err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].BodhisattvaID != "bodhisattva-1" {
		t.Errorf("ListMantras() = %+v, want only bodhisattva-1 mantras", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListMantras_SectFilter covers US-007 AC3b: filtering by sect.
func TestListMantras_SectFilter(t *testing.T) {
	repo, mock := newMockMantraRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, body, audio_url, bodhisattva_id, version, created_at, updated_at FROM mantras WHERE sect = $1 ORDER BY updated_at ASC`)).
		WithArgs("kim_cuong_thua").
		WillReturnRows(sqlmock.NewRows(mantraColumns).
			AddRow("mantra-2", "om-mani-padme-hum", "Om Mani Padme Hum", "kim_cuong_thua", "body", nil, nil, 1, now, now))

	got, err := repo.ListMantras(context.Background(), domain.ListMantrasInput{Sect: "kim_cuong_thua"})
	if err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Sect != "kim_cuong_thua" {
		t.Errorf("ListMantras() = %+v, want only kim_cuong_thua mantras", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListMantras_DeltaSyncAndBodhisattvaFilter covers AC3+AC5 combined:
// both filters can be applied together.
func TestListMantras_DeltaSyncAndBodhisattvaFilter(t *testing.T) {
	repo, mock := newMockMantraRepo(t)
	now := time.Now()
	since := now.Add(-24 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, body, audio_url, bodhisattva_id, version, created_at, updated_at FROM mantras WHERE updated_at > $1 AND bodhisattva_id = $2 ORDER BY updated_at ASC`)).
		WithArgs(since, "bodhisattva-1").
		WillReturnRows(sqlmock.NewRows(mantraColumns))

	if _, err := repo.ListMantras(context.Background(), domain.ListMantrasInput{UpdatedSince: since, BodhisattvaID: "bodhisattva-1"}); err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListMantras_QueryError propagates a query-level failure instead of
// swallowing it.
func TestListMantras_QueryError(t *testing.T) {
	repo, mock := newMockMantraRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, body, audio_url, bodhisattva_id, version, created_at, updated_at FROM mantras ORDER BY updated_at ASC`)).
		WillReturnError(errors.New("connection reset"))

	_, err := repo.ListMantras(context.Background(), domain.ListMantrasInput{})
	if err == nil {
		t.Fatal("ListMantras() expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetMantraBySlug_Found covers US-008 AC1: full detail (including body and sect)
// is returned for a known slug.
func TestGetMantraBySlug_Found(t *testing.T) {
	repo, mock := newMockMantraRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, body, audio_url, bodhisattva_id, version, created_at, updated_at`)).
		WithArgs("chu-dai-bi").
		WillReturnRows(sqlmock.NewRows(mantraColumns).
			AddRow("mantra-1", "chu-dai-bi", "Chú Đại Bi", "bac_tong", "full body text", "https://cdn.example.com/a.mp3", "bodhisattva-1", 2, now, now))

	got, err := repo.GetMantraBySlug(context.Background(), "chu-dai-bi")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if got.Slug != "chu-dai-bi" || got.Body != "full body text" || got.Version != 2 || got.Sect != "bac_tong" {
		t.Errorf("GetMantraBySlug() = %+v, unexpected fields", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetMantraBySlug_NullFields covers AC2: a mantra with no audio_url/
// bodhisattva_id/sect scans cleanly to "" rather than erroring.
func TestGetMantraBySlug_NullFields(t *testing.T) {
	repo, mock := newMockMantraRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, body, audio_url, bodhisattva_id, version, created_at, updated_at`)).
		WithArgs("om-mani-padme-hum").
		WillReturnRows(sqlmock.NewRows(mantraColumns).
			AddRow("mantra-2", "om-mani-padme-hum", "Om Mani Padme Hum", nil, "body", nil, nil, 1, now, now))

	got, err := repo.GetMantraBySlug(context.Background(), "om-mani-padme-hum")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if got.AudioURL != "" || got.BodhisattvaID != "" || got.Sect != "" {
		t.Errorf("GetMantraBySlug() = %+v, want Sect, AudioURL and BodhisattvaID empty", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetMantraBySlug_NotFound covers an unknown slug mapping to
// domain.ErrMantraNotFound rather than a raw sql.ErrNoRows leaking upward.
func TestGetMantraBySlug_NotFound(t *testing.T) {
	repo, mock := newMockMantraRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, title, sect, body, audio_url, bodhisattva_id, version, created_at, updated_at`)).
		WithArgs("ghost-mantra").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetMantraBySlug(context.Background(), "ghost-mantra")
	if !errors.Is(err, domain.ErrMantraNotFound) {
		t.Fatalf("GetMantraBySlug() error = %v, want %v", err, domain.ErrMantraNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
