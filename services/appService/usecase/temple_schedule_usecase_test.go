package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func TestScheduleForTemple(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("db down")

	t.Run("unknown temple returns ErrTempleNotFound and reads no schedule", func(t *testing.T) {
		temples := new(domain.MockTempleRepository)
		schedules := new(domain.MockTempleScheduleRepository)
		temples.On("FindByID", mock.Anything, "t1").Return(nil, domain.ErrTempleNotFound)

		_, err := NewTempleScheduleUsecase(temples, schedules).ScheduleForTemple(ctx, "t1")

		assert.ErrorIs(t, err, domain.ErrTempleNotFound)
		schedules.AssertNotCalled(t, "ListActiveByTemple", mock.Anything, mock.Anything)
	})

	t.Run("temple lookup failure is wrapped", func(t *testing.T) {
		temples := new(domain.MockTempleRepository)
		temples.On("FindByID", mock.Anything, "t1").Return(nil, dbErr)

		_, err := NewTempleScheduleUsecase(temples, new(domain.MockTempleScheduleRepository)).ScheduleForTemple(ctx, "t1")

		assert.ErrorIs(t, err, dbErr)
		assert.NotErrorIs(t, err, domain.ErrTempleNotFound)
	})

	t.Run("temple with no item returns empty non-nil slice", func(t *testing.T) {
		temples := new(domain.MockTempleRepository)
		schedules := new(domain.MockTempleScheduleRepository)
		temples.On("FindByID", mock.Anything, "t1").Return(&domain.Temple{ID: "t1"}, nil)
		schedules.On("ListActiveByTemple", mock.Anything, "t1").Return([]domain.TempleScheduleItem{}, nil)

		got, err := NewTempleScheduleUsecase(temples, schedules).ScheduleForTemple(ctx, "t1")

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("schedule read failure is wrapped", func(t *testing.T) {
		temples := new(domain.MockTempleRepository)
		schedules := new(domain.MockTempleScheduleRepository)
		temples.On("FindByID", mock.Anything, "t1").Return(&domain.Temple{ID: "t1"}, nil)
		schedules.On("ListActiveByTemple", mock.Anything, "t1").Return(nil, dbErr)

		_, err := NewTempleScheduleUsecase(temples, schedules).ScheduleForTemple(ctx, "t1")

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("result is sorted and normalised", func(t *testing.T) {
		temples := new(domain.MockTempleRepository)
		schedules := new(domain.MockTempleScheduleRepository)
		temples.On("FindByID", mock.Anything, "t1").Return(&domain.Temple{ID: "t1"}, nil)
		schedules.On("ListActiveByTemple", mock.Anything, "t1").Return([]domain.TempleScheduleItem{
			{ID: "late", StartTime: 19 * 60, Weekdays: []int{0, 1, 2, 3, 4, 5, 6}},
			{ID: "early", StartTime: 4 * 60, Weekdays: []int{6, 0, 6}, Steps: []domain.TempleScheduleStep{
				{Position: 2, Text: "b"}, {Position: 1, Text: "a"},
			}},
		}, nil)

		got, err := NewTempleScheduleUsecase(temples, schedules).ScheduleForTemple(ctx, "t1")

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "early", got[0].ID)
		assert.Equal(t, []int{0, 6}, got[0].Weekdays)
		assert.Equal(t, "a", got[0].Steps[0].Text)
		assert.Equal(t, "late", got[1].ID)
		assert.Equal(t, []int{}, got[1].Weekdays)
	})
}
