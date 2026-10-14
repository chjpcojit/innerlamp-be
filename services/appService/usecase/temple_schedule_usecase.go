package usecase

import (
	"context"
	"fmt"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// TempleScheduleUsecase answers "what is this temple's standing practice
// schedule". The schedule is plain rows (no dates, no occurrences), so the use
// case only checks the temple exists, reads, and puts the result in order.
type TempleScheduleUsecase struct {
	temples   domain.TempleRepository
	schedules domain.TempleScheduleRepository
}

// NewTempleScheduleUsecase wires the use case.
func NewTempleScheduleUsecase(temples domain.TempleRepository, schedules domain.TempleScheduleRepository) *TempleScheduleUsecase {
	return &TempleScheduleUsecase{temples: temples, schedules: schedules}
}

// ScheduleForTemple returns the active schedule items of a temple, sorted and
// normalised (domain.SortTempleSchedule). An unknown temple is
// domain.ErrTempleNotFound; a temple with no item returns an empty non-nil
// slice.
func (u *TempleScheduleUsecase) ScheduleForTemple(ctx context.Context, templeID string) ([]domain.TempleScheduleItem, error) {
	if _, err := u.temples.FindByID(ctx, templeID); err != nil {
		return nil, fmt.Errorf("find temple %q: %w", templeID, err)
	}
	items, err := u.schedules.ListActiveByTemple(ctx, templeID)
	if err != nil {
		return nil, fmt.Errorf("list schedule of temple %q: %w", templeID, err)
	}
	if items == nil {
		items = []domain.TempleScheduleItem{}
	}
	return domain.SortTempleSchedule(items), nil
}
