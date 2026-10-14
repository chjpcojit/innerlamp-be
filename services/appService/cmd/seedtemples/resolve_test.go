package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// fakeUnits is a hand-written resolver over a tiny dataset: province 79 has ward 26737.
type fakeUnits struct {
	provinces []domain.CurrentProvince
	listErr   error
	wardErr   error
}

func (f fakeUnits) ListCurrentProvinces(context.Context) ([]domain.CurrentProvince, error) {
	return f.provinces, f.listErr
}

func (f fakeUnits) ResolveCurrentWard(_ context.Context, p, w string) (*domain.ResolvedWard, error) {
	if f.wardErr != nil {
		return nil, f.wardErr
	}
	switch {
	case p != "79":
		return nil, domain.ErrAdministrativeUnitNotFound
	case w != "26737":
		return nil, domain.ErrWardNotInProvince
	}
	return &domain.ResolvedWard{ProvinceName: "Hồ Chí Minh", WardFullName: "Phường Tân Định"}, nil
}

var loaded = []domain.CurrentProvince{{Code: "79", Name: "Hồ Chí Minh"}}

func temple(prov, ward, declaredProvince string) []domain.Temple {
	return []domain.Temple{{ID: "temple_south_002", ProvinceCode: prov, WardCode: ward, Province: declaredProvince, Address: "73 Mai Thị Lựu"}}
}

func TestResolveAddresses(t *testing.T) {
	ctx := context.Background()

	t.Run("fills Province and Ward from the dataset (AC 35)", func(t *testing.T) {
		ts := temple("79", "26737", "")
		require.NoError(t, resolveAddresses(ctx, fakeUnits{provinces: loaded}, ts))
		assert.Equal(t, "Hồ Chí Minh", ts[0].Province)
		assert.Equal(t, "Phường Tân Định", ts[0].Ward)
		assert.Equal(t, "73 Mai Thị Lựu", ts[0].Address, "the below-ward address is untouched")
	})

	t.Run("a declared province that matches the dataset is accepted", func(t *testing.T) {
		ts := temple("79", "26737", "Hồ Chí Minh")
		require.NoError(t, resolveAddresses(ctx, fakeUnits{provinces: loaded}, ts))
	})

	t.Run("a declared province that differs is rejected, naming the temple", func(t *testing.T) {
		err := resolveAddresses(ctx, fakeUnits{provinces: loaded}, temple("79", "26737", "Hà Nội"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "temple_south_002")
		assert.Contains(t, err.Error(), "location.province")
	})

	t.Run("unknown province code is rejected", func(t *testing.T) {
		err := resolveAddresses(ctx, fakeUnits{provinces: loaded}, temple("99", "26737", ""))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "province_code")
	})

	t.Run("a ward of another province, or unknown, is rejected", func(t *testing.T) {
		err := resolveAddresses(ctx, fakeUnits{provinces: loaded}, temple("79", "00001", ""))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ward_code")
	})

	t.Run("an empty dataset names seedadminunits and writes nothing (AC 36)", func(t *testing.T) {
		ts := temple("79", "26737", "")
		err := resolveAddresses(ctx, fakeUnits{provinces: nil}, ts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "seedadminunits")
		assert.Empty(t, ts[0].Ward, "nothing was filled in")
	})

	t.Run("a database error is returned, not swallowed", func(t *testing.T) {
		boom := errors.New("connection reset")
		require.ErrorIs(t, resolveAddresses(ctx, fakeUnits{provinces: loaded, wardErr: boom}, temple("79", "26737", "")), boom)
		require.ErrorIs(t, resolveAddresses(ctx, fakeUnits{listErr: boom}, temple("79", "26737", "")), boom)
	})
}
