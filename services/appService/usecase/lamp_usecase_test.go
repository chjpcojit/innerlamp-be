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

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

var lampNow = time.Date(2026, 10, 10, 12, 0, 0, 123456789, time.UTC)

func testLampConfig() config.LampConfig {
	return config.LampConfig{
		MaxActivePerUser: 20, PageSize: 12,
		DedupeWindow: config.Duration(5 * time.Second), DurationsDays: []int{1, 7, 21, 49},
		RecipientMaxChars: 100, PrayerMaxChars: 300,
	}
}

func newLampUsecase(repo *domain.MockLampRepository) *LampUsecase {
	u := NewLampUsecase(repo, testLampConfig())
	u.now = func() time.Time { return lampNow }
	return u
}

func validLampInput() domain.LampInput {
	return domain.LampInput{
		LampType: "OIL_LAMP", Intention: "PEACE", Recipient: strPtr("Bà ngoại"), DurationDays: 7,
	}
}

func assertInvalidLampField(t *testing.T, err error, field string) {
	t.Helper()
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLampInvalid)
	var ve *domain.LampValidationError
	require.True(t, errors.As(err, &ve), "want *LampValidationError, got %T", err)
	assert.Equal(t, field, ve.Field)
}

func TestLampUsecase_Light_Validation(t *testing.T) {
	tests := []struct {
		name  string
		edit  func(in *domain.LampInput)
		field string
	}{
		{"unknown lamp type", func(in *domain.LampInput) { in.LampType = "CANDLE" }, "lamp_type"},
		{"empty lamp type", func(in *domain.LampInput) { in.LampType = "" }, "lamp_type"},
		{"lamp type is case sensitive", func(in *domain.LampInput) { in.LampType = "oil_lamp" }, "lamp_type"},
		{"unknown intention", func(in *domain.LampInput) { in.Intention = "WEALTH" }, "intention"},
		{"empty intention", func(in *domain.LampInput) { in.Intention = "" }, "intention"},
		{"recipient 101 characters", func(in *domain.LampInput) { in.Recipient = strPtr(strings.Repeat("ạ", 101)) }, "recipient"},
		{"prayer 301 characters", func(in *domain.LampInput) { in.Prayer = strPtr(strings.Repeat("ế", 301)) }, "prayer"},
		{"duration not allowed", func(in *domain.LampInput) { in.DurationDays = 5 }, "duration_days"},
		{"duration zero", func(in *domain.LampInput) { in.DurationDays = 0 }, "duration_days"},
		{"duration negative", func(in *domain.LampInput) { in.DurationDays = -7 }, "duration_days"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockLampRepository)
			in := validLampInput()
			tt.edit(&in)

			lamp, created, err := newLampUsecase(repo).Light(context.Background(), 7, in)

			assertInvalidLampField(t, err, tt.field)
			assert.Nil(t, lamp)
			assert.False(t, created)
			repo.AssertNotCalled(t, "Light", mock.Anything, mock.Anything)
		})
	}
}

// v2.0 (BR-LMP-06): a recipient that is absent, empty or only whitespace is not
// an error; the lamp is stored with no recipient.
// VI: v2.0: recipient không gửi, rỗng hoặc toàn khoảng trắng không phải lỗi; đèn
// được lưu không có người nhận.
func TestLampUsecase_Light_BlankRecipientMeansNone(t *testing.T) {
	tests := []struct {
		name      string
		recipient *string
	}{
		{"absent", nil},
		{"empty", strPtr("")},
		{"only whitespace", strPtr("   \t ")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockLampRepository)
			var got domain.LightLampParams
			repo.On("Light", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) { got = args.Get(1).(domain.LightLampParams) }).
				Return(&domain.Lamp{ID: 1}, true, nil)
			in := validLampInput()
			in.Recipient = tt.recipient

			_, created, err := newLampUsecase(repo).Light(context.Background(), 7, in)

			require.NoError(t, err)
			assert.True(t, created)
			assert.Nil(t, got.Lamp.Recipient)
		})
	}
}

