package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterFlow_RestartOverwritesPending_E2E pins the v5.1 restart rules
// over the real binary: a live pending_registration row no longer blocks the
// same phone from starting over — request-otp overwrites the payload, and the
// only throttle is the AC5 send policy (here the 1s cooldown). Another phone's
// live pending still holds the email (AC6), because the email is the account's
// global unique key.
//
// The send cooldown is shortened to 1s so the restart fits a test run;
// everything else is the dev config as shipped.
func TestRegisterFlow_RestartOverwritesPending_E2E(t *testing.T) {
	baseURL, client := newE2EServerWithConfig(t, configOverrides{
		"otp": {"send_cooldown": "1s"},
	})
	api := &apiClient{t: t, baseURL: baseURL, http: client}

	requestOTP := func(phone, email string) *apiResponse {
		return api.post("/v1/auth/register/request-otp", map[string]any{
			"email": email, "password": testPassword, "full_name": "Nguyen Van An", "phone_number": phone,
		}, "")
	}

	t.Run("same phone restarts without waiting out the pending TTL", func(t *testing.T) {
		api.t = t
		phone := api.nextPhone()

		require.Equal(t, http.StatusOK, requestOTP(phone, "first@e2e.example.com").status)

		// Same phone right away: no longer PHONE_TAKEN, but AC5's cooldown still
		// refuses a second send in the same second.
		res := requestOTP(phone, "first@e2e.example.com")
		require.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
		assert.Equal(t, "OTP_RESEND_TOO_SOON", res.errorCode())

		time.Sleep(1100 * time.Millisecond)

		// After the cooldown the restart goes through and overwrites the payload.
		res = requestOTP(phone, "second@e2e.example.com")
		require.Equal(t, http.StatusOK, res.status, "same phone must be able to restart: %s", res.raw)

		res = api.post("/v1/auth/register/verify-otp", map[string]any{"phone_number": phone, "otp_code": "111111"}, "")
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var data struct {
			User map[string]any `json:"user"`
		}
		res.data(&data)
		assert.Equal(t, "second@e2e.example.com", data.User["email"], "the restart's payload replaced the stale one")
	})

	t.Run("another phone's live pending still holds the email", func(t *testing.T) {
		api.t = t
		const email = "shared@e2e.example.com"

		require.Equal(t, http.StatusOK, requestOTP(api.nextPhone(), email).status)

		res := requestOTP(api.nextPhone(), email)
		require.Equal(t, http.StatusConflict, res.status, res.raw)
		assert.Equal(t, "EMAIL_TAKEN", res.errorCode())
	})
}

// TestRegisterFlow_ResendAfterCodeExpired_E2E pins the v5.1 resend rule: once
// the code window has closed but the pending payload is still live, resend-otp
// issues a fresh code instead of OTP_EXPIRED. That is what makes "press resend
// after the 30-minute send lock lifts" a way to resume.
func TestRegisterFlow_ResendAfterCodeExpired_E2E(t *testing.T) {
	baseURL, client := newE2EServerWithConfig(t, configOverrides{
		"otp": {"ttl": "2s", "send_cooldown": "1s"},
	})
	api := &apiClient{t: t, baseURL: baseURL, http: client}
	phone := api.nextPhone()

	require.Equal(t, http.StatusOK, api.post("/v1/auth/register/request-otp", map[string]any{
		"email": "resume@e2e.example.com", "password": testPassword, "full_name": "Nguyen Van An", "phone_number": phone,
	}, "").status)

	// Let the code die without letting the pending payload die (its TTL is the
	// shipped 24h).
	time.Sleep(3 * time.Second)

	// Before v5.1 this answered 400 OTP_EXPIRED.
	res := api.post("/v1/auth/resend-otp", map[string]any{"phone_number": phone, "purpose": "register"}, "")
	require.Equal(t, http.StatusOK, res.status, "resend after the code expired must issue a new code: %s", res.raw)

	res = api.post("/v1/auth/register/verify-otp", map[string]any{"phone_number": phone, "otp_code": "111111"}, "")
	require.Equal(t, http.StatusCreated, res.status, res.raw)
}

// TestRegisterFlow_SendLimitAndVerifyDuringLock_E2E drives US-001 AC5 as
// redesigned in v5.0 over the real binary: after the free first send, every
// request-otp/resend-otp consumes one of three counted sends; the 4th send sets
// a temporary send lock and the 5th is refused with 429 OTP_SEND_LIMITED (with
// locked_until). It also pins B1 — the send lock must NOT block verifying the
// last code, so verify-otp still succeeds while the number is locked out of
// sending.
//
// send_cooldown is shortened to 1s so the four sends fit in a test run; the
// send lock is left long enough to outlast the test's verify call.
func TestRegisterFlow_SendLimitAndVerifyDuringLock_E2E(t *testing.T) {
	baseURL, client := newE2EServerWithConfig(t, configOverrides{
		"otp": {"send_cooldown": "1s", "send_lock_duration": "60s"},
	})
	api := &apiClient{t: t, baseURL: baseURL, http: client}
	phone := api.nextPhone()
	const email = "sendlimit@e2e.example.com"

	request := func() *apiResponse {
		return api.post("/v1/auth/register/request-otp", map[string]any{
			"email": email, "password": testPassword, "full_name": "Nguyen Van An", "phone_number": phone,
		}, "")
	}
	resend := func() *apiResponse {
		return api.post("/v1/auth/resend-otp", map[string]any{
			"phone_number": phone, "purpose": "register",
		}, "")
	}
	waitCooldown := func() { time.Sleep(1100 * time.Millisecond) }

	// Send 1: the free first of the cycle.
	res := request()
	require.Equal(t, http.StatusOK, res.status, res.raw)

	// Sends 2-4: three counted resends. Remaining falls 2, 1, 0.
	for i, wantRemaining := range []int{2, 1, 0} {
		waitCooldown()
		res = resend()
		require.Equalf(t, http.StatusOK, res.status, "resend %d: %s", i+1, res.raw)
		var data struct {
			ResendRemaining int `json:"resend_remaining"`
		}
		res.data(&data)
		assert.Equalf(t, wantRemaining, data.ResendRemaining, "resend %d remaining", i+1)
	}

	// Send 5: the cap is reached, so sending is refused with a deadline.
	waitCooldown()
	res = resend()
	require.Equal(t, http.StatusTooManyRequests, res.status, res.raw)
	assert.Equal(t, "OTP_SEND_LIMITED", res.errorCode())

	lockedUntil, ok := res.details()["locked_until"].(string)
	require.True(t, ok, "OTP_SEND_LIMITED must carry locked_until: %s", res.raw)
	parsed, err := time.Parse(time.RFC3339Nano, lockedUntil)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(60*time.Second), parsed, time.Minute)

	// B1: the send lock must not block verifying the last code that was
	// actually sent — send 4 is still unused and inside its TTL.
	res = api.post("/v1/auth/register/verify-otp", map[string]any{"phone_number": phone, "otp_code": "111111"}, "")
	require.Equal(t, http.StatusCreated, res.status, "verify must work while send-locked: %s", res.raw)
}

