package http

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newProfileMux registers the routes the way cmd/main.go does (minus the auth
// middleware, whose claims the test requests carry directly).
func newProfileMux(uc *mockProfileUsecase) *http.ServeMux {
	h := NewProfileHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/users/me", h.Get)
	mux.HandleFunc("PATCH /v1/users/me", h.Patch)
	mux.HandleFunc("POST /v1/users/me/avatar/upload-url", h.AvatarUploadURL)
	mux.HandleFunc("POST /v1/users/me/avatar/confirm", h.AvatarConfirm)
	mux.HandleFunc("DELETE /v1/users/me/avatar", h.AvatarDelete)
	return mux
}

func plainView() *domain.ProfileView {
	return &domain.ProfileView{Profile: domain.Profile{
		UserID: 7, Email: "an@example.com", PhoneNumber: "0901234567",
	}}
}

func fullView() *domain.ProfileView {
	name, url := "Thích Minh An", "https://storage.example/avatars/7/a.jpg?sig=1"
	exp := time.Date(2026, 10, 11, 12, 15, 0, 0, time.UTC)
	return &domain.ProfileView{
		Profile: domain.Profile{
			UserID: 7, Email: "an@example.com", EmailVerified: true, PhoneNumber: "0901234567",
			DharmaName:       &name,
			OrdinationTemple: &domain.TempleRef{ID: "temple_north_011", Name: "Chùa Bích Động"},
			AvatarObject:     strp("avatars/7/a.jpg"),
		},
		AvatarURL: &url, AvatarURLExpiresAt: &exp,
	}
}

func str(v string) domain.OptionalString { return domain.OptionalString{Set: true, Value: &v} }
func null() domain.OptionalString        { return domain.OptionalString{Set: true} }

// ---- GET /v1/users/me --------------------------------------------------

func TestProfileHandler_Get_FullProfile(t *testing.T) {
	uc := new(mockProfileUsecase)
	uc.On("Get", mock.Anything, int64(7)).Return(fullView(), nil)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodGet, "/v1/users/me", "", "7"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := decode(t, rec)
	assert.Equal(t, true, body["success"])
	data := body["data"].(map[string]any)
	assert.Equal(t, "an@example.com", data["email"])
	assert.Equal(t, true, data["email_verified"])
	assert.Equal(t, "0901234567", data["phone_number"])
	assert.Equal(t, "Thích Minh An", data["dharma_name"])
	assert.Equal(t, map[string]any{"id": "temple_north_011", "name": "Chùa Bích Động"}, data["ordination_temple"])
	assert.Equal(t, "https://storage.example/avatars/7/a.jpg?sig=1", data["avatar_url"])
	assert.Equal(t, "2026-10-11T12:15:00Z", data["avatar_url_expires_at"])
}

func TestProfileHandler_Get_OmitsKeysThatHaveNoValue(t *testing.T) {
	uc := new(mockProfileUsecase)
	uc.On("Get", mock.Anything, int64(7)).Return(plainView(), nil)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodGet, "/v1/users/me", "", "7"))

	require.Equal(t, http.StatusOK, rec.Code)
	data := decode(t, rec)["data"].(map[string]any)
	for _, key := range []string{"dharma_name", "ordination_temple", "avatar_url", "avatar_url_expires_at"} {
		assert.NotContains(t, data, key, "AC2: absent, not null")
	}
	assert.Equal(t, false, data["email_verified"], "email_verified is always present")
	assert.Contains(t, data, "email")
	assert.Contains(t, data, "phone_number")
}

func TestProfileHandler_Get_NeverExposesInternals(t *testing.T) {
	uc := new(mockProfileUsecase)
	uc.On("Get", mock.Anything, int64(7)).Return(fullView(), nil)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodGet, "/v1/users/me", "", "7"))

	data := decode(t, rec)["data"].(map[string]any)
	for _, key := range []string{"id", "user_id", "password_hash", "avatar_object", "created_at", "updated_at"} {
		assert.NotContains(t, data, key, "AC6")
	}
}

