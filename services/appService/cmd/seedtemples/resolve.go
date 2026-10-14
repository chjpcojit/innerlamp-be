package main

import (
	"context"
	"errors"
	"fmt"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// unitResolver is the slice of domain.AdministrativeUnitRepository the seeder
// needs: the dataset must be loaded, and a (province, ward) pair must resolve.
type unitResolver interface {
	ListCurrentProvinces(ctx context.Context) ([]domain.CurrentProvince, error)
	ResolveCurrentWard(ctx context.Context, provinceCode, wardCode string) (*domain.ResolvedWard, error)
}

// resolveAddresses is Step 4 of the seed flow (spec temple v1.7, AC 35–36). For
// every temple it checks ProvinceCode/WardCode against the administrative-units
// dataset and sets Province (short name) and Ward (full name) FROM THE DATASET —
// the file never supplies them. A temple whose file also names a province that
// differs from the dataset's name for its code is rejected (typo guard). It only
// reads, and returns on the first problem so nothing is written for a bad file.
func resolveAddresses(ctx context.Context, r unitResolver, temples []domain.Temple) error {
	provinces, err := r.ListCurrentProvinces(ctx)
	if err != nil {
		return fmt.Errorf("list current provinces: %w", err)
	}
	if len(provinces) == 0 {
		return errors.New("the administrative-units dataset is empty: run `go run ./services/appService/cmd/seedadminunits` before seedtemples")
	}
	for i := range temples {
		t := &temples[i]
		w, err := r.ResolveCurrentWard(ctx, t.ProvinceCode, t.WardCode)
		switch {
		case errors.Is(err, domain.ErrAdministrativeUnitNotFound):
			return fmt.Errorf("temple %q: province_code %q is not a current-scheme province", t.ID, t.ProvinceCode)
		case errors.Is(err, domain.ErrWardNotInProvince):
			return fmt.Errorf("temple %q: ward_code %q is not a ward of province_code %q", t.ID, t.WardCode, t.ProvinceCode)
		case err != nil:
			return fmt.Errorf("temple %q: resolve address: %w", t.ID, err)
		}
		if t.Province != "" && t.Province != w.ProvinceName {
			return fmt.Errorf("temple %q: location.province %q does not match %q, the dataset's name for province_code %q",
				t.ID, t.Province, w.ProvinceName, t.ProvinceCode)
		}
		t.Province, t.Ward = w.ProvinceName, w.WardFullName
	}
	return nil
}