func TestLampUsecase_Light_LengthBoundaries(t *testing.T) {
	decomposed := strings.Repeat("ệ", 100) // ệ typed as e + circumflex + dot below, 100 times
	require.Greater(t, len([]rune(decomposed)), 100, "fixture must exceed 100 code points before NFC")

	tests := []struct {
		name      string
		recipient string
		prayer    *string
		wantErr   string
	}{
		{"recipient 1", "A", nil, ""},
		{"recipient 100", strings.Repeat("ạ", 100), nil, ""},
		{"recipient 100 as decomposed accents counts as 100", decomposed, nil, ""},
		{"recipient 100 plus surrounding spaces", "  " + strings.Repeat("ạ", 100) + "  ", nil, ""},
		{"prayer 300", "A", strPtr(strings.Repeat("ế", 300)), ""},
		{"prayer 301", "A", strPtr(strings.Repeat("ế", 301)), "prayer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockLampRepository)
			if tt.wantErr == "" {
				repo.On("Light", mock.Anything, mock.Anything).Return(&domain.Lamp{ID: 1}, true, nil)
			}
			in := validLampInput()
			in.Recipient = strPtr(tt.recipient)
			in.Prayer = tt.prayer

			_, _, err := newLampUsecase(repo).Light(context.Background(), 7, in)

			if tt.wantErr != "" {
				assertInvalidLampField(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// The length limits are read from config, not from constants: a stricter
// config rejects text the dev defaults accept.
func TestLampUsecase_Light_LengthLimitsComeFromConfig(t *testing.T) {
	cfg := testLampConfig()
	cfg.RecipientMaxChars = 10
	cfg.PrayerMaxChars = 20

	tests := []struct {
		name      string
		recipient string
		prayer    string
		wantField string
	}{
		{"recipient at the configured limit", strings.Repeat("ạ", 10), "", ""},
		{"recipient one over the configured limit", strings.Repeat("ạ", 11), "", "recipient"},
		{"prayer at the configured limit", "A", strings.Repeat("ế", 20), ""},
		{"prayer one over the configured limit", "A", strings.Repeat("ế", 21), "prayer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockLampRepository)
			repo.On("Light", mock.Anything, mock.Anything).Return(&domain.Lamp{ID: 1}, true, nil).Maybe()
			u := NewLampUsecase(repo, cfg)
			u.now = func() time.Time { return lampNow }
			in := validLampInput()
			in.Recipient = strPtr(tt.recipient)
			if tt.prayer != "" {
				in.Prayer = strPtr(tt.prayer)
			}

			_, _, err := u.Light(context.Background(), 7, in)

			if tt.wantField != "" {
				assertInvalidLampField(t, err, tt.wantField)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLampUsecase_Light_PassesNormalizedLampAndPolicy(t *testing.T) {
	repo := new(domain.MockLampRepository)
	var got domain.LightLampParams
	stored := &domain.Lamp{ID: 11, UserID: 7}
	repo.On("Light", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { got = args.Get(1).(domain.LightLampParams) }).
		Return(stored, true, nil)

	in := domain.LampInput{
		LampType: "LOTUS_CANDLE", Intention: "GRATITUDE", Recipient: strPtr("  Thầy Minh  "),
		Prayer: strPtr("  Tri ân  "), DurationDays: 21, IsAnonymous: true,
	}
	lamp, created, err := newLampUsecase(repo).Light(context.Background(), 7, in)

	require.NoError(t, err)
	assert.True(t, created)
	assert.Same(t, stored, lamp)

	assert.Equal(t, int64(7), got.Lamp.UserID)
	assert.Equal(t, domain.LampTypeLotusCandle, got.Lamp.LampType)
	assert.Equal(t, domain.IntentionGratitude, got.Lamp.Intention)
	require.NotNil(t, got.Lamp.Recipient)
	assert.Equal(t, "Thầy Minh", *got.Lamp.Recipient, "recipient is trimmed")
	require.NotNil(t, got.Lamp.Prayer)
	assert.Equal(t, "Tri ân", *got.Lamp.Prayer, "prayer is trimmed")
	assert.Equal(t, 21, got.Lamp.DurationDays)
	assert.True(t, got.Lamp.IsAnonymous)
	assert.Nil(t, got.Lamp.LighterName, "the repository fills the name, not the use case")

	wantLit := time.Date(2026, 10, 10, 12, 0, 0, 123456000, time.UTC)
	assert.Equal(t, wantLit, got.Lamp.LitAt, "lit_at is the server clock at microsecond precision")
	assert.Equal(t, wantLit.Add(21*24*time.Hour), got.Lamp.EndAt, "end_at is lit_at + 21×24h")

	assert.Equal(t, 20, got.MaxActive)
	assert.Equal(t, 5*time.Second, got.DedupeWindow)
}

func TestLampUsecase_Light_EndAtForEveryDuration(t *testing.T) {
	for _, days := range []int{1, 7, 21, 49} {
		repo := new(domain.MockLampRepository)
		var got domain.LightLampParams
		repo.On("Light", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { got = args.Get(1).(domain.LightLampParams) }).
			Return(&domain.Lamp{}, true, nil)

		in := validLampInput()
		in.DurationDays = days
		_, _, err := newLampUsecase(repo).Light(context.Background(), 7, in)

		require.NoError(t, err)
		assert.Equal(t, time.Duration(days)*24*time.Hour, got.Lamp.EndAt.Sub(got.Lamp.LitAt), "%d days", days)
	}
}

func TestLampUsecase_Light_BlankPrayerBecomesNil(t *testing.T) {
	for name, prayer := range map[string]*string{"nil": nil, "empty": strPtr(""), "only spaces": strPtr("  \n ")} {
		t.Run(name, func(t *testing.T) {
			repo := new(domain.MockLampRepository)
			var got domain.LightLampParams
			repo.On("Light", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) { got = args.Get(1).(domain.LightLampParams) }).
				Return(&domain.Lamp{}, true, nil)

			in := validLampInput()
			in.Prayer = prayer
			_, _, err := newLampUsecase(repo).Light(context.Background(), 7, in)

			require.NoError(t, err)
			assert.Nil(t, got.Lamp.Prayer)
		})
	}
}

func TestLampUsecase_Light_RepositoryOutcomes(t *testing.T) {
	t.Run("replayed lamp is returned with created=false", func(t *testing.T) {
		repo := new(domain.MockLampRepository)
		existing := &domain.Lamp{ID: 3}
		repo.On("Light", mock.Anything, mock.Anything).Return(existing, false, nil)

		lamp, created, err := newLampUsecase(repo).Light(context.Background(), 7, validLampInput())

		require.NoError(t, err)
		assert.False(t, created)
		assert.Same(t, existing, lamp)
	})

	t.Run("limit reached passes through", func(t *testing.T) {
		repo := new(domain.MockLampRepository)
		repo.On("Light", mock.Anything, mock.Anything).Return(nil, false, domain.ErrLampLimitReached)

		lamp, _, err := newLampUsecase(repo).Light(context.Background(), 7, validLampInput())

		assert.ErrorIs(t, err, domain.ErrLampLimitReached)
		assert.Nil(t, lamp)
	})

	t.Run("other errors are wrapped", func(t *testing.T) {
		repo := new(domain.MockLampRepository)
		boom := errors.New("db down")
		repo.On("Light", mock.Anything, mock.Anything).Return(nil, false, boom)

		_, _, err := newLampUsecase(repo).Light(context.Background(), 7, validLampInput())

		assert.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, domain.ErrLampInvalid)
	})
}

// ---- Garden ----

func lampAt(id int64, litAt time.Time) domain.GardenLamp {
	return domain.GardenLamp{ID: id, LampType: domain.LampTypeOilLamp, Recipient: strPtr("r"), LitAt: litAt, EndAt: litAt.Add(24 * time.Hour)}
}

func TestLampUsecase_Garden_FilterValidation(t *testing.T) {
	tests := []struct {
		name   string
		filter domain.GardenFilter
		want   error
	}{
		{"unknown lamp type", domain.GardenFilter{LampType: strPtr("CANDLE")}, domain.ErrInvalidLampFilter},
		{"empty lamp type", domain.GardenFilter{LampType: strPtr("")}, domain.ErrInvalidLampFilter},
		{"limit zero", domain.GardenFilter{Limit: strPtr("0")}, domain.ErrInvalidLimit},
		{"limit negative", domain.GardenFilter{Limit: strPtr("-3")}, domain.ErrInvalidLimit},
		{"limit not a number", domain.GardenFilter{Limit: strPtr("many")}, domain.ErrInvalidLimit},
		{"limit empty", domain.GardenFilter{Limit: strPtr("")}, domain.ErrInvalidLimit},
		{"limit overflows", domain.GardenFilter{Limit: strPtr("99999999999999999999")}, domain.ErrInvalidLimit},
		{"cursor garbage", domain.GardenFilter{Cursor: strPtr("!!!")}, domain.ErrInvalidCursor},
		{"cursor empty", domain.GardenFilter{Cursor: strPtr("")}, domain.ErrInvalidCursor},
		{"cursor valid base64 but wrong shape", domain.GardenFilter{Cursor: strPtr("aGVsbG8")}, domain.ErrInvalidCursor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockLampRepository)

			page, err := newLampUsecase(repo).Garden(context.Background(), 7, tt.filter)

			assert.ErrorIs(t, err, tt.want)
			assert.Nil(t, page)
			repo.AssertNotCalled(t, "Garden", mock.Anything, mock.Anything)
		})
	}
}

func TestLampUsecase_Garden_QueryPassedToRepository(t *testing.T) {
	tests := []struct {
		name      string
		filter    domain.GardenFilter
		wantLimit int
		wantType  *domain.LampType
	}{
		{"defaults to page size", domain.GardenFilter{}, 12, nil},
		{"limit below page size is kept", domain.GardenFilter{Limit: strPtr("5")}, 5, nil},
		{"limit above page size is clamped", domain.GardenFilter{Limit: strPtr("50")}, 12, nil},
		{"lamp type filter", domain.GardenFilter{LampType: strPtr("LOTUS_CANDLE")}, 12, lampTypePtr(domain.LampTypeLotusCandle)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockLampRepository)
			var got domain.GardenQuery
			repo.On("Garden", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) { got = args.Get(1).(domain.GardenQuery) }).
				Return(&domain.GardenPage{}, nil)

			_, err := newLampUsecase(repo).Garden(context.Background(), 7, tt.filter)

			require.NoError(t, err)
			assert.Equal(t, tt.wantLimit, got.Limit)
			assert.Equal(t, tt.wantType, got.LampType)
			assert.Nil(t, got.After)
			assert.Equal(t, lampNow.Truncate(time.Microsecond), got.Now)
		})
	}
}

func lampTypePtr(t domain.LampType) *domain.LampType { return &t }

func TestLampUsecase_Garden_PaginationAndCursorRoundTrip(t *testing.T) {
	t0 := time.Date(2026, 10, 10, 11, 0, 0, 500000000, time.UTC)

	t.Run("more rows than limit sets has_more and a cursor that continues after the last shown lamp", func(t *testing.T) {
		repo := new(domain.MockLampRepository)
		// limit 2: the repository returns 3 rows (limit+1).
		rows := []domain.GardenLamp{lampAt(30, t0), lampAt(20, t0.Add(-time.Minute)), lampAt(10, t0.Add(-2*time.Minute))}
		repo.On("Garden", mock.Anything, mock.Anything).Return(&domain.GardenPage{Items: rows, Total: 3}, nil).Once()

		u := newLampUsecase(repo)
		page, err := u.Garden(context.Background(), 7, domain.GardenFilter{Limit: strPtr("2")})

		require.NoError(t, err)
		require.Len(t, page.Items, 2)
		assert.Equal(t, int64(30), page.Items[0].ID)
		assert.Equal(t, int64(20), page.Items[1].ID)
		assert.True(t, page.HasMore)
		require.NotNil(t, page.NextCursor)

		// Feeding the cursor back must resume strictly after lamp 20.
		var got domain.GardenQuery
		repo.On("Garden", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { got = args.Get(1).(domain.GardenQuery) }).
			Return(&domain.GardenPage{}, nil).Once()
		_, err = u.Garden(context.Background(), 7, domain.GardenFilter{Limit: strPtr("2"), Cursor: page.NextCursor})

		require.NoError(t, err)
		require.NotNil(t, got.After)
		assert.Equal(t, int64(20), got.After.ID)
		assert.Equal(t, t0.Add(-time.Minute), got.After.LitAt)
	})

	t.Run("exactly limit rows means no more", func(t *testing.T) {
		repo := new(domain.MockLampRepository)
		rows := []domain.GardenLamp{lampAt(2, t0), lampAt(1, t0.Add(-time.Minute))}
		repo.On("Garden", mock.Anything, mock.Anything).Return(&domain.GardenPage{Items: rows, Total: 2}, nil)

		page, err := newLampUsecase(repo).Garden(context.Background(), 7, domain.GardenFilter{Limit: strPtr("2")})

		require.NoError(t, err)
		assert.Len(t, page.Items, 2)
		assert.False(t, page.HasMore)
		assert.Nil(t, page.NextCursor)
	})

	t.Run("empty garden has empty items, zero counts for every type and no cursor", func(t *testing.T) {
		repo := new(domain.MockLampRepository)
		repo.On("Garden", mock.Anything, mock.Anything).Return(&domain.GardenPage{}, nil)

		page, err := newLampUsecase(repo).Garden(context.Background(), 7, domain.GardenFilter{})

		require.NoError(t, err)
		assert.NotNil(t, page.Items)
		assert.Empty(t, page.Items)
		assert.Equal(t, 0, page.Total)
		assert.Equal(t, map[domain.LampType]int{
			domain.LampTypeOilLamp: 0, domain.LampTypeLotusCandle: 0, domain.LampTypeFlowerLantern: 0,
		}, page.Counts)
		assert.False(t, page.HasMore)
		assert.Nil(t, page.NextCursor)
	})

	t.Run("a type missing from the repository counts is filled with zero", func(t *testing.T) {
		repo := new(domain.MockLampRepository)
		repo.On("Garden", mock.Anything, mock.Anything).Return(&domain.GardenPage{
			Total: 5, Counts: map[domain.LampType]int{domain.LampTypeOilLamp: 5},
		}, nil)

		page, err := newLampUsecase(repo).Garden(context.Background(), 7, domain.GardenFilter{})

		require.NoError(t, err)
		assert.Equal(t, 5, page.Counts[domain.LampTypeOilLamp])
		assert.Equal(t, 0, page.Counts[domain.LampTypeLotusCandle])
		assert.Equal(t, 0, page.Counts[domain.LampTypeFlowerLantern])
	})
}

func TestLampUsecase_Garden_RepositoryError(t *testing.T) {
	repo := new(domain.MockLampRepository)
	boom := errors.New("db down")
	repo.On("Garden", mock.Anything, mock.Anything).Return(nil, boom)

	page, err := newLampUsecase(repo).Garden(context.Background(), 7, domain.GardenFilter{})

	assert.ErrorIs(t, err, boom)
	assert.Nil(t, page)
}

// v2.0 (AC 25): the viewer's id reaches the repository only as ViewerID, which
// marks is_mine; the query is otherwise unchanged.
// VI: v2.0: id người xem chỉ được chuyển xuống repository dưới dạng ViewerID để
// tính is_mine; truy vấn không thay đổi gì khác.
func TestLampUsecase_Garden_PassesViewerID(t *testing.T) {
	repo := new(domain.MockLampRepository)
	var got domain.GardenQuery
	repo.On("Garden", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { got = args.Get(1).(domain.GardenQuery) }).
		Return(&domain.GardenPage{}, nil)

	_, err := newLampUsecase(repo).Garden(context.Background(), 42, domain.GardenFilter{})

	require.NoError(t, err)
	assert.Equal(t, int64(42), got.ViewerID)
	assert.Nil(t, got.LampType)
	assert.Equal(t, 12, got.Limit)
}
