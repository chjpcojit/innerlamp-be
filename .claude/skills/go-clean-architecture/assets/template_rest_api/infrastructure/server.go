package infrastructure

import (
	"net/http"

	httphandler "yourapp/delivery/http"
	"yourapp/repository"
	"yourapp/usecase"
)

// EmailServiceMock is a mock implementation of domain.EmailService
type EmailServiceMock struct{}

// Send sends an email (mock implementation)
func (s *EmailServiceMock) Send(ctx interface{}, to, subject, body string) error {
	// Mock implementation - in production, use actual email service
	return nil
}

// NewHTTPServer creates and configures the HTTP server with all dependencies wired
func NewHTTPServer(db interface{}) *http.Server {
	// Wire up dependencies
	userRepo := repository.NewPostgresUserRepository(db.(*interface{}).(*interface{}).(interface{}))
	emailSvc := &EmailServiceMock{}

	// Create usecases
	createUserUC := usecase.NewCreateUserUseCase(userRepo, emailSvc)

	// Create handlers
	userHandler := httphandler.NewUserHandler(createUserUC)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", userHandler.CreateUser)
	mux.HandleFunc("GET /users", userHandler.GetUser)

	return &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
}
