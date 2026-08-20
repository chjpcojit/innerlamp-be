package usecase

import (
	"context"
	"errors"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

// TestGetSutraBySlug_Success covers AC1: full detail (title, sect, body) is
// returned as-is for a known slug.
func TestGetSutraBySlug_Success(t *testing.T) {
	repo := &domain.MockSutraRepository{}
	logger := newLogger()

	want := &domain.Sutra{ID: "sutra-1", Slug: "kinh-phap-hoa", Title: "Kinh Pháp Hoa", Sect: "Bắc Tông", Body: "full body text"}
	repo.On("GetSutraBySlug", mock.Anything, "kinh-phap-hoa").Return(want, nil)

	uc := NewGetSutraUsecase(repo, logger)

	got, err := uc.GetSutraBySlug(context.Background(), "kinh-phap-hoa")
	if err != nil {
		t.Fatalf("GetSutraBySlug() unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("GetSutraBySlug() = %+v, want %+v", got, want)
	}
	repo.AssertExpectations(t)
}

// TestGetSutraBySlug_AudioURLPassthrough covers AC2: audio_url (present or
// empty) passes through unchanged — the use-case doesn't special-case it.
func TestGetSutraBySlug_AudioURLPassthrough(t *testing.T) {
	repo := &domain.MockSutraRepository{}
	logger := newLogger()

	withAudio := &domain.Sutra{ID: "sutra-1", Slug: "kinh-phap-hoa", AudioURL: "https://cdn.example.com/a.mp3"}
	repo.On("GetSutraBySlug", mock.Anything, "kinh-phap-hoa").Return(withAudio, nil)

	withoutAudio := &domain.Sutra{ID: "sutra-2", Slug: "kinh-tu-niem-xu", AudioURL: ""}
	repo.On("GetSutraBySlug", mock.Anything, "kinh-tu-niem-xu").Return(withoutAudio, nil)

	uc := NewGetSutraUsecase(repo, logger)

	got1, err := uc.GetSutraBySlug(context.Background(), "kinh-phap-hoa")
	if err != nil {
		t.Fatalf("GetSutraBySlug() unexpected error: %v", err)
	}
	if got1.AudioURL != "https://cdn.example.com/a.mp3" {
		t.Errorf("GetSutraBySlug().AudioURL = %q, want the URL", got1.AudioURL)
	}

	got2, err := uc.GetSutraBySlug(context.Background(), "kinh-tu-niem-xu")
	if err != nil {
		t.Fatalf("GetSutraBySlug() unexpected error: %v", err)
	}
	if got2.AudioURL != "" {
		t.Errorf("GetSutraBySlug().AudioURL = %q, want empty", got2.AudioURL)
	}
	repo.AssertExpectations(t)
}

// TestGetSutraBySlug_NotFound propagates ErrSutraNotFound from the
// repository unchanged.
func TestGetSutraBySlug_NotFound(t *testing.T) {
	repo := &domain.MockSutraRepository{}
	logger := newLogger()

	repo.On("GetSutraBySlug", mock.Anything, "ghost-sutra").Return(nil, domain.ErrSutraNotFound)

	uc := NewGetSutraUsecase(repo, logger)

	_, err := uc.GetSutraBySlug(context.Background(), "ghost-sutra")
	if !errors.Is(err, domain.ErrSutraNotFound) {
		t.Fatalf("GetSutraBySlug() error = %v, want %v", err, domain.ErrSutraNotFound)
	}
}

// TestGetSutraBySlug_RepositoryErrorPropagates ensures other repo errors
// surface unchanged rather than being swallowed.
func TestGetSutraBySlug_RepositoryErrorPropagates(t *testing.T) {
	repo := &domain.MockSutraRepository{}
	logger := newLogger()

	wantErr := errors.New("db unavailable")
	repo.On("GetSutraBySlug", mock.Anything, "kinh-phap-hoa").Return(nil, wantErr)

	uc := NewGetSutraUsecase(repo, logger)

	_, err := uc.GetSutraBySlug(context.Background(), "kinh-phap-hoa")
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetSutraBySlug() error = %v, want %v", err, wantErr)
	}
}
