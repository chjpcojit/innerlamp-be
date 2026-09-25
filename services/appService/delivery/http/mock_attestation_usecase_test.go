package http

import (
	"context"

	"github.com/stretchr/testify/mock"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// mockAttestationUsecase is a hand-written testify.Mock implementing
// attestationUsecase — this package's own consumer-defined interface — so the
// handler is tested without a real use case, repository or verifier.
type mockAttestationUsecase struct {
	mock.Mock
}

func (m *mockAttestationUsecase) RequestChallenge(ctx context.Context, platform domain.Platform, purpose domain.ChallengePurpose) (*domain.RequestChallengeResult, error) {
	args := m.Called(ctx, platform, purpose)
	if r := args.Get(0); r != nil {
		return r.(*domain.RequestChallengeResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAttestationUsecase) Verify(ctx context.Context, input domain.VerifyInput) (*domain.Verdict, error) {
	args := m.Called(ctx, input)
	if v := args.Get(0); v != nil {
		return v.(*domain.Verdict), args.Error(1)
	}
	return nil, args.Error(1)
}

// mockAssertionUsecase is a hand-written testify.Mock implementing
// assertionUsecase — the middleware's consumer-defined interface.
type mockAssertionUsecase struct {
	mock.Mock
}

func (m *mockAssertionUsecase) VerifyAssertion(ctx context.Context, input domain.VerifyAssertionInput) error {
	return m.Called(ctx, input).Error(0)
}
