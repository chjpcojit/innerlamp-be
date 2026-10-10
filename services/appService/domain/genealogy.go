package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Genealogy (gia phả) domain — docs/specs/features/genealogy/ v5.2.
//
// Plain data: no json/db tags. The wire shape is delivery's, the SQL shape is
// repository's. Everything here is business vocabulary.

// Role is the caller's relationship to one genealogy. There are exactly two
// (AC81): the owner edits, a viewer only reads. Resolving it is a DB read done
// on every request, never a token claim.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleViewer Role = "viewer"
)

// MemberKind separates people of the clan from their spouses. The set is fixed
// by the genealogy_members.kind CHECK, so these are schema, not configuration.
type MemberKind string

const (
	// MemberKindBlood is a clan member: has a generation and may have children.
	MemberKindBlood MemberKind = "blood"
	// MemberKindInLaw is a spouse of a blood member (dâu / rể): same generation
	// as the attached member, no parent, no children of its own line.
	MemberKindInLaw MemberKind = "in_law"
)

// Gender is the member's gender. Fixed by the CHECK.
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

// Valid reports whether g is male or female.
func (g Gender) Valid() bool { return g == GenderMale || g == GenderFemale }

// InvitationStatus is the lifecycle of an invitation. Fixed by the CHECK.
type InvitationStatus string

const (
	InvitationPending   InvitationStatus = "pending"
	InvitationAccepted  InvitationStatus = "accepted"
	InvitationDeclined  InvitationStatus = "declined"
	InvitationCancelled InvitationStatus = "cancelled"
	InvitationExpired   InvitationStatus = "expired"
)

// MemberRelation says how a new member attaches to an existing one (AC18).
type MemberRelation string

const (
	RelationChild  MemberRelation = "child"
	RelationSpouse MemberRelation = "spouse"
)

// Valid reports whether r is child or spouse.
func (r MemberRelation) Valid() bool { return r == RelationChild || r == RelationSpouse }

// WarningChildBornBeforeParent is the only warning code (AC27).
const WarningChildBornBeforeParent = "CHILD_BORN_BEFORE_PARENT"

