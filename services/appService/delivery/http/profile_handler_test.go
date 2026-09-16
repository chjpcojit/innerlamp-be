package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

type mockProfileUsecase struct {
	mock.Mock
}

func (m *mockProfileUsecase) GetProfile(ctx context.Context, userID int64) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockProfileUsecase) UpdateProfile(ctx context.Context, userID int64, input domain.ProfileUpdateInput) (*domain.User, error) {
	args := m.Called(ctx, userID, input)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

// authenticatedRequest builds a request carrying the same context shape the
// auth middleware attaches (jwtcontext.WithUserID) — US-004 AC6 needs
// requests both with and without this set.
func authenticatedRequest(t *testing.T, method, body string, userID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, "/v1/users/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req = req.WithContext(jwtcontext.WithUserID(req.Context(), userID))
	}
	return req
}

// TestProfileHandler_GetSuccess covers AC1: an authenticated GET returns
// the full profile, with unset fields present as empty strings.
func TestProfileHandler_GetSuccess(t *testing.T) {
	uc := &mockProfileUsecase{}
	uc.On("GetProfile", mock.Anything, int64(7)).Return(&domain.User{
		ID: 7, Email: "new.user@example.com", PhoneNumber: "+84912345678",
	}, nil)

	h := NewProfileHandler(uc)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authenticatedRequest(t, http.MethodGet, "", "7"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("data field missing or wrong type: %v", got["data"])
	}
	if data["email"] != "new.user@example.com" {
		t.Errorf("data.email = %v, want new.user@example.com", data["email"])
	}
	// AC1: unset optional fields must render as "" rather than being
	// omitted or null.
	if v, ok := data["dharma_name"]; !ok || v != "" {
		t.Errorf("data.dharma_name = %v (present=%v), want empty string present", v, ok)
	}
	uc.AssertExpectations(t)
}

// TestProfileHandler_Unauthorized covers AC6: no authenticated user in the
// request context must return 401 without calling the use-case.
func TestProfileHandler_Unauthorized(t *testing.T) {
	uc := &mockProfileUsecase{}
	h := NewProfileHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authenticatedRequest(t, http.MethodGet, "", ""))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	uc.AssertNotCalled(t, "GetProfile", mock.Anything, mock.Anything)
}

// TestProfileHandler_PatchSuccess covers AC2: dharma name / ordination
// place are updated and echoed in the response, without reloading the app.
func TestProfileHandler_PatchSuccess(t *testing.T) {
	uc := &mockProfileUsecase{}
	uc.On("UpdateProfile", mock.Anything, int64(7), domain.ProfileUpdateInput{
		DharmaName: "Tam Dang", OrdinationPlace: "Chua Ba Da",
	}).Return(&domain.User{ID: 7, DharmaName: "Tam Dang", OrdinationPlace: "Chua Ba Da"}, nil)

	h := NewProfileHandler(uc)
	rec := httptest.NewRecorder()
	body := `{"dharma_name":"Tam Dang","ordination_place":"Chua Ba Da"}`
	h.ServeHTTP(rec, authenticatedRequest(t, http.MethodPatch, body, "7"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeResponse(t, rec)
	data := got["data"].(map[string]any)
	if data["dharma_name"] != "Tam Dang" || data["ordination_place"] != "Chua Ba Da" {
		t.Errorf("data = %v, unexpected fields", data)
	}
	uc.AssertExpectations(t)
}

// TestProfileHandler_PatchClearsFields covers AC5: blank fields in the
// request are passed straight through so the use-case clears them.
func TestProfileHandler_PatchClearsFields(t *testing.T) {
	uc := &mockProfileUsecase{}
	uc.On("UpdateProfile", mock.Anything, int64(7), domain.ProfileUpdateInput{}).
		Return(&domain.User{ID: 7}, nil)

	h := NewProfileHandler(uc)
	rec := httptest.NewRecorder()
	body := `{"dharma_name":"","ordination_place":""}`
	h.ServeHTTP(rec, authenticatedRequest(t, http.MethodPatch, body, "7"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	uc.AssertExpectations(t)
}

// TestProfileHandler_PatchRejectsEmailOrPhone covers AC4: email/phone
// number cannot be changed via this endpoint — sending either is rejected
// as an unknown field before the use-case is ever called.
func TestProfileHandler_PatchRejectsEmailOrPhone(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"email field", `{"email":"new@example.com"}`},
		{"phone field", `{"phone":"+84900000000"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &mockProfileUsecase{}
			h := NewProfileHandler(uc)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, authenticatedRequest(t, http.MethodPatch, tt.body, "7"))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			uc.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// TestProfileHandler_PatchUnauthorized covers AC6 for the PATCH path.
func TestProfileHandler_PatchUnauthorized(t *testing.T) {
	uc := &mockProfileUsecase{}
	h := NewProfileHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authenticatedRequest(t, http.MethodPatch, `{}`, ""))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	uc.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything)
}

// TestProfileHandler_MethodNotAllowed covers unsupported verbs on this
// endpoint (only GET/PATCH are defined).
func TestProfileHandler_MethodNotAllowed(t *testing.T) {
	uc := &mockProfileUsecase{}
	h := NewProfileHandler(uc)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authenticatedRequest(t, http.MethodDelete, "", "7"))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
}
