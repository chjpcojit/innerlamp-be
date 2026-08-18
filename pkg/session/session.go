package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"base_project/pkg/jwt"
)

// Errors
var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionExpired       = errors.New("session has expired")
	ErrSessionRevoked       = errors.New("session has been revoked")
	ErrInvalidDeviceID      = errors.New("invalid device ID")
	ErrSessionAlreadyExists = errors.New("session already exists")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
)

// Session represents a user session with JWT rotation support
type Session struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         string             `bson:"user_id"`
	SessionID      string             `bson:"session_id"`    // Unique session identifier
	DeviceID       string             `bson:"device_id"`     // Device fingerprint
	DeviceInfo     *DeviceInfo        `bson:"device_info"`   // Device information
	AccessToken    string             `bson:"access_token"`  // Current access token
	RefreshToken   string             `bson:"refresh_token"` // Current refresh token
	TokenVersion   int                `bson:"token_version"` // Increments on rotation
	IPAddress      string             `bson:"ip_address"`
	UserAgent      string             `bson:"user_agent"`
	IsActive       bool               `bson:"is_active"`
	IsRevoked      bool               `bson:"is_revoked"`
	RevokedAt      *time.Time         `bson:"revoked_at,omitempty"`
	RevokedReason  string             `bson:"revoked_reason,omitempty"`
	LastActivityAt time.Time          `bson:"last_activity_at"`
	ExpiresAt      time.Time          `bson:"expires_at"`
	CreatedAt      time.Time          `bson:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at"`
	Metadata       map[string]any     `bson:"metadata,omitempty"`
}

// DeviceInfo contains device fingerprinting information
type DeviceInfo struct {
	Platform    string `bson:"platform"`     // iOS, Android, Web
	OSVersion   string `bson:"os_version"`   // OS version
	AppVersion  string `bson:"app_version"`  // App version
	DeviceModel string `bson:"device_model"` // Device model
	DeviceName  string `bson:"device_name"`  // User-assigned device name
	ScreenSize  string `bson:"screen_size"`  // Screen resolution
	Timezone    string `bson:"timezone"`     // Device timezone
	Language    string `bson:"language"`     // Device language
	Fingerprint string `bson:"fingerprint"`  // Computed device fingerprint hash
}

// Manager defines the session management interface
type Manager interface {
	// CreateSession creates a new session with tokens
	CreateSession(ctx context.Context, userID, email, role string, deviceInfo *DeviceInfo, ipAddress, userAgent string) (*Session, error)

	// GetSession retrieves a session by session ID
	GetSession(ctx context.Context, sessionID string) (*Session, error)

	// GetSessionByRefreshToken retrieves a session by refresh token
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)

	// RotateTokens rotates access and refresh tokens for a session
	RotateTokens(ctx context.Context, sessionID string) (*Session, error)

	// ValidateSession validates a session and updates last activity
	ValidateSession(ctx context.Context, sessionID, accessToken string) error

	// RevokeSession revokes a specific session
	RevokeSession(ctx context.Context, sessionID, reason string) error

	// RevokeAllUserSessions revokes all sessions for a user
	RevokeAllUserSessions(ctx context.Context, userID, reason string) (int64, error)

	// CleanupExpiredSessions removes expired sessions
	CleanupExpiredSessions(ctx context.Context) (int64, error)

	// GetUserActiveSessions returns all active sessions for a user
	GetUserActiveSessions(ctx context.Context, userID string) ([]*Session, error)

	// RevokeOtherSessions revokes all other sessions except the current one
	RevokeOtherSessions(ctx context.Context, userID, currentSessionID, reason string) (int64, error)
}

// MongoSessionManager implements Manager using MongoDB
type MongoSessionManager struct {
	collection *mongo.Collection
	jwtService jwt.Service
	config     *Config
}

// Config contains session manager configuration
type Config struct {
	SessionTTL         time.Duration // Session time-to-live (default: 24 hours)
	AccessTokenTTL     time.Duration // Access token TTL (default: 15 minutes)
	RefreshTokenTTL    time.Duration // Refresh token TTL (default: 7 days)
	MaxSessionsPerUser int           // Maximum concurrent sessions per user (default: 5)
	// Note: Cleanup handled by MongoDB TTL index automatically - no CleanupInterval needed
}

// DefaultConfig returns default session configuration
// DefaultConfig returns default session configuration
func DefaultConfig() *Config {
	return &Config{
		SessionTTL:         24 * time.Hour,
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    7 * 24 * time.Hour,
		MaxSessionsPerUser: 5,
		// MongoDB TTL index handles cleanup automatically
	}
}

