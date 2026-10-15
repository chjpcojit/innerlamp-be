package e2e

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const subjectResetCode = "InnerLamp — Mã đặt lại mật khẩu"

// resetCode is the code in the newest password-reset mail logged for email.
func resetCode(t *testing.T, serverLog func() string, email string) string {
	t.Helper()
	mails := mailsTo(t, serverLog, email)
	for i := len(mails) - 1; i >= 0; i-- {
		if mails[i].Subject == subjectResetCode {
			m := sixDigits.FindStringSubmatch(mails[i].Text)
			require.NotNil(t, m, "reset mail carries a code")
			return m[1]
		}
	}
	t.Fatalf("no password-reset mail was logged for %s", email)
	return ""
}

// TestPasswordReset_E2E drives login US-003 (AC23–AC30) and its effect on
// US-002 (AC21, AC28) against the real binary: the code goes through the real
// LogMailer log, the reset through the real transaction.
//
// send_cooldown is 30s, longer than the 1.5s response floor, so a second
// request inside it is observable as 429 (AC26).
func TestPasswordReset_E2E(t *testing.T) {
	requireE2E(t)
	dbHost, dbPort, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	baseURL, serverLog, cleanupSrv := startServerWithLog(t, dbHost, dbPort, configOverrides{
		"otp": {"send_cooldown": "30s"},
	})
	t.Cleanup(cleanupSrv)
	db := openDB(t, dbHost, dbPort)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}

	const newPassword = "Brand-new-pass-9"
	request := func(email string) *apiResponse {
		return api.post("/v1/auth/password-reset/request-otp", map[string]any{"email": email}, "")
	}
	confirm := func(email, code, pw string) *apiResponse {
		return api.post("/v1/auth/password-reset/confirm", map[string]any{"email": email, "otp_code": code, "new_password": pw}, "")
	}
	me := func(token string) int {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/v1/users/me", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := api.http.Do(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		return resp.StatusCode
	}

	t.Run("AC23/AC28: reset ends every session on its next request, lifts the sign-in lock, and the new password works", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()
		email := e2eEmail(phone)
		a := api.login(phone, deviceA)
		b := api.login(phone, deviceB)
		require.Equal(t, http.StatusOK, me(a.AccessToken), "before the reset the token works")

		// Lock the email so AC28's "lock is lifted" is observable.
		for range 5 {
			res := api.post("/v1/auth/login", loginBody(phone, "Wrong-pass-123", deviceA), "")
			require.Equal(t, http.StatusUnauthorized, res.status)
		}
		res := api.post("/v1/auth/login", loginBody(phone, testPassword, deviceA), "")
		require.Equal(t, "LOGIN_LOCKED", res.errorCode())

		res = request(email)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, "no-store", res.header.Get("Cache-Control"))
		var body struct {
			Message            string `json:"message"`
			ExpiresInSeconds   int    `json:"expires_in_seconds"`
			ResendAfterSeconds int    `json:"resend_after_seconds"`
		}
		res.data(&body)
		assert.Equal(t, "Nếu email này có tài khoản, mã đã được gửi", body.Message)
		assert.Equal(t, 600, body.ExpiresInSeconds)
		assert.Equal(t, 30, body.ResendAfterSeconds)

		mails := mailsTo(t, serverLog, email)
		last := mails[len(mails)-1]
		require.Equal(t, subjectResetCode, last.Subject)
		code := sixDigits.FindStringSubmatch(last.Text)
		require.NotNil(t, code, "the reset mail carries the code")

		res = confirm(email, code[1], newPassword)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.JSONEq(t, `{"success": true}`, res.raw, "no tokens: the user signs in again")
		assert.Equal(t, "no-store", res.header.Get("Cache-Control"))

		// AC21/AC28: other devices are out on their very next request.
		assert.Equal(t, http.StatusUnauthorized, me(a.AccessToken))
		assert.Equal(t, http.StatusUnauthorized, me(b.AccessToken))
		assert.Equal(t, http.StatusUnauthorized, api.refresh(a.RefreshToken).status)
		assert.Equal(t, http.StatusUnauthorized, api.refresh(b.RefreshToken).status)

		// The old password is gone; the new one works at once, lock or not.
		old := api.post("/v1/auth/login", loginBody(phone, testPassword, deviceA), "")
		assert.Equal(t, http.StatusUnauthorized, old.status)
		fresh := api.post("/v1/auth/login", loginBody(phone, newPassword, deviceA), "")
		require.Equal(t, http.StatusOK, fresh.status, fresh.raw)

		assert.NotContains(t, serverLog(), newPassword, "the password never reaches the log")
	})

	t.Run("AC28: the code is single-use", func(t *testing.T) {
		api.t = t
		email := e2eEmail(api.registerUser())
		require.Equal(t, http.StatusOK, request(email).status)
		code := resetCode(t, serverLog, email)

		require.Equal(t, http.StatusOK, confirm(email, code, newPassword).status)
		res := confirm(email, code, "Another-pass-77")
		assert.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "OTP_EXPIRED", res.errorCode())
	})

	t.Run("AC24/AC25: an email with no account gets the identical body, no mail, and the same minimum time", func(t *testing.T) {
		api.t = t
		known := e2eEmail(api.registerUser())
		unknown := "nobody-" + known

		startKnown := time.Now()
		resKnown := request(known)
		elapsedKnown := time.Since(startKnown)
		startUnknown := time.Now()
		resUnknown := request(unknown)
		elapsedUnknown := time.Since(startUnknown)

		require.Equal(t, http.StatusOK, resKnown.status, resKnown.raw)
		require.Equal(t, http.StatusOK, resUnknown.status, resUnknown.raw)
		assert.JSONEq(t, resKnown.raw, resUnknown.raw)
		assert.Empty(t, mailsTo(t, serverLog, unknown), "nothing is sent to an unknown email")
		assert.GreaterOrEqual(t, elapsedKnown, 1400*time.Millisecond, "password_reset.min_response_time is 1.5s")
		assert.GreaterOrEqual(t, elapsedUnknown, 1400*time.Millisecond)
	})

	t.Run("AC24: an unverified registration in flight is treated as no account", func(t *testing.T) {
		api.t = t
		phone := api.nextPhone()
		api.requestOTP(phone) // pending only, never verified
		email := e2eEmail(phone)
		before := len(mailsTo(t, serverLog, email))

		res := request(email)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Len(t, mailsTo(t, serverLog, email), before, "no reset mail for a pending registration")
		assert.Equal(t, 0, countMatching(t, db, "otp_codes", "email = $1 AND purpose = 'reset_password'", email))
	})

	t.Run("AC26: the send policy answers 429 identically for known and unknown emails", func(t *testing.T) {
		api.t = t
		known := e2eEmail(api.registerUser())
		unknown := "ghost-" + known

		for _, email := range []string{known, unknown} {
			require.Equal(t, http.StatusOK, request(email).status)
			res := request(email) // inside the 30s cooldown
			require.Equal(t, http.StatusTooManyRequests, res.status, email)
			assert.Equal(t, "OTP_RESEND_TOO_SOON", res.errorCode(), email)
		}
	})

	t.Run("AC29: three wrong codes lock the code; the right one is then refused too", func(t *testing.T) {
		api.t = t
		email := e2eEmail(api.registerUser())
		require.Equal(t, http.StatusOK, request(email).status)
		code := resetCode(t, serverLog, email)

		for range 2 {
			res := confirm(email, "000000", newPassword)
			require.Equal(t, http.StatusBadRequest, res.status, res.raw)
			require.Equal(t, "OTP_INVALID", res.errorCode())
		}
		third := confirm(email, "000000", newPassword)
		assert.Equal(t, "OTP_INVALID", third.errorCode(), "the third wrong guess is still an invalid-code answer")
		locked := confirm(email, code, newPassword)
		assert.Equal(t, http.StatusTooManyRequests, locked.status, locked.raw)
		assert.Equal(t, "OTP_LOCKED", locked.errorCode())

		// An email that never asked for a code is expired, not locked/invalid.
		never := confirm("never-asked-"+email, devOTPCode, newPassword)
		assert.Equal(t, http.StatusBadRequest, never.status)
		assert.Equal(t, "OTP_EXPIRED", never.errorCode())
	})

	t.Run("AC27/AC30: bad email, bad password and bad code shape are 400 before any state changes", func(t *testing.T) {
		api.t = t
		email := e2eEmail(api.registerUser())

		res := request("not-an-email")
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "INVALID_EMAIL", res.errorCode())

		res = confirm("not-an-email", devOTPCode, newPassword)
		assert.Equal(t, "INVALID_EMAIL", res.errorCode())
		res = confirm(email, devOTPCode, "short")
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "INVALID_PASSWORD", res.errorCode())
		res = confirm(email, "12ab", newPassword)
		assert.Equal(t, http.StatusBadRequest, res.status)

		assert.Zero(t, countMatching(t, db, "otp_codes", "email = $1 AND purpose = 'reset_password'", email), "rejected requests issued no reset code")
	})
}

