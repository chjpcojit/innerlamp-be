package session

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.SessionTTL != 24*time.Hour {
		t.Errorf("Expected SessionTTL to be 24h, got %v", cfg.SessionTTL)
	}

	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("Expected AccessTokenTTL to be 15m, got %v", cfg.AccessTokenTTL)
	}

	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Errorf("Expected RefreshTokenTTL to be 7d, got %v", cfg.RefreshTokenTTL)
	}

	if cfg.MaxSessionsPerUser != 5 {
		t.Errorf("Expected MaxSessionsPerUser to be 5, got %d", cfg.MaxSessionsPerUser)
	}
}

func TestSession_Struct(t *testing.T) {
	now := time.Now()
	session := &Session{
		ID:             primitive.NewObjectID(),
		UserID:         "user123",
		SessionID:      "session123",
		DeviceID:       "device123",
		AccessToken:    "access_token",
		RefreshToken:   "refresh_token",
		TokenVersion:   1,
		IsActive:       true,
		IsRevoked:      false,
		LastActivityAt: now,
		ExpiresAt:      now.Add(24 * time.Hour),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if session.UserID != "user123" {
		t.Errorf("Expected UserID to be user123, got %s", session.UserID)
	}

	if !session.IsActive {
		t.Error("Expected IsActive to be true")
	}

	if session.IsRevoked {
		t.Error("Expected IsRevoked to be false")
	}

	if session.TokenVersion != 1 {
		t.Errorf("Expected TokenVersion to be 1, got %d", session.TokenVersion)
	}
}

func TestDeviceInfo_Struct(t *testing.T) {
	deviceInfo := &DeviceInfo{
		Platform:    "iOS",
		OSVersion:   "17.0",
		DeviceModel: "iPhone 15 Pro",
	}

	if deviceInfo.Platform != "iOS" {
		t.Errorf("Expected Platform to be iOS, got %s", deviceInfo.Platform)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"SessionNotFound", ErrSessionNotFound, "session not found"},
		{"SessionExpired", ErrSessionExpired, "session has expired"},
		{"SessionRevoked", ErrSessionRevoked, "session has been revoked"},
		{"InvalidDeviceID", ErrInvalidDeviceID, "invalid device ID"},
		{"SessionAlreadyExists", ErrSessionAlreadyExists, "session already exists"},
		{"InvalidRefreshToken", ErrInvalidRefreshToken, "invalid refresh token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("Expected error message %q, got %q", tt.want, tt.err.Error())
			}
		})
	}
}

func TestMongoSessionManager_generateDeviceFingerprint(t *testing.T) {
	mgr := &MongoSessionManager{}

	tests := []struct {
		name       string
		deviceInfo *DeviceInfo
		userAgent  string
	}{
		{
			name:       "Nil device info",
			deviceInfo: nil,
			userAgent:  "Mozilla/5.0",
		},
		{
			name: "With device info",
			deviceInfo: &DeviceInfo{
				Platform:    "iOS",
				OSVersion:   "17.0",
				DeviceModel: "iPhone 15 Pro",
				ScreenSize:  "1179x2556",
			},
			userAgent: "Mozilla/5.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fingerprint := mgr.generateDeviceFingerprint(tt.deviceInfo, tt.userAgent)

			if fingerprint == "" {
				t.Error("Expected non-empty fingerprint")
			}

			// Fingerprint should be a hex string (SHA-256 = 64 chars)
			if len(fingerprint) != 64 {
				t.Errorf("Expected fingerprint length 64, got %d", len(fingerprint))
			}
		})
	}
}

func TestMongoSessionManager_generateDeviceFingerprint_Consistency(t *testing.T) {
	mgr := &MongoSessionManager{}

	deviceInfo := &DeviceInfo{
		Platform:    "iOS",
		OSVersion:   "17.0",
		DeviceModel: "iPhone 15 Pro",
		ScreenSize:  "1179x2556",
	}
	userAgent := "Mozilla/5.0"

	// Generate fingerprint twice with same data
	fingerprint1 := mgr.generateDeviceFingerprint(deviceInfo, userAgent)
	fingerprint2 := mgr.generateDeviceFingerprint(deviceInfo, userAgent)

	if fingerprint1 != fingerprint2 {
		t.Error("Expected consistent fingerprints for same device info")
	}
}

func TestMongoSessionManager_generateDeviceFingerprint_Different(t *testing.T) {
	mgr := &MongoSessionManager{}

	deviceInfo1 := &DeviceInfo{
		Platform:    "iOS",
		OSVersion:   "17.0",
		DeviceModel: "iPhone 15 Pro",
		ScreenSize:  "1179x2556",
	}

	deviceInfo2 := &DeviceInfo{
		Platform:    "Android",
		OSVersion:   "14",
		DeviceModel: "Pixel 8",
		ScreenSize:  "1080x2400",
	}

	userAgent := "Mozilla/5.0"

	fingerprint1 := mgr.generateDeviceFingerprint(deviceInfo1, userAgent)
	fingerprint2 := mgr.generateDeviceFingerprint(deviceInfo2, userAgent)

	if fingerprint1 == fingerprint2 {
		t.Error("Expected different fingerprints for different devices")
	}
}