// NewMongoSessionManager creates a new MongoDB session manager
func NewMongoSessionManager(db *mongo.Database, jwtService jwt.Service, config *Config) (Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	collection := db.Collection("user_sessions")

	// Create indexes
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "session_id", Value: 1}},
			Options: options.Index().
				SetName("idx_session_id").
				SetUnique(true).
				SetBackground(true),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().
				SetName("idx_user_id").
				SetBackground(true),
		},
		{
			Keys: bson.D{{Key: "refresh_token", Value: 1}},
			Options: options.Index().
				SetName("idx_refresh_token").
				SetBackground(true),
		},
		{
			Keys: bson.D{{Key: "device_id", Value: 1}},
			Options: options.Index().
				SetName("idx_device_id").
				SetBackground(true),
		},
		{
			Keys: bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().
				SetName("idx_expires_at").
				SetExpireAfterSeconds(0). // TTL index
				SetBackground(true),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "is_active", Value: 1},
			},
			Options: options.Index().
				SetName("idx_user_active").
				SetBackground(true),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "device_id", Value: 1},
			},
			Options: options.Index().
				SetName("idx_user_device").
				SetBackground(true),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return nil, fmt.Errorf("failed to create session indexes: %w", err)
	}

	return &MongoSessionManager{
		collection: collection,
		jwtService: jwtService,
		config:     config,
	}, nil
}

