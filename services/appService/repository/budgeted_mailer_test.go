package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
	"chjpcoj.com.innerlamp/pkg/ratelimit/store"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newBudgeted(t *testing.T, budget int, now time.Time) (*BudgetedAuthMailer, *domain.MockAuthMailer) {
	t.Helper()
	inner := &domain.MockAuthMailer{}
	m := NewBudgetedAuthMailer(inner, store.NewMemoryStore(), "rl:", budget)
	m.now = func() time.Time { return now }
	return m, inner
}

func TestBudgetedAuthMailer_StopsAtBudgetAcrossMethods(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 22, 30, 0, 0, time.UTC)
	m, inner := newBudgeted(t, 2, now)
	inner.On("SendRegisterCode", mock.Anything, "a@b.com", "123456", time.Minute).Return(nil)
	inner.On("SendAccountExistsNotice", mock.Anything, "a@b.com").Return(nil)

	require.NoError(t, m.SendRegisterCode(ctx, "a@b.com", "123456", time.Minute))
	require.NoError(t, m.SendAccountExistsNotice(ctx, "a@b.com"))

	err := m.SendPasswordResetCode(ctx, "a@b.com", "123456", time.Minute)

	assert.ErrorIs(t, err, domain.ErrRateLimited)
	rl, ok := errors.AsType[*domain.RateLimitedError](err)
	require.True(t, ok)
	assert.Equal(t, 90*time.Minute, rl.RetryAfter, "waits until the next UTC midnight")
	inner.AssertNotCalled(t, "SendPasswordResetCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestBudgetedAuthMailer_NextUTCDayHasFreshBudget(t *testing.T) {
	ctx := context.Background()
	day1 := time.Date(2026, 10, 16, 23, 59, 0, 0, time.UTC)
	m, inner := newBudgeted(t, 1, day1)
	inner.On("SendRegisterCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	require.NoError(t, m.SendRegisterCode(ctx, "a@b.com", "1", time.Minute))
	require.ErrorIs(t, m.SendRegisterCode(ctx, "a@b.com", "1", time.Minute), domain.ErrRateLimited)

	m.now = func() time.Time { return day1.Add(2 * time.Minute) }
	assert.NoError(t, m.SendRegisterCode(ctx, "a@b.com", "1", time.Minute))
}

func TestBudgetedAuthMailer_InnerFailureStillSpendsAndPropagates(t *testing.T) {
	ctx := context.Background()
	m, inner := newBudgeted(t, 5, time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC))
	boom := errors.New("smtp down")
	inner.On("SendRegisterCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(boom)

	err := m.SendRegisterCode(ctx, "a@b.com", "1", time.Minute)

	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, domain.ErrRateLimited)
}

type failingStore struct{}

func (failingStore) Allow(context.Context, string, int64, time.Duration, int64, ratelimit.Algorithm) (*ratelimit.Result, error) {
	return nil, errors.New("redis down")
}

func TestBudgetedAuthMailer_StoreErrorFailsClosed(t *testing.T) {
	inner := &domain.MockAuthMailer{}
	m := NewBudgetedAuthMailer(inner, failingStore{}, "rl:", 5)

	err := m.SendRegisterCode(context.Background(), "a@b.com", "1", time.Minute)

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrRateLimited)
	inner.AssertNotCalled(t, "SendRegisterCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
