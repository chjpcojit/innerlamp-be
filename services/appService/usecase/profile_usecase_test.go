package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

const (
	profileUserID   = int64(42)
	profileUploadID = "0b9c4f3e-5a1d-4c7e-9b2a-6d8e1f0a3c11"
	profileNewID    = "7f3a1b2c-8d4e-4f60-a1b2-c3d4e5f60718"
)

var profileNow = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func profileTestConfig() config.ProfileConfig {
	return config.ProfileConfig{
		DharmaNameMaxChars: 255,
		FullNameMaxChars:   300,
		AvatarMaxBytes:     5 << 20,
		AvatarContentTypes: []string{"image/jpeg", "image/png", "image/webp"},
		UploadURLTTL:       config.Duration(5 * time.Minute),
		AvatarReadURLTTL:   config.Duration(15 * time.Minute),
		Bucket:             "bucket",
		AvatarTmpPrefix:    "avatars/tmp/",
		AvatarPrefix:       "avatars/",
		RateLimit:          config.ProfileRateLimit{Max: 10, Window: config.Duration(time.Minute)},
	}
}

type profileFixture struct {
	uc      *ProfileUsecase
	repo    *domain.MockProfileRepository
	storage *domain.MockAvatarStorage
	// calls records the order of side effects across both mocks.
	calls []string
}

func newProfileFixture(t *testing.T) *profileFixture {
	t.Helper()
	f := &profileFixture{
		repo:    &domain.MockProfileRepository{},
		storage: &domain.MockAvatarStorage{},
	}
	f.uc = NewProfileUsecase(f.repo, f.storage, profileTestConfig(), newLogger())
	f.uc.now = func() time.Time { return profileNow }
	f.uc.newID = func() string { return profileNewID }
	return f
}

// record returns a mock Run hook that appends name to the shared call log.
func (f *profileFixture) record(name string) func(mock.Arguments) {
	return func(mock.Arguments) { f.calls = append(f.calls, name) }
}

func profileFor(avatar *string) *domain.Profile {
	return &domain.Profile{
		UserID: profileUserID, Email: "an@example.com", PhoneNumber: "0901234567",
		AvatarObject: avatar,
	}
}

func set(v string) domain.OptionalString { return domain.OptionalString{Set: true, Value: &v} }
func cleared() domain.OptionalString     { return domain.OptionalString{Set: true} }

// ---- Get ---------------------------------------------------------------

func TestProfileUsecase_Get_NoAvatar(t *testing.T) {
	f := newProfileFixture(t)
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(nil), nil)

	view, err := f.uc.Get(context.Background(), profileUserID)

	require.NoError(t, err)
	assert.Nil(t, view.AvatarURL)
	assert.Nil(t, view.AvatarURLExpiresAt)
	f.storage.AssertNotCalled(t, "SignRead", mock.Anything, mock.Anything, mock.Anything)
}

func TestProfileUsecase_Get_SignsAvatarForConfiguredTTL(t *testing.T) {
	f := newProfileFixture(t)
	key := "avatars/42/7f3a.jpg"
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(&key), nil)
	f.storage.On("SignRead", mock.Anything, key, 15*time.Minute).Return("https://signed/read", nil)

	view, err := f.uc.Get(context.Background(), profileUserID)

	require.NoError(t, err)
	require.NotNil(t, view.AvatarURL)
	assert.Equal(t, "https://signed/read", *view.AvatarURL)
	require.NotNil(t, view.AvatarURLExpiresAt)
	assert.Equal(t, profileNow.Add(15*time.Minute), *view.AvatarURLExpiresAt)
}

func TestProfileUsecase_Get_AccountGone(t *testing.T) {
	f := newProfileFixture(t)
	f.repo.On("Get", mock.Anything, profileUserID).Return(nil, domain.ErrUserNotFound)

	_, err := f.uc.Get(context.Background(), profileUserID)

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestProfileUsecase_Get_SignFailureIsAnError(t *testing.T) {
	f := newProfileFixture(t)
	key := "avatars/42/7f3a.jpg"
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(&key), nil)
	f.storage.On("SignRead", mock.Anything, key, mock.Anything).Return("", errors.New("sign down"))

	view, err := f.uc.Get(context.Background(), profileUserID)

	assert.Error(t, err, "a profile must not silently come back without its avatar")
	assert.Nil(t, view)
}

// ---- Update ------------------------------------------------------------

