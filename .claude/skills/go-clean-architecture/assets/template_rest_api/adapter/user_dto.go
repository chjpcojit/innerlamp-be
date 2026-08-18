package adapter

import "yourapp/domain"

// CreateUserRequest represents the HTTP request to create a user
type CreateUserRequest struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required,min=2"`
}

// UserResponse represents the HTTP response for a user
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// MapCreateUserRequestToInput converts HTTP request to usecase input
func MapCreateUserRequestToInput(req CreateUserRequest) interface{} {
	return struct {
		Email string
		Name  string
	}{
		Email: req.Email,
		Name:  req.Name,
	}
}

// MapUserToResponse converts domain user to HTTP response
func MapUserToResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}
}

// MapUsersToResponses converts multiple domain users to HTTP responses
func MapUsersToResponses(users []*domain.User) []UserResponse {
	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = MapUserToResponse(user)
	}
	return responses
}
