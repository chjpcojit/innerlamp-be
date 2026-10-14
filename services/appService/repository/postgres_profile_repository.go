package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// profileSelect is the one read shape of a profile: users with the chosen temple's
// name joined in. Explicit columns, never SELECT *.
const profileSelect = `
	u.id, u.email, u.email_verified_at IS NOT NULL, u.phone_number, u.full_name, u.dharma_name,
	u.ordination_temple_id, t.name, u.avatar_object`

type profileRepository struct {
	db database.Database
}

// NewProfileRepository returns the Postgres-backed domain.ProfileRepository.
func NewProfileRepository(db database.Database) domain.ProfileRepository {
	return &profileRepository{db: db}
}

func scanProfile(row rowScanner) (*domain.Profile, error) {
	var (
		p                  domain.Profile
		fullName           sql.NullString
		dharmaName         sql.NullString
		templeID, templeNm sql.NullString
		avatar             sql.NullString
	)
	if err := row.Scan(&p.UserID, &p.Email, &p.EmailVerified, &p.PhoneNumber, &fullName, &dharmaName,
		&templeID, &templeNm, &avatar); err != nil {
		return nil, err
	}
	if fullName.Valid {
		p.FullName = &fullName.String
	}
	if dharmaName.Valid {
		p.DharmaName = &dharmaName.String
	}
	if templeID.Valid {
		p.OrdinationTemple = &domain.TempleRef{ID: templeID.String, Name: templeNm.String}
	}
	if avatar.Valid {
		p.AvatarObject = &avatar.String
	}
	return &p, nil
}

// Get returns the profile of userID, or domain.ErrUserNotFound.
func (r *profileRepository) Get(ctx context.Context, userID int64) (*domain.Profile, error) {
	row := r.db.QueryRow(ctx, `SELECT`+profileSelect+`
		FROM users u LEFT JOIN temples t ON t.id = u.ordination_temple_id
		WHERE u.id = $1`, userID)
	p, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get profile %d: %w", userID, err)
	}
	return p, nil
}

// Update sets only the columns patch carries, in ONE statement, and returns the
// row as written. A cleared field is SQL NULL, never ”. All columns of a request
// change together or not at all: a single UPDATE is atomic, so a rejected temple
// also leaves the dharma name alone (AC14).
//
// An empty patch writes nothing — not even updated_at — and just reads.
func (r *profileRepository) Update(ctx context.Context, userID int64, patch domain.ProfilePatch) (*domain.Profile, error) {
	if patch.IsEmpty() {
		return r.Get(ctx, userID)
	}

	// Column names are fixed literals here; only values are parameters.
	args := []any{userID}
	sets := []string{"updated_at = now()"}
	add := func(column string, v domain.OptionalString) {
		if !v.Set {
			return
		}
		args = append(args, nullableStringPtr(v.Value))
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	add("full_name", patch.FullName)
	add("dharma_name", patch.DharmaName)
	add("ordination_temple_id", patch.OrdinationTempleID)

	row := r.db.QueryRow(ctx, `
		WITH u AS (
			UPDATE users SET `+strings.Join(sets, ", ")+`
			WHERE id = $1
			RETURNING id, email, email_verified_at, phone_number, full_name, dharma_name, ordination_temple_id, avatar_object
		)
		SELECT`+profileSelect+`
		FROM u LEFT JOIN temples t ON t.id = u.ordination_temple_id`, args...)
	p, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		// A temple deleted between the use case's TempleExists and this write.
		if isForeignKeyViolation(err, usersOrdinationTempleFKConstraint) {
			return nil, domain.ErrInvalidTemple
		}
		return nil, fmt.Errorf("update profile %d: %w", userID, err)
	}
	return p, nil
}

// SwapAvatar points the user at newObject and returns the key it replaced. The
// row is locked for the read, so two concurrent swaps are serialised and each is
// handed the key it really replaced — otherwise both could read the same old key
// and one image would be deleted twice while another leaked.
func (r *profileRepository) SwapAvatar(ctx context.Context, userID int64, newObject string) (*string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	old, err := lockAvatar(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET avatar_object = $2, updated_at = now() WHERE id = $1`, userID, newObject); err != nil {
		return nil, fmt.Errorf("set avatar %d: %w", userID, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return old, nil
}

// ClearAvatar removes the avatar key and returns it. Nothing is written when
// there was none.
func (r *profileRepository) ClearAvatar(ctx context.Context, userID int64) (*string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	old, err := lockAvatar(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return nil, nil
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET avatar_object = NULL, updated_at = now() WHERE id = $1`, userID); err != nil {
		return nil, fmt.Errorf("clear avatar %d: %w", userID, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return old, nil
}

// lockAvatar reads the user's current avatar key while holding the row lock for
// the rest of tx. domain.ErrUserNotFound if there is no such user.
func lockAvatar(ctx context.Context, tx *sql.Tx, userID int64) (*string, error) {
	var cur sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT avatar_object FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&cur)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock user %d: %w", userID, err)
	}
	if !cur.Valid {
		return nil, nil
	}
	return &cur.String, nil
}

// TempleExists reports whether temples has a row with this id.
func (r *profileRepository) TempleExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM temples WHERE id = $1)`, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check temple exists: %w", err)
	}
	return exists, nil
}
