package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// attestationRepository persists the attestation flow's state — single-use
// challenge nonces and registered iOS keys — on Postgres.
type attestationRepository struct {
	db database.Database
}

// NewAttestationRepository returns the Postgres-backed domain.AttestationRepository
// for the attestation feature. Every SQL statement and transaction boundary of
// challenge minting, key registration and counter advancement lives here.
func NewAttestationRepository(db database.Database) domain.AttestationRepository {
	return &attestationRepository{db: db}
}

// SaveChallenge mints a challenge row — AC1/AC3. expiresAt is computed by the
// use case from config.ChallengeTTL, never defaulted in SQL.
func (r *attestationRepository) SaveChallenge(ctx context.Context, challenge string, platform domain.Platform, purpose domain.ChallengePurpose, expiresAt time.Time) error {
	const query = `
		INSERT INTO attestation_challenges (challenge, platform, purpose, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := r.db.Exec(ctx, query, challenge, string(platform), string(purpose), expiresAt); err != nil {
		return fmt.Errorf("save challenge: %w", err)
	}
	return nil
}

// FindChallenge loads the live challenge for the given purpose — AC6/AC14. The
// row must be unused and unexpired; any of unknown / expired / already-used is
// the single sentinel ErrAttestationChallengeInvalid, so a caller cannot probe
// which state a nonce is in.
func (r *attestationRepository) FindChallenge(ctx context.Context, challenge string, purpose domain.ChallengePurpose) (*domain.AttestationChallenge, error) {
	const query = `
		SELECT id, challenge, platform, purpose, used_at, expires_at, created_at
		FROM attestation_challenges
		WHERE challenge = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
	`
	var (
		c      domain.AttestationChallenge
		usedAt sql.NullTime
	)
	if err := r.db.QueryRow(ctx, query, challenge, string(purpose)).Scan(
		&c.ID, &c.Challenge, &c.Platform, &c.Purpose, &usedAt, &c.ExpiresAt, &c.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAttestationChallengeInvalid
		}
		return nil, fmt.Errorf("find challenge: %w", err)
	}
	if usedAt.Valid {
		c.UsedAt = &usedAt.Time
	}
	return &c, nil
}

// MarkChallengeUsed consumes challenge with a single guarded UPDATE — AC6/AC7.
// The bool reports whether a row was actually marked: false means a concurrent
// request already consumed it (the nonce is one-time).
func (r *attestationRepository) MarkChallengeUsed(ctx context.Context, challenge string) (bool, error) {
	const query = `UPDATE attestation_challenges SET used_at = now() WHERE challenge = $1 AND used_at IS NULL`
	res, err := r.db.Exec(ctx, query, challenge)
	if err != nil {
		return false, fmt.Errorf("mark challenge used: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("mark challenge used rows: %w", err)
	}
	return n > 0, nil
}

// SaveKey upserts the iOS key by key_id and consumes challenge, in one
// transaction — AC8/AC10. Atomicity is the point: a key must never be persisted
// against a still-unused challenge, or the same nonce could be replayed to mint
// a second key.
//
// The upsert updates public_key/receipt/environment/updated_at but NOT
// sign_counter: re-attestation is an idempotent retry of the same registration,
// and the replay guard must not reset (design.md iOS flow Step 4).
func (r *attestationRepository) SaveKey(ctx context.Context, key *domain.AttestationKey, challenge string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const upsertKey = `
		INSERT INTO attestation_keys (key_id, platform, public_key, receipt, sign_counter, environment)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (key_id) DO UPDATE SET
			public_key = EXCLUDED.public_key,
			receipt = EXCLUDED.receipt,
			environment = EXCLUDED.environment,
			updated_at = now()
	`
	if _, err := tx.ExecContext(ctx, upsertKey,
		key.KeyID,
		string(key.Platform),
		key.PublicKey,
		key.Receipt,
		key.SignCounter,
		key.Environment,
	); err != nil {
		return fmt.Errorf("upsert attestation key: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE attestation_challenges SET used_at = now() WHERE challenge = $1 AND used_at IS NULL`, challenge); err != nil {
		return fmt.Errorf("consume challenge: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save key: %w", err)
	}
	return nil
}

// FindKey loads the registered key for keyID — AC13. No row is
// ErrAttestationKeyNotFound.
func (r *attestationRepository) FindKey(ctx context.Context, keyID string) (*domain.AttestationKey, error) {
	const query = `
		SELECT id, key_id, platform, public_key, receipt, sign_counter, environment, user_id, created_at, updated_at
		FROM attestation_keys
		WHERE key_id = $1
	`
	var (
		k      domain.AttestationKey
		userID sql.NullInt64
	)
	if err := r.db.QueryRow(ctx, query, keyID).Scan(
		&k.ID, &k.KeyID, &k.Platform, &k.PublicKey, &k.Receipt, &k.SignCounter, &k.Environment, &userID, &k.CreatedAt, &k.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAttestationKeyNotFound
		}
		return nil, fmt.Errorf("find attestation key: %w", err)
	}
	if userID.Valid {
		k.UserID = &userID.Int64
	}
	return &k, nil
}

// UpdateKeyCounter advances the key's stored counter to newCounter and consumes
// challenge, in one transaction — AC11/AC12. Atomicity keeps the counter from
// being advanced against a still-unused challenge.
func (r *attestationRepository) UpdateKeyCounter(ctx context.Context, keyID string, newCounter int64, challenge string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `UPDATE attestation_keys SET sign_counter = $2, updated_at = now() WHERE key_id = $1`, keyID, newCounter); err != nil {
		return fmt.Errorf("update key counter: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE attestation_challenges SET used_at = now() WHERE challenge = $1 AND used_at IS NULL`, challenge); err != nil {
		return fmt.Errorf("consume challenge: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update key counter: %w", err)
	}
	return nil
}
