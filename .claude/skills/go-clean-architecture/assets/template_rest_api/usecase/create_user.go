package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"yourapp/domain"
)

// CreateUserInput represents the input for creating a user
type CreateUserInput struct {
	Email string
	Name  string
}

// CreateUserUseCase implements the create user business logic
type CreateUserUseCase struct {
	userRepo domain.UserRepository
	emailSvc domain.EmailService
}

// NewCreateUserUseCase creates a new create user usecase
func NewCreateUserUseCase(
	userRepo domain.UserRepository,
	emailSvc domain.EmailService,
) *CreateUserUseCase {
	return &CreateUserUseCase{
		userRepo: userRepo,
		emailSvc: emailSvc,
	}
}

// Execute runs the create user business logic
func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) (*domain.User, error) {
	// Validate input
	if input.Email == "" || input.Name == "" {
		return nil, errors.New("email and name are required")
	}

	// Create user entity
	user := &domain.User{
		ID:    uuid.New().String(),
		Email: input.Email,
		Name:  input.Name,
	}

	// Validate domain entity
	if !user.IsValid() {
		return nil, errors.New("invalid user data")
	}

	// Persist user
	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Send welcome email (non-blocking - could be async)
	_ = uc.emailSvc.Send(
		ctx,
		user.Email,
		"Welcome to Our Service",
		"Thank you for signing up!",
	)

	return user, nil
}