func TestProfileHandler_Get_WithoutClaimsIs401(t *testing.T) {
	uc := new(mockProfileUsecase)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodGet, "/v1/users/me", "", ""))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "UNAUTHORIZED", errCode(t, rec))
	uc.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestProfileHandler_Get_AccountGoneIs401(t *testing.T) {
	uc := new(mockProfileUsecase)
	uc.On("Get", mock.Anything, int64(7)).Return(nil, domain.ErrUserNotFound)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodGet, "/v1/users/me", "", "7"))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "UNAUTHORIZED", errCode(t, rec))
}

func TestProfileHandler_Get_UnexpectedErrorIs500WithoutLeak(t *testing.T) {
	uc := new(mockProfileUsecase)
	uc.On("Get", mock.Anything, int64(7)).Return(nil, errors.New("pq: connection refused at 10.0.0.5"))

	rec := serve(newProfileMux(uc), memorialReq(http.MethodGet, "/v1/users/me", "", "7"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "10.0.0.5")
}

// ---- PATCH /v1/users/me ------------------------------------------------

func TestProfileHandler_Patch_TriState(t *testing.T) {
	tests := []struct {
		name string
		body string
		want domain.ProfilePatch
	}{
		{"both keys absent", `{}`, domain.ProfilePatch{}},
		{"full name", `{"full_name":"Nguyễn Văn An"}`, domain.ProfilePatch{FullName: str("Nguyễn Văn An")}},
		{"full name null is passed on, the use case rejects it", `{"full_name":null}`, domain.ProfilePatch{FullName: null()}},
		{"only dharma name", `{"dharma_name":"Thanh Liên"}`, domain.ProfilePatch{DharmaName: str("Thanh Liên")}},
		{"dharma name null clears", `{"dharma_name":null}`, domain.ProfilePatch{DharmaName: null()}},
		{"dharma name empty string is a value, normalised later", `{"dharma_name":""}`, domain.ProfilePatch{DharmaName: str("")}},
		{"temple null clears", `{"ordination_temple_id":null}`, domain.ProfilePatch{OrdinationTempleID: null()}},
		{"both", `{"dharma_name":"A","ordination_temple_id":"t1"}`,
			domain.ProfilePatch{DharmaName: str("A"), OrdinationTempleID: str("t1")}},
		{"name set, temple cleared", `{"dharma_name":"A","ordination_temple_id":null}`,
			domain.ProfilePatch{DharmaName: str("A"), OrdinationTempleID: null()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := new(mockProfileUsecase)
			var got domain.ProfilePatch
			uc.On("Update", mock.Anything, int64(7), mock.Anything).
				Run(func(args mock.Arguments) { got = args.Get(2).(domain.ProfilePatch) }).
				Return(plainView(), nil)

			rec := serve(newProfileMux(uc), memorialReq(http.MethodPatch, "/v1/users/me", tt.body, "7"))

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProfileHandler_Patch_RejectsFieldsThatAreNotEditable(t *testing.T) {
	for _, body := range []string{
		`{"email":"x@example.com"}`,
		`{"phone_number":"0900000000"}`,
		`{"dharma_name":"A","email":"x@example.com"}`, // the valid field must not slip through
		`{"id":1}`,
		`{"avatar_url":"https://x"}`,
		`{"email_verified":true}`,
	} {
		t.Run(body, func(t *testing.T) {
			uc := new(mockProfileUsecase)

			rec := serve(newProfileMux(uc), memorialReq(http.MethodPatch, "/v1/users/me", body, "7"))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec))
			uc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestProfileHandler_Patch_RejectsMalformedBodies(t *testing.T) {
	for name, body := range map[string]string{
		"empty body":           ``,
		"not json":             `dharma_name=A`,
		"name is a number":     `{"dharma_name":5}`,
		"name is an object":    `{"dharma_name":{"a":1}}`,
		"temple is a boolean":  `{"ordination_temple_id":true}`,
		"two documents":        `{} {}`,
		"truncated":            `{"dharma_name":`,
		"array instead of obj": `[]`,
	} {
		t.Run(name, func(t *testing.T) {
			uc := new(mockProfileUsecase)

			rec := serve(newProfileMux(uc), memorialReq(http.MethodPatch, "/v1/users/me", body, "7"))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec))
			uc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestProfileHandler_Patch_WrongContentTypeIs400(t *testing.T) {
	uc := new(mockProfileUsecase)
	req := memorialReq(http.MethodPatch, "/v1/users/me", `{"dharma_name":"A"}`, "7")
	req.Header.Set("Content-Type", "text/plain")

	rec := serve(newProfileMux(uc), req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec))
}

func TestProfileHandler_Patch_WithoutClaimsIs401(t *testing.T) {
	uc := new(mockProfileUsecase)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodPatch, "/v1/users/me", `{}`, ""))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	uc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestProfileHandler_Patch_BusinessErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantField  string
	}{
		{"full name invalid", domain.ErrInvalidFullName, 400, "INVALID_FULL_NAME", "full_name"},
		{"dharma name too long", domain.ErrInvalidDharmaName, 400, "INVALID_DHARMA_NAME", "dharma_name"},
		{"unknown temple", domain.ErrInvalidTemple, 400, "INVALID_TEMPLE", "ordination_temple_id"},
		{"wrapped sentinel still maps", errors.Join(errors.New("update profile 7"), domain.ErrInvalidTemple), 400, "INVALID_TEMPLE", "ordination_temple_id"},
		{"validation error names its field", &domain.ProfileValidationError{Field: "upload_id", Message: "bad"}, 400, "VALIDATION_ERROR", "upload_id"},
		{"account gone", domain.ErrUserNotFound, 401, "UNAUTHORIZED", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := new(mockProfileUsecase)
			uc.On("Update", mock.Anything, int64(7), mock.Anything).Return(nil, tt.err)

			rec := serve(newProfileMux(uc), memorialReq(http.MethodPatch, "/v1/users/me", `{"dharma_name":"A"}`, "7"))

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			assert.Equal(t, tt.wantCode, errCode(t, rec))
			if tt.wantField != "" {
				details := decode(t, rec)["details"].(map[string]any)
				assert.Equal(t, tt.wantField, details["field"])
			}
		})
	}
}

// ---- POST /v1/users/me/avatar/upload-url --------------------------------

func TestProfileHandler_AvatarUploadURL_ReturnsTicket(t *testing.T) {
	uc := new(mockProfileUsecase)
	var got domain.AvatarUploadRequest
	uc.On("RequestAvatarUpload", mock.Anything, int64(7), mock.Anything).
		Run(func(args mock.Arguments) { got = args.Get(2).(domain.AvatarUploadRequest) }).
		Return(&domain.AvatarUploadTicket{
			UploadID: "0b9c4f3e-5a1d-4c7e-9b2a-6d8e1f0a3c11", URL: "https://storage.example/put?sig=1", Method: "PUT",
			Headers:   map[string]string{"Content-Type": "image/jpeg"},
			ExpiresAt: time.Date(2026, 10, 11, 12, 5, 0, 0, time.UTC),
		}, nil)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodPost, "/v1/users/me/avatar/upload-url",
		`{"content_type":"image/jpeg","size_bytes":2097152}`, "7"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, domain.AvatarUploadRequest{ContentType: "image/jpeg", SizeBytes: 2097152}, got)
	data := decode(t, rec)["data"].(map[string]any)
	assert.Equal(t, "0b9c4f3e-5a1d-4c7e-9b2a-6d8e1f0a3c11", data["upload_id"])
	assert.Equal(t, "https://storage.example/put?sig=1", data["upload_url"])
	assert.Equal(t, "PUT", data["method"])
	assert.Equal(t, map[string]any{"Content-Type": "image/jpeg"}, data["headers"])
	assert.Equal(t, "2026-10-11T12:05:00Z", data["expires_at"])
}

func TestProfileHandler_AvatarUploadURL_ValidationFieldIsReported(t *testing.T) {
	uc := new(mockProfileUsecase)
	uc.On("RequestAvatarUpload", mock.Anything, int64(7), mock.Anything).
		Return(nil, &domain.ProfileValidationError{Field: "size_bytes", Message: "too big"})

	rec := serve(newProfileMux(uc), memorialReq(http.MethodPost, "/v1/users/me/avatar/upload-url",
		`{"content_type":"image/png","size_bytes":99999999}`, "7"))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec))
	details := decode(t, rec)["details"].(map[string]any)
	assert.Equal(t, "size_bytes", details["field"])
}

func TestProfileHandler_AvatarUploadURL_RejectsUnknownFieldsAndBadTypes(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":   `{"content_type":"image/png","size_bytes":1,"user_id":9}`,
		"size is text":    `{"content_type":"image/png","size_bytes":"1"}`,
		"size is decimal": `{"content_type":"image/png","size_bytes":1.5}`,
		"empty":           ``,
	} {
		t.Run(name, func(t *testing.T) {
			uc := new(mockProfileUsecase)

			rec := serve(newProfileMux(uc), memorialReq(http.MethodPost, "/v1/users/me/avatar/upload-url", body, "7"))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			uc.AssertNotCalled(t, "RequestAvatarUpload", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// ---- POST /v1/users/me/avatar/confirm -----------------------------------

func TestProfileHandler_AvatarConfirm_ReturnsProfile(t *testing.T) {
	uc := new(mockProfileUsecase)
	uc.On("ConfirmAvatar", mock.Anything, int64(7), "0b9c4f3e-5a1d-4c7e-9b2a-6d8e1f0a3c11").Return(fullView(), nil)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodPost, "/v1/users/me/avatar/confirm",
		`{"upload_id":"0b9c4f3e-5a1d-4c7e-9b2a-6d8e1f0a3c11"}`, "7"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := decode(t, rec)["data"].(map[string]any)
	assert.Contains(t, data, "avatar_url")
}

func TestProfileHandler_AvatarConfirm_Errors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"object not uploaded", domain.ErrAvatarUploadNotFound, 404, "AVATAR_UPLOAD_NOT_FOUND"},
		{"recorded file outside policy", domain.ErrInvalidAvatar, 400, "INVALID_AVATAR"},
		{"malformed id", &domain.ProfileValidationError{Field: "upload_id", Message: "bad"}, 400, "VALIDATION_ERROR"},
		{"storage down", errors.New("gcs: 503"), 500, "INTERNAL_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := new(mockProfileUsecase)
			uc.On("ConfirmAvatar", mock.Anything, int64(7), mock.Anything).Return(nil, tt.err)

			rec := serve(newProfileMux(uc), memorialReq(http.MethodPost, "/v1/users/me/avatar/confirm",
				`{"upload_id":"x"}`, "7"))

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			assert.Equal(t, tt.wantCode, errCode(t, rec))
		})
	}
}

func TestProfileHandler_AvatarConfirm_RejectsUnknownFields(t *testing.T) {
	uc := new(mockProfileUsecase)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodPost, "/v1/users/me/avatar/confirm",
		`{"upload_id":"x","object_key":"avatars/9/other.jpg"}`, "7"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	uc.AssertNotCalled(t, "ConfirmAvatar", mock.Anything, mock.Anything, mock.Anything)
}

// ---- DELETE /v1/users/me/avatar ------------------------------------------

func TestProfileHandler_AvatarDelete_ReturnsProfileWithoutAvatar(t *testing.T) {
	uc := new(mockProfileUsecase)
	uc.On("RemoveAvatar", mock.Anything, int64(7)).Return(plainView(), nil)

	rec := serve(newProfileMux(uc), memorialReq(http.MethodDelete, "/v1/users/me/avatar", "", "7"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := decode(t, rec)["data"].(map[string]any)
	assert.NotContains(t, data, "avatar_url")
}

func TestProfileHandler_AvatarRoutes_WithoutClaimsAre401(t *testing.T) {
	for _, r := range []struct{ method, path string }{
		{http.MethodPost, "/v1/users/me/avatar/upload-url"},
		{http.MethodPost, "/v1/users/me/avatar/confirm"},
		{http.MethodDelete, "/v1/users/me/avatar"},
	} {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			uc := new(mockProfileUsecase)

			rec := serve(newProfileMux(uc), memorialReq(r.method, r.path, `{}`, ""))

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Empty(t, uc.Calls, "AC27: nothing may run without a token")
		})
	}
}
