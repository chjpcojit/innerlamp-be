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

func intPtr(i int) *int       { return &i }
func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

// validMemorialInput is a full-date solar memorial that passes every rule.
func validMemorialInput() domain.MemorialInput {
	return domain.MemorialInput{
		FullName: "Nguyễn Văn A", Relationship: "FATHER", DeathCalendar: "solar", DeathDate: "2015-03-04",
	}
}

func newMemorialUsecase(repo *domain.MockMemorialRepository) *MemorialUsecase {
	u := NewMemorialUsecase(repo, config.MemorialConfig{MaxPerUser: 100})
	// 2026-10-07 12:00 UTC is already 19:00 in Vietnam, same calendar day.
	u.now = func() time.Time { return time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) }
	return u
}

func assertInvalidField(t *testing.T, err error, field string) {
	t.Helper()
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrMemorialInvalid)
	var ve *domain.MemorialValidationError
	require.True(t, errors.As(err, &ve), "want *MemorialValidationError, got %T", err)
	assert.Equal(t, field, ve.Field)
}

func TestMemorialUsecase_Create_Validation(t *testing.T) {
	tests := []struct {
		name  string
		edit  func(in *domain.MemorialInput)
		field string
	}{
		{"blank full_name", func(in *domain.MemorialInput) { in.FullName = "   " }, "full_name"},
		{"full_name over 100 runes", func(in *domain.MemorialInput) { in.FullName = strings.Repeat("ạ", 101) }, "full_name"},
		{"unknown relationship", func(in *domain.MemorialInput) { in.Relationship = "UNCLE" }, "relationship"},
		{"empty relationship", func(in *domain.MemorialInput) { in.Relationship = "" }, "relationship"},
		{"bad calendar", func(in *domain.MemorialInput) { in.DeathCalendar = "gregorian" }, "death_calendar"},
		{"malformed death_date", func(in *domain.MemorialInput) { in.DeathDate = "04/03/2015" }, "death_date"},
		{"impossible death_date", func(in *domain.MemorialInput) { in.DeathDate = "2015-02-30" }, "death_date"},
		{"death_date before 1900", func(in *domain.MemorialInput) { in.DeathDate = "1899-12-31" }, "death_date"},
		{"death_date in the future", func(in *domain.MemorialInput) { in.DeathDate = "2026-10-08" }, "death_date"},
		{"death_date with death_day", func(in *domain.MemorialInput) { in.DeathDay = intPtr(1) }, "death_date"},
		{"death_date with leap flag", func(in *domain.MemorialInput) { in.DeathLeapMonth = boolPtr(false) }, "death_date"},
		{"no death date at all", func(in *domain.MemorialInput) { in.DeathDate = "" }, "death_date"},
		{"note over 1000 runes", func(in *domain.MemorialInput) { in.Note = strPtr(strings.Repeat("ạ", 1001)) }, "note"},
		{"reminder without days_before", func(in *domain.MemorialInput) {
			in.Reminder = &domain.ReminderInput{Time: "07:30"}
		}, "reminder.days_before"},
		{"reminder days_before 31", func(in *domain.MemorialInput) {
			in.Reminder = &domain.ReminderInput{DaysBefore: intPtr(31), Time: "07:30"}
		}, "reminder.days_before"},
		{"reminder days_before negative", func(in *domain.MemorialInput) {
			in.Reminder = &domain.ReminderInput{DaysBefore: intPtr(-1), Time: "07:30"}
		}, "reminder.days_before"},
		{"reminder time 24:00", func(in *domain.MemorialInput) {
			in.Reminder = &domain.ReminderInput{DaysBefore: intPtr(1), Time: "24:00"}
		}, "reminder.time"},
		{"reminder time single-digit hour", func(in *domain.MemorialInput) {
			in.Reminder = &domain.ReminderInput{DaysBefore: intPtr(1), Time: "7:30"}
		}, "reminder.time"},
		{"reminder timezone unknown", func(in *domain.MemorialInput) {
			in.Reminder = &domain.ReminderInput{DaysBefore: intPtr(1), Time: "07:30", Timezone: "Mars/Olympus"}
		}, "reminder.timezone"},
		{"reminder timezone Local", func(in *domain.MemorialInput) {
			in.Reminder = &domain.ReminderInput{DaysBefore: intPtr(1), Time: "07:30", Timezone: "Local"}
		}, "reminder.timezone"},
		{"OS reminder without reminder", func(in *domain.MemorialInput) { in.AddToOSReminder = true }, "add_to_os_reminder"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockMemorialRepository)
			in := validMemorialInput()
			tt.edit(&in)

			got, err := newMemorialUsecase(repo).Create(context.Background(), 7, in)

			assert.Nil(t, got)
			assertInvalidField(t, err, tt.field)
			repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestMemorialUsecase_Create_PartialDate(t *testing.T) {
	partial := func(cal string, day, month *int, leap *bool) domain.MemorialInput {
		in := validMemorialInput()
		in.DeathDate, in.DeathCalendar = "", cal
		in.DeathDay, in.DeathMonth, in.DeathLeapMonth = day, month, leap
		return in
	}
	tests := []struct {
		name  string
		in    domain.MemorialInput
		field string // "" = valid
	}{
		{"lunar day+month", partial("lunar", intPtr(15), intPtr(7), nil), ""},
		{"lunar leap month", partial("lunar", intPtr(30), intPtr(4), boolPtr(true)), ""},
		{"solar day+month", partial("solar", intPtr(29), intPtr(2), nil), ""},
		{"lunar day 31", partial("lunar", intPtr(31), intPtr(1), nil), "death_day"},
		{"solar 31 April", partial("solar", intPtr(31), intPtr(4), nil), "death_day"},
		{"solar 30 February", partial("solar", intPtr(30), intPtr(2), nil), "death_day"},
		{"day 0", partial("lunar", intPtr(0), intPtr(1), nil), "death_day"},
		{"month 13", partial("lunar", intPtr(1), intPtr(13), nil), "death_month"},
		{"missing month", partial("lunar", intPtr(1), nil, nil), "death_month"},
		{"missing day", partial("lunar", nil, intPtr(1), nil), "death_day"},
		{"neither day nor month", partial("lunar", nil, nil, nil), "death_date"},
		{"solar with leap month", partial("solar", intPtr(1), intPtr(1), boolPtr(true)), "death_leap_month"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockMemorialRepository)
			uc := newMemorialUsecase(repo)

			if tt.field != "" {
				_, err := uc.Create(context.Background(), 7, tt.in)
				assertInvalidField(t, err, tt.field)
				return
			}

			var stored *domain.Memorial
			repo.On("Create", mock.Anything, mock.Anything, 100).Run(func(args mock.Arguments) {
				stored = args.Get(1).(*domain.Memorial)
			}).Return(&domain.Memorial{ID: 1}, nil)

			_, err := uc.Create(context.Background(), 7, tt.in)

			require.NoError(t, err)
			assert.Nil(t, stored.DeathDate, "year unknown: no solar date")
			assert.Equal(t, *tt.in.DeathDay, *stored.DeathDay)
			assert.Equal(t, *tt.in.DeathMonth, *stored.DeathMonth)
			require.NotNil(t, stored.DeathLeapMonth, "leap defaults to false, never nil, in this form")
			wantLeap := tt.in.DeathLeapMonth != nil && *tt.in.DeathLeapMonth
			assert.Equal(t, wantLeap, *stored.DeathLeapMonth)
		})
	}
}

