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

// --- viewers ----------------------------------------------------------------

type genealogyViewerRepository struct {
	db database.Database
}

// NewGenealogyViewerRepository returns the Postgres-backed domain.ViewerRepository.
func NewGenealogyViewerRepository(db database.Database) domain.ViewerRepository {
	return &genealogyViewerRepository{db: db}
}

// List returns the viewers with the member each one stands at: the member whose
// phone number equals the viewer's account number (AC48, AC59).
func (r *genealogyViewerRepository) List(ctx context.Context, genealogyID int64) ([]domain.Viewer, error) {
	rows, err := r.db.Query(ctx, `
		SELECT v.user_id, v.granted_by, v.created_at, u.phone_number, m.id
		FROM genealogy_viewers v
		JOIN users u ON u.id = v.user_id
		LEFT JOIN genealogy_members m ON m.genealogy_id = v.genealogy_id AND m.phone_number = u.phone_number
		WHERE v.genealogy_id = $1
		ORDER BY v.created_at, v.user_id`, genealogyID)
	if err != nil {
		return nil, fmt.Errorf("query viewers: %w", err)
	}
	return scanRows(rows, "viewer", func(s rowScanner) (domain.Viewer, error) {
		v := domain.Viewer{GenealogyID: genealogyID}
		var member sql.NullInt64
		if err := s.Scan(&v.UserID, &v.GrantedBy, &v.CreatedAt, &v.PhoneNumber, &member); err != nil {
			return v, err
		}
		v.MemberID = int64Ptr(member)
		return v, nil
	})
}