// Genealogy is one clan record owned by one user.
type Genealogy struct {
	ID            int64
	OwnerUserID   int64
	Name          string
	AncestralHome string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// LunarDate is a lunar calendar date derived from a stored solar date. Leap is
// true when Month is a leap month (AC40, AC55). It is never stored.
type LunarDate struct {
	Day   int
	Month int
	Leap  bool
}

// Chronicle is the clan's general record (phả ký). Every optional field is a
// pointer so "absent" is distinguishable from "empty" (AC38 replaces it all).
//
// The chạp họ date is stored as a SOLAR date; ChapEnteredAsLunar remembers that
// the user picked it as lunar so the client can show it that way first (AC40).
type Chronicle struct {
	GenealogyID     int64
	Surname         *string
	CurrentLocation *string
	SinceYear       *int
	Origin          *string
	History         *string
	HallName        *string
	HallAddress     *string

	ChapYear           *int
	ChapMonth          *int
	ChapDay            *int
	ChapEnteredAsLunar bool
	// ChapLunar is the lunar equivalent of the stored solar chạp date, derived at
	// read time; nil when there is no date or it is outside the supported range.
	ChapLunar *LunarDate

	HoanhPhi            *string
	HoanhPhiReading     *string
	HoanhPhiMeaning     *string
	CoupletUpper        *string
	CoupletUpperReading *string
	CoupletUpperMeaning *string
	CoupletLower        *string
	CoupletLowerReading *string
	CoupletLowerMeaning *string
	Teaching            *string

	UpdatedAt time.Time
}

// GenealogyMember is one person in the tree.
//
// ParentID is set only for blood members other than the founder. SpouseOfID is
// set only for in-laws. OtherParentID points at an in-law who is a spouse of
// ParentID (which of the father's wives is this child's mother). Generation is
// stored (denormalised) so list and stats need no recursion; Move keeps it
// consistent.
//
// DeathYear plus DeathMonth/DeathDay form a SOLAR death anniversary;
// DeathEnteredAsLunar remembers the user typed it as lunar (AC55).
type GenealogyMember struct {
	ID          int64
	GenealogyID int64

	Kind   MemberKind
	Gender Gender

	FullName       string
	BirthYear      *int
	IsAlive        bool
	DeathYear      *int
	Residence      *string
	DharmaName     *string
	TabletLocation *string
	BurialPlace    *string

	DeathMonth          *int
	DeathDay            *int
	DeathEnteredAsLunar *bool
	// DeathLunar is the lunar equivalent of the stored solar anniversary,
	// derived at read time; nil when absent or out of range.
	DeathLunar *LunarDate

	PhoneNumber *string
	Note        *string

	// IsClanHead is a remembered flag, not a role (AC82): at most one per
	// genealogy, only a living male blood member.
	IsClanHead bool

	ParentID      *int64
	OtherParentID *int64
	SpouseOfID    *int64
	BirthOrder    *int
	Generation    int

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsFounder reports whether m is the progenitor (blood, no parent).
func (m *GenealogyMember) IsFounder() bool {
	return m.Kind == MemberKindBlood && m.ParentID == nil
}

// InLawLabel is "DAU" for an in-law attached to a male member and "RE" for one
// attached to a female member. It is derived from the attached member's gender,
// not stored, so changing that gender never leaves a wrong label (AC32, AC33).
type InLawLabel string

const (
	InLawLabelDau InLawLabel = "DAU"
	InLawLabelRe  InLawLabel = "RE"
)

// MemberView is a member plus what is derived from the tree at read time.
type MemberView struct {
	GenealogyMember
	// InLawLabel is DAU/RE for an in-law, nil for a blood member (AC33).
	InLawLabel *InLawLabel
}

// MemberResult is what a member write returns: the stored member and the
// advisory warnings computed after the write (AC27).
type MemberResult struct {
	MemberView
	Warnings []MemberWarning
}

// MemberWarning is advisory only: the write has already been applied (AC27).
type MemberWarning struct {
	Code            string
	MemberID        int64
	RelatedMemberID int64
}

// Viewer is a user allowed to read one genealogy (accepted an invitation).
type Viewer struct {
	GenealogyID int64
	UserID      int64
	GrantedBy   int64
	CreatedAt   time.Time

	// PhoneNumber is the viewer's account phone number.
	PhoneNumber string
	// MemberID is the member whose phone number equals PhoneNumber (AC59), nil
	// when none does. Derived at read time, never stored.
	MemberID *int64
}

// Invitation asks one person, by phone number, to view the tree, tied to one
// member (AC44). Matching to the invitee is by phone number only (AC75).
type Invitation struct {
	ID          int64
	GenealogyID int64
	MemberID    int64
	PhoneNumber string
	InvitedBy   int64
	Status      InvitationStatus
	CreatedAt   time.Time
	ExpiresAt   time.Time
	RespondedAt *time.Time

	// MemberFullName and GenealogyName are filled by the read paths that need
	// them (the invitee inbox, the owner's list). InvitedByPhone likewise.
	MemberFullName string
	GenealogyName  string
	InvitedByPhone string
}

// GenealogySummary is one row of "my trees" (AC13).
type GenealogySummary struct {
	ID            int64
	Name          string
	AncestralHome string
	MyRole        Role
	CreatedAt     time.Time
}

// GenealogyDetail is the overview (AC14): info, chronicle and counts computed
// at read time from the tree.
type GenealogyDetail struct {
	Genealogy
	MyRole Role
	// MyMemberID is the member whose phone equals the caller's (AC59).
	MyMemberID *int64
	// ClanHeadMemberID is the member carrying the clan-head flag (AC82).
	ClanHeadMemberID *int64
	Chronicle        Chronicle

	// Derived, never stored (AC41).
	FounderName      string
	GenerationsCount int
	MembersCount     int
}

// --- inputs -----------------------------------------------------------------

// MemberInfo is the information about one person that is submitted wherever a
// member is created or edited (AC4–AC8, AC55, AC82). Optional parts are
// pointers so the use case can tell "absent" from "zero".
type MemberInfo struct {
	FullName string
	Gender   *Gender
	// IsAlive is required; nil is a validation error (AC5).
	IsAlive *bool

	BirthYear      *int
	DeathYear      *int
	Residence      *string
	DharmaName     *string
	TabletLocation *string
	BurialPlace    *string

	DeathMonth          *int
	DeathDay            *int
	DeathEnteredAsLunar *bool

	PhoneNumber *string
	Note        *string

	// IsClanHead nil means "unchanged" on an edit and false on a create (AC82).
	IsClanHead *bool
	// BirthOrder nil means "default to max+1" on a child create and "keep" on edit.
	BirthOrder *int
}

// CreateGenealogyInput is a new tree with its founder and the founder's spouses
// (AC1).
type CreateGenealogyInput struct {
	Name          string
	AncestralHome string
	Founder       MemberInfo
	Spouses       []MemberInfo
}

// AddMemberInput attaches a new member to RelatedMemberID (AC18).
type AddMemberInput struct {
	Relation        string
	RelatedMemberID int64
	// OtherParentID is the in-law who is the other parent; only for child (AC57).
	OtherParentID *int64
	Member        MemberInfo
}

// MoveMemberInput re-attaches an existing member (AC76).
type MoveMemberInput struct {
	Relation        string
	RelatedMemberID int64
	OtherParentID   *int64
	BirthOrder      *int
}

// InviteInput invites one member by phone number (AC44).
type InviteInput struct {
	MemberID int64
	// PhoneNumber may be empty when the member already carries one.
	PhoneNumber string
}

// AcceptResult is what accepting an invitation yields (AC71).
type AcceptResult struct {
	GenealogyID int64
	MemberID    int64
	Role        Role
}

// --- computed profile -------------------------------------------------------

// MemberRef is a compact reference to a member (AC66).
type MemberRef struct {
	ID         int64
	FullName   string
	Gender     Gender
	Generation int
	IsAlive    bool
}

// SiblingRef is a sibling, flagged when only one parent is shared.
type SiblingRef struct {
	MemberRef
	IsHalfSibling bool
}

// LineageItem is one step of the chain from the founder down (AC67).
type LineageItem struct {
	ID         int64
	FullName   string
	Generation int
}

// ChiItem is one branch (chi) head on the path to a member (AC68).
type ChiItem struct {
	Ancestor MemberRef
	// Rank is the 1-based position among the sons of the parent, by
	// (birth_order, id).
	Rank     int
	IsEldest bool
}

// KinshipCode is a stable enum for how a member relates to "me" (AC69).
type KinshipCode string

// Kinship says what the selected member is to "me". Label is Vietnamese; it is
// empty when Code is KinshipUnknown.
type Kinship struct {
	Code  KinshipCode
	Label string
}

const (
	KinshipUnknown KinshipCode = "UNKNOWN"
	KinshipSelf    KinshipCode = "SELF"
)

// MemberProfile is everything computed for one selected member (AC64).
type MemberProfile struct {
	Member     MemberView
	MyMemberID *int64
	Age        *int
	Father     *MemberRef
	Mother     *MemberRef
	Spouses    []MemberRef
	Siblings   []SiblingRef
	Children   []MemberRef
	Lineage    []LineageItem
	Chi        *ChiItem
	ChiPath    []ChiItem
	// RelationToMe is nil when MyMemberID is nil.
	RelationToMe *Kinship
}

// --- errors -----------------------------------------------------------------

var (
	// ErrGenealogyNotFound — no genealogy with this id is visible to this user.
	// One error covers "does not exist" and "you have no role in it", so a
	// caller cannot probe which ids exist (AC15).
	ErrGenealogyNotFound = errors.New("genealogy not found")

	// ErrGenealogyForbidden — the caller has a role but the action needs a
	// higher one (a viewer writing, AC81).
	ErrGenealogyForbidden = errors.New("action not allowed for this role")

	// ErrGenealogyLimitReached — the user already owns the maximum (AC10).
	ErrGenealogyLimitReached = errors.New("genealogy limit reached")

	// ErrMemberNotFound — no member with this id in this genealogy; also covers
	// a member of another genealogy (AC22, AC29).
	ErrMemberNotFound = errors.New("member not found")

	// ErrMemberLimitReached — the genealogy already holds the maximum (AC24).
	ErrMemberLimitReached = errors.New("member limit reached")

	// ErrMemberHasDescendants — a blood member with children cannot be deleted
	// (AC35).
	ErrMemberHasDescendants = errors.New("member has descendants")

	// ErrFounderNotDeletable — the founder cannot be deleted (AC36).
	ErrFounderNotDeletable = errors.New("founder cannot be deleted")

	// ErrFounderNotMovable — the founder cannot be re-attached (AC79).
	ErrFounderNotMovable = errors.New("founder cannot be moved")

	// ErrMemberPhoneInUse — another member of the genealogy already carries this
	// phone number (AC60).
	ErrMemberPhoneInUse = errors.New("phone number already used by another member of this genealogy")

	// ErrViewerExists — the user is already the owner or a viewer (AC46, AC71).
	ErrViewerExists = errors.New("user already has access to this genealogy")

	// ErrViewerLimitReached — viewers plus pending invitations hit the cap (AC47).
	ErrViewerLimitReached = errors.New("viewer limit reached")

	// ErrViewerNotFound — that user is not a viewer of this genealogy (AC49).
	ErrViewerNotFound = errors.New("viewer not found")

	// ErrInvitationNotFound — no such invitation, or not addressed to the
	// caller. Same error for both so invitations of others are not probeable
	// (AC71).
	ErrInvitationNotFound = errors.New("invitation not found")

	// ErrInvitationNotPending — already answered, cancelled or expired (AC71).
	ErrInvitationNotPending = errors.New("invitation is no longer pending")

	// ErrInvitationAlreadyPending — an unexpired pending invitation exists for
	// this number in this genealogy (AC46).
	ErrInvitationAlreadyPending = errors.New("invitation already pending")

	// ErrGenealogyInvalid — a submitted field is not allowed. Use
	// GenealogyValidationError to say which; errors.Is still matches this.
	ErrGenealogyInvalid = errors.New("invalid genealogy request")
)

// GenealogyValidationError names the request field that was rejected and why.
type GenealogyValidationError struct {
	// Field is the request field path, e.g. "founder.full_name" or "member_id".
	Field   string
	Message string
}

func (e *GenealogyValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Unwrap lets errors.Is(err, ErrGenealogyInvalid) match.
func (e *GenealogyValidationError) Unwrap() error { return ErrGenealogyInvalid }

// --- ports ------------------------------------------------------------------

// MemberLimits are the size caps a member write is checked against inside the
// repository's transaction, so they hold under concurrent writes (AC24, AC25).
type MemberLimits struct {
	MaxMembers    int
	MaxGeneration int
	MaxSpouses    int
}

// GenealogyRepository stores trees, chronicles and resolves roles.
type GenealogyRepository interface {
	// Create inserts the tree, an empty chronicle, the founder and the spouses
	// in one transaction, unless the owner already owns maxPerUser trees
	// (ErrGenealogyLimitReached), also under concurrent calls (AC1, AC10, AC12).
	// Returns the new genealogy id. ErrMemberPhoneInUse on a duplicate phone.
	Create(ctx context.Context, ownerUserID int64, in CreateGenealogyParams, maxPerUser int) (int64, error)

	// ListByUser returns trees the user owns or views, newest first (AC13).
	ListByUser(ctx context.Context, userID int64) ([]GenealogySummary, error)

	// ResolveRole returns the caller's role in the tree, or
	// ErrGenealogyNotFound when there is none (Step A, AC15).
	ResolveRole(ctx context.Context, genealogyID, userID int64) (Role, error)

	// FindDetail returns the overview with chronicle, counts and clan head, or
	// ErrGenealogyNotFound. MyRole/MyMemberID are set by the use case.
	FindDetail(ctx context.Context, genealogyID int64) (*GenealogyDetail, error)

	// UpdateInfo changes name and hometown (AC16).
	UpdateInfo(ctx context.Context, genealogyID int64, name, ancestralHome string) error

	// Delete removes the tree and everything under it (AC17).
	Delete(ctx context.Context, genealogyID int64) error

	// UpsertChronicle replaces every chronicle field (AC38).
	UpsertChronicle(ctx context.Context, c *Chronicle) error
}

// CreateGenealogyParams is CreateGenealogyInput after validation and
// normalisation.
type CreateGenealogyParams struct {
	Name          string
	AncestralHome string
	Founder       GenealogyMember
	Spouses       []GenealogyMember
}

// GenealogyMemberRepository stores members.
type GenealogyMemberRepository interface {
	// List returns every member of the tree ordered by generation, id (AC28).
	List(ctx context.Context, genealogyID int64) ([]GenealogyMember, error)

	// FindByID returns the member of this tree or ErrMemberNotFound (AC29).
	FindByID(ctx context.Context, genealogyID, memberID int64) (*GenealogyMember, error)

	// Add attaches m to relatedID in one transaction that serialises writes to
	// the tree, enforces the limits, resolves generation and default
	// birth_order, and swaps the clan-head flag (Step M-2). m.OtherParentID is
	// checked against relatedID (AC57). It returns ErrMemberNotFound when
	// relatedID is not in the tree, ErrMemberLimitReached at the member cap,
	// ErrMemberPhoneInUse on a duplicate number, and a *GenealogyValidationError
	// on related_member_id when the spouse or generation cap would be exceeded.
	Add(ctx context.Context, genealogyID int64, relation MemberRelation, relatedID int64, m *GenealogyMember, limits MemberLimits) (*GenealogyMember, error)

	// Update replaces the editable fields of m, swaps the clan-head flag, and
	// cancels the pending invitation of a member whose phone number changed or
	// was cleared (AC30, AC82). It never changes relationships.
	Update(ctx context.Context, genealogyID int64, m *GenealogyMember) (*GenealogyMember, error)

	// Delete removes a blood member without children together with its
	// in-laws, or a single in-law (Step D-2). ErrMemberHasDescendants,
	// ErrFounderNotDeletable, ErrMemberNotFound.
	Delete(ctx context.Context, genealogyID, memberID int64) error

	// Move re-attaches a member and shifts the generation of its whole branch
	// in one transaction (Step R-2, AC76–AC79).
	Move(ctx context.Context, genealogyID, memberID int64, in MoveMemberInput, limits MemberLimits) (*GenealogyMember, error)

	// FindMemberIDByPhone returns the member of the tree whose phone number is
	// phone, or nil (AC59).
	FindMemberIDByPhone(ctx context.Context, genealogyID int64, phone string) (*int64, error)
}

// ViewerRepository stores viewer grants.
type ViewerRepository interface {
	// List returns the viewers of the tree with the member each one stands at,
	// matched by phone number (AC48).
	List(ctx context.Context, genealogyID int64) ([]Viewer, error)

	// Remove deletes one viewer grant or returns ErrViewerNotFound (AC49, AC50).
	Remove(ctx context.Context, genealogyID, userID int64) error
}

// InvitationRepository stores invitations. Writes that also change viewers or
// members are single transactions that lock the genealogy row first, so Create
// and Accept cannot deadlock.
type InvitationRepository interface {
	// Create is Step C-2: validates the member and number, writes the number
	// onto the member when it has none, and inserts the invitation, enforcing
	// the viewer cap inside the transaction (AC44–AC47, AC60).
	Create(ctx context.Context, genealogyID, invitedBy int64, in InviteInput, maxViewers int, ttl time.Duration) (*Invitation, error)

	// ListByGenealogy returns pending, accepted, declined and expired
	// invitations, newest first (AC73).
	ListByGenealogy(ctx context.Context, genealogyID int64) ([]Invitation, error)

	// ListMine returns the pending, unexpired invitations addressed to phone
	// (AC70).
	ListMine(ctx context.Context, phone string) ([]Invitation, error)

	// Cancel marks a pending invitation of this tree cancelled, or returns
	// ErrInvitationNotFound (AC74).
	Cancel(ctx context.Context, genealogyID, invitationID int64) error

	// Accept is Step I-2: grants the caller viewer access and marks the
	// invitation accepted in one transaction (AC71).
	Accept(ctx context.Context, invitationID, userID int64, phone string, maxViewers int) (*AcceptResult, error)

	// Decline marks a pending invitation addressed to phone declined (AC72).
	Decline(ctx context.Context, invitationID int64, phone string) error
}

// UserPhoneLookup is the minimum the genealogy use cases need from the user
// store: a user's own phone number (AC59, AC71).
type UserPhoneLookup interface {
	// FindPhoneByID returns the normalised phone number of an account.
	FindPhoneByID(ctx context.Context, userID int64) (string, error)
}
