package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// LampType is the kind of lamp. The set is fixed by the lamps.lamp_type CHECK,
// so these constants are schema, not configuration.
type LampType string

const (
	LampTypeOilLamp       LampType = "OIL_LAMP"       // đèn dầu
	LampTypeLotusCandle   LampType = "LOTUS_CANDLE"   // nến sen
	LampTypeFlowerLantern LampType = "FLOWER_LANTERN" // hoa đăng
)

// AllLampTypes lists every lamp type, in the order the garden counts report them.
var AllLampTypes = []LampType{LampTypeOilLamp, LampTypeLotusCandle, LampTypeFlowerLantern}

// Valid reports whether t is one of the three lamp types.
func (t LampType) Valid() bool {
	switch t {
	case LampTypeOilLamp, LampTypeLotusCandle, LampTypeFlowerLantern:
		return true
	}
	return false
}

// Intention is the wish a lamp is lit for. The set is fixed by the
// lamps.intention CHECK.
type Intention string

const (
	IntentionPeace           Intention = "PEACE"            // cầu an
	IntentionSalvation       Intention = "SALVATION"        // cầu siêu
	IntentionMeritDedication Intention = "MERIT_DEDICATION" // hồi hướng
	IntentionGratitude       Intention = "GRATITUDE"        // tri ân
)

// Valid reports whether i is one of the four intentions.
func (i Intention) Valid() bool {
	switch i {
	case IntentionPeace, IntentionSalvation, IntentionMeritDedication, IntentionGratitude:
		return true
	}
	return false
}

// AnonymousLighterName is what the garden shows when a lamp carries no name:
// the lighter hid it, or had no dharma name.
const AnonymousLighterName = "Vô danh"

// Lamp is one lit lamp. It maps the lamps table.
//
// Plain data: no json/db tags — the wire shape is delivery's, the SQL shape is
// repository's.
type Lamp struct {
	ID     int64
	UserID int64

	LampType  LampType
	Intention Intention
	// Recipient is nil when the lamp is dedicated to no one in particular
	// (clients show "all beings").
	// VI: Recipient là nil khi đèn không dành riêng cho ai (client hiển thị
	// "Tất cả chúng sanh").
	Recipient *string
	// Prayer is nil when the lighter wrote none.
	Prayer *string

	DurationDays int
	// IsAnonymous records that the lighter chose to hide their name.
	IsAnonymous bool
	// LighterName is the dharma name copied onto the lamp when it was lit. It is
	// nil when the lamp is anonymous or the lighter had no dharma name, and it
	// never follows later profile changes.
	LighterName *string

	LitAt time.Time
	// EndAt is LitAt + DurationDays×24h. A lamp is lit while EndAt is later than now.
	EndAt time.Time
}

// DisplayName is the lighter's name as other users see it.
func (l Lamp) DisplayName() string { return displayLighterName(l.LighterName) }

// LampInput is what the user submits to light a lamp: the raw request, before
// validation. Recipient and Prayer are pointers so "absent" is distinguishable
// from ""; both treat absent, empty and whitespace-only alike.
// VI: Recipient và Prayer là con trỏ để phân biệt "không gửi" với ""; cả hai
// coi không gửi, rỗng và toàn khoảng trắng là như nhau.
type LampInput struct {
	LampType     string
	Intention    string
	Recipient    *string
	Prayer       *string
	DurationDays int
	IsAnonymous  bool
}

// GardenLamp is one lamp as the garden shows it to every signed-in user. It
// deliberately has no intention, prayer or owner: those are not for others.
type GardenLamp struct {
	ID       int64
	LampType LampType
	// Recipient is nil when the lamp has no recipient.
	// VI: Recipient là nil khi đèn không có người nhận.
	Recipient *string
	// IsMine is true when the viewer lit this lamp. It is the only trace of the
	// owner that leaves the repository: the owner's id is never loaded.
	// VI: IsMine là true khi người đang xem là người thắp đèn này. Đây là dấu vết
	// duy nhất về chủ đèn ra khỏi repository: user_id của chủ không bao giờ được đọc.
	IsMine bool
	// LighterName is nil for an anonymous lamp or one lit without a dharma name.
	LighterName *string
	// LitAt orders the garden and builds the paging cursor; it is not shown.
	LitAt time.Time
	EndAt time.Time
}

// DisplayName is the lighter's name as other users see it.
func (g GardenLamp) DisplayName() string { return displayLighterName(g.LighterName) }

