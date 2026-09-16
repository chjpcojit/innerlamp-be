package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// tokenRole is the JWT "role" claim assigned to every newly registered user.
// There is no role system yet at MVP — this is a placeholder default.
const tokenRole = "user"

// RegisterUsecase implements US-001 (Đăng ký tài khoản).
type RegisterUsecase struct {
	users      domain.UserRepository
	jwtService jwt.Service
	logger     logging.Logger
}

// NewRegisterUsecase wires the register use-case with its dependencies.
func NewRegisterUsecase(users domain.UserRepository, jwtService jwt.Service, logger logging.Logger) *RegisterUsecase {
	return &RegisterUsecase{users: users, jwtService: jwtService, logger: logger}
}

// Register hashes the password, persists the user/device/session, and returns
// the freshly issued JWT pair — AC1/AC6/AC7.
//
// Input format (email shape, phone format, password length) is enforced by the
// delivery layer's struct tags before this is called; what stays here is
// canonicalization the business needs — lowercasing the email so uniqueness is
// case-insensitive — not format checking.
func (uc *RegisterUsecase) Register(ctx context.Context, input domain.RegisterInput) (*domain.RegisterResult, error) {
	email := validation.NormalizeEmail(input.Email)
	phone := strings.TrimSpace(input.PhoneNumber)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Email:           email,
		PasswordHash:    string(passwordHash),
		PhoneNumber:     phone,
		DharmaName:      strings.TrimSpace(input.DharmaName),
		OrdinationPlace: strings.TrimSpace(input.OrdinationPlace),
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

	session, err := uc.users.Register(ctx, user, device, func(userID string) (string, string, error) {
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

	uc.logger.WithContext(ctx).WithFields(logging.Fields{"user_id": user.ID}).Info("user registered")

	return &domain.RegisterResult{
		User:         user,
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
	}, nil
}

// deviceFingerprint hashes device metadata into the fingerprint stored on
// user_devices.device_id, mirroring pkg/session's approach.
func deviceFingerprint(d domain.DeviceInfo) string {
	raw := strings.Join([]string{d.Platform, d.OSVersion, d.DeviceModel, d.UserAgent}, "|")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
