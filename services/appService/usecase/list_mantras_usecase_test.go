package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

// TestListMantras_Success covers AC1: the full list from the repository is
// returned as-is.
func TestListMantras_Success(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	want := []*domain.Mantra{
		{ID: "mantra-1", Slug: "chu-dai-bi", Title: "Chú Đại Bi", BodhisattvaID: "bodhisattva-1"},
		{ID: "mantra-2", Slug: "om-mani-padme-hum", Title: "Om Mani Padme Hum"},
	}
	repo.On("ListMantras", mock.Anything, domain.ListMantrasInput{}).Return(want, nil)

	uc := NewListMantrasUsecase(repo, logger)

	got, err := uc.ListMantras(context.Background(), domain.ListMantrasInput{})
	if err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ListMantras() = %+v, want %+v", got, want)
	}
	repo.AssertExpectations(t)
}

// TestListMantras_NoBodhisattvaNoError covers AC2: a mantra with no
// bodhisattva_id passes through without error.
func TestListMantras_NoBodhisattvaNoError(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	want := []*domain.Mantra{{ID: "mantra-2", Slug: "om-mani-padme-hum", BodhisattvaID: ""}}
	repo.On("ListMantras", mock.Anything, domain.ListMantrasInput{}).Return(want, nil)

	uc := NewListMantrasUsecase(repo, logger)

	got, err := uc.ListMantras(context.Background(), domain.ListMantrasInput{})
	if err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].BodhisattvaID != "" {
		t.Errorf("ListMantras() = %+v, want BodhisattvaID empty", got)
	}
	repo.AssertExpectations(t)
}

// TestListMantras_BodhisattvaFilter covers AC3: a non-empty BodhisattvaID is
// passed straight through for the repository to filter on.
func TestListMantras_BodhisattvaFilter(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	input := domain.ListMantrasInput{BodhisattvaID: "bodhisattva-1"}
	filtered := []*domain.Mantra{{ID: "mantra-1", BodhisattvaID: "bodhisattva-1"}}
	repo.On("ListMantras", mock.Anything, input).Return(filtered, nil)

	uc := NewListMantrasUsecase(repo, logger)

	got, err := uc.ListMantras(context.Background(), input)
	if err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].BodhisattvaID != "bodhisattva-1" {
		t.Errorf("ListMantras() = %+v, want only bodhisattva-1 mantras", got)
	}
	repo.AssertExpectations(t)
}

// TestListMantras_DeltaSync covers AC5: a non-zero UpdatedSince is passed
// straight through for the repository to filter on.
func TestListMantras_DeltaSync(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	input := domain.ListMantrasInput{UpdatedSince: since}
	updated := []*domain.Mantra{{ID: "mantra-3", Slug: "chu-vang-sanh", UpdatedAt: since.Add(time.Hour)}}
	repo.On("ListMantras", mock.Anything, input).Return(updated, nil)

	uc := NewListMantrasUsecase(repo, logger)

	got, err := uc.ListMantras(context.Background(), input)
	if err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "mantra-3" {
		t.Errorf("ListMantras() = %+v, want only mantra-3", got)
	}
	repo.AssertExpectations(t)
}

// TestListMantras_SectFilter covers US-007 AC3b: sect is passed through to repo.
func TestListMantras_SectFilter(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	input := domain.ListMantrasInput{Sect: "kim_cuong_thua"}
	filtered := []*domain.Mantra{{ID: "mantra-2", Sect: "kim_cuong_thua"}}
	repo.On("ListMantras", mock.Anything, input).Return(filtered, nil)

	uc := NewListMantrasUsecase(repo, logger)

	got, err := uc.ListMantras(context.Background(), input)
	if err != nil {
		t.Fatalf("ListMantras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Sect != "kim_cuong_thua" {
		t.Errorf("ListMantras() = %+v, want only kim_cuong_thua mantras", got)
	}
	repo.AssertExpectations(t)
}

// TestListMantras_RepositoryErrorPropagates ensures repo errors surface
// unchanged rather than being swallowed.
func TestListMantras_RepositoryErrorPropagates(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	wantErr := errors.New("db unavailable")
	repo.On("ListMantras", mock.Anything, mock.Anything).Return(nil, wantErr)

	uc := NewListMantrasUsecase(repo, logger)

	_, err := uc.ListMantras(context.Background(), domain.ListMantrasInput{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("ListMantras() error = %v, want %v", err, wantErr)
	}
}
