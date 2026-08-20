package usecase

import (
	"context"
	"errors"
	"testing"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

// TestGetProfile_Success covers AC1: an authenticated user's full profile
// is returned as-is from the repository.
func TestGetProfile_Success(t *testing.T) {
	repo := &domain.MockUserRepository{}
	logger := newLogger()

	want := &domain.User{ID: 7, Email: "new.user@example.com", PhoneNumber: "+84912345678", DharmaName: "Tam Dang"}
	repo.On("FindByID", mock.Anything, int64(7)).Return(want, nil)

	uc := NewProfileUsecase(repo, logger)

	got, err := uc.GetProfile(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetProfile() unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("GetProfile() = %+v, want %+v", got, want)
	}
	repo.AssertExpectations(t)
}

// TestGetProfile_NotFound propagates a not-found error from the repository.
func TestGetProfile_NotFound(t *testing.T) {
	repo := &domain.MockUserRepository{}
	logger := newLogger()

	repo.On("FindByID", mock.Anything, int64(99)).Return(nil, domain.ErrUserNotFound)

	uc := NewProfileUsecase(repo, logger)

	_, err := uc.GetProfile(context.Background(), 99)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("GetProfile() error = %v, want %v", err, domain.ErrUserNotFound)
	}
}

// TestUpdateProfile_Success covers AC2: dharma name / ordination place are
// updated and reflected in the returned user.
func TestUpdateProfile_Success(t *testing.T) {
	repo := &domain.MockUserRepository{}
	logger := newLogger()

	input := domain.ProfileUpdateInput{DharmaName: "Tam Dang", OrdinationPlace: "Chua Ba Da"}
	updated := &domain.User{ID: 7, DharmaName: "Tam Dang", OrdinationPlace: "Chua Ba Da"}
	repo.On("UpdateProfile", mock.Anything, int64(7), input).Return(updated, nil)

	uc := NewProfileUsecase(repo, logger)

	got, err := uc.UpdateProfile(context.Background(), 7, input)
	if err != nil {
		t.Fatalf("UpdateProfile() unexpected error: %v", err)
	}
	if got.DharmaName != "Tam Dang" || got.OrdinationPlace != "Chua Ba Da" {
		t.Errorf("UpdateProfile() = %+v, unexpected fields", got)
	}
	repo.AssertExpectations(t)
}

// TestUpdateProfile_ClearsFields covers AC5: blank fields pass through
// unchanged to the repository, which is what clears the columns.
func TestUpdateProfile_ClearsFields(t *testing.T) {
	repo := &domain.MockUserRepository{}
	logger := newLogger()

	input := domain.ProfileUpdateInput{DharmaName: "", OrdinationPlace: ""}
	cleared := &domain.User{ID: 7}
	repo.On("UpdateProfile", mock.Anything, int64(7), input).Return(cleared, nil)

	uc := NewProfileUsecase(repo, logger)

	got, err := uc.UpdateProfile(context.Background(), 7, input)
	if err != nil {
		t.Fatalf("UpdateProfile() unexpected error: %v", err)
	}
	if got.DharmaName != "" || got.OrdinationPlace != "" {
		t.Errorf("UpdateProfile() = %+v, want cleared fields", got)
	}
	repo.AssertExpectations(t)
}

// TestUpdateProfile_RepositoryErrorPropagates ensures repo errors surface
// unchanged rather than being swallowed.
func TestUpdateProfile_RepositoryErrorPropagates(t *testing.T) {
	repo := &domain.MockUserRepository{}
	logger := newLogger()

	repo.On("UpdateProfile", mock.Anything, int64(7), mock.Anything).Return(nil, domain.ErrUserNotFound)

	uc := NewProfileUsecase(repo, logger)

	_, err := uc.UpdateProfile(context.Background(), 7, domain.ProfileUpdateInput{})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("UpdateProfile() error = %v, want %v", err, domain.ErrUserNotFound)
	}
}
