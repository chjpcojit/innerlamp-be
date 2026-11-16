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
	uc.On("SetPhoto", mock.Anything, int64(7), int64(9), mock.Anything).Return(sampleMemorial(), nil)

	rec := serve(newMemorialMux(uc), photoUploadRequest(t, "7", []byte("\x89PNG\r\n\x1a\n")))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	uc.AssertExpectations(t)
}

func TestMemorialHandler_UploadPhoto_NotFoundBeforeReadingBody(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(nil, domain.ErrMemorialNotFound)

	rec := serve(newMemorialMux(uc), photoUploadRequest(t, "7", []byte("anything")))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	uc.AssertNotCalled(t, "SetPhoto", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMemorialHandler_UploadPhoto_UnsupportedType(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)
	uc.On("SetPhoto", mock.Anything, int64(7), int64(9), mock.Anything).Return(nil, domain.ErrPhotoUnsupportedType)

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

// photoUploadRequestField is photoUploadRequest with a custom part name, so a
// test can send a multipart body with no "photo" part at all.
func photoUploadRequestField(t *testing.T, field string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(field, "photo.png")
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPut, "/v1/memorials/9/photo", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	claims := &jwtclaims.CustomClaims{UserID: "7", TokenType: jwtclaims.TokenTypeAccess, RegisteredClaims: jwt.RegisteredClaims{}}
	return req.WithContext(jwtcontext.WithClaims(req.Context(), claims))
}

// The cap is on the file, not on the multipart body: a file of exactly the
// limit must be accepted even though the body around it is a few bytes larger.
func TestMemorialHandler_UploadPhoto_FileAtLimit(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)
	uc.On("SetPhoto", mock.Anything, int64(7), int64(9), mock.Anything).Return(sampleMemorial(), nil)

	rec := serve(newMemorialMux(uc), photoUploadRequest(t, "7", bytes.Repeat([]byte("x"), 5<<20)))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestMemorialHandler_UploadPhoto_FileOverLimitByOne(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)

	rec := serve(newMemorialMux(uc), photoUploadRequest(t, "7", bytes.Repeat([]byte("x"), 5<<20+1)))

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	uc.AssertNotCalled(t, "SetPhoto", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMemorialHandler_UploadPhoto_MissingPartEnvelope(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)

	rec := serve(newMemorialMux(uc), photoUploadRequestField(t, "file", []byte("x")))

	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	body := decodeErrorBody(t, rec)
	assert.Equal(t, "VALIDATION_ERROR", body.Code)
	assert.Equal(t, "photo", body.Details["field"])
}

func TestMemorialHandler_UploadPhoto_EmptyFileEnvelope(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(sampleMemorial(), nil)

	rec := serve(newMemorialMux(uc), photoUploadRequest(t, "7", nil))

	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	body := decodeErrorBody(t, rec)
	assert.Equal(t, "VALIDATION_ERROR", body.Code)
	assert.Equal(t, "photo", body.Details["field"])
}

// An expired signature is a refresh signal: 403 with its own code, never 400
// and never "not found".
func TestMemorialHandler_Get_ExpiredPhotoURLIsForbidden(t *testing.T) {
	uc := new(mockMemorialUsecase)
	uc.On("Get", mock.Anything, int64(7), int64(9)).Return(nil, domain.ErrPhotoURLExpired)

	rec := serve(newMemorialMux(uc), memorialReq(http.MethodGet, "/v1/memorials/9", "", "7"))

	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	assert.Equal(t, "PHOTO_URL_EXPIRED", errCode(t, rec))
}
