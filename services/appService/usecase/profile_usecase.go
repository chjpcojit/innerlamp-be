package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// avatarExtensions maps an allowed content type to the extension of the stored
// object key. It follows the protocol (a media type has one canonical suffix),
// not the environment, so it is a constant rather than config. A type added to
// profile.avatar_content_types with no entry here is stored without a suffix.
var avatarExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ProfileUsecase serves a user's own profile: read, partial edit, and the
// avatar upload / confirm / remove flow (docs/specs/features/profile).
type ProfileUsecase struct {
	repo    domain.ProfileRepository
	storage domain.AvatarStorage
	cfg     config.ProfileConfig
	logger  logging.Logger

	now   func() time.Time
	newID func() string
}

// NewProfileUsecase wires the use case.
func NewProfileUsecase(repo domain.ProfileRepository, storage domain.AvatarStorage, cfg config.ProfileConfig, logger logging.Logger) *ProfileUsecase {
	return &ProfileUsecase{
		repo: repo, storage: storage, cfg: cfg, logger: logger,
		now:   time.Now,
		newID: uuid.NewString,
	}
}

func invalidProfile(field, message string) error {
	return &domain.ProfileValidationError{Field: field, Message: message}
}

// Get returns the profile of userID with its avatar address signed (AC1–AC6).
// domain.ErrUserNotFound means the account vanished after the token was issued.
func (u *ProfileUsecase) Get(ctx context.Context, userID int64) (*domain.ProfileView, error) {
	p, err := u.repo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return u.view(ctx, p)
}

// Update applies a partial edit (AC8–AC16). Everything is validated before the
// first write, so a bad field leaves every field of the request unapplied.
//
// Field order follows the request: full name, dharma name, then temple. A full name that is null, blank or over the limit is domain.ErrInvalidFullName. A dharma name over
// the limit is domain.ErrInvalidDharmaName; an unknown temple is
// domain.ErrInvalidTemple.
func (u *ProfileUsecase) Update(ctx context.Context, userID int64, in domain.ProfilePatch) (*domain.ProfileView, error) {
	patch, err := u.normalizePatch(ctx, in)
	if err != nil {
		return nil, err
	}

	var p *domain.Profile
	if patch.IsEmpty() {
		p, err = u.repo.Get(ctx, userID)
	} else {
		p, err = u.repo.Update(ctx, userID, patch)
	}
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return u.view(ctx, p)
}

// normalizePatch validates in and returns the patch to store. A key that was
// absent stays absent; null, blank and whitespace become "clear".
func (u *ProfileUsecase) normalizePatch(ctx context.Context, in domain.ProfilePatch) (domain.ProfilePatch, error) {
	var out domain.ProfilePatch

	if in.FullName.Set {
		if in.FullName.Value == nil {
			return domain.ProfilePatch{}, domain.ErrInvalidFullName
		}
		name, err := normalizeFullName(*in.FullName.Value, u.cfg.FullNameMaxChars)
		if err != nil {
			return domain.ProfilePatch{}, err
		}
		out.FullName = domain.OptionalString{Set: true, Value: &name}
	}

	if in.DharmaName.Set {
		out.DharmaName.Set = true
		if in.DharmaName.Value != nil {
			name, err := normalizeDharmaName(*in.DharmaName.Value, u.cfg.DharmaNameMaxChars)
			if err != nil {
				return domain.ProfilePatch{}, err
			}
			out.DharmaName.Value = name
		}
	}

	if in.OrdinationTempleID.Set {
		out.OrdinationTempleID.Set = true
		if in.OrdinationTempleID.Value != nil {
			if id := strings.TrimSpace(*in.OrdinationTempleID.Value); id != "" {
				exists, err := u.repo.TempleExists(ctx, id)
				if err != nil {
					return domain.ProfilePatch{}, fmt.Errorf("check temple: %w", err)
				}
				if !exists {
					return domain.ProfilePatch{}, domain.ErrInvalidTemple
				}
				out.OrdinationTempleID.Value = &id
			}
		}
	}
	return out, nil
}

// RequestAvatarUpload validates the claimed file and signs a URL that accepts
// exactly that upload into the user's temporary folder (AC18–AC20). Nothing is
// stored server side: the temporary key is derived again from the token's user
// id and the upload id at confirm time.
func (u *ProfileUsecase) RequestAvatarUpload(ctx context.Context, userID int64, req domain.AvatarUploadRequest) (*domain.AvatarUploadTicket, error) {
	if !u.contentTypeAllowed(req.ContentType) {
		return nil, invalidProfile("content_type", "content type is not allowed")
	}
	if req.SizeBytes < 1 || req.SizeBytes > u.cfg.AvatarMaxBytes {
		return nil, invalidProfile("size_bytes", fmt.Sprintf("must be between 1 and %d", u.cfg.AvatarMaxBytes))
	}

	uploadID := u.newID()
	ttl := u.cfg.UploadURLTTL.Duration()
	up, err := u.storage.SignUpload(ctx, u.tmpKey(userID, uploadID), req.ContentType, u.cfg.AvatarMaxBytes, ttl)
	if err != nil {
		return nil, fmt.Errorf("sign avatar upload: %w", err)
	}
	return &domain.AvatarUploadTicket{
		UploadID:  uploadID,
		URL:       up.URL,
		Method:    up.Method,
		Headers:   up.Headers,
		ExpiresAt: u.now().Add(ttl),
	}, nil
}