func TestProfileUsecase_Update_NormalisesDharmaName(t *testing.T) {
	f := newProfileFixture(t)
	decomposed := "  " + norm.NFD.String("Thanh Liên") + "\t"
	want := domain.ProfilePatch{DharmaName: set(norm.NFC.String("Thanh Liên"))}
	f.repo.On("Update", mock.Anything, profileUserID, want).Return(profileFor(nil), nil)

	_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{DharmaName: set(decomposed)})

	require.NoError(t, err)
	f.repo.AssertExpectations(t)
}

func TestProfileUsecase_Update_BlankOrNullClears(t *testing.T) {
	for name, in := range map[string]domain.OptionalString{
		"null":       cleared(),
		"empty":      set(""),
		"whitespace": set(" \t "),
	} {
		t.Run(name, func(t *testing.T) {
			f := newProfileFixture(t)
			f.repo.On("Update", mock.Anything, profileUserID, domain.ProfilePatch{DharmaName: cleared()}).
				Return(profileFor(nil), nil)

			_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{DharmaName: in})

			require.NoError(t, err)
			f.repo.AssertExpectations(t)
		})
	}
}

func TestProfileUsecase_Update_DharmaNameLengthBoundary(t *testing.T) {
	// Counted in characters after NFC, so 255 accented letters are fine whether
	// they were typed composed or decomposed, and 256 are not.
	composed255 := strings.Repeat("ệ", 255)
	decomposed255 := norm.NFD.String(composed255)
	require.Greater(t, len(decomposed255), 255*3, "fixture must exceed the limit in bytes and runes")

	for name, v := range map[string]string{"composed": composed255, "decomposed": decomposed255} {
		t.Run("255 "+name, func(t *testing.T) {
			f := newProfileFixture(t)
			f.repo.On("Update", mock.Anything, profileUserID, domain.ProfilePatch{DharmaName: set(composed255)}).
				Return(profileFor(nil), nil)

			_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{DharmaName: set(v)})

			require.NoError(t, err)
		})
	}

	t.Run("256", func(t *testing.T) {
		f := newProfileFixture(t)

		_, err := f.uc.Update(context.Background(), profileUserID,
			domain.ProfilePatch{DharmaName: set(strings.Repeat("ệ", 256))})

		assert.ErrorIs(t, err, domain.ErrInvalidDharmaName)
		f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestProfileUsecase_Update_UnknownTempleWritesNothing(t *testing.T) {
	f := newProfileFixture(t)
	f.repo.On("TempleExists", mock.Anything, "temple_nope").Return(false, nil)

	// The dharma name is valid; the unknown temple must still stop the whole request.
	_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{
		DharmaName: set("Thanh Liên"), OrdinationTempleID: set("temple_nope"),
	})

	assert.ErrorIs(t, err, domain.ErrInvalidTemple)
	f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestProfileUsecase_Update_InvalidNameDoesNotLookUpTemple(t *testing.T) {
	f := newProfileFixture(t)

	_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{
		DharmaName: set(strings.Repeat("a", 256)), OrdinationTempleID: set("temple_ok"),
	})

	assert.ErrorIs(t, err, domain.ErrInvalidDharmaName)
	f.repo.AssertNotCalled(t, "TempleExists", mock.Anything, mock.Anything)
}

func TestProfileUsecase_Update_ClearingTempleSkipsLookup(t *testing.T) {
	for name, in := range map[string]domain.OptionalString{"null": cleared(), "blank": set("  ")} {
		t.Run(name, func(t *testing.T) {
			f := newProfileFixture(t)
			f.repo.On("Update", mock.Anything, profileUserID, domain.ProfilePatch{OrdinationTempleID: cleared()}).
				Return(profileFor(nil), nil)

			_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{OrdinationTempleID: in})

			require.NoError(t, err)
			f.repo.AssertNotCalled(t, "TempleExists", mock.Anything, mock.Anything)
		})
	}
}

func TestProfileUsecase_Update_KnownTempleIsStored(t *testing.T) {
	f := newProfileFixture(t)
	f.repo.On("TempleExists", mock.Anything, "temple_north_011").Return(true, nil)
	f.repo.On("Update", mock.Anything, profileUserID, domain.ProfilePatch{OrdinationTempleID: set("temple_north_011")}).
		Return(profileFor(nil), nil)

	_, err := f.uc.Update(context.Background(), profileUserID,
		domain.ProfilePatch{OrdinationTempleID: set(" temple_north_011 ")})

	require.NoError(t, err)
	f.repo.AssertExpectations(t)
}

