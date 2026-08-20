package usecase

import (
	"context"
	"errors"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

// TestGetMantraBySlug_Success covers AC1: full detail (title, body,
// bodhisattva_id) is returned as-is for a known slug.
func TestGetMantraBySlug_Success(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	want := &domain.Mantra{ID: "mantra-1", Slug: "chu-dai-bi", Title: "Chú Đại Bi", Body: "full body text", BodhisattvaID: "bodhisattva-1"}
	repo.On("GetMantraBySlug", mock.Anything, "chu-dai-bi").Return(want, nil)

	uc := NewGetMantraUsecase(repo, logger)

	got, err := uc.GetMantraBySlug(context.Background(), "chu-dai-bi")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("GetMantraBySlug() = %+v, want %+v", got, want)
	}
	repo.AssertExpectations(t)
}

// TestGetMantraBySlug_AudioURLPassthrough covers AC2: audio_url (present or
// empty) passes through unchanged.
func TestGetMantraBySlug_AudioURLPassthrough(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	withAudio := &domain.Mantra{ID: "mantra-1", Slug: "chu-dai-bi", AudioURL: "https://cdn.example.com/a.mp3"}
	repo.On("GetMantraBySlug", mock.Anything, "chu-dai-bi").Return(withAudio, nil)

	withoutAudio := &domain.Mantra{ID: "mantra-2", Slug: "om-mani-padme-hum", AudioURL: ""}
	repo.On("GetMantraBySlug", mock.Anything, "om-mani-padme-hum").Return(withoutAudio, nil)

	uc := NewGetMantraUsecase(repo, logger)

	got1, err := uc.GetMantraBySlug(context.Background(), "chu-dai-bi")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if got1.AudioURL != "https://cdn.example.com/a.mp3" {
		t.Errorf("GetMantraBySlug().AudioURL = %q, want the URL", got1.AudioURL)
	}

	got2, err := uc.GetMantraBySlug(context.Background(), "om-mani-padme-hum")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if got2.AudioURL != "" {
		t.Errorf("GetMantraBySlug().AudioURL = %q, want empty", got2.AudioURL)
	}
	repo.AssertExpectations(t)
}

// TestGetMantraBySlug_NotFound propagates ErrMantraNotFound from the
// repository unchanged.
func TestGetMantraBySlug_NotFound(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	repo.On("GetMantraBySlug", mock.Anything, "ghost-mantra").Return(nil, domain.ErrMantraNotFound)

	uc := NewGetMantraUsecase(repo, logger)

	_, err := uc.GetMantraBySlug(context.Background(), "ghost-mantra")
	if !errors.Is(err, domain.ErrMantraNotFound) {
		t.Fatalf("GetMantraBySlug() error = %v, want %v", err, domain.ErrMantraNotFound)
	}
}

// TestGetMantraBySlug_RepositoryErrorPropagates ensures other repo errors
// surface unchanged rather than being swallowed.
func TestGetMantraBySlug_RepositoryErrorPropagates(t *testing.T) {
	repo := &domain.MockMantraRepository{}
	logger := newLogger()

	wantErr := errors.New("db unavailable")
	repo.On("GetMantraBySlug", mock.Anything, "chu-dai-bi").Return(nil, wantErr)

	uc := NewGetMantraUsecase(repo, logger)

	_, err := uc.GetMantraBySlug(context.Background(), "chu-dai-bi")
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetMantraBySlug() error = %v, want %v", err, wantErr)
	}
}