// TestRegisterFlow_OrdinationTemple_E2E pins v7.0 over the real binary: the
// client sends ordination_temple_id, an unknown id is refused up front with
// INVALID_TEMPLE (nothing written), a valid id travels pending -> users and
// comes back in the verify-otp user object, and omitting it is fine (NULL).
func TestRegisterFlow_OrdinationTemple_E2E(t *testing.T) {
	requireE2E(t)
	dbHost, dbPort, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	baseURL, cleanupSrv := startServer(t, dbHost, dbPort, nil)
	t.Cleanup(cleanupSrv)

	db := openDB(t, dbHost, dbPort)
	execSQL(t, db, `INSERT INTO temples (id, name, type, region, sect, established_year, heritage_rank, heritage_priority, latitude, longitude, address)
		VALUES ('temple_e2e_001', 'Chùa E2E', 'CHUA', 'north', 'Thiền', 1000, 'CHUA_XEP_HANG', 7, 21.0, 105.8, 'Hà Nội')`)

	api := &apiClient{t: t, baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
	request := func(phone, email string, extra map[string]any) *apiResponse {
		body := map[string]any{"email": email, "password": testPassword, "full_name": "Nguyen Van An", "phone_number": phone}
		for k, v := range extra {
			body[k] = v
		}
		return api.post("/v1/auth/register/request-otp", body, "")
	}
	verify := func(phone string) *apiResponse {
		return api.post("/v1/auth/register/verify-otp", map[string]any{"phone_number": phone, "otp_code": "111111"}, "")
	}

	t.Run("unknown id is refused and writes nothing", func(t *testing.T) {
		api.t = t
		res := request(api.nextPhone(), "badtemple@e2e.example.com", map[string]any{"ordination_temple_id": "temple_missing"})
		require.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "INVALID_TEMPLE", res.errorCode())
		assert.Zero(t, countRows(t, db, "pending_registration"))
	})

	t.Run("full_name is required and blank is refused without writing", func(t *testing.T) {
		api.t = t
		missing := api.post("/v1/auth/register/request-otp", map[string]any{"email": "noname@e2e.example.com", "password": testPassword, "phone_number": api.nextPhone()}, "")
		require.Equal(t, http.StatusBadRequest, missing.status, missing.raw)
		assert.Equal(t, "VALIDATION_ERROR", missing.errorCode())

		blank := request(api.nextPhone(), "blankname@e2e.example.com", map[string]any{"full_name": "   "})
		require.Equal(t, http.StatusBadRequest, blank.status, blank.raw)
		assert.Equal(t, "INVALID_FULL_NAME", blank.errorCode())
		assert.Zero(t, countRows(t, db, "pending_registration"))
	})

	t.Run("legacy ordination_place is rejected", func(t *testing.T) {
		api.t = t
		res := request(api.nextPhone(), "legacy@e2e.example.com", map[string]any{"ordination_place": "Chùa Hoằng Pháp"})
		require.Equal(t, http.StatusBadRequest, res.status, res.raw)
		assert.Equal(t, "VALIDATION_ERROR", res.errorCode())
	})

	t.Run("valid id reaches the user", func(t *testing.T) {
		api.t = t
		phone := api.nextPhone()
		res := request(phone, "withtemple@e2e.example.com", map[string]any{"ordination_temple_id": "temple_e2e_001"})
		require.Equal(t, http.StatusOK, res.status, res.raw)

		res = verify(phone)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var data struct {
			User map[string]any `json:"user"`
		}
		res.data(&data)
		assert.Equal(t, "Nguyen Van An", data.User["full_name"])
		assert.Equal(t, "temple_e2e_001", data.User["ordination_temple_id"])
		assert.NotContains(t, data.User, "ordination_place")
	})

	t.Run("no temple chosen is NULL and omitted", func(t *testing.T) {
		api.t = t
		phone := api.nextPhone()
		require.Equal(t, http.StatusOK, request(phone, "notemple@e2e.example.com", nil).status)

		res := verify(phone)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var data struct {
			User map[string]any `json:"user"`
		}
		res.data(&data)
		assert.NotContains(t, data.User, "ordination_temple_id")
	})
}
