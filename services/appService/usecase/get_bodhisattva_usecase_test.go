package usecase

import (
	"context"
	"errors"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

// TestGetBodhisattvaBySlug_Success covers AC1: name, description, image, and
// related mantras (fetched via MantraRepository filtered by BodhisattvaID)
// are all returned together.
func TestGetBodhisattvaBySlug_Success(t *testing.T) {
	bodhisattvas := &domain.MockBodhisattvaRepository{}
	mantras := &domain.MockMantraRepository{}
	logger := newLogger()

	want := &domain.Bodhisattva{ID: "bodhisattva-1", Slug: "quan-the-am", Name: "Quán Thế Âm Bồ Tát", Description: "..."}
	bodhisattvas.On("GetBodhisattvaBySlug", mock.Anything, "quan-the-am").Return(want, nil)

	wantMantras := []*domain.Mantra{{ID: "mantra-1", Slug: "chu-dai-bi", BodhisattvaID: "bodhisattva-1"}}
	mantras.On("ListMantras", mock.Anything, domain.ListMantrasInput{BodhisattvaID: "bodhisattva-1"}).Return(wantMantras, nil)

	uc := NewGetBodhisattvaUsecase(bodhisattvas, mantras, logger)

	gotBodhisattva, gotMantras, err := uc.GetBodhisattvaBySlug(context.Background(), "quan-the-am")
	if err != nil {
		t.Fatalf("GetBodhisattvaBySlug() unexpected error: %v", err)
	}
	if gotBodhisattva != want {
		t.Errorf("GetBodhisattvaBySlug() bodhisattva = %+v, want %+v", gotBodhisattva, want)
	}
	if len(gotMantras) != 1 || gotMantras[0] != wantMantras[0] {
		t.Errorf("GetBodhisattvaBySlug() mantras = %+v, want %+v", gotMantras, wantMantras)
	}
	bodhisattvas.AssertExpectations(t)
	mantras.AssertExpectations(t)
}

// TestGetBodhisattvaBySlug_NoMantrasReturnsEmptyNotError covers AC2: a
// bodhisattva with no related mantras returns an empty (non-nil) slice, not
// an error.
func TestGetBodhisattvaBySlug_NoMantrasReturnsEmptyNotError(t *testing.T) {
	bodhisattvas := &domain.MockBodhisattvaRepository{}
	mantras := &domain.MockMantraRepository{}
	logger := newLogger()

	want := &domain.Bodhisattva{ID: "bodhisattva-2", Slug: "dia-tang", Name: "Địa Tạng Vương Bồ Tát"}
	bodhisattvas.On("GetBodhisattvaBySlug", mock.Anything, "dia-tang").Return(want, nil)
	mantras.On("ListMantras", mock.Anything, domain.ListMantrasInput{BodhisattvaID: "bodhisattva-2"}).Return([]*domain.Mantra{}, nil)

	uc := NewGetBodhisattvaUsecase(bodhisattvas, mantras, logger)

	_, gotMantras, err := uc.GetBodhisattvaBySlug(context.Background(), "dia-tang")
	if err != nil {
		t.Fatalf("GetBodhisattvaBySlug() unexpected error: %v", err)
	}
	if gotMantras == nil || len(gotMantras) != 0 {
		t.Errorf("GetBodhisattvaBySlug() mantras = %v, want empty non-nil slice", gotMantras)
	}
	bodhisattvas.AssertExpectations(t)
	mantras.AssertExpectations(t)
}

// TestGetBodhisattvaBySlug_NotFound propagates ErrBodhisattvaNotFound
// without ever calling MantraRepository — there's no bodhisattva to fetch
// related mantras for.
func TestGetBodhisattvaBySlug_NotFound(t *testing.T) {
	bodhisattvas := &domain.MockBodhisattvaRepository{}
	mantras := &domain.MockMantraRepository{}
	logger := newLogger()

	bodhisattvas.On("GetBodhisattvaBySlug", mock.Anything, "ghost").Return(nil, domain.ErrBodhisattvaNotFound)

	uc := NewGetBodhisattvaUsecase(bodhisattvas, mantras, logger)

	_, _, err := uc.GetBodhisattvaBySlug(context.Background(), "ghost")
	if !errors.Is(err, domain.ErrBodhisattvaNotFound) {
		t.Fatalf("GetBodhisattvaBySlug() error = %v, want %v", err, domain.ErrBodhisattvaNotFound)
	}
	mantras.AssertNotCalled(t, "ListMantras", mock.Anything, mock.Anything)
}

// TestGetBodhisattvaBySlug_BodhisattvaRepositoryErrorPropagates ensures
// non-not-found bodhisattva repo errors surface unchanged.
func TestGetBodhisattvaBySlug_BodhisattvaRepositoryErrorPropagates(t *testing.T) {
	bodhisattvas := &domain.MockBodhisattvaRepository{}
	mantras := &domain.MockMantraRepository{}
	logger := newLogger()

	wantErr := errors.New("db unavailable")
	bodhisattvas.On("GetBodhisattvaBySlug", mock.Anything, "quan-the-am").Return(nil, wantErr)

	uc := NewGetBodhisattvaUsecase(bodhisattvas, mantras, logger)

	_, _, err := uc.GetBodhisattvaBySlug(context.Background(), "quan-the-am")
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetBodhisattvaBySlug() error = %v, want %v", err, wantErr)
	}
	mantras.AssertNotCalled(t, "ListMantras", mock.Anything, mock.Anything)
}

// TestGetBodhisattvaBySlug_MantraRepositoryErrorPropagates ensures a
// failure fetching related mantras surfaces unchanged rather than silently
// returning the bodhisattva with an empty mantra list.
func TestGetBodhisattvaBySlug_MantraRepositoryErrorPropagates(t *testing.T) {
	bodhisattvas := &domain.MockBodhisattvaRepository{}
	mantras := &domain.MockMantraRepository{}
	logger := newLogger()

	want := &domain.Bodhisattva{ID: "bodhisattva-1", Slug: "quan-the-am"}
	bodhisattvas.On("GetBodhisattvaBySlug", mock.Anything, "quan-the-am").Return(want, nil)

	wantErr := errors.New("db unavailable")
	mantras.On("ListMantras", mock.Anything, domain.ListMantrasInput{BodhisattvaID: "bodhisattva-1"}).Return(nil, wantErr)

	uc := NewGetBodhisattvaUsecase(bodhisattvas, mantras, logger)

	_, _, err := uc.GetBodhisattvaBySlug(context.Background(), "quan-the-am")
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetBodhisattvaBySlug() error = %v, want %v", err, wantErr)
	}
}