// ConfirmAvatar turns an uploaded temporary object into the user's avatar
// (AC22–AC25). The order is the point: copy to a fresh key first, so the row
// never points at an object that does not exist; write the row; and only then
// delete the old image, so a failure in between leaves the previous avatar
// intact and at worst an unreferenced file (TD-002).
func (u *ProfileUsecase) ConfirmAvatar(ctx context.Context, userID int64, uploadID string) (*domain.ProfileView, error) {
	// Only the canonical form this API issues is accepted: uuid.Parse alone also
	// takes braced and URN spellings, which would put odd text into an object key.
	if parsed, err := uuid.Parse(uploadID); err != nil || parsed.String() != uploadID {
		return nil, invalidProfile("upload_id", "must be an upload id returned by upload-url")
	}

	tmp := u.tmpKey(userID, uploadID)
	meta, err := u.storage.Stat(ctx, tmp)
	if err != nil {
		if errors.Is(err, domain.ErrObjectNotFound) {
			return nil, domain.ErrAvatarUploadNotFound
		}
		return nil, fmt.Errorf("stat avatar upload: %w", err)
	}
	// Judge what storage recorded, not what the client claimed at upload-url.
	if !u.contentTypeAllowed(meta.ContentType) || meta.Size < 1 || meta.Size > u.cfg.AvatarMaxBytes {
		return nil, domain.ErrInvalidAvatar
	}

	newKey := u.avatarKey(userID, meta.ContentType)
	if err := u.storage.Copy(ctx, tmp, newKey); err != nil {
		return nil, fmt.Errorf("copy avatar: %w", err)
	}

	old, err := u.repo.SwapAvatar(ctx, userID, newKey)
	if err != nil {
		return nil, fmt.Errorf("swap avatar: %w", err)
	}
	u.deleteOrphanable(ctx, userID, old)

	return u.Get(ctx, userID)
}

// RemoveAvatar clears the avatar and deletes its image (AC26). A user with no
// avatar is not an error: the unchanged profile comes back.
func (u *ProfileUsecase) RemoveAvatar(ctx context.Context, userID int64) (*domain.ProfileView, error) {
	old, err := u.repo.ClearAvatar(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("clear avatar: %w", err)
	}
	u.deleteOrphanable(ctx, userID, old)

	return u.Get(ctx, userID)
}

// deleteOrphanable removes an image the row no longer points at. A failure is
// logged and swallowed: the user's data is already correct, and failing the
// request would only make a client retry something that has succeeded. The file
// is left behind as garbage — a decision recorded as TD-002.
func (u *ProfileUsecase) deleteOrphanable(ctx context.Context, userID int64, key *string) {
	if key == nil {
		return
	}
	if err := u.storage.Delete(ctx, *key); err != nil {
		// user_id and key only: no PII in the log.
		u.logger.WithContext(ctx).WithError(err).Warn("left an unreferenced avatar object behind",
			"user_id", userID, "object", *key)
	}
}

// view attaches a signed read address to p when it has an avatar. A signing
// failure fails the call: returning the profile minus its image would look like
// "no avatar" to the client.
func (u *ProfileUsecase) view(ctx context.Context, p *domain.Profile) (*domain.ProfileView, error) {
	v := &domain.ProfileView{Profile: *p}
	if p.AvatarObject == nil {
		return v, nil
	}
	ttl := u.cfg.AvatarReadURLTTL.Duration()
	url, err := u.storage.SignRead(ctx, *p.AvatarObject, ttl)
	if err != nil {
		return nil, fmt.Errorf("sign avatar read url: %w", err)
	}
	exp := u.now().Add(ttl)
	v.AvatarURL = &url
	v.AvatarURLExpiresAt = &exp
	return v, nil
}

func (u *ProfileUsecase) contentTypeAllowed(ct string) bool {
	for _, allowed := range u.cfg.AvatarContentTypes {
		if ct == allowed {
			return true
		}
	}
	return false
}

// tmpKey is where a client's upload lands. It is built from the token's user id,
// so another user's upload id resolves to a key in the caller's own folder and
// is simply not found (AC23).
func (u *ProfileUsecase) tmpKey(userID int64, uploadID string) string {
	return u.cfg.AvatarTmpPrefix + strconv.FormatInt(userID, 10) + "/" + uploadID
}

// avatarKey is a fresh key for every confirm, so a replacement never reuses the
// previous name (no stale cache, and "delete the old one" cannot hit the new one).
func (u *ProfileUsecase) avatarKey(userID int64, contentType string) string {
	return u.cfg.AvatarPrefix + strconv.FormatInt(userID, 10) + "/" + u.newID() + avatarExtensions[contentType]
}