// TestAuthRateLimit_E2E drives login AC31–AC34: per-IP budgets on every
// /v1/auth/* group, the forwarded visitor address, and the daily mail budget.
func TestAuthRateLimit_E2E(t *testing.T) {
	requireE2E(t)
	dbHost, dbPort, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	baseURL, _, cleanupSrv := startServerWithLog(t, dbHost, dbPort, configOverrides{
		"auth_rate_limit": {
			"send":    map[string]any{"max": 100000, "window": "1h"},
			"verify":  map[string]any{"max": 100000, "window": "1h"},
			"login":   map[string]any{"max": 3, "window": "1h"},
			"refresh": map[string]any{"max": 2, "window": "1h"},
		},
		"mailer": {"daily_budget": 3},
	})
	t.Cleanup(cleanupSrv)
	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}

	webKey := func() string {
		raw, err := os.ReadFile(baseConfigPath)
		require.NoError(t, err)
		var doc struct {
			Server struct {
				WebAPIKeySecret string `yaml:"web_api_key_secret"`
			} `yaml:"server"`
			DevSecrets map[string]string `yaml:"dev_secrets"`
		}
		require.NoError(t, yaml.Unmarshal(raw, &doc))
		return doc.DevSecrets[doc.Server.WebAPIKeySecret]
	}()
	require.NotEmpty(t, webKey)

	login := func(email string, headers map[string]string) *apiResponse {
		return api.postWithHeaders("/v1/auth/login", loginBodyEmail(email, "Wrong-pass-123", deviceA), headers)
	}

	t.Run("AC31: past the budget is 429 RATE_LIMITED with retry_after_seconds, and no handler runs", func(t *testing.T) {
		api.t = t
		email := "limit-a@e2e.example.com"
		for i := range 3 {
			res := login(email, nil)
			require.Equal(t, http.StatusUnauthorized, res.status, "request %d", i+1)
		}
		res := login(email, nil)
		require.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
		assert.Equal(t, "RATE_LIMITED", res.errorCode())
		assert.Positive(t, res.details()["retry_after_seconds"])
		assert.NotEmpty(t, res.header.Get("Retry-After"))
		assert.Equal(t, "no-store", res.header.Get("Cache-Control"))
	})

	t.Run("AC34: with a valid key each forwarded visitor has their own bucket; without a key the header is ignored", func(t *testing.T) {
		api.t = t
		// The peer's own budget is spent by the subtest above; a forwarded
		// visitor is a different bucket.
		keyed := func(visitor string) map[string]string {
			return map[string]string{"X-API-Key": webKey, "X-InnerLamp-Client-IP": visitor}
		}
		for range 3 {
			require.Equal(t, http.StatusUnauthorized, login("v1@e2e.example.com", keyed("203.0.113.10")).status)
		}
		assert.Equal(t, http.StatusTooManyRequests, login("v1@e2e.example.com", keyed("203.0.113.10")).status)
		assert.Equal(t, http.StatusUnauthorized, login("v2@e2e.example.com", keyed("203.0.113.11")).status, "another visitor, another bucket")

		// No key: the header is attacker-controlled and ignored — the peer's
		// spent bucket answers, whatever address is claimed.
		spoofed := login("v3@e2e.example.com", map[string]string{"X-InnerLamp-Client-IP": "203.0.113.99"})
		assert.Equal(t, http.StatusTooManyRequests, spoofed.status, "an unkeyed caller cannot pick its own bucket")

		// A wrong key is refused outright, before any counting.
		wrong := login("v4@e2e.example.com", map[string]string{"X-API-Key": "nope", "X-InnerLamp-Client-IP": "203.0.113.12"})
		assert.Equal(t, http.StatusUnauthorized, wrong.status)
		assert.Equal(t, "INVALID_API_KEY", wrong.errorCode())
	})

	t.Run("AC31: refresh has its own budget; logout is not limited", func(t *testing.T) {
		api.t = t
		keyed := map[string]string{"X-API-Key": webKey, "X-InnerLamp-Client-IP": "203.0.113.20"}
		for range 2 {
			res := api.postWithHeaders("/v1/auth/refresh", map[string]any{"refresh_token": "x"}, keyed)
			require.Equal(t, http.StatusUnauthorized, res.status)
		}
		res := api.postWithHeaders("/v1/auth/refresh", map[string]any{"refresh_token": "x"}, keyed)
		assert.Equal(t, http.StatusTooManyRequests, res.status)

		for range 5 {
			res := api.postWithHeaders("/v1/auth/logout", nil, keyed)
			assert.Equal(t, http.StatusUnauthorized, res.status, "no bearer is 401, never 429")
		}
	})

	t.Run("AC33: the fourth email of the day is 429 RATE_LIMITED, whatever route would send it", func(t *testing.T) {
		api.t = t
		keyed := func(n int) map[string]string {
			return map[string]string{"X-API-Key": webKey, "X-InnerLamp-Client-IP": "203.0.113." + strings.Repeat("1", 1) + string(rune('0'+n))}
		}
		register := func(i int) *apiResponse {
			phone := api.nextPhone()
			return api.postWithHeaders("/v1/auth/register/request-otp", map[string]any{
				"email": e2eEmail(phone), "password": testPassword, "full_name": "Nguyen Van An",
				"phone_country": "VN", "phone_number": "0" + strings.TrimPrefix(phone, "+84"),
			}, keyed(i))
		}
		for i := range 3 {
			res := register(i)
			require.Equal(t, http.StatusOK, res.status, "mail %d: %s", i+1, res.raw)
		}
		res := register(3)
		require.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
		assert.Equal(t, "RATE_LIMITED", res.errorCode())
		assert.Positive(t, res.details()["retry_after_seconds"])
	})
}
