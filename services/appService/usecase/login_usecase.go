package usecase

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// LoginUsecase implements US-002 (Đăng nhập).
type LoginUsecase struct {
	users      domain.UserRepository
	jwtService jwt.Service
	logger     logging.Logger
}

// NewLoginUsecase wires the login use-case with its dependencies.
func NewLoginUsecase(users domain.UserRepository, jwtService jwt.Service, logger logging.Logger) *LoginUsecase {
	return &LoginUsecase{users: users, jwtService: jwtService, logger: logger}
}

// Login verifies email/password and issues a device-scoped session —
// AC1/AC4/AC5. AC2/AC3: wrong password and unknown email both collapse to
// domain.ErrInvalidCredentials so neither leaks which one was true.
// AC6 (rate limiting) is deliberately not handled here — the spec places
// it at the middleware/gateway layer, not in business logic.
func (uc *LoginUsecase) Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, error) {
	email := validation.NormalizeEmail(input.Email)

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	device := &domain.Device{
		DeviceID:    deviceFingerprint(input.Device),
		Platform:    input.Device.Platform,
		OSVersion:   input.Device.OSVersion,
		AppVersion:  input.Device.AppVersion,
		DeviceModel: input.Device.DeviceModel,
		DeviceName:  input.Device.DeviceName,
		IPAddress:   input.Device.IPAddress,
		UserAgent:   input.Device.UserAgent,
	}

	session, err := uc.users.CreateSession(ctx, user.ID, device, func(userID string) (string, string, error) {
		accessToken, err := uc.jwtService.GenerateAccessToken(userID, user.Email, tokenRole, nil)
		if err != nil {
			return "", "", fmt.Errorf("generate access token: %w", err)
		}
		refreshToken, err := uc.jwtService.GenerateRefreshToken(userID, nil)
		if err != nil {
			return "", "", fmt.Errorf("generate refresh token: %w", err)
		}
		return accessToken, refreshToken, nil
	})
	if err != nil {
		return nil, err
	}

	uc.logger.WithContext(ctx).WithFields(logging.Fields{"user_id": user.ID}).Info("user logged in")

	return &domain.LoginResult{
		User:         user,
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
	}, nil
}
