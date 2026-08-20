package http

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// avatarUsecase is the subset of *usecase.AvatarUsecase this handler
// depends on, defined here (the consumer) so it can be unit tested with a
// mock instead of the real use-case.
type avatarUsecase interface {
	UpdateAvatar(ctx context.Context, userID int64, reader io.Reader, contentType string) (*domain.User, error)
}

// maxAvatarBytes caps the accepted upload size (5MB) — a request body
// larger than this is rejected before it ever reaches storage.
const maxAvatarBytes = 5 << 20

// allowedAvatarContentTypes is the sniffed-content-type allowlist for
// avatar uploads. The client-supplied Content-Type header is never trusted
// for this decision (or passed to storage) — only http.DetectContentType's
// read of the actual body bytes is.
var allowedAvatarContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// AvatarHandler implements POST /v1/users/me/avatar (US-004 AC3). The
// request body is the raw image bytes — kept separate from PATCH
// /v1/users/me per the spec's technical note, since that endpoint only ever
// accepts JSON.
type AvatarHandler struct {
	usecase avatarUsecase
}

// NewAvatarHandler wires the HTTP handler to its use-case.
func NewAvatarHandler(uc avatarUsecase) *AvatarHandler {
	return &AvatarHandler{usecase: uc}
}

func (h *AvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, utils.ErrorTypeValidation, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	userID, ok := authenticatedUserID(r)
	if !ok {
		writeUnauthorized(w)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarBytes)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		if utils.IsRequestTooLarge(err) {
			utils.WriteValidationError(w, "FILE_TOO_LARGE", "avatar image exceeds the maximum allowed size", nil)
			return
		}
		utils.WriteValidationError(w, "INVALID_IMAGE", "unable to read uploaded image", nil)
		return
	}

	detectedType := http.DetectContentType(data)
	if !allowedAvatarContentTypes[detectedType] {
		utils.WriteValidationError(w, "INVALID_IMAGE", "unsupported image type, must be JPEG, PNG or WebP", map[string]any{"detected_type": detectedType})
		return
	}

	user, err := h.usecase.UpdateAvatar(r.Context(), userID, bytes.NewReader(data), detectedType)
	if err != nil {
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	utils.WriteSuccessResponse(w, newProfileResponse(user))
}
