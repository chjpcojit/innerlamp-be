package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newTempleSuggestionRepo(t *testing.T) (domain.TempleSuggestionRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	return NewTempleSuggestionRepository(db), db
}

func TestTempleSuggestionRepository_Create_StoresPending(t *testing.T) {
	repo, db := newTempleSuggestionRepo(t)
	ctx := context.Background()

	got, err := repo.Create(ctx, &domain.TempleSuggestion{
		Name: "Chùa Mới", ProvinceCode: "79", Province: "Hồ Chí Minh",
		WardCode: "26737", Ward: "Phường Tân Định", Address: "12 Nguyễn Huệ",
		Status: domain.TempleSuggestionCreated, // ignored: a new row is always pending
	})
	require.NoError(t, err)
	assert.NotZero(t, got.ID)
	assert.Equal(t, domain.TempleSuggestionPending, got.Status)
	assert.False(t, got.CreatedAt.IsZero())

	assert.Equal(t, "79", got.ProvinceCode)
	assert.Equal(t, "Phường Tân Định", got.Ward)

	var name, provinceCode, province, wardCode, ward, address, status string
	var reviewedBy, templeID, reason *string
	require.NoError(t, db.QueryRow(ctx,
		`SELECT name, province_code, province, ward_code, ward, address, status, reviewed_by::text, temple_id, reject_reason
		 FROM temple_suggestions WHERE id = $1`, got.ID,
	).Scan(&name, &provinceCode, &province, &wardCode, &ward, &address, &status, &reviewedBy, &templeID, &reason))
	assert.Equal(t, "Chùa Mới", name)
	assert.Equal(t, "79", provinceCode)
	assert.Equal(t, "Hồ Chí Minh", province)
	assert.Equal(t, "26737", wardCode)
	assert.Equal(t, "Phường Tân Định", ward)
	assert.Equal(t, "12 Nguyễn Huệ", address)
	assert.Equal(t, "pending", status)
	assert.Nil(t, reviewedBy)
	assert.Nil(t, templeID)
	assert.Nil(t, reason)
}

func TestTempleSuggestionRepository_Create_DuplicatesAreSeparateRows(t *testing.T) {
	repo, _ := newTempleSuggestionRepo(t)
	ctx := context.Background()
	in := &domain.TempleSuggestion{
		Name: "Chùa Trùng", ProvinceCode: "46", Province: "Huế", WardCode: "19000", Ward: "Phường Phú Xuân", Address: "Đường 1",
	}

	a, err := repo.Create(ctx, in)
	require.NoError(t, err)
	b, err := repo.Create(ctx, in)
	require.NoError(t, err)

	assert.NotEqual(t, a.ID, b.ID)
}

func TestTempleSuggestionRepository_ReviewCheckRejectsHalfReviewedRows(t *testing.T) {
	repo, db := newTempleSuggestionRepo(t)
	ctx := context.Background()
	s, err := repo.Create(ctx, &domain.TempleSuggestion{
		Name: "N", ProvinceCode: "01", Province: "P", WardCode: "00004", Ward: "W", Address: "A",
	})
	require.NoError(t, err)

	// created without a temple / reviewer violates chk_temple_suggestions_review.
	_, err = db.Exec(ctx, `UPDATE temple_suggestions SET status = 'created' WHERE id = $1`, s.ID)
	assert.Error(t, err)
	// an unknown status violates chk_temple_suggestions_status.
	_, err = db.Exec(ctx, `UPDATE temple_suggestions SET status = 'approved' WHERE id = $1`, s.ID)
	assert.Error(t, err)
}

func TestTempleSuggestionRepository_AddressCodesCheck(t *testing.T) {
	repo, db := newTempleSuggestionRepo(t)
	ctx := context.Background()
	s, err := repo.Create(ctx, &domain.TempleSuggestion{
		Name: "N", ProvinceCode: "01", Province: "Hà Nội", WardCode: "00004", Ward: "Phường Ba Đình", Address: "A",
	})
	require.NoError(t, err)

	// A half-filled address (a code without the other code or the ward name)
	// violates chk_temple_suggestions_address_codes.
	_, err = db.Exec(ctx, `UPDATE temple_suggestions SET ward_code = NULL WHERE id = $1`, s.ID)
	assert.Error(t, err)
	_, err = db.Exec(ctx, `UPDATE temple_suggestions SET ward = NULL WHERE id = $1`, s.ID)
	assert.Error(t, err)

	// A suggestion submitted before the codes existed (all three NULL, province
	// as typed) is still a valid row.
	_, err = db.Exec(ctx, `INSERT INTO temple_suggestions (name, province, address) VALUES ('Cũ', 'Quảng Ninh', 'Thôn 1')`)
	assert.NoError(t, err)
}
