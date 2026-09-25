package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newAttestationRepo opens the real Postgres test DB and returns a repository
// over it. The schema is bootstrapped and truncated by connectTestDB.
func newAttestationRepo(t *testing.T) (domain.AttestationRepository, database.Database) {
	t.Helper()
	db := connectTestDB(t)
	return NewAttestationRepository(db), db
}

func saveChallenge(t *testing.T, repo domain.AttestationRepository, challenge string, platform domain.Platform, purpose domain.ChallengePurpose) {
	t.Helper()
	require.NoError(t, repo.SaveChallenge(context.Background(), challenge, platform, purpose, time.Now().Add(time.Hour)))
}

func TestAttestationRepository_SaveAndFindChallenge(t *testing.T) {
	repo, _ := newAttestationRepo(t)
	ctx := context.Background()

	saveChallenge(t, repo, "nonce-attest", domain.PlatformAndroid, domain.ChallengePurposeAttest)
	saveChallenge(t, repo, "nonce-assert", domain.PlatformIOS, domain.ChallengePurposeAssert)

	// Found with the right purpose.
	c, err := repo.FindChallenge(ctx, "nonce-attest", domain.ChallengePurposeAttest)
	require.NoError(t, err)
	require.Equal(t, domain.PlatformAndroid, c.Platform)
	require.Equal(t, domain.ChallengePurposeAttest, c.Purpose)

	// The same nonce under the wrong purpose is invalid (purpose filtering).
	_, err = repo.FindChallenge(ctx, "nonce-attest", domain.ChallengePurposeAssert)
	require.ErrorIs(t, err, domain.ErrAttestationChallengeInvalid)
}

func TestAttestationRepository_FindChallenge_ExpiredAndUsed(t *testing.T) {
	repo, _ := newAttestationRepo(t)
	ctx := context.Background()

	// Expired: expires_at is in the past.
	require.NoError(t, repo.SaveChallenge(ctx, "expired", domain.PlatformAndroid, domain.ChallengePurposeAttest, time.Now().Add(-time.Minute)))
	_, err := repo.FindChallenge(ctx, "expired", domain.ChallengePurposeAttest)
	require.ErrorIs(t, err, domain.ErrAttestationChallengeInvalid)

	// Used: MarkChallengeUsed then FindChallenge fails.
	saveChallenge(t, repo, "used", domain.PlatformAndroid, domain.ChallengePurposeAttest)
	marked, err := repo.MarkChallengeUsed(ctx, "used")
	require.NoError(t, err)
	require.True(t, marked)
	_, err = repo.FindChallenge(ctx, "used", domain.ChallengePurposeAttest)
	require.ErrorIs(t, err, domain.ErrAttestationChallengeInvalid)
}

func TestAttestationRepository_MarkChallengeUsed(t *testing.T) {
	repo, _ := newAttestationRepo(t)
	ctx := context.Background()

	saveChallenge(t, repo, "race", domain.PlatformAndroid, domain.ChallengePurposeAttest)

	// First mark wins.
	marked, err := repo.MarkChallengeUsed(ctx, "race")
	require.NoError(t, err)
	require.True(t, marked)

	// Second mark affects no rows (the nonce is one-time).
	marked, err = repo.MarkChallengeUsed(ctx, "race")
	require.NoError(t, err)
	require.False(t, marked)

	// Unknown challenge also affects no rows.
	marked, err = repo.MarkChallengeUsed(ctx, "does-not-exist")
	require.NoError(t, err)
	require.False(t, marked)
}

func TestAttestationRepository_SaveKey_UpsertIsIdempotent(t *testing.T) {
	repo, _ := newAttestationRepo(t)
	ctx := context.Background()

	saveChallenge(t, repo, "ch1", domain.PlatformIOS, domain.ChallengePurposeAttest)
	key := &domain.AttestationKey{
		KeyID:       "key-1",
		Platform:    domain.PlatformIOS,
		PublicKey:   []byte("pk-v1"),
		Receipt:     []byte("rcpt"),
		SignCounter: 0,
		Environment: "production",
	}
	require.NoError(t, repo.SaveKey(ctx, key, "ch1"))

	// Re-registering the same key (idempotent retry, AC10) updates the row
	// rather than erroring.
	require.NoError(t, repo.SaveKey(ctx, &domain.AttestationKey{
		KeyID:       "key-1",
		Platform:    domain.PlatformIOS,
		PublicKey:   []byte("pk-v2"),
		Receipt:     []byte("rcpt-v2"),
		SignCounter: 0,
		Environment: "development",
	}, "ch1"))

	got, err := repo.FindKey(ctx, "key-1")
	require.NoError(t, err)
	require.Equal(t, []byte("pk-v2"), got.PublicKey)
	require.Equal(t, []byte("rcpt-v2"), got.Receipt)
	require.Equal(t, "development", got.Environment)
}

func TestAttestationRepository_FindKey_NotFound(t *testing.T) {
	repo, _ := newAttestationRepo(t)
	_, err := repo.FindKey(context.Background(), "no-such-key")
	require.ErrorIs(t, err, domain.ErrAttestationKeyNotFound)
}

func TestAttestationRepository_UpdateKeyCounter(t *testing.T) {
	repo, _ := newAttestationRepo(t)
	ctx := context.Background()

	saveChallenge(t, repo, "ch1", domain.PlatformIOS, domain.ChallengePurposeAssert)
	require.NoError(t, repo.SaveKey(ctx, &domain.AttestationKey{
		KeyID:       "key-1",
		Platform:    domain.PlatformIOS,
		PublicKey:   []byte("pk"),
		SignCounter: 5,
		Environment: "production",
	}, "ch1"))

	require.NoError(t, repo.UpdateKeyCounter(ctx, "key-1", 6, "ch1"))

	got, err := repo.FindKey(ctx, "key-1")
	require.NoError(t, err)
	require.Equal(t, int64(6), got.SignCounter)

	// The challenge consumed by the counter update is no longer findable.
	_, err = repo.FindChallenge(ctx, "ch1", domain.ChallengePurposeAssert)
	require.ErrorIs(t, err, domain.ErrAttestationChallengeInvalid)
}
