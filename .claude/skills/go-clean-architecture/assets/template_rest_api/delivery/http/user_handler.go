package http

import (
	"encoding/json"
	"net/http"

	"yourapp/adapter"
	"yourapp/usecase"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	createUserUC *usecase.CreateUserUseCase
}

// NewUserHandler creates a new user HTTP handler
func NewUserHandler(createUserUC *usecase.CreateUserUseCase) *UserHandler {
	return &UserHandler{
		createUserUC: createUserUC,
	}
}

// CreateUser handles HTTP POST /users requests
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req adapter.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Execute usecase
	user, err := h.createUserUC.Execute(r.Context(), usecase.CreateUserInput{
		Email: req.Email,
		Name:  req.Name,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(adapter.MapUserToResponse(user))
}

// GetUser handles HTTP GET /users/:id requests
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
