package domain

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ ArticleRepository = (*MockArticleRepository)(nil)

// MockArticleRepository is a hand-written mock of ArticleRepository (see
// mock_temple_repository.go for the pattern).
type MockArticleRepository struct {
	mock.Mock
}

// ListTop mocks the ListTop method.
func (m *MockArticleRepository) ListTop(ctx context.Context, limit int) ([]Article, error) {
	args := m.Called(ctx, limit)
	if a := args.Get(0); a != nil {
		return a.([]Article), args.Error(1)
	}
	return nil, args.Error(1)
}

// List mocks the List method.
func (m *MockArticleRepository) List(ctx context.Context, offset, limit int) ([]Article, int, error) {
	args := m.Called(ctx, offset, limit)
	if a := args.Get(0); a != nil {
		return a.([]Article), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

// FindByID mocks the FindByID method.
func (m *MockArticleRepository) FindByID(ctx context.Context, id string) (*Article, error) {
	args := m.Called(ctx, id)
	if a := args.Get(0); a != nil {
		return a.(*Article), args.Error(1)
	}
	return nil, args.Error(1)
}

// Count mocks the Count method.
func (m *MockArticleRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// InsertMissing mocks the InsertMissing method.
func (m *MockArticleRepository) InsertMissing(ctx context.Context, articles []Article) (int, error) {
	args := m.Called(ctx, articles)
	return args.Int(0), args.Error(1)
}

// FillMissingImages mocks the FillMissingImages method.
func (m *MockArticleRepository) FillMissingImages(ctx context.Context, articles []Article) (int, error) {
	args := m.Called(ctx, articles)
	return args.Int(0), args.Error(1)
}
