// Package repository holds backofficeService's Postgres adapters.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

type adminRepository struct {
	db database.Database
}

// NewAdminRepository returns the Postgres-backed domain.AdminRepository.
func NewAdminRepository(db database.Database) domain.AdminRepository {
	return &adminRepository{db: db}
}

const adminColumns = `id, email, password_hash, name, role, is_active, must_change_password, tokens_valid_after, temp_password_expires_at, last_login_at`

func (r *adminRepository) findOne(ctx context.Context, where string, arg any) (*domain.Admin, error) {
	var a domain.Admin
	var validAfter, tempExpires, lastLogin sql.NullTime
	err := r.db.QueryRow(ctx, `SELECT `+adminColumns+` FROM admins WHERE `+where, arg).
		Scan(&a.ID, &a.Email, &a.PasswordHash, &a.Name, &a.Role, &a.IsActive, &a.MustChangePassword, &validAfter, &tempExpires, &lastLogin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrAdminNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find admin: %w", err)
	}
	if validAfter.Valid {
		a.TokensValidAfter = &validAfter.Time
	}
	if tempExpires.Valid {
		a.TempPasswordExpiresAt = &tempExpires.Time
	}
	if lastLogin.Valid {
		a.LastLoginAt = &lastLogin.Time
	}
	return &a, nil
}

// FindByEmail returns the admin with the given (already normalized) email.
func (r *adminRepository) FindByEmail(ctx context.Context, email string) (*domain.Admin, error) {
	return r.findOne(ctx, `email = $1`, email)
}

// FindByID returns the admin with the given id.
func (r *adminRepository) FindByID(ctx context.Context, id int64) (*domain.Admin, error) {
	return r.findOne(ctx, `id = $1`, id)
}

// UpdatePassword stores a new hash and clears must_change_password.
func (r *adminRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	if _, err := r.db.Exec(ctx,
		`UPDATE admins SET password_hash = $2, must_change_password = FALSE, tokens_valid_after = now(), temp_password_expires_at = NULL, updated_at = now() WHERE id = $1`,
		id, passwordHash); err != nil {
		return fmt.Errorf("update admin password: %w", err)
	}
	return nil
}

// CreateBootstrapIfEmpty inserts the default admin only when no admin exists.
// The emptiness check and the insert are one statement, so two instances
// starting together cannot both create one.
func (r *adminRepository) CreateBootstrapIfEmpty(ctx context.Context, email, passwordHash, name string) (bool, error) {
	res, err := r.db.Exec(ctx,
		`INSERT INTO admins (email, password_hash, name, must_change_password)
		 SELECT $1, $2, $3, TRUE WHERE NOT EXISTS (SELECT 1 FROM admins)
		 ON CONFLICT (email) DO NOTHING`,
		email, passwordHash, name)
	if err != nil {
		return false, fmt.Errorf("create bootstrap admin: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("bootstrap admin rows affected: %w", err)
	}
	return n > 0, nil
}

// TouchLastLogin records a successful login.
func (r *adminRepository) TouchLastLogin(ctx context.Context, id int64, at time.Time) error {
	if _, err := r.db.Exec(ctx, `UPDATE admins SET last_login_at = $2, updated_at = $2 WHERE id = $1`, id, at); err != nil {
		return fmt.Errorf("touch admin last login: %w", err)
	}
	return nil
}

// Create inserts a new admin and returns it.
func (r *adminRepository) Create(ctx context.Context, email, passwordHash, name string) (*domain.Admin, error) {
	a := domain.Admin{Email: email, PasswordHash: passwordHash, Name: name, Role: domain.RoleAdmin, IsActive: true}
	err := r.db.QueryRow(ctx,
		`INSERT INTO admins (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`,
		email, passwordHash, name).Scan(&a.ID)
	if err != nil {
		return nil, fmt.Errorf("create admin: %w", err)
	}
	return &a, nil
}

// CreateAccount inserts an account that must change its password on first login.
// ON CONFLICT ... RETURNING yields no row for a duplicate email, which is
// reported as ErrEmailTaken without parsing driver error codes.
func (r *adminRepository) CreateAccount(ctx context.Context, email, passwordHash, name, role string, tempExpiresAt *time.Time) (*domain.Admin, error) {
	a := domain.Admin{Email: email, PasswordHash: passwordHash, Name: name, Role: role, IsActive: true,
		MustChangePassword: true, TempPasswordExpiresAt: tempExpiresAt}
	err := r.db.QueryRow(ctx,
		`INSERT INTO admins (email, password_hash, name, role, must_change_password, temp_password_expires_at)
		 VALUES ($1, $2, $3, $4, TRUE, $5)
		 ON CONFLICT (email) DO NOTHING RETURNING id`,
		email, passwordHash, name, role, tempExpiresAt).Scan(&a.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrEmailTaken
	}
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return &a, nil
}

// ResetTempPassword swaps in a temporary password and revokes every session.
func (r *adminRepository) ResetTempPassword(ctx context.Context, id int64, passwordHash string, tempExpiresAt *time.Time) (*domain.Admin, error) {
	a := domain.Admin{ID: id, MustChangePassword: true, TempPasswordExpiresAt: tempExpiresAt}
	err := r.db.QueryRow(ctx,
		`UPDATE admins SET password_hash = $2, must_change_password = TRUE, temp_password_expires_at = $3,
		        tokens_valid_after = now(), updated_at = now()
		 WHERE id = $1 RETURNING email, name, role, is_active`,
		id, passwordHash, tempExpiresAt).Scan(&a.Email, &a.Name, &a.Role, &a.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrAdminNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reset temp password: %w", err)
	}
	return &a, nil
}

// DeleteByID removes an account.
func (r *adminRepository) DeleteByID(ctx context.Context, id int64) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM admins WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete admin: %w", err)
	}
	return nil
}
