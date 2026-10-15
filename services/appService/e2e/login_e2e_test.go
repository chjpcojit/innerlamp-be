package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"chjpcoj.com.innerlamp/pkg/jwt"
)

// TestLoginFlow_E2E drives US-002 (docs/specs/features/login/requirements.md,
// AC1–AC15) against the real binary and a fresh Postgres. Every subtest
// registers its own phone number, so they share one server without sharing
// state.
//
// What is here and not in the per-layer tests: that cmd/main.go wraps logout
// in the access-token middleware (AC15), that config values reach the running
// server (AC7 lock threshold/duration, AC8 token lifetimes), and flows that
// span several HTTP calls with real DB state in between (AC4, AC9–AC14).
func TestLoginFlow_E2E(t *testing.T) {
	baseURL, client := newE2EServer(t)
	api := &apiClient{t: t, baseURL: baseURL, http: client}

	t.Run("AC1/AC8: login returns tokens, user and the configured lifetimes; local phone form accepted", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()

		// AC1: 0xxxxxxxxx normalises to the same +84 account.
		localForm := "0" + strings.TrimPrefix(phone, "+84")
		res := api.post("/v1/auth/login", loginBody(localForm, testPassword, deviceA), "")
		require.Equal(t, http.StatusOK, res.status, res.raw)

		var data struct {
			AccessToken  string         `json:"access_token"`
			RefreshToken string         `json:"refresh_token"`
			User         map[string]any `json:"user"`
		}
		res.data(&data)
		assert.Equal(t, phone, data.User["phone_number"])

		access := decodeClaims(t, data.AccessToken)
		assert.Equal(t, "access", access["token_type"])
		assert.EqualValues(t, 15*60, access["exp"].(float64)-access["iat"].(float64), "AC8: access token lives 15m")
		sid, _ := access["extra"].(map[string]any)["sid"].(string)
		assert.NotEmpty(t, sid, "AC8: access token carries sid")

		refresh := decodeClaims(t, data.RefreshToken)
		assert.Equal(t, "refresh", refresh["token_type"])
		assert.EqualValues(t, 168*3600, refresh["exp"].(float64)-refresh["iat"].(float64), "AC8: refresh token lives 7 days")
	})

	t.Run("AC2: wrong password, unknown phone and pending-only phone are byte-identical 401s", func(t *testing.T) {
		api.t = t
		registered := api.registerUser()
		pending := api.nextPhone()
		api.requestOTP(pending) // pending_registration only, never verified

		bodies := map[string]string{}
		for name, phone := range map[string]string{
			"wrong password": registered,
			"unknown phone":  api.nextPhone(),
			"pending only":   pending,
		} {
			pw := "Wrong-pass-123"
			res := api.post("/v1/auth/login", loginBody(phone, pw, deviceA), "")
			require.Equal(t, http.StatusUnauthorized, res.status, name)
			assert.Equal(t, "INVALID_CREDENTIALS", res.errorCode(), name)
			bodies[name] = res.raw
		}
		assert.Equal(t, bodies["wrong password"], bodies["unknown phone"])
		assert.Equal(t, bodies["wrong password"], bodies["pending only"])
	})

	t.Run("AC3/AC4: a new device keeps the old one; the same device again supersedes only itself", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()

		a1 := api.login(phone, deviceA)
		b := api.login(phone, deviceB)
		a2 := api.login(phone, deviceA)

		assert.Equal(t, http.StatusUnauthorized, api.refresh(a1.RefreshToken).status, "AC4: device A's first session is superseded")
		assert.Equal(t, http.StatusOK, api.refresh(b.RefreshToken).status, "AC3: device B untouched")
		assert.Equal(t, http.StatusOK, api.refresh(a2.RefreshToken).status, "AC4: device A's new session works")
	})

	t.Run("AC5/AC6: malformed requests are 400 and never count toward the lock", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()

		for i := 0; i < 5; i++ {
			res := api.post("/v1/auth/login", map[string]any{
				"phone_number": phone, "password": testPassword, "device": map[string]any{"platform": "ios"},
			}, "")
			require.Equal(t, http.StatusBadRequest, res.status)
			require.Equal(t, "INVALID_DEVICE", res.errorCode())

			res = api.post("/v1/auth/login", loginBody(phone, "", deviceA), "")
			require.Equal(t, http.StatusBadRequest, res.status)
			require.Equal(t, "INVALID_PASSWORD", res.errorCode())

			res = api.post("/v1/auth/login", loginBody("not-a-phone", testPassword, deviceA), "")
			require.Equal(t, http.StatusBadRequest, res.status)
			require.Equal(t, "INVALID_PHONE", res.errorCode())
		}

		api.login(phone, deviceA) // 200 — 15 rejected requests did not lock it
	})

	t.Run("AC7: fifth failure locks for 15 minutes, even the correct password and even with no account", func(t *testing.T) {
		api.t = t
		for name, phone := range map[string]string{
			"registered": api.registerUser(),
			"no account": api.nextPhone(),
		} {
			for i := 0; i < 5; i++ {
				res := api.post("/v1/auth/login", loginBody(phone, "Wrong-pass-123", deviceA), "")
				require.Equal(t, http.StatusUnauthorized, res.status, "%s attempt %d", name, i+1)
			}

			res := api.post("/v1/auth/login", loginBody(phone, testPassword, deviceA), "")
			require.Equal(t, http.StatusTooManyRequests, res.status, name)
			require.Equal(t, "LOGIN_LOCKED", res.errorCode(), name)

			lockedUntil, err := time.Parse(time.RFC3339Nano, res.details()["locked_until"].(string))
			require.NoError(t, err)
			assert.WithinDuration(t, time.Now().Add(15*time.Minute), lockedUntil, time.Minute, name)
		}
	})

	t.Run("AC9/AC10/AC11: rotate, reject bad tokens, and revoke on reuse of the previous token", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()
		other := api.login(phone, deviceB)
		first := api.login(phone, deviceA)

		res := api.refresh(first.RefreshToken)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var second tokenPair
		res.data(&second)
		assert.NotEqual(t, first.RefreshToken, second.RefreshToken, "AC9: refresh token rotates")
		assert.Equal(t, sidOf(t, first.AccessToken), sidOf(t, second.AccessToken), "AC9: same session, same sid")

		for name, token := range map[string]string{
			"garbage":                 "not.a.jwt",
			"access token as refresh": second.AccessToken,
		} {
			res := api.refresh(token)
			assert.Equal(t, http.StatusUnauthorized, res.status, name)
			assert.Equal(t, "REFRESH_TOKEN_INVALID", res.errorCode(), name)
		}

		res = api.refresh(first.RefreshToken) // AC11: the previous generation again
		assert.Equal(t, http.StatusUnauthorized, res.status)
		assert.Equal(t, "REFRESH_TOKEN_INVALID", res.errorCode())

		assert.Equal(t, http.StatusUnauthorized, api.refresh(second.RefreshToken).status, "AC11: the session itself is revoked")
		assert.Equal(t, http.StatusOK, api.refresh(other.RefreshToken).status, "AC11: other device untouched")
	})

	t.Run("AC12: a device-less verify-otp session refreshes and logs out", func(t *testing.T) {
		api.t = t
		phone := api.nextPhone()
		api.requestOTP(phone)
		reg := api.verifyOTP(phone)
		require.NotEmpty(t, sidOf(t, reg.AccessToken), "US-001 access token carries sid")

		res := api.refresh(reg.RefreshToken)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var rotated tokenPair
		res.data(&rotated)

		res = api.post("/v1/auth/logout", nil, rotated.AccessToken)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, http.StatusUnauthorized, api.refresh(rotated.RefreshToken).status)
	})

	t.Run("AC13/AC14: logout is 200 {success:true}, idempotent, and ends only its own session", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()
		other := api.login(phone, deviceB)
		mine := api.login(phone, deviceA)

		for i := 0; i < 2; i++ { // second call: AC14 idempotent
			res := api.post("/v1/auth/logout", nil, mine.AccessToken)
			require.Equal(t, http.StatusOK, res.status, res.raw)
			assert.Equal(t, "application/json", res.contentType)
			assert.JSONEq(t, `{"success": true}`, res.raw)
		}

		assert.Equal(t, http.StatusUnauthorized, api.refresh(mine.RefreshToken).status, "AC13: refresh after logout fails")
		assert.Equal(t, http.StatusOK, api.refresh(other.RefreshToken).status, "AC13: other device still logged in")
	})

	t.Run("AC14: a genuine token naming another user's sid revokes nothing", func(t *testing.T) {
		api.t = t
		victim := api.login(api.registerUser(), deviceA)
		attacker := api.login(api.registerUser(), deviceA)
		attackerID := decodeClaims(t, attacker.AccessToken)["user_id"].(string)

		forged := mintAccessToken(t, attackerID, map[string]any{"sid": sidOf(t, victim.AccessToken)}, 15*time.Minute)
		res := api.post("/v1/auth/logout", nil, forged)
		require.Equal(t, http.StatusOK, res.status, res.raw)

		assert.Equal(t, http.StatusOK, api.refresh(victim.RefreshToken).status, "victim's session survives")
		assert.Equal(t, http.StatusOK, api.refresh(attacker.RefreshToken).status, "attacker's own session untouched too")
	})

	// AC15 alone cannot tell whether cmd/main.go wraps logout: LoginHandler.Logout
	// also answers 401 UNAUTHORIZED when the context has no claims, so every case
	// below stays 401 with the middleware removed. What catches a missing wrap is
	// the positive path — AC12 and AC13 send a valid access token and need 200,
	// which only happens if the middleware put the claims there (checked
	// 2026-09-21 by unwrapping the route: AC12 and AC13 fail, AC15 still passes).
	// Likewise RequireTokenType "access" is not observable here: a refresh token
	// carries no sid, so the handler rejects it even if the middleware lets it in.
	t.Run("AC15: logout rejects every unusable bearer without touching state", func(t *testing.T) {
		api.t = t
		phone := api.registerUser()
		victim := api.login(phone, deviceA)
		userID := decodeClaims(t, victim.AccessToken)["user_id"].(string)

		// pkg/jwt refuses a non-positive TTL, so an expired token is minted with
		// the shortest one and outlived. exp has 1-second precision and the
		// middleware applies no leeway.
		expired := mintAccessToken(t, userID, map[string]any{"sid": sidOf(t, victim.AccessToken)}, time.Second)
		time.Sleep(2 * time.Second)

		cases := map[string]string{
			"no Authorization header":  "",
			"malformed token":          "not.a.jwt",
			"refresh token as bearer":  victim.RefreshToken,
			"access token without sid": mintAccessToken(t, userID, nil, 15*time.Minute),
			"expired access token":     expired,
		}
		for name, bearer := range cases {
			res := api.post("/v1/auth/logout", nil, bearer)
			assert.Equal(t, http.StatusUnauthorized, res.status, name)
			assert.Equal(t, "UNAUTHORIZED", res.errorCode(), name)
		}

		assert.Equal(t, http.StatusOK, api.refresh(victim.RefreshToken).status, "no rejected logout revoked the session")
	})

	t.Run("AC20/AC21/AC22: verify-otp, login and refresh are no-store and report expires_in", func(t *testing.T) {
		api.t = t
		phone := api.nextPhone()
		api.requestOTP(phone)

		verify := api.post("/v1/auth/register/verify-otp", map[string]any{"email": e2eEmail(phone), "otp_code": devOTPCode}, "")
		require.Equal(t, http.StatusCreated, verify.status, verify.raw)
		login := api.post("/v1/auth/login", loginBody(phone, testPassword, deviceA), "")
		require.Equal(t, http.StatusOK, login.status, login.raw)
		var pair struct {
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
		}
		login.data(&pair)
		refresh := api.refresh(pair.RefreshToken)
		require.Equal(t, http.StatusOK, refresh.status, refresh.raw)

		for name, res := range map[string]*apiResponse{"verify-otp": verify, "login": login, "refresh": refresh} {
			assert.Equal(t, "no-store", res.header.Get("Cache-Control"), name+": AC20")
			assert.Equal(t, "no-cache", res.header.Get("Pragma"), name+": AC20")
			var body struct {
				ExpiresIn int64          `json:"expires_in"`
				User      map[string]any `json:"user"`
			}
			res.data(&body)
			assert.EqualValues(t, 15*60, body.ExpiresIn, name+": AC21/AC22 equals the access token lifetime")
			if name == "refresh" {
				assert.Nil(t, body.User, "AC22: refresh carries no user")
			}
		}

		// AC20 also covers error responses.
		bad := api.refresh("not-a-token")
		require.Equal(t, http.StatusUnauthorized, bad.status, bad.raw)
		assert.Equal(t, "no-store", bad.header.Get("Cache-Control"), "AC20: errors are no-store too")
	})
}