// LighterNameAtLighting is the name copied onto a lamp when it is lit: none for
// an anonymous lamp, otherwise the lighter's dharma name at that moment (nil
// when they have none). The account's full name is never used.
// VI: LighterNameAtLighting là tên được chép lên đèn lúc thắp: không có tên nếu
// đèn ẩn danh, ngược lại là pháp danh tại thời điểm đó (nil nếu chưa có pháp
// danh). Không bao giờ dùng họ tên của tài khoản.
func LighterNameAtLighting(dharmaName *string, anonymous bool) *string {
	if anonymous || dharmaName == nil || *dharmaName == "" {
		return nil
	}
	name := *dharmaName
	return &name
}

func displayLighterName(name *string) string {
	if name == nil || *name == "" {
		return AnonymousLighterName
	}
	return *name
}

// Cursor marks a position in the garden: the last lamp of the previous page.
// The garden is ordered LitAt DESC, ID DESC, so "after" means strictly older.
type Cursor struct {
	LitAt time.Time
	ID    int64
}

// GardenFilter is the raw garden query before validation. A nil field means the
// query parameter was absent (an empty string is a different, invalid, value).
type GardenFilter struct {
	LampType *string
	Limit    *string
	Cursor   *string
}

// GardenQuery is a validated garden request.
type GardenQuery struct {
	// LampType restricts the list when non-nil; counts ignore it.
	LampType *LampType
	// Limit is the page size, already clamped to the configured page size.
	Limit int
	// After continues the list strictly after this lamp when non-nil.
	After *Cursor
	// Now is the instant that decides which lamps are lit (EndAt > Now).
	Now time.Time
	// ViewerID is the signed-in user reading the garden. It only sets
	// GardenLamp.IsMine and never filters the list.
	// VI: ViewerID là user đang xem vườn đèn; chỉ dùng để tính GardenLamp.IsMine,
	// không bao giờ dùng để lọc danh sách.
	ViewerID int64
}

// GardenPage is one page of the garden plus the counts of every lit lamp.
type GardenPage struct {
	// Items are the lit lamps, newest first. The repository returns up to
	// Limit+1 of them; the extra one only tells the use case there is a next page.
	Items []GardenLamp
	// Total is every lit lamp of every user and type, ignoring the filter.
	Total int
	// Counts has one key per lamp type, 0 when none is lit.
	Counts map[LampType]int

	// HasMore and NextCursor are filled by the use case, not the repository.
	HasMore    bool
	NextCursor *string
}

// LightLampParams is everything the repository needs to light a lamp atomically.
type LightLampParams struct {
	// Lamp is the validated lamp; Lamp.LitAt is the instant everything compares to.
	// ID and LighterName are filled by the repository.
	Lamp Lamp
	// MaxActive is how many lit lamps one user may have.
	MaxActive int
	// DedupeWindow is how long an identical lamp from the same user counts as
	// the same submission.
	DedupeWindow time.Duration
}

var (
	// ErrLampLimitReached — the user already has the maximum number of lit lamps.
	ErrLampLimitReached = errors.New("lamp limit reached")

	// ErrInvalidLampFilter — the garden's lamp_type is present but not a lamp type.
	ErrInvalidLampFilter = errors.New("lamp_type must be one of OIL_LAMP, LOTUS_CANDLE, FLOWER_LANTERN")

	// ErrInvalidLimit — the garden's limit is not a positive integer.
	ErrInvalidLimit = errors.New("limit must be a positive integer")

	// ErrInvalidCursor — the garden's cursor was not issued by this endpoint.
	ErrInvalidCursor = errors.New("cursor is not valid")

	// ErrLampInvalid — a submitted field is not allowed. Use LampValidationError
	// to say which one; errors.Is still matches this.
	ErrLampInvalid = errors.New("invalid lamp")
)

// LampValidationError names the request field that was rejected and why.
type LampValidationError struct {
	// Field is the request field name, e.g. "recipient".
	Field   string
	Message string
}

func (e *LampValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Unwrap lets errors.Is(err, ErrLampInvalid) match.
func (e *LampValidationError) Unwrap() error { return ErrLampInvalid }

// LampRepository is the port the lamp use case stores through.
type LampRepository interface {
	// Light stores p.Lamp as a lit lamp for p.Lamp.UserID, atomically with the
	// checks that guard it: an identical lamp from the same user lit within
	// p.DedupeWindow is returned instead (created == false, nothing written),
	// and ErrLampLimitReached is returned when the user already has
	// p.MaxActive lit lamps. Concurrent calls for one user can never exceed
	// MaxActive. The lighter's current dharma name is copied onto the lamp
	// unless it is anonymous.
	Light(ctx context.Context, p LightLampParams) (lamp *Lamp, created bool, err error)

	// Garden returns the lit lamps matching q (up to q.Limit+1 of them) together
	// with the total and per-type counts, all read from one snapshot.
	Garden(ctx context.Context, q GardenQuery) (*GardenPage, error)
}
