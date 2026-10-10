package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func validSuggestionInput() domain.TempleSuggestionInput {
	return domain.TempleSuggestionInput{Name: "Chùa Một Cột", Province: "Hà Nội", Address: "Phố Chùa Một Cột, Ba Đình"}
}

func TestTempleSuggestionUsecase_Create_Success(t *testing.T) {
	repo := new(domain.MockTempleSuggestionRepository)
	uc := NewTempleSuggestionUsecase(repo)
	now := time.Now()

	// Surrounding whitespace is trimmed; the text is otherwise stored as typed.
	repo.On("Create", mock.Anything, &domain.TempleSuggestion{
		Name: "Chùa  Một Cột", Province: "hà nội", Address: "Phố Chùa Một Cột",
	}).Return(&domain.TempleSuggestion{
		ID: 7, Name: "Chùa  Một Cột", Province: "hà nội", Address: "Phố Chùa Một Cột",
		Status: domain.TempleSuggestionPending, CreatedAt: now,
	}, nil)

	got, err := uc.Create(context.Background(), domain.TempleSuggestionInput{
		Name: "  Chùa  Một Cột ", Province: "\thà nội\n", Address: " Phố Chùa Một Cột  ",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(7), got.ID)
	assert.Equal(t, domain.TempleSuggestionPending, got.Status)
	repo.AssertExpectations(t)
}

func TestTempleSuggestionUsecase_Create_Validation(t *testing.T) {
	long := func(n int) string { return strings.Repeat("ạ", n) } // multi-byte: counts runes, not bytes

	tests := []struct {
		name      string
		mutate    func(in *domain.TempleSuggestionInput)
		wantField string
	}{
		{"empty name", func(in *domain.TempleSuggestionInput) { in.Name = "" }, "name"},
		{"blank province", func(in *domain.TempleSuggestionInput) { in.Province = "   " }, "province"},
		{"blank address", func(in *domain.TempleSuggestionInput) { in.Address = " \t " }, "address"},
		{"name too long", func(in *domain.TempleSuggestionInput) { in.Name = long(256) }, "name"},
		{"province too long", func(in *domain.TempleSuggestionInput) { in.Province = long(101) }, "province"},
		{"address too long", func(in *domain.TempleSuggestionInput) { in.Address = long(501) }, "address"},
		{"first offender wins", func(in *domain.TempleSuggestionInput) { in.Province = ""; in.Address = "" }, "province"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockTempleSuggestionRepository)
			in := validSuggestionInput()
			tt.mutate(&in)

			_, err := NewTempleSuggestionUsecase(repo).Create(context.Background(), in)

			require.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrTempleSuggestionInvalid)
			bad, ok := errors.AsType[*domain.TempleSuggestionValidationError](err)
			require.True(t, ok)
			assert.Equal(t, tt.wantField, bad.Field)
			repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestTempleSuggestionUsecase_Create_AtLimitIsAccepted(t *testing.T) {
	repo := new(domain.MockTempleSuggestionRepository)
	repo.On("Create", mock.Anything, mock.Anything).Return(&domain.TempleSuggestion{ID: 1}, nil)
	in := domain.TempleSuggestionInput{
		Name: strings.Repeat("a", domain.TempleSuggestionNameMax), Province: strings.Repeat("a", domain.TempleSuggestionProvinceMax),
		Address: strings.Repeat("ạ", domain.TempleSuggestionAddressMax),
	}

	_, err := NewTempleSuggestionUsecase(repo).Create(context.Background(), in)

	require.NoError(t, err)
}

func TestTempleSuggestionUsecase_Create_RepositoryErrorIsWrapped(t *testing.T) {
	repo := new(domain.MockTempleSuggestionRepository)
	boom := errors.New("db down")
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, boom)

	_, err := NewTempleSuggestionUsecase(repo).Create(context.Background(), validSuggestionInput())

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, domain.ErrTempleSuggestionInvalid)
}