// --- fixtures --------------------------------------------------------------

const testPassword = "Passw0rd!e2e"

var (
	deviceA = map[string]any{"platform": "ios", "device_model": "iPhone15,2", "os_version": "17.4"}
	deviceB = map[string]any{"platform": "android", "device_model": "Pixel 8", "os_version": "14"}
)

// phoneSeq makes every subtest's phone number unique within one run; the
// database is fresh per run, so collisions across runs cannot happen.
var phoneSeq atomic.Int64

type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func loginBody(phone, password string, device map[string]any) map[string]any {
	return map[string]any{"phone_number": phone, "password": password, "device": device}
}

// --- HTTP client -----------------------------------------------------------

type apiClient struct {
	t       *testing.T
	baseURL string
	http    *http.Client
}

type apiResponse struct {
	t           *testing.T
	status      int
	header      http.Header
	contentType string
	raw         string
	envelope    struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   *struct {
			Code string `json:"code"`
		} `json:"error"`
		Details map[string]any `json:"details"`
	}
}

func (r *apiResponse) data(dst any) {
	r.t.Helper()
	require.True(r.t, r.envelope.Success, r.raw)
	require.NoError(r.t, json.Unmarshal(r.envelope.Data, dst))
}

func (r *apiResponse) errorCode() string {
	if r.envelope.Error == nil {
		return ""
	}
	return r.envelope.Error.Code
}

