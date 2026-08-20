package usecase

import (
	"context"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// ListBodhisattvasUsecase implements US-009 (Xem danh sách Chư Phật & Bồ Tát).
type ListBodhisattvasUsecase struct {
	bodhisattvas domain.BodhisattvaRepository
	logger       logging.Logger
}

// NewListBodhisattvasUsecase wires the list-bodhisattvas use-case with its dependencies.
func NewListBodhisattvasUsecase(bodhisattvas domain.BodhisattvaRepository, logger logging.Logger) *ListBodhisattvasUsecase {
	return &ListBodhisattvasUsecase{bodhisattvas: bodhisattvas, logger: logger}
}

// ListBodhisattvas returns bodhisattvas matching input — AC1 (list), AC2 (no
// image_url passes through without error, the client applies a
// placeholder), AC4 (delta sync when input.UpdatedSince is non-zero).
// Serving cache first when offline (AC3) is a client-side concern.
func (uc *ListBodhisattvasUsecase) ListBodhisattvas(ctx context.Context, input domain.ListBodhisattvasInput) ([]*domain.Bodhisattva, error) {
	return uc.bodhisattvas.ListBodhisattvas(ctx, input)
}