// Remove deletes one viewer grant (AC49, AC50).
func (r *genealogyViewerRepository) Remove(ctx context.Context, genealogyID, userID int64) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM genealogy_viewers WHERE genealogy_id = $1 AND user_id = $2`, genealogyID, userID)
	if err != nil {
		return fmt.Errorf("remove viewer: %w", err)
	}
	return expectOneRow(res, domain.ErrViewerNotFound)
}

// --- invitations ------------------------------------------------------------

type genealogyInvitationRepository struct {
	db database.Database
}

// NewGenealogyInvitationRepository returns the Postgres-backed
// domain.InvitationRepository.
func NewGenealogyInvitationRepository(db database.Database) domain.InvitationRepository {
	return &genealogyInvitationRepository{db: db}
}

// invitationSelect reads an invitation with the names the screens need. A
// pending invitation past its expiry is reported as expired, so the status is
// right without any cleanup job (AC75).
const invitationSelect = `
	SELECT i.id, i.genealogy_id, i.member_id, i.phone_number, i.invited_by,
	       CASE WHEN i.status = 'pending' AND i.expires_at <= now() THEN 'expired' ELSE i.status END,
	       i.created_at, i.expires_at, i.responded_at,
	       m.full_name, g.name, coalesce(u.phone_number, '')
	FROM genealogy_invitations i
	JOIN genealogy_members m ON m.id = i.member_id AND m.genealogy_id = i.genealogy_id
	JOIN genealogies g ON g.id = i.genealogy_id
	LEFT JOIN users u ON u.id = i.invited_by`

func scanInvitation(s rowScanner) (domain.Invitation, error) {
	var (
		inv       domain.Invitation
		status    string
		responded sql.NullTime
	)
	if err := s.Scan(&inv.ID, &inv.GenealogyID, &inv.MemberID, &inv.PhoneNumber, &inv.InvitedBy, &status,
		&inv.CreatedAt, &inv.ExpiresAt, &responded, &inv.MemberFullName, &inv.GenealogyName, &inv.InvitedByPhone); err != nil {
		return inv, err
	}
	inv.Status = domain.InvitationStatus(status)
	if responded.Valid {
		t := responded.Time
		inv.RespondedAt = &t
	}
	return inv, nil
}

// Create is Step C-2. Everything runs under the tree's row lock so the viewer
// cap, the one-number-one-member rule and the one-pending-invitation rule are
// decided against a stable tree.
func (r *genealogyInvitationRepository) Create(ctx context.Context, genealogyID, invitedBy int64, in domain.InviteInput, maxViewers int, ttl time.Duration) (*domain.Invitation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockGenealogy(ctx, tx, genealogyID); err != nil {
		return nil, err
	}

	var (
		alive       bool
		memberPhone sql.NullString
	)
	err = tx.QueryRowContext(ctx,
		`SELECT is_alive, phone_number FROM genealogy_members WHERE id = $1 AND genealogy_id = $2`,
		in.MemberID, genealogyID).Scan(&alive, &memberPhone)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read member: %w", err)
	}
	if !alive {
		return nil, genValidation("member_id", "only a living member can be invited")
	}

	phone := in.PhoneNumber
	switch {
	case phone == "" && !memberPhone.Valid:
		return nil, genValidation("phone_number", "phone_number is required: the member has none")
	case phone == "":
		phone = memberPhone.String
	case memberPhone.Valid && memberPhone.String != phone:
		return nil, genValidation("phone_number", "phone_number differs from the number recorded on the member")
	}

	// A lapsed pending invitation for this number is marked expired so a new one
	// can be created (the unique index only admits one pending per number).
	if _, err := tx.ExecContext(ctx, `
		UPDATE genealogy_invitations SET status = 'expired', responded_at = now()
		WHERE genealogy_id = $1 AND phone_number = $2 AND status = 'pending' AND expires_at <= now()`,
		genealogyID, phone); err != nil {
		return nil, fmt.Errorf("expire old invitations: %w", err)
	}

	var has bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM genealogies g JOIN users u ON u.id = g.owner_user_id
			WHERE g.id = $1 AND u.phone_number = $2
		) OR EXISTS (
			SELECT 1 FROM genealogy_viewers v JOIN users u ON u.id = v.user_id
			WHERE v.genealogy_id = $1 AND u.phone_number = $2
		)`, genealogyID, phone).Scan(&has); err != nil {
		return nil, fmt.Errorf("check existing access: %w", err)
	}
	if has {
		return nil, domain.ErrViewerExists
	}

	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM genealogy_invitations
		               WHERE genealogy_id = $1 AND phone_number = $2 AND status = 'pending')`,
		genealogyID, phone).Scan(&has); err != nil {
		return nil, fmt.Errorf("check pending invitation: %w", err)
	}
	if has {
		return nil, domain.ErrInvitationAlreadyPending
	}

	if !memberPhone.Valid {
		if _, err := tx.ExecContext(ctx, `
			UPDATE genealogy_members SET phone_number = $2, updated_at = now()
			WHERE id = $1 AND genealogy_id = $3`, in.MemberID, phone, genealogyID); err != nil {
			if isUniqueViolation(err, constraintMemberPhone) {
				return nil, domain.ErrMemberPhoneInUse
			}
			return nil, fmt.Errorf("write phone onto member: %w", err)
		}
	}

	var used int
	if err := tx.QueryRowContext(ctx, `
		SELECT (SELECT count(*) FROM genealogy_viewers WHERE genealogy_id = $1)
		     + (SELECT count(*) FROM genealogy_invitations
		        WHERE genealogy_id = $1 AND status = 'pending' AND expires_at > now())`,
		genealogyID).Scan(&used); err != nil {
		return nil, fmt.Errorf("count viewers and invitations: %w", err)
	}
	if used >= maxViewers {
		return nil, domain.ErrViewerLimitReached
	}

	var id int64
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO genealogy_invitations (genealogy_id, member_id, phone_number, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, now() + $5 * interval '1 second')
		RETURNING id`,
		genealogyID, in.MemberID, phone, invitedBy, int64(ttl/time.Second)).Scan(&id); err != nil {
		if isUniqueViolation(err, constraintInvitationActive) {
			return nil, domain.ErrInvitationAlreadyPending
		}
		return nil, fmt.Errorf("insert invitation: %w", err)
	}

	inv, err := scanInvitation(tx.QueryRowContext(ctx, invitationSelect+` WHERE i.id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("read invitation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &inv, nil
}

// ListByGenealogy returns pending, accepted, declined and expired invitations,
// newest first. Cancelled ones are hidden (AC73).
func (r *genealogyInvitationRepository) ListByGenealogy(ctx context.Context, genealogyID int64) ([]domain.Invitation, error) {
	rows, err := r.db.Query(ctx,
		invitationSelect+` WHERE i.genealogy_id = $1 AND i.status <> 'cancelled' ORDER BY i.created_at DESC, i.id DESC`,
		genealogyID)
	if err != nil {
		return nil, fmt.Errorf("query invitations: %w", err)
	}
	return scanRows(rows, "invitation", scanInvitation)
}

// ListMine is the invitee's inbox: pending, unexpired invitations addressed to
// phone (AC70).
func (r *genealogyInvitationRepository) ListMine(ctx context.Context, phone string) ([]domain.Invitation, error) {
	rows, err := r.db.Query(ctx,
		invitationSelect+` WHERE i.phone_number = $1 AND i.status = 'pending' AND i.expires_at > now()
		 ORDER BY i.created_at DESC, i.id DESC`, phone)
	if err != nil {
		return nil, fmt.Errorf("query inbox: %w", err)
	}
	return scanRows(rows, "invitation", scanInvitation)
}

// Cancel marks a pending invitation of this tree cancelled (AC74).
func (r *genealogyInvitationRepository) Cancel(ctx context.Context, genealogyID, invitationID int64) error {
	res, err := r.db.Exec(ctx, `
		UPDATE genealogy_invitations SET status = 'cancelled', responded_at = now()
		WHERE id = $1 AND genealogy_id = $2 AND status = 'pending'`, invitationID, genealogyID)
	if err != nil {
		return fmt.Errorf("cancel invitation: %w", err)
	}
	return expectOneRow(res, domain.ErrInvitationNotFound)
}

// Accept is Step I-2: grant viewer access and mark the invitation accepted in
// one transaction. The tree is locked before the invitation, the same order as
// Create, so the two cannot deadlock.
func (r *genealogyInvitationRepository) Accept(ctx context.Context, invitationID, userID int64, phone string, maxViewers int) (*domain.AcceptResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var (
		genealogyID, memberID, invitedBy int64
	)
	err = tx.QueryRowContext(ctx, `
		SELECT genealogy_id, member_id, invited_by FROM genealogy_invitations
		WHERE id = $1 AND phone_number = $2`, invitationID, phone).Scan(&genealogyID, &memberID, &invitedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrInvitationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read invitation: %w", err)
	}

	if err := lockGenealogy(ctx, tx, genealogyID); err != nil {
		if errors.Is(err, domain.ErrGenealogyNotFound) {
			return nil, domain.ErrInvitationNotFound
		}
		return nil, err
	}

	var pending bool
	if err := tx.QueryRowContext(ctx, `
		SELECT status = 'pending' AND expires_at > now()
		FROM genealogy_invitations WHERE id = $1 FOR UPDATE`, invitationID).Scan(&pending); err != nil {
		return nil, fmt.Errorf("lock invitation: %w", err)
	}
	if !pending {
		return nil, domain.ErrInvitationNotPending
	}

	var has bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM genealogies WHERE id = $1 AND owner_user_id = $2)
		    OR EXISTS (SELECT 1 FROM genealogy_viewers WHERE genealogy_id = $1 AND user_id = $2)`,
		genealogyID, userID).Scan(&has); err != nil {
		return nil, fmt.Errorf("check existing access: %w", err)
	}
	if has {
		return nil, domain.ErrViewerExists
	}

	var viewers int
	if err := tx.QueryRowContext(ctx,
		`SELECT count(*) FROM genealogy_viewers WHERE genealogy_id = $1`, genealogyID).Scan(&viewers); err != nil {
		return nil, fmt.Errorf("count viewers: %w", err)
	}
	if viewers >= maxViewers {
		return nil, domain.ErrViewerLimitReached
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO genealogy_viewers (genealogy_id, user_id, granted_by) VALUES ($1, $2, $3)`,
		genealogyID, userID, invitedBy); err != nil {
		return nil, fmt.Errorf("insert viewer: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE genealogy_invitations SET status = 'accepted', responded_at = now() WHERE id = $1`,
		invitationID); err != nil {
		return nil, fmt.Errorf("mark accepted: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &domain.AcceptResult{GenealogyID: genealogyID, MemberID: memberID, Role: domain.RoleViewer}, nil
}

// Decline marks a pending, unexpired invitation addressed to phone declined
// (AC72). When nothing was updated it tells "not yours / not there" apart from
// "no longer pending".
func (r *genealogyInvitationRepository) Decline(ctx context.Context, invitationID int64, phone string) error {
	res, err := r.db.Exec(ctx, `
		UPDATE genealogy_invitations SET status = 'declined', responded_at = now()
		WHERE id = $1 AND phone_number = $2 AND status = 'pending' AND expires_at > now()`,
		invitationID, phone)
	if err != nil {
		return fmt.Errorf("decline invitation: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n > 0 {
		return nil
	}
	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM genealogy_invitations WHERE id = $1 AND phone_number = $2)`,
		invitationID, phone).Scan(&exists); err != nil {
		return fmt.Errorf("check invitation: %w", err)
	}
	if exists {
		return domain.ErrInvitationNotPending
	}
	return domain.ErrInvitationNotFound
}

// --- user phone lookup ------------------------------------------------------

type userPhoneLookup struct {
	db database.Database
}

// NewUserPhoneLookup returns the Postgres-backed domain.UserPhoneLookup the
// genealogy use cases use to learn the caller's own phone number.
func NewUserPhoneLookup(db database.Database) domain.UserPhoneLookup {
	return &userPhoneLookup{db: db}
}

// FindPhoneByID returns the account's phone number, or domain.ErrUserNotFound.
func (r *userPhoneLookup) FindPhoneByID(ctx context.Context, userID int64) (string, error) {
	var phone string
	err := r.db.QueryRow(ctx, `SELECT phone_number FROM users WHERE id = $1`, userID).Scan(&phone)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrUserNotFound
	}
	if err != nil {
		return "", fmt.Errorf("find phone: %w", err)
	}
	return phone, nil
}