func (r *apiResponse) details() map[string]any { return r.envelope.Details }

func (c *apiClient) post(path string, body any, bearer string) *apiResponse {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, reader)
	require.NoError(c.t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.http.Do(req)
	require.NoError(c.t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)

	res := &apiResponse{t: c.t, status: resp.StatusCode, header: resp.Header, contentType: resp.Header.Get("Content-Type"), raw: string(raw)}
	require.NoErrorf(c.t, json.Unmarshal(raw, &res.envelope), "%s %s: body is not the JSON envelope: %q", path, resp.Status, raw)
	return res
}

func (c *apiClient) nextPhone() string {
	return fmt.Sprintf("+849%08d", 10000000+phoneSeq.Add(1))
}

// devOTPCode is config.dev.yaml's otp.dev_fixed_code (⚠️ DEV ONLY): every code
// the E2E server issues is this value, so the harness need not read the
// mailer log.
const devOTPCode = "111111"

// e2eEmail is the account email the harness registers for phone. Login is
// still phone-based, so tests keep passing phones around and derive the email.
func e2eEmail(phone string) string {
	return strings.TrimPrefix(phone, "+") + "@e2e.example.com"
}

// requestOTP runs onboarding step 1 for phone (an E.164 "+84…" number from
// nextPhone), typed the way a VN user would: with the trunk 0.
func (c *apiClient) requestOTP(phone string) {
	c.t.Helper()
	res := c.post("/v1/auth/register/request-otp", map[string]any{
		"email":         e2eEmail(phone),
		"password":      testPassword,
		"full_name":     "Nguyen Van An",
		"phone_country": "VN",
		"phone_number":  "0" + strings.TrimPrefix(phone, "+84"),
	}, "")
	require.Equal(c.t, http.StatusOK, res.status, res.raw)
}

