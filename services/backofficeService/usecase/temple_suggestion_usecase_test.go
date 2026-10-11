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

	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

type suggestionFixture struct {
	repo  *domain.MockTempleSuggestionRepository
	wards *domain.MockWardResolver
	uc    *TempleSuggestionUsecase
}

func newSuggestionFixture() *suggestionFixture {
	f := &suggestionFixture{repo: &domain.MockTempleSuggestionRepository{}, wards: &domain.MockWardResolver{}}
	f.uc = NewTempleSuggestionUsecase(f.repo, f.wards, 20)
	f.uc.now = func() time.Time { return time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC) }
	return f
}

func validNewTemple() domain.NewTemple {
	return domain.NewTemple{
		Name: "  Chùa Mới  ", Type: "CHUA", Region: "south", Sect: "Bắc tông",
		EstablishedYear: 1990, HeritageRank: "DI_TICH_CAP_TINH",
		Latitude: 10.77, Longitude: 106.69,
		ProvinceCode: "79", WardCode: "26740", Address: "  12 Nguyễn Huệ ",
		ImageURL: "/static/images/x.jpg",
	}
}

func TestTempleSuggestionUsecase_List(t *testing.T) {
	ctx := context.Background()

	t.Run("defaults to pending, page 1", func(t *testing.T) {
		f := newSuggestionFixture()
		f.repo.On("List", ctx, domain.TempleSuggestionPending, 0, 20).
			Return([]domain.TempleSuggestion{{ID: 1}}, 41, nil)
		page, err := f.uc.List(ctx, "", 1)
		require.NoError(t, err)
		assert.Equal(t, 1, page.Page)
		assert.Equal(t, 20, page.PageSize)
		assert.Equal(t, 41, page.Total)
		assert.Equal(t, 3, page.TotalPages)
		assert.Len(t, page.Items, 1)
	})

	t.Run("page offset and past-the-end is empty not error", func(t *testing.T) {
		f := newSuggestionFixture()
		f.repo.On("List", ctx, domain.TempleSuggestionCreated, 40, 20).Return([]domain.TempleSuggestion(nil), 41, nil)
		page, err := f.uc.List(ctx, "created", 3)
		require.NoError(t, err)
		assert.NotNil(t, page.Items)
		assert.Empty(t, page.Items)
	})

	t.Run("unknown status", func(t *testing.T) {
		f := newSuggestionFixture()
		_, err := f.uc.List(ctx, "bogus", 1)
		assert.ErrorIs(t, err, domain.ErrInvalidStatus)
		f.repo.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("page below 1", func(t *testing.T) {
		f := newSuggestionFixture()
		_, err := f.uc.List(ctx, "", 0)
		assert.ErrorIs(t, err, domain.ErrInvalidPage)
	})

	t.Run("empty queue has zero pages", func(t *testing.T) {
		f := newSuggestionFixture()
		f.repo.On("List", ctx, domain.TempleSuggestionPending, 0, 20).Return([]domain.TempleSuggestion(nil), 0, nil)
		page, err := f.uc.List(ctx, "", 1)
		require.NoError(t, err)
		assert.Equal(t, 0, page.TotalPages)
	})
}

func TestTempleSuggestionUsecase_Get(t *testing.T) {
	f := newSuggestionFixture()
	f.repo.On("FindByID", mock.Anything, int64(9)).Return(nil, domain.ErrSuggestionNotFound)
	_, err := f.uc.Get(context.Background(), 9)
	assert.ErrorIs(t, err, domain.ErrSuggestionNotFound)
}

func TestTempleSuggestionUsecase_CreateTemple_Success(t *testing.T) {
	ctx := context.Background()
	f := newSuggestionFixture()
	f.wards.On("ResolveCurrentWard", ctx, "79", "26740").
		Return(&domain.ResolvedWard{ProvinceName: "Hồ Chí Minh", WardFullName: "Phường Sài Gòn"}, nil)

	var got domain.NewTemple
	f.repo.On("CreateTemple", ctx, int64(5), int64(7), domain.RoleContentEditor, mock.Anything, 6).
		Run(func(a mock.Arguments) { got = a.Get(4).(domain.NewTemple) }).
		Return(&domain.CreatedTemple{}, nil)

	_, err := f.uc.CreateTemple(ctx, 5, 7, domain.RoleContentEditor, validNewTemple())
	require.NoError(t, err)

	assert.Equal(t, "Chùa Mới", got.Name, "trimmed")
	assert.Equal(t, "12 Nguyễn Huệ", got.Address, "trimmed")
	assert.Equal(t, "Hồ Chí Minh", got.Province, "name from dataset")
	assert.Equal(t, "Phường Sài Gòn", got.Ward, "name from dataset")
	assert.NotNil(t, got.Aliases)
	assert.NotNil(t, got.VisitingNotes)
}

func TestTempleSuggestionUsecase_CreateTemple_IgnoresClientNames(t *testing.T) {
	ctx := context.Background()
	f := newSuggestionFixture()
	f.wards.On("ResolveCurrentWard", ctx, "79", "26740").
		Return(&domain.ResolvedWard{ProvinceName: "Hồ Chí Minh", WardFullName: "Phường Sài Gòn"}, nil)
	var got domain.NewTemple
	f.repo.On("CreateTemple", ctx, int64(1), int64(1), domain.RoleAdmin, mock.Anything, mock.Anything).
		Run(func(a mock.Arguments) { got = a.Get(4).(domain.NewTemple) }).
		Return(&domain.CreatedTemple{}, nil)

	in := validNewTemple()
	in.Province, in.Ward = "Hà Nội", "Phường Giả"
	_, err := f.uc.CreateTemple(ctx, 1, 1, domain.RoleAdmin, in)
	require.NoError(t, err)
	assert.Equal(t, "Hồ Chí Minh", got.Province)
	assert.Equal(t, "Phường Sài Gòn", got.Ward)
}

func TestTempleSuggestionUsecase_CreateTemple_Validation(t *testing.T) {
	long := strings.Repeat("a", 256)
	cases := []struct {
		name  string
		mod   func(*domain.NewTemple)
		field string
	}{
		{"empty name", func(n *domain.NewTemple) { n.Name = "  " }, "name"},
		{"name too long", func(n *domain.NewTemple) { n.Name = long }, "name"},
		{"empty type", func(n *domain.NewTemple) { n.Type = "" }, "type"},
		{"bad type", func(n *domain.NewTemple) { n.Type = "NHA_THO" }, "type"},
		{"empty region", func(n *domain.NewTemple) { n.Region = "" }, "region"},
		{"bad region", func(n *domain.NewTemple) { n.Region = "east" }, "region"},
		{"empty sect", func(n *domain.NewTemple) { n.Sect = "" }, "sect"},
		{"year zero", func(n *domain.NewTemple) { n.EstablishedYear = 0 }, "established_year"},
		{"year future", func(n *domain.NewTemple) { n.EstablishedYear = 2027 }, "established_year"},
		{"empty rank", func(n *domain.NewTemple) { n.HeritageRank = "" }, "heritage_rank"},
		{"bad rank", func(n *domain.NewTemple) { n.HeritageRank = "X" }, "heritage_rank"},
		{"lat high", func(n *domain.NewTemple) { n.Latitude = 90.5 }, "latitude"},
		{"lat low", func(n *domain.NewTemple) { n.Latitude = -91 }, "latitude"},
		{"lng high", func(n *domain.NewTemple) { n.Longitude = 180.1 }, "longitude"},
		{"empty province_code", func(n *domain.NewTemple) { n.ProvinceCode = " " }, "province_code"},
		{"empty ward_code", func(n *domain.NewTemple) { n.WardCode = "" }, "ward_code"},
		{"empty address", func(n *domain.NewTemple) { n.Address = "  " }, "address"},
		{"bad image", func(n *domain.NewTemple) { n.ImageURL = "ftp://x/y.jpg" }, "image_url"},
		{"bad thumbnail", func(n *domain.NewTemple) { n.ThumbnailURL = "images/y.jpg" }, "thumbnail_url"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newSuggestionFixture()
			in := validNewTemple()
			tc.mod(&in)
			_, err := f.uc.CreateTemple(context.Background(), 1, 1, domain.RoleAdmin, in)
			var ie *domain.TempleInputError
			require.ErrorAs(t, err, &ie)
			assert.Equal(t, tc.field, ie.Field)
			assert.ErrorIs(t, err, domain.ErrTempleInputInvalid)
			f.wards.AssertNotCalled(t, "ResolveCurrentWard", mock.Anything, mock.Anything, mock.Anything)
			f.repo.AssertNotCalled(t, "CreateTemple", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestTempleSuggestionUsecase_CreateTemple_AddressCodes(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		resolver error
		field    string
	}{
		{"unknown province", domain.ErrAdministrativeUnitNotFound, "province_code"},
		{"ward of another province", domain.ErrWardNotInProvince, "ward_code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newSuggestionFixture()
			f.wards.On("ResolveCurrentWard", ctx, "79", "26740").Return(nil, tc.resolver)
			_, err := f.uc.CreateTemple(ctx, 1, 1, domain.RoleAdmin, validNewTemple())
			var ie *domain.TempleInputError
			require.ErrorAs(t, err, &ie)
			assert.Equal(t, tc.field, ie.Field)
			f.repo.AssertNotCalled(t, "CreateTemple", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}

	t.Run("resolver failure is not an input error", func(t *testing.T) {
		f := newSuggestionFixture()
		boom := errors.New("db down")
		f.wards.On("ResolveCurrentWard", ctx, "79", "26740").Return(nil, boom)
		_, err := f.uc.CreateTemple(ctx, 1, 1, domain.RoleAdmin, validNewTemple())
		assert.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, domain.ErrTempleInputInvalid)
	})
}

func TestTempleSuggestionUsecase_CreateTemple_RepoErrorsPassThrough(t *testing.T) {
	ctx := context.Background()
	for _, e := range []error{domain.ErrSuggestionNotFound, domain.ErrSuggestionAlreadyCreated, domain.ErrReopenForbidden} {
		f := newSuggestionFixture()
		f.wards.On("ResolveCurrentWard", ctx, "79", "26740").Return(&domain.ResolvedWard{ProvinceName: "p", WardFullName: "w"}, nil)
		f.repo.On("CreateTemple", ctx, int64(1), int64(1), domain.RoleContentEditor, mock.Anything, 6).Return(nil, e)
		_, err := f.uc.CreateTemple(ctx, 1, 1, domain.RoleContentEditor, validNewTemple())
		assert.ErrorIs(t, err, e)
	}
}
