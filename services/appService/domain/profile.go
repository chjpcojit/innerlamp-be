package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Profile is a user's own hồ sơ cá nhân (docs/specs/features/profile). It is the
// read model of GET /v1/users/me: users LEFT JOIN temples.
//
// Plain data: no json/db tag. AvatarObject is a storage object KEY, never a URL —
// the readable address is a signed URL minted per response (ProfileView).
type Profile struct {
	UserID           int64
	Email            string
	EmailVerified    bool
	PhoneNumber      string
	FullName         *string
	DharmaName       *string
	OrdinationTemple *TempleRef
	AvatarObject     *string
}

// TempleRef is the chosen ordination temple as the profile shows it.
type TempleRef struct {
	ID   string
	Name string
}

// ProfileView is a Profile together with the signed address of its avatar.
// AvatarURL and AvatarURLExpiresAt are both nil when the user has no avatar.
type ProfileView struct {
	Profile            Profile
	AvatarURL          *string
	AvatarURLExpiresAt *time.Time
}

// OptionalString distinguishes the three things a PATCH field can say:
//
//	Set == false             the key was absent: keep the stored value
//	Set == true, Value == nil  the key was null / blank: clear the stored value
//	Set == true, Value != nil  the key carried a value
//
// A plain *string cannot tell "absent" from "null".
type OptionalString struct {
	Set   bool
	Value *string
}

// ProfilePatch is a partial update of the editable text fields. Email and
// phone number are deliberately not here: they are read-only (AC15).
type ProfilePatch struct {
	// FullName is required: absent keeps, a value replaces, null/blank is
	// rejected with ErrInvalidFullName (it can be edited, not cleared).
	FullName           OptionalString
	DharmaName         OptionalString
	OrdinationTempleID OptionalString
}

// IsEmpty reports whether the patch changes nothing (body `{}`, AC13).
func (p ProfilePatch) IsEmpty() bool {
	return !p.FullName.Set && !p.DharmaName.Set && !p.OrdinationTempleID.Set
}

// AvatarUploadRequest is what a client claims about the file it is about to
// upload. Storage re-checks content type and size on confirm; this claim only
// bounds what the signed URL will accept.
type AvatarUploadRequest struct {
	ContentType string
	SizeBytes   int64
}

// AvatarUploadTicket is the permission to PUT one object into the temporary
// folder. Headers must be sent verbatim: they are part of the signature.
type AvatarUploadTicket struct {
	UploadID  string
	URL       string
	Method    string
	Headers   map[string]string
	ExpiresAt time.Time
}

// ObjectMeta is what storage recorded about an uploaded object.
type ObjectMeta struct {
	ContentType string
	Size        int64
}

var (
	// ErrProfileInvalid — a submitted profile field is not allowed. Use
	// ProfileValidationError to say which one; errors.Is still matches this.
	ErrProfileInvalid = errors.New("invalid profile")

	// ErrInvalidAvatar — the uploaded object's recorded content type or size is
	// outside the policy (AC24). The avatar is left unchanged.
	ErrInvalidAvatar = errors.New("avatar is not an allowed image")

	// ErrAvatarUploadNotFound — the user's temporary folder holds no object for
	// the given upload id: never uploaded, expired, or another user's id (AC23).
	ErrAvatarUploadNotFound = errors.New("avatar upload not found")

	// ErrObjectNotFound — storage has no object at the key. Raised by
	// AvatarStorage; the use case maps it to ErrAvatarUploadNotFound.
	ErrObjectNotFound = errors.New("storage object not found")
)

// ProfileValidationError names the request field that was rejected and why.
type ProfileValidationError struct {
	// Field is the request field name, e.g. "upload_id".
	Field   string
	Message string
}

func (e *ProfileValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Unwrap lets errors.Is(err, ErrProfileInvalid) match.
func (e *ProfileValidationError) Unwrap() error { return ErrProfileInvalid }

// ProfileRepository is the port the profile use case stores through.
type ProfileRepository interface {
	// Get returns the profile of userID, the temple name joined in.
	// Returns ErrUserNotFound if the account no longer exists — AC7.
	Get(ctx context.Context, userID int64) (*Profile, error)

	// Update applies patch to the user's row in ONE statement, touching only the
	// columns the patch sets (a cleared field becomes SQL NULL), and returns the
	// profile as it is afterwards. A foreign-key violation on the temple returns
	// ErrInvalidTemple — AC11, AC14. An empty patch changes nothing.
	Update(ctx context.Context, userID int64, patch ProfilePatch) (*Profile, error)

	// SwapAvatar points the user at newObject and returns the key it replaced
	// (nil if there was none). The read and the write share one transaction with
	// the row locked, so two concurrent swaps each get the key they really
	// replaced — AC25.
	SwapAvatar(ctx context.Context, userID int64, newObject string) (oldObject *string, err error)

	// ClearAvatar removes the user's avatar key and returns the key it removed
	// (nil if there was none; nothing is written then) — AC26.
	ClearAvatar(ctx context.Context, userID int64) (oldObject *string, err error)

	// TempleExists reports whether id is a temple in the catalogue — AC11.
	TempleExists(ctx context.Context, id string) (bool, error)
}

// SignedUpload is where and how a client uploads one object: send Method to URL
// with exactly Headers. The method is the store's choice (a real bucket takes a
// PUT of a signed URL, the dev emulator a POST to its upload endpoint), so it
// travels with the URL instead of being fixed by the caller.
type SignedUpload struct {
	URL     string
	Method  string
	Headers map[string]string
}

// AvatarStorage is the narrow port to object storage. It is declared here so the
// use case depends on this shape, not on pkg/storage.
type AvatarStorage interface {
	// SignUpload returns how a client uploads one object to key with
	// contentType, at most maxBytes, within ttl. Whether the address is signed
	// (and so enforces maxBytes and ttl) depends on the store: see the adapter.
	SignUpload(ctx context.Context, key, contentType string, maxBytes int64, ttl time.Duration) (*SignedUpload, error)

	// SignRead returns a GET URL for key valid for ttl.
	SignRead(ctx context.Context, key string, ttl time.Duration) (string, error)

	// Stat returns the metadata storage recorded for key, or ErrObjectNotFound.
	Stat(ctx context.Context, key string) (*ObjectMeta, error)

	// Copy duplicates srcKey to dstKey; srcKey is left in place.
	Copy(ctx context.Context, srcKey, dstKey string) error

	// Delete removes key. Deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
}