func TestMemorialUsecase_Create_Success(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	in := validMemorialInput()
	in.FullName = "  Nguyễn Văn A  "
	in.Note = strPtr("  nhớ ơn  ")
	in.Reminder = &domain.ReminderInput{DaysBefore: intPtr(3), Time: "07:30"}
	in.AddToOSReminder = true

	var stored *domain.Memorial
	repo.On("Create", mock.Anything, mock.Anything, 100).Run(func(args mock.Arguments) {
		stored = args.Get(1).(*domain.Memorial)
	}).Return(&domain.Memorial{ID: 9, UserID: 7}, nil)

	got, err := newMemorialUsecase(repo).Create(context.Background(), 7, in)

	require.NoError(t, err)
	assert.Equal(t, int64(9), got.ID)
	assert.Equal(t, int64(7), stored.UserID, "owner comes from the argument, never the input")
	assert.Equal(t, "Nguyễn Văn A", stored.FullName, "name is trimmed")
	assert.Equal(t, "nhớ ơn", *stored.Note)
	assert.Equal(t, "2015-03-04", stored.DeathDate.Format(time.DateOnly))
	assert.Nil(t, stored.DeathDay)
	assert.Nil(t, stored.DeathLeapMonth, "year known: partial fields stay nil")
	assert.Equal(t, domain.Reminder{DaysBefore: 3, Time: "07:30", Timezone: "Asia/Ho_Chi_Minh"}, *stored.Reminder,
		"timezone defaults to Asia/Ho_Chi_Minh")
	assert.True(t, stored.AddToOSReminder)
}

func TestMemorialUsecase_Create_BlankNoteIsNil(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	in := validMemorialInput()
	in.Note = strPtr("   ")
	var stored *domain.Memorial
	repo.On("Create", mock.Anything, mock.Anything, 100).Run(func(args mock.Arguments) {
		stored = args.Get(1).(*domain.Memorial)
	}).Return(&domain.Memorial{}, nil)

	_, err := newMemorialUsecase(repo).Create(context.Background(), 7, in)

	require.NoError(t, err)
	assert.Nil(t, stored.Note)
}

func TestMemorialUsecase_Create_DeathDateTodayIsAllowed(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	repo.On("Create", mock.Anything, mock.Anything, 100).Return(&domain.Memorial{}, nil)
	in := validMemorialInput()
	// 2026-10-07 12:00 UTC: today in Vietnam is 2026-10-07 (19:00 local).
	in.DeathDate = "2026-10-07"

	_, err := newMemorialUsecase(repo).Create(context.Background(), 7, in)

	require.NoError(t, err)
}

