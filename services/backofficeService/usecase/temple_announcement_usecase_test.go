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

	"chjpcoj.com.innerlamp/services/backofficeService/config"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

var annNow = time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)

func newAnnUC() (*TempleAnnouncementUsecase, *domain.MockAnnouncementRepository, *domain.MockIssuingBodyRepository) {
	ar, br := &domain.MockAnnouncementRepository{}, &domain.MockIssuingBodyRepository{}
	uc := NewTempleAnnouncementUsecase(ar, br, config.TempleAnnouncementConfig{
		TitleMaxChars: 10, BodyMaxChars: 20, MaxValidityDays: 30,
		IssuingBodyNameMaxChars: 8, PageSize: 2, MaxPageSize: 5,
	})
	uc.now = func() time.Time { return annNow }
	return uc, ar, br
}

func validNew() domain.NewAnnouncement {
	return domain.NewAnnouncement{IssuingBodyID: "tib_1", Title: "Lễ Vu Lan", Body: "Nội dung", ExpiresAt: annNow.Add(48 * time.Hour)}
}

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	var ie *domain.AnnouncementInputError
	require.True(t, errors.As(err, &ie), "want AnnouncementInputError, got %v", err)
	return ie.Field
}

func TestAnnouncementState_Boundary(t *testing.T) {
	a := domain.Announcement{ExpiresAt: annNow}
	assert.Equal(t, domain.AnnouncementExpired, a.StateAt(annNow), "expires_at == now is already expired")
	assert.Equal(t, domain.AnnouncementActive, a.StateAt(annNow.Add(-time.Nanosecond)))
	rev := annNow.Add(-time.Hour)
	a.RevokedAt = &rev
	assert.Equal(t, domain.AnnouncementRevoked, a.StateAt(annNow.Add(-time.Minute)), "revoked wins over any expiry")
}

func TestPublish_Success(t *testing.T) {
	uc, ar, _ := newAnnUC()
	ctx := context.Background()
	ar.On("TempleExists", ctx, "temple_1").Return(true, nil)
	ar.On("IssuingBodyBelongs", ctx, "tib_1", "temple_1").Return(true, nil)
	in := validNew()
	want := domain.Announcement{
		TempleID: "temple_1", IssuingBodyID: "tib_1", Title: "Lễ Vu Lan", Body: "Nội dung",
		Source: domain.AnnouncementSourceAdmin, PublishedAt: annNow, ExpiresAt: in.ExpiresAt, CreatedBy: strp("7"),
	}
	stored := want
	stored.ID = "tann_x"
	ar.On("Insert", ctx, want).Return(&stored, nil)

	got, err := uc.Publish(ctx, "temple_1", "7", in)
	require.NoError(t, err)
	assert.Equal(t, "tann_x", got.ID)
	assert.Equal(t, domain.AnnouncementActive, got.State)
	ar.AssertExpectations(t)
}

func strp(s string) *string { return &s }

func TestPublish_Validation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*domain.NewAnnouncement)
		field  string
	}{
		{"blank title", func(n *domain.NewAnnouncement) { n.Title = "   " }, "title"},
		{"blank body", func(n *domain.NewAnnouncement) { n.Body = "" }, "body"},
		{"blank issuing body", func(n *domain.NewAnnouncement) { n.IssuingBodyID = " " }, "issuing_body_id"},
		{"missing expires_at", func(n *domain.NewAnnouncement) { n.ExpiresAt = time.Time{} }, "expires_at"},
		{"title 11 chars", func(n *domain.NewAnnouncement) { n.Title = strings.Repeat("a", 11) }, "title"},
		{"title 11 accented runes is over, even though bytes are more", func(n *domain.NewAnnouncement) { n.Title = strings.Repeat("ệ", 11) }, "title"},
		{"body 21 chars", func(n *domain.NewAnnouncement) { n.Body = strings.Repeat("b", 21) }, "body"},
		{"expires in the past", func(n *domain.NewAnnouncement) { n.ExpiresAt = annNow.Add(-time.Second) }, "expires_at"},
		{"expires == now", func(n *domain.NewAnnouncement) { n.ExpiresAt = annNow }, "expires_at"},
		{"expires beyond max validity", func(n *domain.NewAnnouncement) { n.ExpiresAt = annNow.Add(30*24*time.Hour + time.Second) }, "expires_at"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, ar, _ := newAnnUC()
			in := validNew()
			tc.mutate(&in)
			_, err := uc.Publish(context.Background(), "temple_1", "7", in)
			require.ErrorIs(t, err, domain.ErrInvalidAnnouncement)
			assert.Equal(t, tc.field, fieldOf(t, err))
			ar.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything)
			ar.AssertNotCalled(t, "TempleExists", mock.Anything, mock.Anything)
		})
	}

	t.Run("accented title of exactly the limit passes", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(true, nil)
		ar.On("IssuingBodyBelongs", mock.Anything, "tib_1", "t").Return(true, nil)
		ar.On("Insert", mock.Anything, mock.Anything).Return(&domain.Announcement{ID: "x"}, nil)
		in := validNew()
		in.Title = strings.Repeat("ệ", 10)
		_, err := uc.Publish(context.Background(), "t", "7", in)
		require.NoError(t, err)
	})
	t.Run("expires exactly at max validity passes", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(true, nil)
		ar.On("IssuingBodyBelongs", mock.Anything, "tib_1", "t").Return(true, nil)
		ar.On("Insert", mock.Anything, mock.Anything).Return(&domain.Announcement{ID: "x"}, nil)
		in := validNew()
		in.ExpiresAt = annNow.Add(30 * 24 * time.Hour)
		_, err := uc.Publish(context.Background(), "t", "7", in)
		require.NoError(t, err)
	})
}

