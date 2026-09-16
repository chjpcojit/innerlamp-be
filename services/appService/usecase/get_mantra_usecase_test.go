package usecase

import (
	"context"
	"errors"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

// TestGetMantraBySlug_Success covers AC1: full detail (title, body,
// bodhisattva_id) and linked bodhisattva is returned for a known slug.
func TestGetMantraBySlug_Success(t *testing.T) {
	mantraRepo := &domain.MockMantraRepository{}
	bodhisattvaRepo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	wantMantra := &domain.Mantra{ID: "mantra-1", Slug: "chu-dai-bi", Title: "Chú Đại Bi", Body: "full body text", BodhisattvaID: "bodhisattva-1", Sect: "bac_tong"}
	wantBodhisattva := &domain.Bodhisattva{ID: "bodhisattva-1", Slug: "quan-the-am", Name: "Quán Thế Âm Bồ Tát"}

	mantraRepo.On("GetMantraBySlug", mock.Anything, "chu-dai-bi").Return(wantMantra, nil)
	bodhisattvaRepo.On("GetBodhisattvaByID", mock.Anything, "bodhisattva-1").Return(wantBodhisattva, nil)

	uc := NewGetMantraUsecase(mantraRepo, bodhisattvaRepo, logger)

	gotMantra, gotBodhisattva, err := uc.GetMantraBySlug(context.Background(), "chu-dai-bi")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if gotMantra != wantMantra {
		t.Errorf("GetMantraBySlug() mantra = %+v, want %+v", gotMantra, wantMantra)
	}
	if gotBodhisattva != wantBodhisattva {
		t.Errorf("GetMantraBySlug() bodhisattva = %+v, want %+v", gotBodhisattva, wantBodhisattva)
	}
	mantraRepo.AssertExpectations(t)
	bodhisattvaRepo.AssertExpectations(t)
}

// TestGetMantraBySlug_NoBodhisattvaNoError covers mantras with no bodhisattva_id.
func TestGetMantraBySlug_NoBodhisattvaNoError(t *testing.T) {
	mantraRepo := &domain.MockMantraRepository{}
	bodhisattvaRepo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	wantMantra := &domain.Mantra{ID: "mantra-2", Slug: "om-mani-padme-hum", Title: "Om Mani Padme Hum", BodhisattvaID: ""}
	mantraRepo.On("GetMantraBySlug", mock.Anything, "om-mani-padme-hum").Return(wantMantra, nil)

	uc := NewGetMantraUsecase(mantraRepo, bodhisattvaRepo, logger)

	gotMantra, gotBodhisattva, err := uc.GetMantraBySlug(context.Background(), "om-mani-padme-hum")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if gotMantra != wantMantra {
		t.Errorf("GetMantraBySlug() mantra = %+v, want %+v", gotMantra, wantMantra)
	}
	if gotBodhisattva != nil {
		t.Errorf("GetMantraBySlug() bodhisattva = %+v, want nil", gotBodhisattva)
	}
	bodhisattvaRepo.AssertNotCalled(t, "GetBodhisattvaByID", mock.Anything, mock.Anything)
}

// TestGetMantraBySlug_OrphanBodhisattvaID covers case where bodhisattva_id does not exist.
func TestGetMantraBySlug_OrphanBodhisattvaID(t *testing.T) {
	mantraRepo := &domain.MockMantraRepository{}
	bodhisattvaRepo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	wantMantra := &domain.Mantra{ID: "mantra-1", Slug: "chu-dai-bi", BodhisattvaID: "ghost-id"}
	mantraRepo.On("GetMantraBySlug", mock.Anything, "chu-dai-bi").Return(wantMantra, nil)
	bodhisattvaRepo.On("GetBodhisattvaByID", mock.Anything, "ghost-id").Return(nil, domain.ErrBodhisattvaNotFound)

	uc := NewGetMantraUsecase(mantraRepo, bodhisattvaRepo, logger)

	gotMantra, gotBodhisattva, err := uc.GetMantraBySlug(context.Background(), "chu-dai-bi")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if gotMantra != wantMantra {
		t.Errorf("GetMantraBySlug() mantra = %+v, want %+v", gotMantra, wantMantra)
	}
	if gotBodhisattva != nil {
		t.Errorf("GetMantraBySlug() bodhisattva = %+v, want nil on not found", gotBodhisattva)
	}
}

// TestGetMantraBySlug_AudioURLPassthrough covers AC2: audio_url (present or
// empty) passes through unchanged.
func TestGetMantraBySlug_AudioURLPassthrough(t *testing.T) {
	mantraRepo := &domain.MockMantraRepository{}
	bodhisattvaRepo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	withAudio := &domain.Mantra{ID: "mantra-1", Slug: "chu-dai-bi", AudioURL: "https://cdn.example.com/a.mp3"}
	mantraRepo.On("GetMantraBySlug", mock.Anything, "chu-dai-bi").Return(withAudio, nil)

	withoutAudio := &domain.Mantra{ID: "mantra-2", Slug: "om-mani-padme-hum", AudioURL: ""}
	mantraRepo.On("GetMantraBySlug", mock.Anything, "om-mani-padme-hum").Return(withoutAudio, nil)

	uc := NewGetMantraUsecase(mantraRepo, bodhisattvaRepo, logger)

	got1, _, err := uc.GetMantraBySlug(context.Background(), "chu-dai-bi")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if got1.AudioURL != "https://cdn.example.com/a.mp3" {
		t.Errorf("GetMantraBySlug().AudioURL = %q, want the URL", got1.AudioURL)
	}

	got2, _, err := uc.GetMantraBySlug(context.Background(), "om-mani-padme-hum")
	if err != nil {
		t.Fatalf("GetMantraBySlug() unexpected error: %v", err)
	}
	if got2.AudioURL != "" {
		t.Errorf("GetMantraBySlug().AudioURL = %q, want empty", got2.AudioURL)
	}
}

// TestGetMantraBySlug_NotFound propagates ErrMantraNotFound from the
// repository unchanged.
func TestGetMantraBySlug_NotFound(t *testing.T) {
	mantraRepo := &domain.MockMantraRepository{}
	bodhisattvaRepo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	mantraRepo.On("GetMantraBySlug", mock.Anything, "ghost-mantra").Return(nil, domain.ErrMantraNotFound)

	uc := NewGetMantraUsecase(mantraRepo, bodhisattvaRepo, logger)

	_, _, err := uc.GetMantraBySlug(context.Background(), "ghost-mantra")
	if !errors.Is(err, domain.ErrMantraNotFound) {
		t.Fatalf("GetMantraBySlug() error = %v, want %v", err, domain.ErrMantraNotFound)
	}
}

// TestGetMantraBySlug_RepositoryErrorPropagates ensures other repo errors
// surface unchanged rather than being swallowed.
func TestGetMantraBySlug_RepositoryErrorPropagates(t *testing.T) {
	mantraRepo := &domain.MockMantraRepository{}
	bodhisattvaRepo := &domain.MockBodhisattvaRepository{}
	logger := newLogger()

	wantErr := errors.New("db unavailable")
	mantraRepo.On("GetMantraBySlug", mock.Anything, "chu-dai-bi").Return(nil, wantErr)

	uc := NewGetMantraUsecase(mantraRepo, bodhisattvaRepo, logger)

	_, _, err := uc.GetMantraBySlug(context.Background(), "chu-dai-bi")
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetMantraBySlug() error = %v, want %v", err, wantErr)
	}
}
