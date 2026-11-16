package http

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	jwtclaims "chjpcoj.com.innerlamp/pkg/jwt/claims"
	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func photoUploadRequest(t *testing.T, userID string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("photo", "photo.png")
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPut, "/v1/memorials/9/photo", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	claims := &jwtclaims.CustomClaims{UserID: userID, TokenType: jwtclaims.TokenTypeAccess, RegisteredClaims: jwt.RegisteredClaims{}}
	return req.WithContext(jwtcontext.WithClaims(req.Context(), claims))
}

func TestMemorialHandler_UploadPhoto(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)
	uc.On("SetPhoto", mock.Anything, int64(7), int64(9), mock.Anything, "image/png", int64(8)).Return(sampleMemorial(), nil)

	rec := serve(newMemorialMux(uc), photoUploadRequest(t, "7", []byte("\x89PNG\r\n\x1a\n")))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	uc.AssertExpectations(t)
}

func TestMemorialHandler_UploadPhoto_NotFoundBeforeReadingBody(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(nil, domain.ErrMemorialNotFound)

	rec := serve(newMemorialMux(uc), photoUploadRequest(t, "7", []byte("anything")))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	uc.AssertNotCalled(t, "SetPhoto", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMemorialHandler_UploadPhoto_UnsupportedType(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)
	uc.On("SetPhoto", mock.Anything, int64(7), int64(9), mock.Anything, "text/plain; charset=utf-8", int64(5)).Return(nil, domain.ErrPhotoUnsupportedType)

	rec := serve(newMemorialMux(uc), photoUploadRequest(t, "7", []byte("hello")))

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestMemorialHandler_UploadPhoto_TooLarge(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)
	h := NewMemorialHandler(uc).WithPhotoLimit(10)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /v1/memorials/{id}/photo", h.UploadPhoto)

	rec := serve(mux, photoUploadRequest(t, "7", bytes.Repeat([]byte("x"), 100)))

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
}

func TestMemorialHandler_DeletePhoto(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("RemovePhoto", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/memorials/9/photo", nil)
	claims := &jwtclaims.CustomClaims{UserID: "7", TokenType: jwtclaims.TokenTypeAccess, RegisteredClaims: jwt.RegisteredClaims{}}
	req = req.WithContext(jwtcontext.WithClaims(req.Context(), claims))

	rec := serve(newMemorialMux(uc), req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	uc.AssertExpectations(t)
}