func TestProfileUsecase_Update_EmptyPatchOnlyReads(t *testing.T) {
	f := newProfileFixture(t)
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(nil), nil)

	view, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{})

	require.NoError(t, err)
	assert.Equal(t, "an@example.com", view.Profile.Email)
	f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestProfileUsecase_Update_TempleLookupErrorIsNotInvalidTemple(t *testing.T) {
	f := newProfileFixture(t)
	f.repo.On("TempleExists", mock.Anything, "t1").Return(false, errors.New("db down"))

	_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{OrdinationTempleID: set("t1")})

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrInvalidTemple, "a DB failure must be a 500, not a 400")
}

func TestProfileUsecase_Update_ForeignKeyRaceSurfacesAsInvalidTemple(t *testing.T) {
	f := newProfileFixture(t)
	f.repo.On("TempleExists", mock.Anything, "t1").Return(true, nil)
	f.repo.On("Update", mock.Anything, profileUserID, mock.Anything).Return(nil, domain.ErrInvalidTemple)

	_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{OrdinationTempleID: set("t1")})

	assert.ErrorIs(t, err, domain.ErrInvalidTemple)
}

// ---- RequestAvatarUpload -----------------------------------------------

func TestProfileUsecase_RequestAvatarUpload_IssuesTicketInTempFolder(t *testing.T) {
	f := newProfileFixture(t)
	wantKey := "avatars/tmp/42/" + profileNewID
	f.storage.On("SignUpload", mock.Anything, wantKey, "image/jpeg", int64(5<<20), 5*time.Minute).
		Return(&domain.SignedUpload{
			URL: "https://signed/put", Method: "POST", Headers: map[string]string{"Content-Type": "image/jpeg"},
		}, nil)

	ticket, err := f.uc.RequestAvatarUpload(context.Background(), profileUserID,
		domain.AvatarUploadRequest{ContentType: "image/jpeg", SizeBytes: 2 << 20})

	require.NoError(t, err)
	assert.Equal(t, profileNewID, ticket.UploadID)
	assert.Equal(t, "https://signed/put", ticket.URL)
	assert.Equal(t, "POST", ticket.Method, "the method is whatever the store asked for, not a fixed PUT")
	assert.Equal(t, map[string]string{"Content-Type": "image/jpeg"}, ticket.Headers)
	assert.Equal(t, profileNow.Add(5*time.Minute), ticket.ExpiresAt)
}