// CreateSession creates a new session with tokens
func (m *MongoSessionManager) CreateSession(ctx context.Context, userID, email, role string, deviceInfo *DeviceInfo, ipAddress, userAgent string) (*Session, error) {
	// Generate device fingerprint
	deviceID := m.generateDeviceFingerprint(deviceInfo, userAgent)
	if deviceInfo != nil {
		deviceInfo.Fingerprint = deviceID
	}

	// Check if we need to revoke old sessions (enforce max sessions limit)
	if err := m.enforceSessionLimit(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to enforce session limit: %w", err)
	}

	// Generate tokens
	sessionID := primitive.NewObjectID().Hex()

	accessToken, err := m.jwtService.GenerateAccessToken(userID, email, role, map[string]any{
		"session_id": sessionID,
		"device_id":  deviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := m.jwtService.GenerateRefreshToken(userID, map[string]any{
		"session_id": sessionID,
		"device_id":  deviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	now := time.Now()
	session := &Session{
		ID:             primitive.NewObjectID(),
		UserID:         userID,
		SessionID:      sessionID,
		DeviceID:       deviceID,
		DeviceInfo:     deviceInfo,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		TokenVersion:   1,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		IsActive:       true,
		IsRevoked:      false,
		LastActivityAt: now,
		ExpiresAt:      now.Add(m.config.SessionTTL),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err = m.collection.InsertOne(ctx, session)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrSessionAlreadyExists
		}
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by session ID
func (m *MongoSessionManager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	var session Session
	err := m.collection.FindOne(ctx, bson.M{"session_id": sessionID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	// Check if session is expired or revoked
	if session.IsRevoked {
		return nil, ErrSessionRevoked
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// GetSessionByRefreshToken retrieves a session by refresh token
func (m *MongoSessionManager) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	var session Session
	err := m.collection.FindOne(ctx, bson.M{"refresh_token": refreshToken}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	// Check if session is expired or revoked
	if session.IsRevoked {
		return nil, ErrSessionRevoked
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// RotateTokens rotates access and refresh tokens for a session
func (m *MongoSessionManager) RotateTokens(ctx context.Context, sessionID string) (*Session, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Validate refresh token
	claims, err := m.jwtService.ValidateToken(session.RefreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Generate new tokens
	accessToken, err := m.jwtService.GenerateAccessToken(session.UserID, claims.Email, claims.Role, map[string]any{
		"session_id": sessionID,
		"device_id":  session.DeviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	refreshToken, err := m.jwtService.GenerateRefreshToken(session.UserID, map[string]any{
		"session_id": sessionID,
		"device_id":  session.DeviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	// Update session with new tokens
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"access_token":     accessToken,
			"refresh_token":    refreshToken,
			"token_version":    session.TokenVersion + 1,
			"last_activity_at": now,
			"updated_at":       now,
		},
	}

	result, err := m.collection.UpdateOne(ctx, bson.M{"session_id": sessionID}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, ErrSessionNotFound
	}

	// Return updated session
	return m.GetSession(ctx, sessionID)
}

// ValidateSession validates a session and updates last activity
func (m *MongoSessionManager) ValidateSession(ctx context.Context, sessionID, accessToken string) error {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Validate access token
	_, err = m.jwtService.ValidateToken(accessToken)
	if err != nil {
		return fmt.Errorf("invalid access token: %w", err)
	}

	// Check if token matches stored token
	if session.AccessToken != accessToken {
		return ErrInvalidRefreshToken
	}

	// Update last activity
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"last_activity_at": now,
			"updated_at":       now,
		},
	}

	_, err = m.collection.UpdateOne(ctx, bson.M{"session_id": sessionID}, update)
	return err
}

// RevokeSession revokes a specific session
func (m *MongoSessionManager) RevokeSession(ctx context.Context, sessionID, reason string) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"is_active":      false,
			"is_revoked":     true,
			"revoked_at":     &now,
			"revoked_reason": reason,
			"updated_at":     now,
		},
	}

	result, err := m.collection.UpdateOne(ctx, bson.M{"session_id": sessionID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// RevokeAllUserSessions revokes all sessions for a user
func (m *MongoSessionManager) RevokeAllUserSessions(ctx context.Context, userID, reason string) (int64, error) {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"is_active":      false,
			"is_revoked":     true,
			"revoked_at":     &now,
			"revoked_reason": reason,
			"updated_at":     now,
		},
	}

	result, err := m.collection.UpdateMany(ctx, bson.M{"user_id": userID, "is_revoked": false}, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

// CleanupExpiredSessions removes expired sessions
func (m *MongoSessionManager) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"expires_at": bson.M{"$lt": time.Now()}},
			{"is_revoked": true, "revoked_at": bson.M{"$lt": time.Now().Add(-30 * 24 * time.Hour)}}, // Keep revoked for 30 days
		},
	}

	result, err := m.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// GetUserActiveSessions returns all active sessions for a user
func (m *MongoSessionManager) GetUserActiveSessions(ctx context.Context, userID string) ([]*Session, error) {
	filter := bson.M{
		"user_id":    userID,
		"is_active":  true,
		"is_revoked": false,
		"expires_at": bson.M{"$gt": time.Now()},
	}

	cursor, err := m.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "last_activity_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []*Session
	if err = cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

// RevokeOtherSessions revokes all other sessions except the current one
func (m *MongoSessionManager) RevokeOtherSessions(ctx context.Context, userID, currentSessionID, reason string) (int64, error) {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"is_active":      false,
			"is_revoked":     true,
			"revoked_at":     &now,
			"revoked_reason": reason,
			"updated_at":     now,
		},
	}

	filter := bson.M{
		"user_id":    userID,
		"session_id": bson.M{"$ne": currentSessionID},
		"is_revoked": false,
	}

	result, err := m.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

// enforceSessionLimit enforces maximum concurrent sessions per user
func (m *MongoSessionManager) enforceSessionLimit(ctx context.Context, userID string) error {
	if m.config.MaxSessionsPerUser <= 0 {
		return nil // No limit
	}

	// Get active sessions count
	filter := bson.M{
		"user_id":    userID,
		"is_active":  true,
		"is_revoked": false,
		"expires_at": bson.M{"$gt": time.Now()},
	}

	count, err := m.collection.CountDocuments(ctx, filter)
	if err != nil {
		return err
	}

	// If at limit, revoke the oldest session
	if count >= int64(m.config.MaxSessionsPerUser) {
		opts := options.FindOne().SetSort(bson.D{{Key: "last_activity_at", Value: 1}}) // Oldest first
		var oldestSession Session
		err = m.collection.FindOne(ctx, filter, opts).Decode(&oldestSession)
		if err != nil && err != mongo.ErrNoDocuments {
			return err
		}

		if err == nil {
			_ = m.RevokeSession(ctx, oldestSession.SessionID, "session limit exceeded")
		}
	}

	return nil
}

// generateDeviceFingerprint generates a unique device fingerprint
func (m *MongoSessionManager) generateDeviceFingerprint(deviceInfo *DeviceInfo, userAgent string) string {
	if deviceInfo == nil {
		// Fallback to user agent hash
		hash := sha256.Sum256([]byte(userAgent))
		return hex.EncodeToString(hash[:])
	}

	// Create fingerprint from device info
	fingerprintData := fmt.Sprintf("%s|%s|%s|%s|%s",
		deviceInfo.Platform,
		deviceInfo.OSVersion,
		deviceInfo.DeviceModel,
		deviceInfo.ScreenSize,
		userAgent,
	)

	hash := sha256.Sum256([]byte(fingerprintData))
	return hex.EncodeToString(hash[:])
}