func TestPublish_TempleAndBodyChecks(t *testing.T) {
	t.Run("temple missing", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(false, nil)
		_, err := uc.Publish(context.Background(), "t", "7", validNew())
		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
		ar.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything)
	})
	t.Run("issuing body of another temple", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(true, nil)
		ar.On("IssuingBodyBelongs", mock.Anything, "tib_1", "t").Return(false, nil)
		_, err := uc.Publish(context.Background(), "t", "7", validNew())
		assert.ErrorIs(t, err, domain.ErrInvalidIssuingBody)
		ar.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything)
	})
	t.Run("temple vanished between check and insert", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(true, nil)
		ar.On("IssuingBodyBelongs", mock.Anything, "tib_1", "t").Return(true, nil)
		ar.On("Insert", mock.Anything, mock.Anything).Return(nil, domain.ErrTempleNotFound)
		_, err := uc.Publish(context.Background(), "t", "7", validNew())
		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
	})
	t.Run("db error is returned", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(false, errors.New("boom"))
		_, err := uc.Publish(context.Background(), "t", "7", validNew())
		assert.EqualError(t, err, "boom")
	})
}

func TestEdit(t *testing.T) {
	existing := func() *domain.Announcement {
		return &domain.Announcement{
			ID: "tann_1", TempleID: "t", IssuingBodyID: "tib_1", Title: "Cũ", Body: "Cũ",
			Source: "admin", PublishedAt: annNow.Add(-time.Hour), ExpiresAt: annNow.Add(time.Hour),
		}
	}
	t.Run("updates only the given fields", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("FindByID", mock.Anything, "tann_1").Return(existing(), nil)
		want := *existing()
		want.Title = "Mới"
		stored := want
		ar.On("Update", mock.Anything, want).Return(&stored, nil)
		got, err := uc.Edit(context.Background(), "tann_1", domain.AnnouncementPatch{Title: strp("Mới")})
		require.NoError(t, err)
		assert.Equal(t, "Mới", got.Title)
		assert.Equal(t, domain.AnnouncementActive, got.State)
		ar.AssertNotCalled(t, "IssuingBodyBelongs", mock.Anything, mock.Anything, mock.Anything)
	})
	t.Run("changing issuing body checks it against the announcement's temple", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("FindByID", mock.Anything, "tann_1").Return(existing(), nil)
		ar.On("IssuingBodyBelongs", mock.Anything, "tib_9", "t").Return(false, nil)
		_, err := uc.Edit(context.Background(), "tann_1", domain.AnnouncementPatch{IssuingBodyID: strp("tib_9")})
		assert.ErrorIs(t, err, domain.ErrInvalidIssuingBody)
		ar.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})
	t.Run("not found", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("FindByID", mock.Anything, "nope").Return(nil, domain.ErrAnnouncementNotFound)
		_, err := uc.Edit(context.Background(), "nope", domain.AnnouncementPatch{Title: strp("x")})
		assert.ErrorIs(t, err, domain.ErrAnnouncementNotFound)
	})
	t.Run("revoked is refused and nothing is written", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		a := existing()
		a.RevokedAt = &annNow
		ar.On("FindByID", mock.Anything, "tann_1").Return(a, nil)
		_, err := uc.Edit(context.Background(), "tann_1", domain.AnnouncementPatch{Title: strp("x")})
		assert.ErrorIs(t, err, domain.ErrAnnouncementRevoked)
		ar.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})
	t.Run("validation as publish", func(t *testing.T) {
		past := annNow.Add(-time.Minute)
		for name, p := range map[string]domain.AnnouncementPatch{
			"blank title":     {Title: strp(" ")},
			"blank body":      {Body: strp("")},
			"long title":      {Title: strp(strings.Repeat("a", 11))},
			"past expires_at": {ExpiresAt: &past},
			"blank body id":   {IssuingBodyID: strp("")},
		} {
			uc, ar, _ := newAnnUC()
			ar.On("FindByID", mock.Anything, "tann_1").Return(existing(), nil)
			_, err := uc.Edit(context.Background(), "tann_1", p)
			assert.ErrorIs(t, err, domain.ErrInvalidAnnouncement, name)
			ar.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		}
	})
}

