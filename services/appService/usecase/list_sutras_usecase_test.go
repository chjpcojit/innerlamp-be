package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

// TestListSutras_Success covers AC1: the full list from the repository is
// returned as-is.
func TestListSutras_Success(t *testing.T) {
	repo := &domain.MockSutraRepository{}
	logger := newLogger()

	want := []*domain.Sutra{
		{ID: "sutra-1", Slug: "kinh-phap-hoa", Title: "Kinh Pháp Hoa", Sect: "Bắc Tông"},
		{ID: "sutra-2", Slug: "kinh-tu-niem-xu", Title: "Kinh Tứ Niệm Xứ", Sect: "Nam Tông"},
	}
	repo.On("ListSutras", mock.Anything, domain.ListSutrasInput{}).Return(want, nil)

	uc := NewListSutrasUsecase(repo, logger)

	got, err := uc.ListSutras(context.Background(), domain.ListSutrasInput{})
	if err != nil {
		t.Fatalf("ListSutras() unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ListSutras() = %+v, want %+v", got, want)
	}
	repo.AssertExpectations(t)
}

// TestListSutras_FullSync covers AC2: a zero-valued UpdatedSince (first
// open, no cache yet) is passed straight through so the repository returns
// the whole table.
func TestListSutras_FullSync(t *testing.T) {
	repo := &domain.MockSutraRepository{}
	logger := newLogger()

	repo.On("ListSutras", mock.Anything, domain.ListSutrasInput{}).Return([]*domain.Sutra{}, nil)

	uc := NewListSutrasUsecase(repo, logger)

	if _, err := uc.ListSutras(context.Background(), domain.ListSutrasInput{}); err != nil {
		t.Fatalf("ListSutras() unexpected error: %v", err)
	}
	repo.AssertExpectations(t)
}

// TestListSutras_DeltaSync covers AC3/AC6: a non-zero UpdatedSince is
// passed straight through for the repository to filter on.
func TestListSutras_DeltaSync(t *testing.T) {
	repo := &domain.MockSutraRepository{}
	logger := newLogger()

	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	input := domain.ListSutrasInput{UpdatedSince: since}
	updated := []*domain.Sutra{{ID: "sutra-3", Slug: "kinh-kim-cang", UpdatedAt: since.Add(time.Hour)}}
	repo.On("ListSutras", mock.Anything, input).Return(updated, nil)

	uc := NewListSutrasUsecase(repo, logger)

	got, err := uc.ListSutras(context.Background(), input)
	if err != nil {
		t.Fatalf("ListSutras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "sutra-3" {
		t.Errorf("ListSutras() = %+v, want only sutra-3", got)
	}
	repo.AssertExpectations(t)
}

// TestListSutras_SectFilter covers AC5: a non-empty Sect is passed straight
// through for the repository to filter on.
func TestListSutras_SectFilter(t *testing.T) {
	repo := &domain.MockSutraRepository{}
	logger := newLogger()

	input := domain.ListSutrasInput{Sect: "Kim Cương Thừa"}
	filtered := []*domain.Sutra{{ID: "sutra-4", Sect: "Kim Cương Thừa"}}
	repo.On("ListSutras", mock.Anything, input).Return(filtered, nil)

	uc := NewListSutrasUsecase(repo, logger)

	got, err := uc.ListSutras(context.Background(), input)
	if err != nil {
		t.Fatalf("ListSutras() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Sect != "Kim Cương Thừa" {
		t.Errorf("ListSutras() = %+v, want only Kim Cương Thừa sutras", got)
	}
	repo.AssertExpectations(t)
}

// TestListSutras_RepositoryErrorPropagates ensures repo errors surface
// unchanged rather than being swallowed — needed so the handler can map to
// a 5xx without the client mistaking it for "no results".
func TestListSutras_RepositoryErrorPropagates(t *testing.T) {
	repo := &domain.MockSutraRepository{}
	logger := newLogger()

	wantErr := errors.New("db unavailable")
	repo.On("ListSutras", mock.Anything, mock.Anything).Return(nil, wantErr)

	uc := NewListSutrasUsecase(repo, logger)

	_, err := uc.ListSutras(context.Background(), domain.ListSutrasInput{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("ListSutras() error = %v, want %v", err, wantErr)
	}
}
