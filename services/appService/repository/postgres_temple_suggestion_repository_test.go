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
		Name: "Chùa Mới", Province: "Quảng Ninh", Address: "Thôn 1, xã A",
		Status: domain.TempleSuggestionCreated, // ignored: a new row is always pending
	})
	require.NoError(t, err)
	assert.NotZero(t, got.ID)
	assert.Equal(t, domain.TempleSuggestionPending, got.Status)
	assert.False(t, got.CreatedAt.IsZero())

	var name, province, address, status string
	var reviewedBy, templeID, reason *string
	require.NoError(t, db.QueryRow(ctx,
		`SELECT name, province, address, status, reviewed_by::text, temple_id, reject_reason FROM temple_suggestions WHERE id = $1`, got.ID,
	).Scan(&name, &province, &address, &status, &reviewedBy, &templeID, &reason))
	assert.Equal(t, "Chùa Mới", name)
	assert.Equal(t, "Quảng Ninh", province)
	assert.Equal(t, "Thôn 1, xã A", address)
	assert.Equal(t, "pending", status)
	assert.Nil(t, reviewedBy)
	assert.Nil(t, templeID)
	assert.Nil(t, reason)
}

func TestTempleSuggestionRepository_Create_DuplicatesAreSeparateRows(t *testing.T) {
	repo, _ := newTempleSuggestionRepo(t)
	ctx := context.Background()
	in := &domain.TempleSuggestion{Name: "Chùa Trùng", Province: "Huế", Address: "Đường 1"}

	a, err := repo.Create(ctx, in)
	require.NoError(t, err)
	b, err := repo.Create(ctx, in)
	require.NoError(t, err)

	assert.NotEqual(t, a.ID, b.ID)
}

func TestTempleSuggestionRepository_ReviewCheckRejectsHalfReviewedRows(t *testing.T) {
	repo, db := newTempleSuggestionRepo(t)
	ctx := context.Background()
	s, err := repo.Create(ctx, &domain.TempleSuggestion{Name: "N", Province: "P", Address: "A"})
	require.NoError(t, err)

	// created without a temple / reviewer violates chk_temple_suggestions_review.
	_, err = db.Exec(ctx, `UPDATE temple_suggestions SET status = 'created' WHERE id = $1`, s.ID)
	assert.Error(t, err)
	// an unknown status violates chk_temple_suggestions_status.
	_, err = db.Exec(ctx, `UPDATE temple_suggestions SET status = 'approved' WHERE id = $1`, s.ID)
	assert.Error(t, err)
}
