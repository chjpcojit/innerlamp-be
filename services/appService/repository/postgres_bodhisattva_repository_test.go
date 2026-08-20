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

func newMockBodhisattvaRepo(t *testing.T) (domain.BodhisattvaRepository, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return NewBodhisattvaRepository(&fakeDatabase{db: sqlDB}), mock
}

var bodhisattvaColumns = []string{"id", "slug", "name", "description", "image_url", "version", "created_at", "updated_at"}

// TestListBodhisattvas_Success covers AC1: the full list, no filters, is
// returned.
func TestListBodhisattvas_Success(t *testing.T) {
	repo, mock := newMockBodhisattvaRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, name, description, image_url, version, created_at, updated_at FROM bodhisattvas ORDER BY updated_at ASC`)).
		WillReturnRows(sqlmock.NewRows(bodhisattvaColumns).
			AddRow("bodhisattva-1", "quan-the-am", "Quán Thế Âm Bồ Tát", "desc", "https://cdn.example.com/qta.png", 1, now, now).
			AddRow("bodhisattva-2", "dia-tang", "Địa Tạng Vương Bồ Tát", "desc 2", nil, 1, now, now))

	got, err := repo.ListBodhisattvas(context.Background(), domain.ListBodhisattvasInput{})
	if err != nil {
		t.Fatalf("ListBodhisattvas() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListBodhisattvas() returned %d bodhisattvas, want 2", len(got))
	}
	if got[0].ID != "bodhisattva-1" || got[0].ImageURL != "https://cdn.example.com/qta.png" {
		t.Errorf("ListBodhisattvas()[0] = %+v, unexpected fields", got[0])
	}
	// nullable image_url must surface as "" rather than panicking on scan —
	// AC2.
	if got[1].ImageURL != "" {
		t.Errorf("ListBodhisattvas()[1].ImageURL = %q, want empty", got[1].ImageURL)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListBodhisattvas_EmptyResultReturnsEmptySlice ensures no rows produces
// an empty (non-nil) slice, not nil.
func TestListBodhisattvas_EmptyResultReturnsEmptySlice(t *testing.T) {
	repo, mock := newMockBodhisattvaRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, name, description, image_url, version, created_at, updated_at FROM bodhisattvas ORDER BY updated_at ASC`)).
		WillReturnRows(sqlmock.NewRows(bodhisattvaColumns))

	got, err := repo.ListBodhisattvas(context.Background(), domain.ListBodhisattvasInput{})
	if err != nil {
		t.Fatalf("ListBodhisattvas() unexpected error: %v", err)
	}
	if got == nil {
		t.Error("ListBodhisattvas() = nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Errorf("ListBodhisattvas() returned %d bodhisattvas, want 0", len(got))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListBodhisattvas_DeltaSync covers AC4.
func TestListBodhisattvas_DeltaSync(t *testing.T) {
	repo, mock := newMockBodhisattvaRepo(t)
	now := time.Now()
	since := now.Add(-24 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, name, description, image_url, version, created_at, updated_at FROM bodhisattvas WHERE updated_at > $1 ORDER BY updated_at ASC`)).
		WithArgs(since).
		WillReturnRows(sqlmock.NewRows(bodhisattvaColumns).
			AddRow("bodhisattva-3", "van-thu", "Văn Thù Bồ Tát", "desc", nil, 2, now, now))

	got, err := repo.ListBodhisattvas(context.Background(), domain.ListBodhisattvasInput{UpdatedSince: since})
	if err != nil {
		t.Fatalf("ListBodhisattvas() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "bodhisattva-3" {
		t.Errorf("ListBodhisattvas() = %+v, want only bodhisattva-3", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestListBodhisattvas_QueryError propagates a query-level failure instead
// of swallowing it.
func TestListBodhisattvas_QueryError(t *testing.T) {
	repo, mock := newMockBodhisattvaRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, name, description, image_url, version, created_at, updated_at FROM bodhisattvas ORDER BY updated_at ASC`)).
		WillReturnError(errors.New("connection reset"))

	_, err := repo.ListBodhisattvas(context.Background(), domain.ListBodhisattvasInput{})
	if err == nil {
		t.Fatal("ListBodhisattvas() expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetBodhisattvaBySlug_Found covers US-010 AC1: name, description, and
// image are returned for a known slug.
func TestGetBodhisattvaBySlug_Found(t *testing.T) {
	repo, mock := newMockBodhisattvaRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, name, description, image_url, version, created_at, updated_at`)).
		WithArgs("quan-the-am").
		WillReturnRows(sqlmock.NewRows(bodhisattvaColumns).
			AddRow("bodhisattva-1", "quan-the-am", "Quán Thế Âm Bồ Tát", "full description", "https://cdn.example.com/qta.png", 2, now, now))

	got, err := repo.GetBodhisattvaBySlug(context.Background(), "quan-the-am")
	if err != nil {
		t.Fatalf("GetBodhisattvaBySlug() unexpected error: %v", err)
	}
	if got.Slug != "quan-the-am" || got.Description != "full description" || got.Version != 2 {
		t.Errorf("GetBodhisattvaBySlug() = %+v, unexpected fields", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetBodhisattvaBySlug_NullImageURL covers AC2: a bodhisattva with no
// image_url scans cleanly to "" rather than erroring.
func TestGetBodhisattvaBySlug_NullImageURL(t *testing.T) {
	repo, mock := newMockBodhisattvaRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, name, description, image_url, version, created_at, updated_at`)).
		WithArgs("dia-tang").
		WillReturnRows(sqlmock.NewRows(bodhisattvaColumns).
			AddRow("bodhisattva-2", "dia-tang", "Địa Tạng Vương Bồ Tát", "desc", nil, 1, now, now))

	got, err := repo.GetBodhisattvaBySlug(context.Background(), "dia-tang")
	if err != nil {
		t.Fatalf("GetBodhisattvaBySlug() unexpected error: %v", err)
	}
	if got.ImageURL != "" {
		t.Errorf("GetBodhisattvaBySlug().ImageURL = %q, want empty", got.ImageURL)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetBodhisattvaBySlug_NotFound covers an unknown slug mapping to
// domain.ErrBodhisattvaNotFound rather than a raw sql.ErrNoRows leaking
// upward.
func TestGetBodhisattvaBySlug_NotFound(t *testing.T) {
	repo, mock := newMockBodhisattvaRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, slug, name, description, image_url, version, created_at, updated_at`)).
		WithArgs("ghost-bodhisattva").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetBodhisattvaBySlug(context.Background(), "ghost-bodhisattva")
	if !errors.Is(err, domain.ErrBodhisattvaNotFound) {
		t.Fatalf("GetBodhisattvaBySlug() error = %v, want %v", err, domain.ErrBodhisattvaNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
