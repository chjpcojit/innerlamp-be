package domain

import "context"

// User represents a user entity in the domain
type User struct {
	ID    string
	Email string
	Name  string
}

// IsValid validates the user entity
func (u *User) IsValid() bool {
	return u.Email != "" && u.Name != ""
}

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetAll(ctx context.Context) ([]*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

// EmailService defines the interface for sending emails
type EmailService interface {
	Send(ctx context.Context, to, subject, body string) error
}
