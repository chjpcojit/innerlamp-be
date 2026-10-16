package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
)

// Subjects the appService email mailer uses (repository/email_auth_mailer.go).
// Asserted on the dev LogMailer output to tell a code email from the
// account-exists notice.
const (
	subjectRegisterCode  = "InnerLamp — Mã xác minh đăng ký"
	subjectAccountExists = "InnerLamp — Email của bạn đã có tài khoản"
)

// countMatching is count(*) of table WHERE where, with bind args.
func countMatching(t *testing.T, db database.Database, table, where string, args ...any) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var n int
	require.NoError(t, db.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE "+where, args...).Scan(&n))
	return n
}

// loggedMail is one email the dev LogMailer wrote to the server log instead
// of sending it.
type loggedMail struct {
	To      string `json:"to_email"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
}

var sixDigits = regexp.MustCompile(`\b(\d{6})\b`)

// mailsTo returns, oldest first, every email the server logged for to.
func mailsTo(t *testing.T, serverLog func() string, to string) []loggedMail {
	t.Helper()
	var out []loggedMail
	sc := bufio.NewScanner(strings.NewReader(serverLog()))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if !strings.Contains(string(line), `"to_email"`) {
			continue
		}
		var m loggedMail
		if json.Unmarshal(line, &m) == nil && m.To == to {
			out = append(out, m)
		}
	}
	require.NoError(t, sc.Err())
	return out
}

// codeFromLastMail reads the code out of the newest registration email logged
// for to — the way a user reads it from their inbox.
func codeFromLastMail(t *testing.T, serverLog func() string, to string) string {
	t.Helper()
	mails := mailsTo(t, serverLog, to)
	require.NotEmpty(t, mails, "no email logged for %s", to)
	last := mails[len(mails)-1]
	require.Equal(t, subjectRegisterCode, last.Subject)
	m := sixDigits.FindStringSubmatch(last.Text)
	require.NotNil(t, m, "no 6-digit code in the mail body: %q", last.Text)
	return m[1]
}

// TestRegisterFlow_Email_E2E drives onboarding v8.0 (email + emailed code)
// over the real binary and a real Postgres. The code is read from the server
// log, where the dev LogMailer (mailer.provider "log") writes the whole email.
// In dev every code is otp.dev_fixed_code ("111111"), so the read value is
// that constant — what this proves is that the code travels by email, not
// that it is random (AC20 is a unit-test concern).
//
// send_cooldown is shortened to 1s so a resend fits a test run; everything
// else is the dev config as shipped.
func TestRegisterFlow_Email_E2E(t *testing.T) {
	requireE2E(t)
	const cooldown = 1
	dbHost, dbPort, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	baseURL, serverLog, cleanupSrv := startServerWithLog(t, dbHost, dbPort, configOverrides{
		"otp": {"send_cooldown": "1s"},
	})
	t.Cleanup(cleanupSrv)
	db := openDB(t, dbHost, dbPort)

	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}

	// phone is an E.164 "+84…" number from nextPhone, typed the way a VN user would.
	requestOTP := func(email, phone string) *apiResponse {
		return api.post("/v1/auth/register/request-otp", map[string]any{
			"email":         email,
			"password":      testPassword,
			"full_name":     "Nguyen Van An",
			"phone_country": "VN",
			"phone_number":  "0" + strings.TrimPrefix(phone, "+84"),
		}, "")
	}
	resendOTP := func(email string) *apiResponse {
		return api.post("/v1/auth/register/resend-otp", map[string]any{"email": email}, "")
	}
	verifyOTP := func(email, code string) *apiResponse {
		return api.post("/v1/auth/register/verify-otp", map[string]any{"email": email, "otp_code": code}, "")
	}
	waitCooldown := func() { time.Sleep(cooldown*time.Second + 100*time.Millisecond) }
	pendingRows := func(email string) int {
		return countMatching(t, db, "pending_registration", "email = $1", email)
	}

	// Set by the happy path, reused by the "already taken" cases.
	var accountEmail, accountPhone string

	t.Run("register by email with the emailed code (AC1, AC14)", func(t *testing.T) {
		api.t = t
		accountEmail, accountPhone = "new.user@e2e.example.com", api.nextPhone()
		usersBefore := countRows(t, db, "users")

		// Mixed case on the wire; stored and mailed lower-case (AC1).
		res := requestOTP("New.User@E2E.example.com", accountPhone)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var step1 struct {
			ExpiresInSeconds   int `json:"expires_in_seconds"`
			ResendAfterSeconds int `json:"resend_after_seconds"`
		}
		res.data(&step1)
		assert.Equal(t, 600, step1.ExpiresInSeconds, "otp.ttl")
		assert.Equal(t, cooldown, step1.ResendAfterSeconds, "otp.send_cooldown")
		assert.Equal(t, usersBefore, countRows(t, db, "users"), "step 1 creates no user")
		assert.Equal(t, 1, pendingRows(accountEmail))

		code := codeFromLastMail(t, serverLog, accountEmail)

		res = verifyOTP(accountEmail, code)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		assert.Equal(t, "no-store", res.header.Get("Cache-Control"))
		var data struct {
			AccessToken  string         `json:"access_token"`
			RefreshToken string         `json:"refresh_token"`
			ExpiresIn    int            `json:"expires_in"`
			User         map[string]any `json:"user"`
		}
		res.data(&data)
		assert.NotEmpty(t, data.AccessToken)
		assert.NotEmpty(t, data.RefreshToken)
		assert.Positive(t, data.ExpiresIn)
		assert.Equal(t, accountEmail, data.User["email"])
		assert.Equal(t, accountPhone, data.User["phone_number"], "E.164")
		assert.Equal(t, "Nguyen Van An", data.User["full_name"])

		assert.Equal(t, usersBefore+1, countRows(t, db, "users"))
		assert.Zero(t, pendingRows(accountEmail), "pending row is deleted")
		assert.Equal(t, 1, countMatching(t, db, "users", "email = $1 AND email_verified_at IS NOT NULL", accountEmail))

		// AC19: the password never reaches the log.
		assert.NotContains(t, serverLog(), testPassword)
	})

	t.Run("an email with an account gets the same 200 and can never create one (AC2, AC16)", func(t *testing.T) {
		api.t = t
		require.NotEmpty(t, accountEmail, "depends on the happy path")
		usersBefore := countRows(t, db, "users")

		fresh := requestOTP("fresh.user@e2e.example.com", api.nextPhone())
		require.Equal(t, http.StatusOK, fresh.status, fresh.raw)

		taken := requestOTP(accountEmail, api.nextPhone())
		require.Equal(t, http.StatusOK, taken.status, taken.raw)
		assert.JSONEq(t, fresh.raw, taken.raw, "the body must not reveal that the email has an account")

		mails := mailsTo(t, serverLog, accountEmail)
		require.NotEmpty(t, mails)
		last := mails[len(mails)-1]
		assert.Equal(t, subjectAccountExists, last.Subject, "the owner gets the notice, not a code")
		assert.Nil(t, sixDigits.FindStringSubmatch(last.Text), "the notice carries no code")

		// The dev fixed code is what the server issued; it still must not work.
		res := verifyOTP(accountEmail, devOTPCode)
		require.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "OTP_INVALID", res.errorCode())
		assert.Equal(t, usersBefore, countRows(t, db, "users"), "no account created")
	})

	t.Run("a phone that belongs to an account is 409 PHONE_TAKEN (AC7)", func(t *testing.T) {
		api.t = t
		require.NotEmpty(t, accountPhone, "depends on the happy path")
		const email = "phone.taken@e2e.example.com"

		res := requestOTP(email, accountPhone)
		require.Equal(t, http.StatusConflict, res.status, res.raw)
		assert.Equal(t, "PHONE_TAKEN", res.errorCode())
		assert.Zero(t, pendingRows(email), "nothing written")
		assert.Empty(t, mailsTo(t, serverLog, email), "nothing sent")
	})

	t.Run("an expired pending registration is 400 PENDING_REGISTRATION_EXPIRED (AC12, AC17)", func(t *testing.T) {
		api.t = t
		const email = "expired@e2e.example.com"
		require.Equal(t, http.StatusOK, requestOTP(email, api.nextPhone()).status)
		code := codeFromLastMail(t, serverLog, email)

		// pending_registration_ttl is 24h; age the row instead of waiting.
		execSQL(t, db, `UPDATE pending_registration SET expires_at = now() - interval '1 second' WHERE email = $1`, email)

		res := verifyOTP(email, code)
		require.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "PENDING_REGISTRATION_EXPIRED", res.errorCode())

		waitCooldown()
		res = resendOTP(email)
		require.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "PENDING_REGISTRATION_EXPIRED", res.errorCode())

		res = resendOTP("never.requested@e2e.example.com")
		require.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "PENDING_REGISTRATION_EXPIRED", res.errorCode())
	})

	t.Run("the third wrong code locks it: 429 OTP_LOCKED (AC15)", func(t *testing.T) {
		api.t = t
		const email = "locked@e2e.example.com"
		require.Equal(t, http.StatusOK, requestOTP(email, api.nextPhone()).status)
		code := codeFromLastMail(t, serverLog, email)
		wrong := "000000"
		if code == wrong {
			wrong = "999999"
		}

		for i := 1; i <= 3; i++ {
			res := verifyOTP(email, wrong)
			require.Equalf(t, http.StatusBadRequest, res.status, "wrong code %d: %s", i, res.raw)
			assert.Equalf(t, "OTP_INVALID", res.errorCode(), "wrong code %d", i)
		}

		// Even the right code is refused now.
		res := verifyOTP(email, code)
		require.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
		assert.Equal(t, "OTP_LOCKED", res.errorCode())
		lockedUntil, ok := res.details()["locked_until"].(string)
		require.True(t, ok, "OTP_LOCKED must carry locked_until: %s", res.raw)
		parsed, err := time.Parse(time.RFC3339Nano, lockedUntil)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().Add(30*time.Minute), parsed, time.Minute, "otp.lock_duration")

		assert.Zero(t, countMatching(t, db, "users", "email = $1", email))
	})

	t.Run("resend issues a new code, then the cooldown applies (AC9, AC11)", func(t *testing.T) {
		api.t = t
		const email = "resend@e2e.example.com"
		require.Equal(t, http.StatusOK, requestOTP(email, api.nextPhone()).status)

		// Inside the cooldown: refused, nothing sent.
		res := resendOTP(email)
		require.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
		assert.Equal(t, "OTP_RESEND_TOO_SOON", res.errorCode())
		assert.Contains(t, res.details(), "retry_after_seconds")
		require.Len(t, mailsTo(t, serverLog, email), 1)

		waitCooldown()
		res = resendOTP(email)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var data struct {
			ExpiresInSeconds   int  `json:"expires_in_seconds"`
			ResendAfterSeconds int  `json:"resend_after_seconds"`
			ResendRemaining    *int `json:"resend_remaining"`
		}
		res.data(&data)
		assert.Equal(t, 600, data.ExpiresInSeconds)
		assert.Equal(t, cooldown, data.ResendAfterSeconds)
		require.NotNil(t, data.ResendRemaining, res.raw)
		assert.Equal(t, 2, *data.ResendRemaining)
		require.Len(t, mailsTo(t, serverLog, email), 2, "the resend emailed a new code")

		// Right after the resend the cooldown applies again.
		res = resendOTP(email)
		require.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
		assert.Equal(t, "OTP_RESEND_TOO_SOON", res.errorCode())

		res = verifyOTP(email, codeFromLastMail(t, serverLog, email))
		require.Equal(t, http.StatusCreated, res.status, res.raw)
	})

	t.Run("a different payload cannot hijack a pending registration with a live code (AC24)", func(t *testing.T) {
		api.t = t
		const email = "victim@e2e.example.com"
		const attackerPassword = "attacker-secret-1"
		victimPhone, attackerPhone := api.nextPhone(), api.nextPhone()

		require.Equal(t, http.StatusOK, requestOTP(email, victimPhone).status)
		victimCode := codeFromLastMail(t, serverLog, email)
		mailsBefore := len(mailsTo(t, serverLog, email))
		var hashBefore string
		require.NoError(t, db.QueryRow(context.Background(),
			`SELECT password_hash FROM pending_registration WHERE email = $1`, email).Scan(&hashBefore))

		waitCooldown() // past the cooldown, so only the AC24 guard can refuse
		res := api.post("/v1/auth/register/request-otp", map[string]any{
			"email":         email,
			"password":      attackerPassword,
			"full_name":     "Nguyen Van An",
			"phone_country": "VN",
			"phone_number":  "0" + strings.TrimPrefix(attackerPhone, "+84"),
		}, "")
		require.Equal(t, http.StatusConflict, res.status, res.raw)
		assert.Equal(t, "REGISTRATION_IN_PROGRESS", res.errorCode())
		assert.Contains(t, res.details(), "retry_after_seconds")
		assert.Len(t, mailsTo(t, serverLog, email), mailsBefore, "no mail for a rejected request")

		var phone, hashAfter string
		require.NoError(t, db.QueryRow(context.Background(),
			`SELECT phone_number, password_hash FROM pending_registration WHERE email = $1`, email).Scan(&phone, &hashAfter))
		assert.Equal(t, victimPhone, phone)
		assert.Equal(t, hashBefore, hashAfter, "the victim's password hash is untouched")

		// The victim's code still works and the account is the victim's.
		res = verifyOTP(email, victimCode)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		assert.Equal(t, 1, countMatching(t, db, "users", "email = $1 AND phone_number = $2", email, victimPhone))

		device := deviceA
		res = api.post("/v1/auth/login", loginBodyEmail(email, testPassword, device), "")
		assert.Equal(t, http.StatusOK, res.status, res.raw)
		res = api.post("/v1/auth/login", loginBodyEmail(email, attackerPassword, device), "")
		assert.Equal(t, http.StatusUnauthorized, res.status, res.raw)
	})
}