func TestRevoke(t *testing.T) {
	t.Run("revokes with now", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		rev := annNow
		ar.On("Revoke", mock.Anything, "tann_1", annNow).Return(&domain.Announcement{ID: "tann_1", ExpiresAt: annNow.Add(time.Hour), RevokedAt: &rev}, nil)
		got, err := uc.Revoke(context.Background(), "tann_1")
		require.NoError(t, err)
		assert.Equal(t, domain.AnnouncementRevoked, got.State)
	})
	t.Run("second revoke returns the original revoked_at (idempotent)", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		orig := annNow.Add(-24 * time.Hour)
		ar.On("Revoke", mock.Anything, "tann_1", annNow).Return(&domain.Announcement{ID: "tann_1", RevokedAt: &orig}, nil)
		got, err := uc.Revoke(context.Background(), "tann_1")
		require.NoError(t, err)
		assert.Equal(t, orig, *got.RevokedAt)
	})
	t.Run("not found", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("Revoke", mock.Anything, "nope", annNow).Return(nil, domain.ErrAnnouncementNotFound)
		_, err := uc.Revoke(context.Background(), "nope")
		assert.ErrorIs(t, err, domain.ErrAnnouncementNotFound)
	})
}

func TestList(t *testing.T) {
	rev := annNow
	items := []domain.Announcement{
		{ID: "a", ExpiresAt: annNow.Add(time.Hour)},
		{ID: "b", ExpiresAt: annNow},
		{ID: "c", ExpiresAt: annNow.Add(time.Hour), RevokedAt: &rev},
	}
	t.Run("derives state, default page size, offset", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(true, nil)
		ar.On("ListByTemple", mock.Anything, "t", 2, 2).Return(items, 5, nil)
		p, err := uc.List(context.Background(), "t", 2, 0)
		require.NoError(t, err)
		assert.Equal(t, 2, p.PageSize)
		assert.Equal(t, 3, p.TotalPages)
		assert.Equal(t, []domain.AnnouncementState{domain.AnnouncementActive, domain.AnnouncementExpired, domain.AnnouncementRevoked},
			[]domain.AnnouncementState{p.Items[0].State, p.Items[1].State, p.Items[2].State})
	})
	t.Run("page_size is capped at max", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(true, nil)
		ar.On("ListByTemple", mock.Anything, "t", 0, 5).Return([]domain.Announcement(nil), 0, nil)
		p, err := uc.List(context.Background(), "t", 1, 999)
		require.NoError(t, err)
		assert.Equal(t, 5, p.PageSize)
		assert.NotNil(t, p.Items)
	})
	t.Run("invalid page", func(t *testing.T) {
		uc, _, _ := newAnnUC()
		_, err := uc.List(context.Background(), "t", 0, 0)
		assert.ErrorIs(t, err, domain.ErrInvalidPage)
		_, err = uc.List(context.Background(), "t", 1, -1)
		assert.ErrorIs(t, err, domain.ErrInvalidPage)
	})
	t.Run("temple missing", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(false, nil)
		_, err := uc.List(context.Background(), "t", 1, 0)
		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
	})
}

func TestCreateIssuingBody(t *testing.T) {
	t.Run("trims and creates", func(t *testing.T) {
		uc, ar, br := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(true, nil)
		br.On("Insert", mock.Anything, "t", "Ban Trị").Return(&domain.IssuingBody{ID: "tib_1", TempleID: "t", Name: "Ban Trị"}, nil)
		b, err := uc.CreateIssuingBody(context.Background(), "t", "  Ban Trị ")
		require.NoError(t, err)
		assert.Equal(t, "tib_1", b.ID)
	})
	t.Run("blank and too long (runes)", func(t *testing.T) {
		for _, name := range []string{"  ", strings.Repeat("ệ", 9)} {
			uc, _, br := newAnnUC()
			_, err := uc.CreateIssuingBody(context.Background(), "t", name)
			require.ErrorIs(t, err, domain.ErrInvalidIssuingBodyName)
			assert.Equal(t, "name", fieldOf(t, err))
			br.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything, mock.Anything)
		}
	})
	t.Run("temple missing", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(false, nil)
		_, err := uc.CreateIssuingBody(context.Background(), "t", "Ban")
		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
	})
	t.Run("duplicate name in another case maps to exists", func(t *testing.T) {
		uc, ar, br := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(true, nil)
		br.On("Insert", mock.Anything, "t", "BAN").Return(nil, domain.ErrIssuingBodyExists)
		_, err := uc.CreateIssuingBody(context.Background(), "t", "BAN")
		assert.ErrorIs(t, err, domain.ErrIssuingBodyExists)
	})
}

func TestListIssuingBodies(t *testing.T) {
	t.Run("lists", func(t *testing.T) {
		uc, ar, br := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(true, nil)
		br.On("ListByTemple", mock.Anything, "t").Return([]domain.IssuingBody{{ID: "tib_1"}}, nil)
		got, err := uc.ListIssuingBodies(context.Background(), "t")
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})
	t.Run("temple missing", func(t *testing.T) {
		uc, ar, _ := newAnnUC()
		ar.On("TempleExists", mock.Anything, "t").Return(false, nil)
		_, err := uc.ListIssuingBodies(context.Background(), "t")
		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
	})
}