// verifyOTP completes onboarding step 2 with the dev fixed code.
func (c *apiClient) verifyOTP(phone string) tokenPair {
	c.t.Helper()
	res := c.post("/v1/auth/register/verify-otp", map[string]any{"email": e2eEmail(phone), "otp_code": devOTPCode}, "")
	require.Equal(c.t, http.StatusCreated, res.status, res.raw)
	var pair tokenPair
	res.data(&pair)
	return pair
}

func (c *apiClient) registerUser() string {
	c.t.Helper()
	phone := c.nextPhone()
	c.requestOTP(phone)
	c.verifyOTP(phone)
	return phone
}

func (c *apiClient) login(phone string, device map[string]any) tokenPair {
	c.t.Helper()
	res := c.post("/v1/auth/login", loginBody(phone, testPassword, device), "")
	require.Equal(c.t, http.StatusOK, res.status, res.raw)
	var pair tokenPair
	res.data(&pair)
	return pair
}

func (c *apiClient) refresh(token string) *apiResponse {
	c.t.Helper()
	return c.post("/v1/auth/refresh", map[string]any{"refresh_token": token}, "")
}

// --- tokens ----------------------------------------------------------------

// decodeClaims reads a JWT payload without verifying it — the server already
// did that; the test only inspects what it was given.
func decodeClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))
	return claims
}

func sidOf(t *testing.T, accessToken string) string {
	t.Helper()
	extra, _ := decodeClaims(t, accessToken)["extra"].(map[string]any)
	sid, _ := extra["sid"].(string)
	return sid
}

// mintAccessToken signs a token the server will accept as genuine, using the
// dev config's own secret, issuer and audience — the only way to reach the
// "valid signature but no sid" and "expired" branches over HTTP.
func mintAccessToken(t *testing.T, userID string, extra map[string]any, ttl time.Duration) string {
	t.Helper()
	raw, err := os.ReadFile(baseConfigPath)
	require.NoError(t, err)
	var doc struct {
		JWT struct {
			Algorithm       string `yaml:"algorithm"`
			SecretKeySecret string `yaml:"secret_key_secret"`
			Issuer          string `yaml:"issuer"`
			Audience        string `yaml:"audience"`
		} `yaml:"jwt"`
		DevSecrets map[string]string `yaml:"dev_secrets"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &doc))

	svc, err := jwt.New(&jwt.Config{
		Algorithm:              jwt.Algorithm(doc.JWT.Algorithm),
		SecretKey:              doc.DevSecrets[doc.JWT.SecretKeySecret],
		AccessTokenExpiration:  ttl,
		RefreshTokenExpiration: time.Hour,
		Issuer:                 doc.JWT.Issuer,
		Audience:               doc.JWT.Audience,
	})
	require.NoError(t, err)
	token, err := svc.GenerateAccessToken(userID, "", "user", extra)
	require.NoError(t, err)
	return token
}
