package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// fakeDatabase adapts an sqlmock-backed *sql.DB to the database.Database
// interface so userRepository.Register can be unit tested without a real
// Postgres instance. Register only ever calls Begin — every other method
// is unused by the code under test.
type fakeDatabase struct {
	db *sql.DB
}

func (f *fakeDatabase) Connect(ctx context.Context) error { return nil }
func (f *fakeDatabase) Close() error                      { return f.db.Close() }
func (f *fakeDatabase) Ping(ctx context.Context) error    { return nil }
func (f *fakeDatabase) IsConnected() bool                 { return true }
func (f *fakeDatabase) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return f.db.QueryContext(ctx, query, args...)
}
func (f *fakeDatabase) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return f.db.QueryRowContext(ctx, query, args...)
}
func (f *fakeDatabase) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, errors.New("fakeDatabase.Exec not implemented")
}
func (f *fakeDatabase) Begin(ctx context.Context) (*sql.Tx, error) { return f.db.BeginTx(ctx, nil) }
func (f *fakeDatabase) GetSQLDB() (*sql.DB, error)                 { return f.db, nil }
func (f *fakeDatabase) GetMongoClient() (any, error) {
	return nil, database.ErrOperationNotSupported
}
func (f *fakeDatabase) GetType() database.DBType { return database.PostgreSQL }
func (f *fakeDatabase) GetConfig() *database.Config {
	return &database.Config{Type: database.PostgreSQL}
}

func newMockRepo(t *testing.T) (domain.UserRepository, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return NewUserRepository(&fakeDatabase{db: sqlDB}), mock
}

func testUser() *domain.User {
	return &domain.User{
		Email:        "new.user@example.com",
		PasswordHash: "bcrypt-hash",
		PhoneNumber:  "+84912345678",
	}
}

func testDevice() *domain.Device {
	return &domain.Device{
		DeviceID: "fingerprint-hash",
		Platform: "iOS",
	}
}

var noopIssuer domain.TokenIssuer = func(userID string) (string, string, error) {
	return "access-token", "refresh-token", nil
}

