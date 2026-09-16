package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

// TestListBodhisattvas_Success covers AC1: the full list from the
// repository is returned as-is.
func TestListBodhisattvas_Success(t *testing.T) {
	repo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	want := []*domain.Bodhisattva{
		{ID: "bodhisattva-1", Slug: "quan-the-am", Name: "Quán Thế Âm Bồ Tát", ImageURL: "https://cdn.example.com/qta.png"},
		{ID: "bodhisattva-2", Slug: "dia-tang", Name: "Địa Tạng Vương Bồ Tát"},
	}
	repo.On("ListBodhisattvas", mock.Anything, domain.ListBodhisattvasInput{}).Return(want, nil)

	uc := NewListBodhisattvasUsecase(repo, logger)

	got, err := uc.ListBodhisattvas(context.Background(), domain.ListBodhisattvasInput{})
	if err != nil {
		t.Fatalf("ListBodhisattvas() unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ListBodhisattvas() = %+v, want %+v", got, want)
	}
	repo.AssertExpectations(t)
}

// TestListBodhisattvas_NoImageURLNoError covers AC2: a bodhisattva with no
// image_url passes through without error — the client applies the
// placeholder.
func TestListBodhisattvas_NoImageURLNoError(t *testing.T) {
	repo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	want := []*domain.Bodhisattva{{ID: "bodhisattva-2", Slug: "dia-tang", ImageURL: ""}}
	repo.On("ListBodhisattvas", mock.Anything, domain.ListBodhisattvasInput{}).Return(want, nil)

	uc := NewListBodhisattvasUsecase(repo, logger)

	got, err := uc.ListBodhisattvas(context.Background(), domain.ListBodhisattvasInput{})
	if err != nil {
		t.Fatalf("ListBodhisattvas() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ImageURL != "" {
		t.Errorf("ListBodhisattvas() = %+v, want ImageURL empty", got)
	}
	repo.AssertExpectations(t)
}

// TestListBodhisattvas_DeltaSync covers AC4: a non-zero UpdatedSince is
// passed straight through for the repository to filter on.
func TestListBodhisattvas_DeltaSync(t *testing.T) {
	repo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	input := domain.ListBodhisattvasInput{UpdatedSince: since}
	updated := []*domain.Bodhisattva{{ID: "bodhisattva-3", Slug: "van-thu", UpdatedAt: since.Add(time.Hour)}}
	repo.On("ListBodhisattvas", mock.Anything, input).Return(updated, nil)

	uc := NewListBodhisattvasUsecase(repo, logger)

	got, err := uc.ListBodhisattvas(context.Background(), input)
	if err != nil {
		t.Fatalf("ListBodhisattvas() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "bodhisattva-3" {
		t.Errorf("ListBodhisattvas() = %+v, want only bodhisattva-3", got)
	}
	repo.AssertExpectations(t)
}

// TestListBodhisattvas_SectFilter covers US-009 AC2b: sect is passed through to repo.
func TestListBodhisattvas_SectFilter(t *testing.T) {
	repo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	input := domain.ListBodhisattvasInput{Sect: "bac_tong"}
	filtered := []*domain.Bodhisattva{{ID: "bodhisattva-1", Sect: "bac_tong"}}
	repo.On("ListBodhisattvas", mock.Anything, input).Return(filtered, nil)

	uc := NewListBodhisattvasUsecase(repo, logger)

	got, err := uc.ListBodhisattvas(context.Background(), input)
	if err != nil {
		t.Fatalf("ListBodhisattvas() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Sect != "bac_tong" {
		t.Errorf("ListBodhisattvas() = %+v, want only bac_tong bodhisattvas", got)
	}
	repo.AssertExpectations(t)
}

// TestListBodhisattvas_RepositoryErrorPropagates ensures repo errors
// surface unchanged rather than being swallowed.
func TestListBodhisattvas_RepositoryErrorPropagates(t *testing.T) {
	repo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	wantErr := errors.New("db unavailable")
	repo.On("ListBodhisattvas", mock.Anything, mock.Anything).Return(nil, wantErr)

	uc := NewListBodhisattvasUsecase(repo, logger)

	_, err := uc.ListBodhisattvas(context.Background(), domain.ListBodhisattvasInput{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("ListBodhisattvas() error = %v, want %v", err, wantErr)
	}
}
