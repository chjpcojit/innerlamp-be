package repository

import (
	"context"
	"database/sql"
	"errors"

	"yourapp/domain"
)

// PostgresUserRepository implements domain.UserRepository using PostgreSQL
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(db *sql.DB) domain.UserRepository {
	return &PostgresUserRepository{db: db}
}

// Create inserts a new user into the database
func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	if !user.IsValid() {
		return errors.New("invalid user")
	}

	query := "INSERT INTO users (id, email, name) VALUES ($1, $2, $3)"
	_, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.Name)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := "SELECT id, email, name FROM users WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, id)

	var user domain.User
	if err := row.Scan(&user.ID, &user.Email, &user.Name); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// GetAll retrieves all users
func (r *PostgresUserRepository) GetAll(ctx context.Context) ([]*domain.User, error) {
	query := "SELECT id, email, name FROM users"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// Update updates an existing user
func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	if !user.IsValid() {
		return errors.New("invalid user")
	}

	query := "UPDATE users SET email = $1, name = $2 WHERE id = $3"
	result, err := r.db.ExecContext(ctx, query, user.Email, user.Name, user.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// Delete removes a user by ID
func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM users WHERE id = $1"
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}