func TestMemorialUsecase_Create_TodayFollowsVietnamZone(t *testing.T) {
	repo := new(domain.MockMemorialRepository)
	repo.On("Create", mock.Anything, mock.Anything, 100).Return(&domain.Memorial{}, nil)
	uc := newMemorialUsecase(repo)
	// 20:00 UTC on the 7th is 03:00 on the 8th in Vietnam, so the 8th is "today".
	uc.now = func() time.Time { return time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC) }
	in := validMemorialInput()
	in.DeathDate = "2026-10-08"

	_, err := uc.Create(context.Background(), 7, in)

	require.NoError(t, err)
}

func TestMemorialUsecase_Create_RepoErrors(t *testing.T) {
	t.Run("limit reached passes through", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		repo.On("Create", mock.Anything, mock.Anything, 100).Return(nil, domain.ErrMemorialLimitReached)

		_, err := newMemorialUsecase(repo).Create(context.Background(), 7, validMemorialInput())

		assert.ErrorIs(t, err, domain.ErrMemorialLimitReached)
	})
	t.Run("infrastructure error is wrapped", func(t *testing.T) {
		boom := errors.New("db down")
		repo := new(domain.MockMemorialRepository)
		repo.On("Create", mock.Anything, mock.Anything, 100).Return(nil, boom)

		_, err := newMemorialUsecase(repo).Create(context.Background(), 7, validMemorialInput())

		assert.ErrorIs(t, err, boom)
	})
}

func TestMemorialUsecase_List(t *testing.T) {
	t.Run("returns the user's items", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		repo.On("ListByUser", mock.Anything, int64(7)).Return([]domain.Memorial{{ID: 2}, {ID: 1}}, nil)

		got, err := newMemorialUsecase(repo).List(context.Background(), 7)

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, int64(2), got[0].ID)
	})
	t.Run("none is an empty slice, not nil", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		repo.On("ListByUser", mock.Anything, int64(7)).Return(nil, nil)

		got, err := newMemorialUsecase(repo).List(context.Background(), 7)

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
	t.Run("repository error", func(t *testing.T) {
		boom := errors.New("db down")
		repo := new(domain.MockMemorialRepository)
		repo.On("ListByUser", mock.Anything, int64(7)).Return(nil, boom)

		_, err := newMemorialUsecase(repo).List(context.Background(), 7)

		assert.ErrorIs(t, err, boom)
	})
}

func TestMemorialUsecase_Get(t *testing.T) {
	t.Run("found, looked up by id and owner", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		repo.On("FindByIDAndUser", mock.Anything, int64(5), int64(7)).Return(&domain.Memorial{ID: 5}, nil)

		got, err := newMemorialUsecase(repo).Get(context.Background(), 7, 5)

		require.NoError(t, err)
		assert.Equal(t, int64(5), got.ID)
	})
	t.Run("not found (also: someone else's)", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		repo.On("FindByIDAndUser", mock.Anything, int64(5), int64(8)).Return(nil, domain.ErrMemorialNotFound)

		_, err := newMemorialUsecase(repo).Get(context.Background(), 8, 5)

		assert.ErrorIs(t, err, domain.ErrMemorialNotFound)
	})
}

func TestMemorialUsecase_Update(t *testing.T) {
	t.Run("validation failure never reaches the repository", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		in := validMemorialInput()
		in.FullName = ""

		_, err := newMemorialUsecase(repo).Update(context.Background(), 7, 5, in)

		assertInvalidField(t, err, "full_name")
		repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})
	t.Run("sends id and owner from the arguments", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		var sent *domain.Memorial
		repo.On("Update", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			sent = args.Get(1).(*domain.Memorial)
		}).Return(&domain.Memorial{ID: 5}, nil)

		got, err := newMemorialUsecase(repo).Update(context.Background(), 7, 5, validMemorialInput())

		require.NoError(t, err)
		assert.Equal(t, int64(5), got.ID)
		assert.Equal(t, int64(5), sent.ID)
		assert.Equal(t, int64(7), sent.UserID)
	})
	t.Run("not found", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		repo.On("Update", mock.Anything, mock.Anything).Return(nil, domain.ErrMemorialNotFound)

		_, err := newMemorialUsecase(repo).Update(context.Background(), 8, 5, validMemorialInput())

		assert.ErrorIs(t, err, domain.ErrMemorialNotFound)
	})
}

func TestMemorialUsecase_Delete(t *testing.T) {
	t.Run("deleted", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		repo.On("Delete", mock.Anything, int64(5), int64(7)).Return(nil)

		require.NoError(t, newMemorialUsecase(repo).Delete(context.Background(), 7, 5))
	})
	t.Run("not found", func(t *testing.T) {
		repo := new(domain.MockMemorialRepository)
		repo.On("Delete", mock.Anything, int64(5), int64(8)).Return(domain.ErrMemorialNotFound)

		err := newMemorialUsecase(repo).Delete(context.Background(), 8, 5)

		assert.ErrorIs(t, err, domain.ErrMemorialNotFound)
	})
}
