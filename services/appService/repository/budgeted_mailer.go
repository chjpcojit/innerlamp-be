package repository

import (
	"context"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// dailyBudgetWindow is the counting window of the mail budget. The budget is
// "per UTC day" (login AC33), so the key embeds the date and the window only
// has to outlive it.
const dailyBudgetWindow = 24 * time.Hour

// BudgetedAuthMailer is a decorator over domain.AuthMailer that spends one unit
// of a daily budget before every send and refuses with *domain.RateLimitedError
// once it is gone (login AC33, NFR-ONB-02). It protects the free tier of the
// mail provider and bounds what an attacker can make the service send.
//
// The counter lives in the same shared rate-limit store as the per-IP limits,
// so the budget holds across instances when that store is Redis. A store error
// is returned as an ordinary send failure: failing closed means a broken
// counter cannot be turned into unlimited mail.
type BudgetedAuthMailer struct {
	inner  domain.AuthMailer
	store  ratelimit.Store
	prefix string
	budget int64
	now    func() time.Time
}

// NewBudgetedAuthMailer wraps inner. prefix is the key prefix of the shared
// store (the same one the HTTP limiters use); budget is mailer.daily_budget.
func NewBudgetedAuthMailer(inner domain.AuthMailer, store ratelimit.Store, prefix string, budget int) *BudgetedAuthMailer {
	return &BudgetedAuthMailer{inner: inner, store: store, prefix: prefix, budget: int64(budget), now: time.Now}
}

var _ domain.AuthMailer = (*BudgetedAuthMailer)(nil)

func (m *BudgetedAuthMailer) spend(ctx context.Context) error {
	now := m.now().UTC()
	key := m.prefix + "mail:daily:" + now.Format("2006-01-02")
	res, err := m.store.Allow(ctx, key, m.budget, dailyBudgetWindow, m.budget, ratelimit.FixedWindow)
	if err != nil {
		return fmt.Errorf("mail budget: %w", err)
	}
	if !res.Allowed {
		// Until the next UTC midnight, not the store's own window: the budget
		// is per calendar day.
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		return &domain.RateLimitedError{RetryAfter: next.Sub(now)}
	}
	return nil
}

func (m *BudgetedAuthMailer) SendRegisterCode(ctx context.Context, to, code string, ttl time.Duration) error {
	if err := m.spend(ctx); err != nil {
		return err
	}
	return m.inner.SendRegisterCode(ctx, to, code, ttl)
}

func (m *BudgetedAuthMailer) SendAccountExistsNotice(ctx context.Context, to string) error {
	if err := m.spend(ctx); err != nil {
		return err
	}
	return m.inner.SendAccountExistsNotice(ctx, to)
}

func (m *BudgetedAuthMailer) SendPasswordResetCode(ctx context.Context, to, code string, ttl time.Duration) error {
	if err := m.spend(ctx); err != nil {
		return err
	}
	return m.inner.SendPasswordResetCode(ctx, to, code, ttl)
}