func TestProfileUsecase_RequestAvatarUpload_Validation(t *testing.T) {
	max := int64(5 << 20)
	tests := []struct {
		name      string
		req       domain.AvatarUploadRequest
		wantField string // "" = accepted
	}{
		{"jpeg ok", domain.AvatarUploadRequest{ContentType: "image/jpeg", SizeBytes: 1}, ""},
		{"png ok", domain.AvatarUploadRequest{ContentType: "image/png", SizeBytes: 1}, ""},
		{"webp ok", domain.AvatarUploadRequest{ContentType: "image/webp", SizeBytes: 1}, ""},
		{"exactly the maximum is accepted", domain.AvatarUploadRequest{ContentType: "image/png", SizeBytes: max}, ""},
		{"one byte over the maximum", domain.AvatarUploadRequest{ContentType: "image/png", SizeBytes: max + 1}, "size_bytes"},
		{"zero bytes", domain.AvatarUploadRequest{ContentType: "image/png", SizeBytes: 0}, "size_bytes"},
		{"negative bytes", domain.AvatarUploadRequest{ContentType: "image/png", SizeBytes: -1}, "size_bytes"},
		{"gif not allowed", domain.AvatarUploadRequest{ContentType: "image/gif", SizeBytes: 1}, "content_type"},
		{"empty type", domain.AvatarUploadRequest{ContentType: "", SizeBytes: 1}, "content_type"},
		{"type is case sensitive", domain.AvatarUploadRequest{ContentType: "IMAGE/JPEG", SizeBytes: 1}, "content_type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newProfileFixture(t)
			f.storage.On("SignUpload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(&domain.SignedUpload{URL: "https://signed/put", Method: "PUT"}, nil).Maybe()

			_, err := f.uc.RequestAvatarUpload(context.Background(), profileUserID, tt.req)

			if tt.wantField == "" {
				require.NoError(t, err)
				return
			}
			var bad *domain.ProfileValidationError
			require.ErrorAs(t, err, &bad)
			assert.Equal(t, tt.wantField, bad.Field)
			f.storage.AssertNotCalled(t, "SignUpload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestProfileUsecase_RequestAvatarUpload_SignFailure(t *testing.T) {
	f := newProfileFixture(t)
	f.storage.On("SignUpload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("sign down"))

	ticket, err := f.uc.RequestAvatarUpload(context.Background(), profileUserID,
		domain.AvatarUploadRequest{ContentType: "image/png", SizeBytes: 10})

	assert.Error(t, err)
	assert.Nil(t, ticket)
}

// ---- ConfirmAvatar -----------------------------------------------------

func tmpKey() string { return "avatars/tmp/42/" + profileUploadID }

func TestProfileUsecase_ConfirmAvatar_ReplacesInOrder(t *testing.T) {
	f := newProfileFixture(t)
	oldKey := "avatars/42/old.jpg"
	newKey := "avatars/42/" + profileNewID + ".jpg"

	f.storage.On("Stat", mock.Anything, tmpKey()).Run(f.record("stat")).
		Return(&domain.ObjectMeta{ContentType: "image/jpeg", Size: 1 << 20}, nil)
	f.storage.On("Copy", mock.Anything, tmpKey(), newKey).Run(f.record("copy")).Return(nil)
	f.repo.On("SwapAvatar", mock.Anything, profileUserID, newKey).Run(f.record("swap")).Return(&oldKey, nil)
	f.storage.On("Delete", mock.Anything, oldKey).Run(f.record("delete-old")).Return(nil)
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(&newKey), nil)
	f.storage.On("SignRead", mock.Anything, newKey, 15*time.Minute).Return("https://signed/new", nil)

	view, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

	require.NoError(t, err)
	// copy before the DB write (the row never points at a missing object), and the
	// old image is deleted only after the row points at the new one.
	assert.Equal(t, []string{"stat", "copy", "swap", "delete-old"}, f.calls)
	require.NotNil(t, view.AvatarURL)
	assert.Equal(t, "https://signed/new", *view.AvatarURL)
	f.storage.AssertNotCalled(t, "Delete", mock.Anything, tmpKey())
}

func TestProfileUsecase_ConfirmAvatar_FirstAvatarDeletesNothing(t *testing.T) {
	f := newProfileFixture(t)
	newKey := "avatars/42/" + profileNewID + ".png"
	f.storage.On("Stat", mock.Anything, tmpKey()).Return(&domain.ObjectMeta{ContentType: "image/png", Size: 10}, nil)
	f.storage.On("Copy", mock.Anything, tmpKey(), newKey).Return(nil)
	f.repo.On("SwapAvatar", mock.Anything, profileUserID, newKey).Return(nil, nil)
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(&newKey), nil)
	f.storage.On("SignRead", mock.Anything, newKey, mock.Anything).Return("u", nil)

	_, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

	require.NoError(t, err)
	f.storage.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestProfileUsecase_ConfirmAvatar_ObjectKeyExtensionFollowsContentType(t *testing.T) {
	for ct, ext := range map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"} {
		t.Run(ct, func(t *testing.T) {
			f := newProfileFixture(t)
			newKey := "avatars/42/" + profileNewID + ext
			f.storage.On("Stat", mock.Anything, tmpKey()).Return(&domain.ObjectMeta{ContentType: ct, Size: 10}, nil)
			f.storage.On("Copy", mock.Anything, tmpKey(), newKey).Return(nil)
			f.repo.On("SwapAvatar", mock.Anything, profileUserID, newKey).Return(nil, nil)
			f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(&newKey), nil)
			f.storage.On("SignRead", mock.Anything, newKey, mock.Anything).Return("u", nil)

			_, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

			require.NoError(t, err)
			f.storage.AssertExpectations(t)
		})
	}
}

func TestProfileUsecase_ConfirmAvatar_MalformedUploadID(t *testing.T) {
	for name, id := range map[string]string{
		"empty":          "",
		"not a uuid":     "abc",
		"path traversal": "../43/" + profileUploadID,
		"uppercase":      strings.ToUpper(profileUploadID),
		"braced":         "{" + profileUploadID + "}",
	} {
		t.Run(name, func(t *testing.T) {
			f := newProfileFixture(t)

			_, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, id)

			var bad *domain.ProfileValidationError
			require.ErrorAs(t, err, &bad)
			assert.Equal(t, "upload_id", bad.Field)
			f.storage.AssertNotCalled(t, "Stat", mock.Anything, mock.Anything)
		})
	}
}

func TestProfileUsecase_ConfirmAvatar_NoObject(t *testing.T) {
	f := newProfileFixture(t)
	f.storage.On("Stat", mock.Anything, tmpKey()).Return(nil, domain.ErrObjectNotFound)

	_, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

	assert.ErrorIs(t, err, domain.ErrAvatarUploadNotFound)
	f.storage.AssertNotCalled(t, "Copy", mock.Anything, mock.Anything, mock.Anything)
	f.repo.AssertNotCalled(t, "SwapAvatar", mock.Anything, mock.Anything, mock.Anything)
}

func TestProfileUsecase_ConfirmAvatar_StatFailureIsNotNotFound(t *testing.T) {
	f := newProfileFixture(t)
	f.storage.On("Stat", mock.Anything, tmpKey()).Return(nil, errors.New("storage down"))

	_, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrAvatarUploadNotFound)
}