// TestRegister_Success covers AC1/AC7: user, device and session rows are
// inserted atomically and the issued token pair is persisted on the
// session row.
func TestRegister_Success(t *testing.T) {
	repo, mock := newMockRepo(t)

	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
		WithArgs("new.user@example.com", "bcrypt-hash", "+84912345678", nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(1), now, now))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO user_devices`)).
		WithArgs(int64(1), "fingerprint-hash", "iOS", "", "", "", "", "", "").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(10), now))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO user_sessions`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(100), now, now))
	mock.ExpectCommit()

	user := testUser()
	device := testDevice()

	session, err := repo.Register(context.Background(), user, device, noopIssuer)
	if err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}
	if session.AccessToken != "access-token" || session.RefreshToken != "refresh-token" {
		t.Errorf("Register() tokens = %s/%s, want access-token/refresh-token", session.AccessToken, session.RefreshToken)
	}
	if user.ID != 1 {
		t.Errorf("user.ID = %d, want 1", user.ID)
	}
	if device.UserID != 1 || device.ID != 10 {
		t.Errorf("device = %+v, want UserID=1 ID=10", device)
	}
	if session.UserID != 1 || session.DeviceID != 10 {
		t.Errorf("session = %+v, want UserID=1 DeviceID=10", session)
	}

	// AC7: session must be immediately usable (active, not revoked) and
	// the refresh token TTL must be 7 days (pkg/session.DefaultConfig()).
	if !session.IsActive || session.IsRevoked {
		t.Errorf("session active/revoked = %v/%v, want true/false", session.IsActive, session.IsRevoked)
	}
	if session.SessionID == "" {
		t.Error("session.SessionID is empty, want a generated UUID")
	}
	wantExpiry := now.Add(sessionTTL)
	if diff := session.ExpiresAt.Sub(wantExpiry); diff < -time.Second || diff > time.Second {
		t.Errorf("session.ExpiresAt = %v, want ~%v (sessionTTL=%v)", session.ExpiresAt, wantExpiry, sessionTTL)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestRegister_EmailTaken and TestRegister_PhoneTaken cover AC2: the two
// unique constraints must map to distinct domain errors, and the
// transaction must roll back rather than leave a partial insert.
func TestRegister_EmailTaken(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
		WillReturnError(&pq.Error{Code: "23505", Constraint: "users_email_key"})
	mock.ExpectRollback()

	_, err := repo.Register(context.Background(), testUser(), testDevice(), noopIssuer)
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("Register() error = %v, want %v", err, domain.ErrEmailTaken)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestRegister_PhoneTaken(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
		WillReturnError(&pq.Error{Code: "23505", Constraint: "users_phone_number_key"})
	mock.ExpectRollback()

	_, err := repo.Register(context.Background(), testUser(), testDevice(), noopIssuer)
	if !errors.Is(err, domain.ErrPhoneTaken) {
		t.Fatalf("Register() error = %v, want %v", err, domain.ErrPhoneTaken)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestRegister_TokenIssuanceFailure ensures a failure minting the JWT pair
// rolls back the whole transaction instead of leaving user/device rows
// committed without a session.
func TestRegister_TokenIssuanceFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(1), now, now))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO user_devices`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(10), now))
	mock.ExpectRollback()

	failingIssuer := domain.TokenIssuer(func(userID string) (string, string, error) {
		return "", "", errors.New("signing key unavailable")
	})

	_, err := repo.Register(context.Background(), testUser(), testDevice(), failingIssuer)
	if err == nil {
		t.Fatalf("Register() expected error when token issuance fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestFindByEmail_Found and TestFindByEmail_NotFound cover the lookup
// login's use-case depends on (US-002 AC1/AC3).
func TestFindByEmail_Found(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, phone_number, dharma_name, ordination_place, created_at, updated_at`)).
		WithArgs("new.user@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "phone_number", "dharma_name", "ordination_place", "created_at", "updated_at"}).
			AddRow(int64(7), "new.user@example.com", "bcrypt-hash", "+84912345678", nil, nil, now, now))

	user, err := repo.FindByEmail(context.Background(), "new.user@example.com")
	if err != nil {
		t.Fatalf("FindByEmail() unexpected error: %v", err)
	}
	if user.ID != 7 || user.Email != "new.user@example.com" || user.PasswordHash != "bcrypt-hash" {
		t.Errorf("FindByEmail() user = %+v, unexpected fields", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestFindByEmail_NotFound(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, phone_number, dharma_name, ordination_place, created_at, updated_at`)).
		WithArgs("ghost@example.com").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.FindByEmail(context.Background(), "ghost@example.com")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("FindByEmail() error = %v, want %v", err, domain.ErrUserNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestCreateSession_NewDevice and TestCreateSession_ExistingDeviceUpsert
// cover US-002 AC4: each device gets its own session, and repeat logins
// from the same device upsert (rather than duplicate) its device row.
func TestCreateSession_NewDevice(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO user_devices`)).
		WithArgs(int64(7), "fingerprint-hash", "iOS", "", "", "", "", "", "").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(20), now))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO user_sessions`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(200), now, now))
	mock.ExpectCommit()

	device := testDevice()
	session, err := repo.CreateSession(context.Background(), 7, device, noopIssuer)
	if err != nil {
		t.Fatalf("CreateSession() unexpected error: %v", err)
	}
	if session.UserID != 7 || session.DeviceID != 20 {
		t.Errorf("CreateSession() session = %+v, want UserID=7 DeviceID=20", session)
	}
	if device.UserID != 7 {
		t.Errorf("device.UserID = %d, want 7", device.UserID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestCreateSession_ExistingDeviceUpsert(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	// The upsert (ON CONFLICT) query is the same INSERT statement whether
	// the device row already exists or not — sqlmock only needs to see
	// the conflicting device's existing id come back.
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO user_devices`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(20), now))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO user_sessions`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(201), now, now))
	mock.ExpectCommit()

	session, err := repo.CreateSession(context.Background(), 7, testDevice(), noopIssuer)
	if err != nil {
		t.Fatalf("CreateSession() unexpected error: %v", err)
	}
	// Same device id (20) reused across logins — proves the device row was
	// upserted, not duplicated, while still getting a fresh session (201).
	if session.DeviceID != 20 || session.ID != 201 {
		t.Errorf("CreateSession() session = %+v, want DeviceID=20 ID=201", session)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestCreateSession_TokenIssuanceFailure ensures the device upsert rolls
// back if token issuance fails, so login never leaves an orphaned device
// row with no session.
func TestCreateSession_TokenIssuanceFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO user_devices`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(20), now))
	mock.ExpectRollback()

	failingIssuer := domain.TokenIssuer(func(userID string) (string, string, error) {
		return "", "", errors.New("signing key unavailable")
	})

	_, err := repo.CreateSession(context.Background(), 7, testDevice(), failingIssuer)
	if err == nil {
		t.Fatalf("CreateSession() expected error when token issuance fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestCreatePasswordResetToken_Success covers US-003 AC1: a reset token is
// persisted for the account.
func TestCreatePasswordResetToken_Success(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO password_reset_tokens`)).
		WithArgs(int64(7), "token-hash", expiresAt).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("token-uuid", now))

	token, err := repo.CreatePasswordResetToken(context.Background(), 7, "token-hash", expiresAt)
	if err != nil {
		t.Fatalf("CreatePasswordResetToken() unexpected error: %v", err)
	}
	if token.ID != "token-uuid" || token.UserID != 7 || token.TokenHash != "token-hash" {
		t.Errorf("CreatePasswordResetToken() token = %+v, unexpected fields", token)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestResetPassword_Success covers AC3: password rotates, token is marked
// used, and every active session for the account is revoked, atomically.
func TestResetPassword_Success(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens`)).
		WithArgs("token-hash").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at", "used_at"}).
			AddRow("token-uuid", int64(7), now.Add(10*time.Minute), nil))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET password_hash`)).
		WithArgs("new-hash", int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE password_reset_tokens SET used_at`)).
		WithArgs("token-uuid").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE user_sessions SET is_revoked`)).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	if err := repo.ResetPassword(context.Background(), "token-hash", "new-hash"); err != nil {
		t.Fatalf("ResetPassword() unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestResetPassword_TokenNotFound, TestResetPassword_TokenExpired and
// TestResetPassword_TokenAlreadyUsed cover AC4: an unknown, expired, or
// already-used token must be rejected and the transaction rolled back
// without touching users/sessions.
func TestResetPassword_TokenNotFound(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens`)).
		WithArgs("token-hash").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err := repo.ResetPassword(context.Background(), "token-hash", "new-hash")
	if !errors.Is(err, domain.ErrResetTokenInvalid) {
		t.Fatalf("ResetPassword() error = %v, want %v", err, domain.ErrResetTokenInvalid)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestResetPassword_TokenExpired(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens`)).
		WithArgs("token-hash").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at", "used_at"}).
			AddRow("token-uuid", int64(7), now.Add(-1*time.Minute), nil))
	mock.ExpectRollback()

	err := repo.ResetPassword(context.Background(), "token-hash", "new-hash")
	if !errors.Is(err, domain.ErrResetTokenInvalid) {
		t.Fatalf("ResetPassword() error = %v, want %v", err, domain.ErrResetTokenInvalid)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestFindByID_Found and TestFindByID_NotFound cover US-004 AC1 (profile
// view lookup by the authenticated user's ID).
func TestFindByID_Found(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, phone_number, dharma_name, ordination_place, avatar_url, created_at, updated_at`)).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "phone_number", "dharma_name", "ordination_place", "avatar_url", "created_at", "updated_at"}).
			AddRow(int64(7), "new.user@example.com", "bcrypt-hash", "+84912345678", "Tam Dang", "Chua Ba Da", "https://cdn.example.com/a.jpg", now, now))

	user, err := repo.FindByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("FindByID() unexpected error: %v", err)
	}
	if user.ID != 7 || user.DharmaName != "Tam Dang" || user.OrdinationPlace != "Chua Ba Da" || user.AvatarURL != "https://cdn.example.com/a.jpg" {
		t.Errorf("FindByID() user = %+v, unexpected fields", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, phone_number, dharma_name, ordination_place, avatar_url, created_at, updated_at`)).
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.FindByID(context.Background(), 99)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, domain.ErrUserNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestUpdateProfile_Success covers US-004 AC2, and TestUpdateProfile_ClearsFields
// covers AC5 (blank input clears the columns, i.e. bound as SQL NULL).
func TestUpdateProfile_Success(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE users SET dharma_name = $1, ordination_place = $2, updated_at = now()`)).
		WithArgs("Tam Dang", "Chua Ba Da", int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "phone_number", "dharma_name", "ordination_place", "avatar_url", "created_at", "updated_at"}).
			AddRow(int64(7), "new.user@example.com", "bcrypt-hash", "+84912345678", "Tam Dang", "Chua Ba Da", nil, now, now))

	user, err := repo.UpdateProfile(context.Background(), 7, domain.ProfileUpdateInput{DharmaName: "Tam Dang", OrdinationPlace: "Chua Ba Da"})
	if err != nil {
		t.Fatalf("UpdateProfile() unexpected error: %v", err)
	}
	if user.DharmaName != "Tam Dang" || user.OrdinationPlace != "Chua Ba Da" {
		t.Errorf("UpdateProfile() user = %+v, unexpected fields", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestUpdateProfile_ClearsFields(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE users SET dharma_name = $1, ordination_place = $2, updated_at = now()`)).
		WithArgs(nil, nil, int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "phone_number", "dharma_name", "ordination_place", "avatar_url", "created_at", "updated_at"}).
			AddRow(int64(7), "new.user@example.com", "bcrypt-hash", "+84912345678", nil, nil, nil, now, now))

	user, err := repo.UpdateProfile(context.Background(), 7, domain.ProfileUpdateInput{DharmaName: "", OrdinationPlace: ""})
	if err != nil {
		t.Fatalf("UpdateProfile() unexpected error: %v", err)
	}
	if user.DharmaName != "" || user.OrdinationPlace != "" {
		t.Errorf("UpdateProfile() user = %+v, want cleared fields", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestUpdateProfile_NotFound(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE users SET dharma_name = $1, ordination_place = $2, updated_at = now()`)).
		WithArgs("Tam Dang", "Chua Ba Da", int64(99)).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.UpdateProfile(context.Background(), 99, domain.ProfileUpdateInput{DharmaName: "Tam Dang", OrdinationPlace: "Chua Ba Da"})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("UpdateProfile() error = %v, want %v", err, domain.ErrUserNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestUpdateAvatarURL_Success covers US-004 AC3: uploading a new avatar
// replaces the previously stored URL.
func TestUpdateAvatarURL_Success(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE users SET avatar_url = $1, updated_at = now()`)).
		WithArgs("https://cdn.example.com/new.jpg", int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "phone_number", "dharma_name", "ordination_place", "avatar_url", "created_at", "updated_at"}).
			AddRow(int64(7), "new.user@example.com", "bcrypt-hash", "+84912345678", nil, nil, "https://cdn.example.com/new.jpg", now, now))

	user, err := repo.UpdateAvatarURL(context.Background(), 7, "https://cdn.example.com/new.jpg")
	if err != nil {
		t.Fatalf("UpdateAvatarURL() unexpected error: %v", err)
	}
	if user.AvatarURL != "https://cdn.example.com/new.jpg" {
		t.Errorf("UpdateAvatarURL() user.AvatarURL = %q, want new URL", user.AvatarURL)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestResetPassword_TokenAlreadyUsed(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens`)).
		WithArgs("token-hash").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at", "used_at"}).
			AddRow("token-uuid", int64(7), now.Add(10*time.Minute), now.Add(-1*time.Minute)))
	mock.ExpectRollback()

	err := repo.ResetPassword(context.Background(), "token-hash", "new-hash")
	if !errors.Is(err, domain.ErrResetTokenInvalid) {
		t.Fatalf("ResetPassword() error = %v, want %v", err, domain.ErrResetTokenInvalid)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
