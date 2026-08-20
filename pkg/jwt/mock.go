package jwt

import (
	"chjpcoj.com.innerlamp/pkg/jwt/claims"

	"github.com/stretchr/testify/mock"
)

// MockService is a mock implementation of Service for testing
type MockService struct {
	mock.Mock
}

// GenerateAccessToken mocks the GenerateAccessToken method
func (m *MockService) GenerateAccessToken(userID, email, role string, extra map[string]any) (string, error) {
	args := m.Called(userID, email, role, extra)
	return args.String(0), args.Error(1)
}

// GenerateRefreshToken mocks the GenerateRefreshToken method
func (m *MockService) GenerateRefreshToken(userID string, extra map[string]any) (string, error) {
	args := m.Called(userID, extra)
	return args.String(0), args.Error(1)
}

// ValidateToken mocks the ValidateToken method
func (m *MockService) ValidateToken(tokenString string) (*claims.CustomClaims, error) {
	args := m.Called(tokenString)
	if c := args.Get(0); c != nil {
		return c.(*claims.CustomClaims), args.Error(1)
	}
	return nil, args.Error(1)
}

// ParseToken mocks the ParseToken method
func (m *MockService) ParseToken(tokenString string) (*claims.CustomClaims, error) {
	args := m.Called(tokenString)
	if c := args.Get(0); c != nil {
		return c.(*claims.CustomClaims), args.Error(1)
	}
	return nil, args.Error(1)
}

// RefreshAccessToken mocks the RefreshAccessToken method
func (m *MockService) RefreshAccessToken(refreshToken string, email, role string) (string, error) {
	args := m.Called(refreshToken, email, role)
	return args.String(0), args.Error(1)
}