func TestProfileUsecase_ConfirmAvatar_RecordedMetaOutsidePolicy(t *testing.T) {
	max := int64(5 << 20)
	tests := []struct {
		name string
		meta domain.ObjectMeta
	}{
		{"type not allowed", domain.ObjectMeta{ContentType: "text/html", Size: 10}},
		{"one byte over", domain.ObjectMeta{ContentType: "image/png", Size: max + 1}},
		{"empty object", domain.ObjectMeta{ContentType: "image/png", Size: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newProfileFixture(t)
			meta := tt.meta
			f.storage.On("Stat", mock.Anything, tmpKey()).Return(&meta, nil)

			_, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

			assert.ErrorIs(t, err, domain.ErrInvalidAvatar)
			f.storage.AssertNotCalled(t, "Copy", mock.Anything, mock.Anything, mock.Anything)
			f.repo.AssertNotCalled(t, "SwapAvatar", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestProfileUsecase_ConfirmAvatar_ExactlyMaxBytesIsAccepted(t *testing.T) {
	f := newProfileFixture(t)
	newKey := "avatars/42/" + profileNewID + ".png"
	f.storage.On("Stat", mock.Anything, tmpKey()).Return(&domain.ObjectMeta{ContentType: "image/png", Size: 5 << 20}, nil)
	f.storage.On("Copy", mock.Anything, tmpKey(), newKey).Return(nil)
	f.repo.On("SwapAvatar", mock.Anything, profileUserID, newKey).Return(nil, nil)
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(&newKey), nil)
	f.storage.On("SignRead", mock.Anything, newKey, mock.Anything).Return("u", nil)

	_, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

	require.NoError(t, err)
}

func TestProfileUsecase_ConfirmAvatar_CopyFailureLeavesRowUntouched(t *testing.T) {
	f := newProfileFixture(t)
	f.storage.On("Stat", mock.Anything, tmpKey()).Return(&domain.ObjectMeta{ContentType: "image/png", Size: 10}, nil)
	f.storage.On("Copy", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("copy failed"))

	_, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

	require.Error(t, err)
	f.repo.AssertNotCalled(t, "SwapAvatar", mock.Anything, mock.Anything, mock.Anything)
	f.storage.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestProfileUsecase_ConfirmAvatar_SwapFailureKeepsOldImage(t *testing.T) {
	f := newProfileFixture(t)
	f.storage.On("Stat", mock.Anything, tmpKey()).Return(&domain.ObjectMeta{ContentType: "image/png", Size: 10}, nil)
	f.storage.On("Copy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.repo.On("SwapAvatar", mock.Anything, profileUserID, mock.Anything).Return(nil, errors.New("db down"))

	_, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

	require.Error(t, err)
	f.storage.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestProfileUsecase_ConfirmAvatar_OldImageDeleteFailureStillSucceeds(t *testing.T) {
	f := newProfileFixture(t)
	oldKey := "avatars/42/old.jpg"
	newKey := "avatars/42/" + profileNewID + ".png"
	f.storage.On("Stat", mock.Anything, tmpKey()).Return(&domain.ObjectMeta{ContentType: "image/png", Size: 10}, nil)
	f.storage.On("Copy", mock.Anything, tmpKey(), newKey).Return(nil)
	f.repo.On("SwapAvatar", mock.Anything, profileUserID, newKey).Return(&oldKey, nil)
	f.storage.On("Delete", mock.Anything, oldKey).Return(errors.New("delete failed"))
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(&newKey), nil)
	f.storage.On("SignRead", mock.Anything, newKey, mock.Anything).Return("u", nil)

	view, err := f.uc.ConfirmAvatar(context.Background(), profileUserID, profileUploadID)

	require.NoError(t, err, "TD-002: an orphaned old file is accepted, the request still succeeds")
	require.NotNil(t, view.AvatarURL)
}

// ---- RemoveAvatar ------------------------------------------------------

func TestProfileUsecase_RemoveAvatar_ClearsRowThenDeletesObject(t *testing.T) {
	f := newProfileFixture(t)
	oldKey := "avatars/42/old.jpg"
	f.repo.On("ClearAvatar", mock.Anything, profileUserID).Run(f.record("clear")).Return(&oldKey, nil)
	f.storage.On("Delete", mock.Anything, oldKey).Run(f.record("delete")).Return(nil)
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(nil), nil)

	view, err := f.uc.RemoveAvatar(context.Background(), profileUserID)

	require.NoError(t, err)
	assert.Equal(t, []string{"clear", "delete"}, f.calls)
	assert.Nil(t, view.AvatarURL)
}

func TestProfileUsecase_RemoveAvatar_NoAvatarIsNotAnError(t *testing.T) {
	f := newProfileFixture(t)
	f.repo.On("ClearAvatar", mock.Anything, profileUserID).Return(nil, nil)
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(nil), nil)

	view, err := f.uc.RemoveAvatar(context.Background(), profileUserID)

	require.NoError(t, err)
	assert.Nil(t, view.AvatarURL)
	f.storage.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestProfileUsecase_RemoveAvatar_ObjectDeleteFailureStillSucceeds(t *testing.T) {
	f := newProfileFixture(t)
	oldKey := "avatars/42/old.jpg"
	f.repo.On("ClearAvatar", mock.Anything, profileUserID).Return(&oldKey, nil)
	f.storage.On("Delete", mock.Anything, oldKey).Return(errors.New("delete failed"))
	f.repo.On("Get", mock.Anything, profileUserID).Return(profileFor(nil), nil)

	_, err := f.uc.RemoveAvatar(context.Background(), profileUserID)

	require.NoError(t, err)
}

func TestProfileUsecase_RemoveAvatar_ClearFailureDeletesNothing(t *testing.T) {
	f := newProfileFixture(t)
	f.repo.On("ClearAvatar", mock.Anything, profileUserID).Return(nil, errors.New("db down"))

	_, err := f.uc.RemoveAvatar(context.Background(), profileUserID)

	require.Error(t, err)
	f.storage.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestProfileUsecase_Update_NormalisesFullName(t *testing.T) {
	f := newProfileFixture(t)
	decomposed := "  " + norm.NFD.String("Nguyễn Văn An") + "\t"
	want := domain.ProfilePatch{FullName: set(norm.NFC.String("Nguyễn Văn An"))}
	f.repo.On("Update", mock.Anything, profileUserID, want).Return(profileFor(nil), nil)

	_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{FullName: set(decomposed)})

	require.NoError(t, err)
	f.repo.AssertExpectations(t)
}

// The full name is required: null, blank and over-limit are all rejected and
// nothing is written.
func TestProfileUsecase_Update_FullNameCannotBeClearedOrTooLong(t *testing.T) {
	for name, in := range map[string]domain.OptionalString{
		"null":       cleared(),
		"empty":      set(""),
		"whitespace": set(" \t "),
		"too long":   set(strings.Repeat("a", 301)),
	} {
		t.Run(name, func(t *testing.T) {
			f := newProfileFixture(t)

			_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{FullName: in})

			require.ErrorIs(t, err, domain.ErrInvalidFullName)
			f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestProfileUsecase_Update_FullNameAtLimitIsAccepted(t *testing.T) {
	f := newProfileFixture(t)
	name := strings.Repeat("a", 300)
	f.repo.On("Update", mock.Anything, profileUserID, domain.ProfilePatch{FullName: set(name)}).Return(profileFor(nil), nil)

	_, err := f.uc.Update(context.Background(), profileUserID, domain.ProfilePatch{FullName: set(name)})

	require.NoError(t, err)
}
