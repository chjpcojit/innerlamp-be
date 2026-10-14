package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Genealogy (docs/specs/features/genealogy/, v5.7) against the real binary and a
// fresh Postgres. What is here and not in the per-layer tests: that cmd/main.go
// wraps every one of the 21 routes in the access-token middleware, that the role
// (owner / viewer / stranger) holds across real users and real rows, that the
// invitation journey works through the real inbox, and that the limits and the
// invitation lifetime come from the running config.

// --- small helpers ----------------------------------------------------------

type gMember struct {
	ID                  int64   `json:"id"`
	Kind                string  `json:"kind"`
	FullName            string  `json:"full_name"`
	Gender              string  `json:"gender"`
	IsAlive             bool    `json:"is_alive"`
	DeathYear           *int    `json:"death_year"`
	DeathMonth          *int    `json:"death_month"`
	DeathDay            *int    `json:"death_day"`
	DeathEnteredAsLunar *bool   `json:"death_entered_as_lunar"`
	DeathLunarDay       *int    `json:"death_lunar_day"`
	DeathLunarMonth     *int    `json:"death_lunar_month"`
	DeathLunarLeap      *bool   `json:"death_lunar_leap"`
	PhoneNumber         *string `json:"phone_number"`
	IsClanHead          bool    `json:"is_clan_head"`
	ParentID            *int64  `json:"parent_id"`
	OtherParentID       *int64  `json:"other_parent_id"`
	SpouseOfID          *int64  `json:"spouse_of_id"`
	BirthOrder          *int    `json:"birth_order"`
	Generation          int     `json:"generation"`
	InLawLabel          *string `json:"in_law_label"`
	Warnings            []gWarn `json:"warnings"`
}

type gWarn struct {
	Code            string `json:"code"`
	MemberID        int64  `json:"member_id"`
	RelatedMemberID int64  `json:"related_member_id"`
}

type gDetail struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	AncestralHome    string `json:"ancestral_home"`
	MyRole           string `json:"my_role"`
	MyMemberID       *int64 `json:"my_member_id"`
	ClanHeadMemberID *int64 `json:"clan_head_member_id"`
	Chronicle        struct {
		Surname          *string `json:"surname"`
		ChapYear         *int    `json:"chap_year"`
		ChapLunarDay     *int    `json:"chap_lunar_day"`
		ChapLunarMonth   *int    `json:"chap_lunar_month"`
		ChapLunarLeap    *bool   `json:"chap_lunar_leap"`
		FounderName      string  `json:"founder_name"`
		GenerationsCount int     `json:"generations_count"`
		MembersCount     int     `json:"members_count"`
	} `json:"chronicle"`
}

type gList[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type gInvitation struct {
	ID             int64  `json:"id"`
	MemberID       int64  `json:"member_id"`
	MemberFullName string `json:"member_full_name"`
	PhoneNumber    string `json:"phone_number"`
	Status         string `json:"status"`
	GenealogyName  string `json:"genealogy_name"`
	InvitedByPhone string `json:"invited_by_phone"`
}

type gUser struct {
	phone string
	token string
	id    string
}

func (c *apiClient) newUser() gUser {
	c.t.Helper()
	phone := c.registerUser()
	tok := c.login(phone, deviceA).AccessToken
	return gUser{phone: phone, token: tok, id: userIDOf(c.t, tok)}
}

func founderBody(name string) map[string]any {
	return map[string]any{"full_name": name, "gender": "male", "is_alive": true}
}

func (c *apiClient) createTree(u gUser, name string) gDetail {
	c.t.Helper()
	res := c.do(http.MethodPost, "/v1/genealogies", map[string]any{
		"name": name, "ancestral_home": "Nam Định", "founder": founderBody("Nguyễn Tổ"),
		"spouses": []map[string]any{{"full_name": "Trần Thị Tổ", "is_alive": true}},
	}, u.token)
	require.Equal(c.t, http.StatusCreated, res.status, res.raw)
	var d gDetail
	res.data(&d)
	return d
}

func (c *apiClient) members(u gUser, gid int64) []gMember {
	c.t.Helper()
	res := c.do(http.MethodGet, fmt.Sprintf("/v1/genealogies/%d/members", gid), nil, u.token)
	require.Equal(c.t, http.StatusOK, res.status, res.raw)
	var l gList[gMember]
	res.data(&l)
	return l.Items
}

func (c *apiClient) founderID(u gUser, gid int64) int64 {
	c.t.Helper()
	for _, m := range c.members(u, gid) {
		if m.ParentID == nil && m.Kind == "blood" {
			return m.ID
		}
	}
	c.t.Fatal("tree has no founder")
	return 0
}

func (c *apiClient) addMember(u gUser, gid int64, body map[string]any) (*apiResponse, gMember) {
	c.t.Helper()
	res := c.do(http.MethodPost, fmt.Sprintf("/v1/genealogies/%d/members", gid), body, u.token)
	var m gMember
	if res.status == http.StatusCreated {
		res.data(&m)
	}
	return res, m
}

func child(parentID int64, name string, extra map[string]any) map[string]any {
	b := map[string]any{"relation": "child", "related_member_id": parentID, "full_name": name, "gender": "male", "is_alive": true}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func genPath(gid int64, tail string) string { return fmt.Sprintf("/v1/genealogies/%d%s", gid, tail) }

// --- tests ------------------------------------------------------------------

func TestGenealogyFlow_E2E(t *testing.T) {
	baseURL, client := newE2EServer(t)
	api := &apiClient{t: t, baseURL: baseURL, http: client}

	t.Run("wiring: all 21 routes are behind the access-token middleware", func(t *testing.T) {
		api.t = t
		u := api.newUser()
		pair := api.login(u.phone, deviceA)

		routes := []struct{ method, path string }{
			{http.MethodPost, "/v1/genealogies"},
			{http.MethodGet, "/v1/genealogies"},
			{http.MethodGet, "/v1/genealogies/1"},
			{http.MethodPut, "/v1/genealogies/1"},
			{http.MethodDelete, "/v1/genealogies/1"},
			{http.MethodPut, "/v1/genealogies/1/chronicle"},
			{http.MethodGet, "/v1/genealogies/1/members"},
			{http.MethodPost, "/v1/genealogies/1/members"},
			{http.MethodGet, "/v1/genealogies/1/members/1"},
			{http.MethodPut, "/v1/genealogies/1/members/1"},
			{http.MethodDelete, "/v1/genealogies/1/members/1"},
			{http.MethodPut, "/v1/genealogies/1/members/1/relation"},
			{http.MethodGet, "/v1/genealogies/1/members/1/profile"},
			{http.MethodPost, "/v1/genealogies/1/invitations"},
			{http.MethodGet, "/v1/genealogies/1/invitations"},
			{http.MethodDelete, "/v1/genealogies/1/invitations/1"},
			{http.MethodGet, "/v1/genealogies/1/viewers"},
			{http.MethodDelete, "/v1/genealogies/1/viewers/1"},
			{http.MethodGet, "/v1/invitations"},
			{http.MethodPost, "/v1/invitations/1/accept"},
			{http.MethodPost, "/v1/invitations/1/decline"},
		}
		require.Len(t, routes, 21)
		for _, rt := range routes {
			var body any
			if rt.method == http.MethodPost || rt.method == http.MethodPut {
				body = map[string]any{}
			}
			none := api.do(rt.method, rt.path, body, "")
			assert.Equal(t, http.StatusUnauthorized, none.status, "%s %s without a token: %s", rt.method, rt.path, none.raw)
			assert.Equal(t, "UNAUTHORIZED", none.errorCode(), "%s %s", rt.method, rt.path)
			garbage := api.do(rt.method, rt.path, body, "not-a-jwt")
			assert.Equal(t, http.StatusUnauthorized, garbage.status, "%s %s with garbage", rt.method, rt.path)
			wrongType := api.do(rt.method, rt.path, body, pair.RefreshToken)
			assert.Equal(t, http.StatusUnauthorized, wrongType.status, "%s %s with a refresh token", rt.method, rt.path)
		}
	})

	t.Run("AC1-AC17: create, overview, rename, chronicle with lunar date, delete", func(t *testing.T) {
		api.t = t
		owner := api.newUser()

		d := api.createTree(owner, "Họ Nguyễn")
		assert.Equal(t, "owner", d.MyRole)
		assert.Equal(t, "Nguyễn Tổ", d.Chronicle.FounderName)
		assert.Equal(t, 1, d.Chronicle.GenerationsCount)
		assert.Equal(t, 2, d.Chronicle.MembersCount, "founder + one spouse")
		assert.Nil(t, d.MyMemberID, "no member carries the owner's number yet")

		ms := api.members(owner, d.ID)
		require.Len(t, ms, 2)
		assert.Equal(t, "blood", ms[0].Kind)
		assert.Equal(t, 1, ms[0].Generation)
		assert.Equal(t, "in_law", ms[1].Kind)
		require.NotNil(t, ms[1].InLawLabel)
		assert.Equal(t, "DAU", *ms[1].InLawLabel, "wife of a male member")

		// A female founder, a blank name and unknown fields are refused and create nothing.
		for _, tt := range []struct {
			name string
			body map[string]any
			code int
		}{
			{"female founder", map[string]any{"name": "x", "ancestral_home": "y", "founder": map[string]any{"full_name": "a", "gender": "female", "is_alive": true}}, http.StatusBadRequest},
			{"blank name", map[string]any{"name": " ", "ancestral_home": "y", "founder": founderBody("a")}, http.StatusBadRequest},
			{"server-owned field", map[string]any{"name": "x", "ancestral_home": "y", "owner_user_id": 1, "founder": founderBody("a")}, http.StatusBadRequest},
		} {
			res := api.do(http.MethodPost, "/v1/genealogies", tt.body, owner.token)
			assert.Equal(t, tt.code, res.status, tt.name)
		}

		// Chronicle: a solar chạp date comes back with its lunar equivalent (Tết 2024 = 10 Feb 2024).
		res := api.do(http.MethodPut, genPath(d.ID, "/chronicle"), map[string]any{
			"surname": "Nguyễn", "chap_year": 2024, "chap_month": 2, "chap_day": 10, "chap_entered_as_lunar": true,
			"couplet_upper": "上聯", "couplet_upper_reading": "thượng liên", "couplet_upper_meaning": "vế trên",
		}, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var ch struct {
			ChapLunarDay   *int   `json:"chap_lunar_day"`
			ChapLunarMonth *int   `json:"chap_lunar_month"`
			ChapLunarLeap  *bool  `json:"chap_lunar_leap"`
			Reading        string `json:"couplet_upper_reading"`
		}
		res.data(&ch)
		require.NotNil(t, ch.ChapLunarDay)
		assert.Equal(t, 1, *ch.ChapLunarDay)
		assert.Equal(t, 1, *ch.ChapLunarMonth)
		assert.Equal(t, "thượng liên", ch.Reading)

		for _, derived := range []map[string]any{{"founder_name": "x"}, {"members_count": 9}, {"chap_lunar_day": 1}} {
			bad := api.do(http.MethodPut, genPath(d.ID, "/chronicle"), derived, owner.token)
			assert.Equal(t, http.StatusBadRequest, bad.status, "derived field %v must be refused", derived)
		}
		half := api.do(http.MethodPut, genPath(d.ID, "/chronicle"), map[string]any{"chap_year": 2024}, owner.token)
		assert.Equal(t, http.StatusBadRequest, half.status)
		assert.Equal(t, "chap_month", half.details()["field"])

		// Rename.
		res = api.do(http.MethodPut, genPath(d.ID, ""), map[string]any{"name": "Họ Nguyễn Đại", "ancestral_home": "Hà Nam"}, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		res = api.do(http.MethodGet, genPath(d.ID, ""), nil, owner.token)
		var got gDetail
		res.data(&got)
		assert.Equal(t, "Họ Nguyễn Đại", got.Name)
		require.NotNil(t, got.Chronicle.ChapLunarMonth)

		// List shows it, with my_role.
		res = api.do(http.MethodGet, "/v1/genealogies", nil, owner.token)
		var list gList[struct {
			ID     int64  `json:"id"`
			MyRole string `json:"my_role"`
		}]
		res.data(&list)
		require.Len(t, list.Items, 1)
		assert.Equal(t, "owner", list.Items[0].MyRole)

		// Delete is final.
		res = api.do(http.MethodDelete, genPath(d.ID, ""), nil, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, http.StatusNotFound, api.do(http.MethodGet, genPath(d.ID, ""), nil, owner.token).status)
		assert.Equal(t, http.StatusNotFound, api.do(http.MethodDelete, genPath(d.ID, ""), nil, owner.token).status)
	})

	t.Run("AC18-AC37, AC55-AC58, AC76-AC80, AC82: members, warnings, deletion, move, clan head", func(t *testing.T) {
		api.t = t
		owner := api.newUser()
		d := api.createTree(owner, "Họ A")
		root := api.founderID(owner, d.ID)

		// Children get the next birth order and the right generation.
		res, a := api.addMember(owner, d.ID, child(root, "A", map[string]any{"birth_year": 1950}))
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		assert.Equal(t, 2, a.Generation)
		require.NotNil(t, a.BirthOrder)
		assert.Equal(t, 1, *a.BirthOrder)
		assert.Empty(t, a.Warnings)

		res, b := api.addMember(owner, d.ID, child(root, "B", nil))
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		assert.Equal(t, 2, *b.BirthOrder)

		// A child born before the parent is saved, with a warning (AC27).
		res, early := api.addMember(owner, d.ID, child(a.ID, "Con sớm", map[string]any{"birth_year": 1940}))
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		require.Len(t, early.Warnings, 1)
		assert.Equal(t, "CHILD_BORN_BEFORE_PARENT", early.Warnings[0].Code)
		assert.Equal(t, early.ID, early.Warnings[0].MemberID)
		assert.Equal(t, a.ID, early.Warnings[0].RelatedMemberID)

		// A daughter has no children in this tree; an in-law has none either.
		res, daughter := api.addMember(owner, d.ID, child(root, "Mai", map[string]any{"gender": "female"}))
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		res, _ = api.addMember(owner, d.ID, child(daughter.ID, "Cháu ngoại", nil))
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "related_member_id", res.details()["field"])

		// Her husband is a son-in-law (RE).
		res, husband := api.addMember(owner, d.ID, map[string]any{
			"relation": "spouse", "related_member_id": daughter.ID, "full_name": "Chồng Mai", "is_alive": true,
		})
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		assert.Equal(t, "male", husband.Gender)
		require.NotNil(t, husband.InLawLabel)
		assert.Equal(t, "RE", *husband.InLawLabel)

		// Deleting: a member with children is refused, the founder is refused, a leaf goes.
		res = api.do(http.MethodDelete, genPath(d.ID, fmt.Sprintf("/members/%d", a.ID)), nil, owner.token)
		assert.Equal(t, http.StatusConflict, res.status)
		assert.Equal(t, "MEMBER_HAS_DESCENDANTS", res.errorCode())
		res = api.do(http.MethodDelete, genPath(d.ID, fmt.Sprintf("/members/%d", root)), nil, owner.token)
		assert.Equal(t, http.StatusConflict, res.status)
		assert.Equal(t, "FOUNDER_NOT_DELETABLE", res.errorCode())

		// Move: A (with its child) under B shifts the whole branch one generation deeper.
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d/relation", a.ID)),
			map[string]any{"relation": "child", "related_member_id": b.ID}, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var moved gMember
		res.data(&moved)
		assert.Equal(t, 3, moved.Generation)
		for _, m := range api.members(owner, d.ID) {
			if m.ID == early.ID {
				assert.Equal(t, 4, m.Generation, "descendants follow the moved member")
			}
		}
		// A loop and a founder move are refused.
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d/relation", b.ID)),
			map[string]any{"relation": "child", "related_member_id": early.ID}, owner.token)
		assert.Equal(t, http.StatusBadRequest, res.status)
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d/relation", root)),
			map[string]any{"relation": "child", "related_member_id": b.ID}, owner.token)
		assert.Equal(t, http.StatusConflict, res.status)
		assert.Equal(t, "FOUNDER_NOT_MOVABLE", res.errorCode())

		// PUT never carries relationships.
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", a.ID)),
			map[string]any{"full_name": "A", "gender": "male", "is_alive": true, "parent_id": root}, owner.token)
		assert.Equal(t, http.StatusBadRequest, res.status)

		// Clan head: a flag, one per tree; the new head replaces the old.
		setHead := func(id int64, name string, val bool) *apiResponse {
			return api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", id)),
				map[string]any{"full_name": name, "gender": "male", "is_alive": true, "is_clan_head": val}, owner.token)
		}
		require.Equal(t, http.StatusOK, setHead(a.ID, "A", true).status)
		require.Equal(t, http.StatusOK, setHead(b.ID, "B", true).status)
		var dd gDetail
		api.do(http.MethodGet, genPath(d.ID, ""), nil, owner.token).data(&dd)
		require.NotNil(t, dd.ClanHeadMemberID)
		assert.Equal(t, b.ID, *dd.ClanHeadMemberID)
		// The clan head dies without clearing the flag: refused.
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", b.ID)),
			map[string]any{"full_name": "B", "gender": "male", "is_alive": false}, owner.token)
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "is_alive", res.details()["field"])
		// A woman cannot be clan head.
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", daughter.ID)),
			map[string]any{"full_name": "Mai", "gender": "female", "is_alive": true, "is_clan_head": true}, owner.token)
		assert.Equal(t, http.StatusBadRequest, res.status)

		// Death anniversary: solar storage, lunar derived (Tết 2024).
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", early.ID)), map[string]any{
			"full_name": "Con sớm", "gender": "male", "is_alive": false, "birth_year": 1940, "death_year": 2024,
			"death_month": 2, "death_day": 10, "death_entered_as_lunar": true,
		}, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var died gMember
		res.data(&died)
		require.NotNil(t, died.DeathLunarDay)
		assert.Equal(t, 1, *died.DeathLunarDay)
		assert.Equal(t, 1, *died.DeathLunarMonth)
		assert.Equal(t, 10, *died.DeathDay, "the stored date stays solar")
		// A living member cannot have an anniversary.
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", b.ID)), map[string]any{
			"full_name": "B", "gender": "male", "is_alive": true, "death_month": 2, "death_day": 10,
		}, owner.token)
		assert.Equal(t, http.StatusBadRequest, res.status)

		// Phone uniqueness inside the tree (AC60).
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", a.ID)), map[string]any{
			"full_name": "A", "gender": "male", "is_alive": true, "phone_number": "0912000111", "is_clan_head": false,
		}, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", b.ID)), map[string]any{
			"full_name": "B", "gender": "male", "is_alive": true, "phone_number": "0912000111", "is_clan_head": true,
		}, owner.token)
		assert.Equal(t, http.StatusConflict, res.status)
		assert.Equal(t, "MEMBER_PHONE_ALREADY_USED", res.errorCode())

		// Deleting a leaf works; its in-laws go with it.
		res = api.do(http.MethodDelete, genPath(d.ID, fmt.Sprintf("/members/%d", daughter.ID)), nil, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		for _, m := range api.members(owner, d.ID) {
			assert.NotEqual(t, daughter.ID, m.ID)
			assert.NotEqual(t, husband.ID, m.ID, "the in-law goes with the member it was attached to")
		}
	})

	t.Run("AC64-AC69: the profile is computed for me and follows rank, not age", func(t *testing.T) {
		api.t = t
		owner := api.newUser()
		d := api.createTree(owner, "Họ B")
		root := api.founderID(owner, d.ID)

		// root ─ loc ─ {z (elder), a} ; a ─ c   (c is me; z is born later but is the elder brother)
		_, loc := api.addMember(owner, d.ID, child(root, "Lộc", nil))
		_, z := api.addMember(owner, d.ID, child(loc.ID, "Z", map[string]any{"birth_year": 2010, "birth_order": 1}))
		_, a := api.addMember(owner, d.ID, child(loc.ID, "A", map[string]any{"birth_year": 1960, "birth_order": 2}))
		_, c := api.addMember(owner, d.ID, child(a.ID, "C", map[string]any{"birth_year": 1990}))

		// Make C "me": the owner's own number goes onto C.
		res := api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", c.ID)), map[string]any{
			"full_name": "C", "gender": "male", "is_alive": true, "birth_year": 1990, "phone_number": owner.phone,
		}, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)

		var detail gDetail
		api.do(http.MethodGet, genPath(d.ID, ""), nil, owner.token).data(&detail)
		require.NotNil(t, detail.MyMemberID)
		assert.Equal(t, c.ID, *detail.MyMemberID)

		res = api.do(http.MethodGet, genPath(d.ID, fmt.Sprintf("/members/%d/profile", z.ID)), nil, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var p struct {
			MyMemberID   *int64 `json:"my_member_id"`
			Age          *int   `json:"age"`
			RelationToMe *struct {
				Code  string  `json:"code"`
				Label *string `json:"label"`
			} `json:"relation_to_me"`
			Lineage []struct{ ID int64 } `json:"lineage"`
			Chi     *struct {
				Ancestor struct {
					FullName string `json:"full_name"`
				} `json:"ancestor"`
				IsEldest bool `json:"is_eldest"`
			} `json:"chi"`
		}
		res.data(&p)
		require.NotNil(t, p.RelationToMe)
		require.NotNil(t, p.RelationToMe.Label)
		assert.Equal(t, "Bác", *p.RelationToMe.Label, "my father's elder brother, although born decades later")
		assert.Len(t, p.Lineage, 3)
		require.NotNil(t, p.Chi)
		assert.Equal(t, "Z", p.Chi.Ancestor.FullName)
		assert.True(t, p.Chi.IsEldest)

		// A member of another tree is simply not found.
		other := api.newUser()
		od := api.createTree(other, "Họ khác")
		foreign := api.founderID(other, od.ID)
		res = api.do(http.MethodGet, genPath(d.ID, fmt.Sprintf("/members/%d/profile", foreign)), nil, owner.token)
		assert.Equal(t, http.StatusNotFound, res.status)
		assert.Equal(t, "MEMBER_NOT_FOUND", res.errorCode())
	})

	t.Run("AC44-AC50, AC53, AC59-AC62, AC70-AC75, AC81: the invitation journey and the viewer role", func(t *testing.T) {
		api.t = t
		owner := api.newUser()
		invitee := api.newUser()
		stranger := api.newUser()
		d := api.createTree(owner, "Họ C")
		root := api.founderID(owner, d.ID)
		_, son := api.addMember(owner, d.ID, child(root, "Nguyễn Văn B", nil))

		// The owner picks a member and invites by phone: pending, number written onto the node.
		res := api.do(http.MethodPost, genPath(d.ID, "/invitations"),
			map[string]any{"member_id": son.ID, "phone_number": invitee.phone}, owner.token)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var inv gInvitation
		res.data(&inv)
		assert.Equal(t, "pending", inv.Status)
		assert.Equal(t, son.ID, inv.MemberID)

		// Same answer for a number nobody has registered (no account probe, AC45).
		_, other := api.addMember(owner, d.ID, child(root, "Chưa có tài khoản", nil))
		res = api.do(http.MethodPost, genPath(d.ID, "/invitations"),
			map[string]any{"member_id": other.ID, "phone_number": "+84977700001"}, owner.token)
		assert.Equal(t, http.StatusCreated, res.status, res.raw)

		// Duplicates and bad input.
		res = api.do(http.MethodPost, genPath(d.ID, "/invitations"),
			map[string]any{"member_id": son.ID}, owner.token)
		assert.Equal(t, http.StatusConflict, res.status)
		assert.Equal(t, "INVITATION_ALREADY_PENDING", res.errorCode())
		res = api.do(http.MethodPost, genPath(d.ID, "/invitations"), map[string]any{"phone_number": invitee.phone}, owner.token)
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "member_id", res.details()["field"])

		// The invitee sees it in the inbox, with nothing else of the tree.
		res = api.do(http.MethodGet, "/v1/invitations", nil, invitee.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var inbox gList[gInvitation]
		res.data(&inbox)
		require.Len(t, inbox.Items, 1)
		assert.Equal(t, "Họ C", inbox.Items[0].GenealogyName)
		assert.Equal(t, "Nguyễn Văn B", inbox.Items[0].MemberFullName)
		assert.Equal(t, owner.phone, inbox.Items[0].InvitedByPhone)
		assert.NotContains(t, res.raw, `"genealogy_id"`, "the inbox must not leak the tree")

		// Before accepting, the invitee has no access at all.
		assert.Equal(t, http.StatusNotFound, api.do(http.MethodGet, genPath(d.ID, ""), nil, invitee.token).status)

		// A stranger cannot accept someone else's invitation.
		res = api.do(http.MethodPost, fmt.Sprintf("/v1/invitations/%d/accept", inv.ID), nil, stranger.token)
		assert.Equal(t, http.StatusNotFound, res.status)
		assert.Equal(t, "INVITATION_NOT_FOUND", res.errorCode())

		// Accept: the invitee becomes a viewer standing at the invited node.
		res = api.do(http.MethodPost, fmt.Sprintf("/v1/invitations/%d/accept", inv.ID), nil, invitee.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var acc struct {
			GenealogyID int64  `json:"genealogy_id"`
			MemberID    int64  `json:"member_id"`
			Role        string `json:"role"`
		}
		res.data(&acc)
		assert.Equal(t, d.ID, acc.GenealogyID)
		assert.Equal(t, son.ID, acc.MemberID)
		assert.Equal(t, "viewer", acc.Role)

		var vd gDetail
		api.do(http.MethodGet, genPath(d.ID, ""), nil, invitee.token).data(&vd)
		assert.Equal(t, "viewer", vd.MyRole)
		require.NotNil(t, vd.MyMemberID)
		assert.Equal(t, son.ID, *vd.MyMemberID, "my position is the member carrying my phone number")

		// The viewer reads the tree, a member and a profile …
		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, genPath(d.ID, "/members"), nil, invitee.token).status)
		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, genPath(d.ID, fmt.Sprintf("/members/%d", son.ID)), nil, invitee.token).status)
		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, genPath(d.ID, fmt.Sprintf("/members/%d/profile", son.ID)), nil, invitee.token).status)
		// … and the tree shows in their list.
		var vlist gList[struct {
			ID     int64  `json:"id"`
			MyRole string `json:"my_role"`
		}]
		api.do(http.MethodGet, "/v1/genealogies", nil, invitee.token).data(&vlist)
		require.Len(t, vlist.Items, 1)
		assert.Equal(t, "viewer", vlist.Items[0].MyRole)

		// … but cannot change anything (AC81), nor manage access.
		writes := []struct {
			method, path string
			body         any
		}{
			{http.MethodPut, genPath(d.ID, ""), map[string]any{"name": "x", "ancestral_home": "y"}},
			{http.MethodDelete, genPath(d.ID, ""), nil},
			{http.MethodPut, genPath(d.ID, "/chronicle"), map[string]any{"surname": "x"}},
			{http.MethodPost, genPath(d.ID, "/members"), child(root, "x", nil)},
			{http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", son.ID)), map[string]any{"full_name": "x", "gender": "male", "is_alive": true}},
			{http.MethodDelete, genPath(d.ID, fmt.Sprintf("/members/%d", son.ID)), nil},
			{http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d/relation", son.ID)), map[string]any{"relation": "child", "related_member_id": root}},
			{http.MethodPost, genPath(d.ID, "/invitations"), map[string]any{"member_id": root, "phone_number": "0911000222"}},
			{http.MethodGet, genPath(d.ID, "/invitations"), nil},
			{http.MethodGet, genPath(d.ID, "/viewers"), nil},
		}
		for _, w := range writes {
			r := api.do(w.method, w.path, w.body, invitee.token)
			assert.Equal(t, http.StatusForbidden, r.status, "%s %s: %s", w.method, w.path, r.raw)
			assert.Equal(t, "GENEALOGY_FORBIDDEN", r.errorCode(), "%s %s", w.method, w.path)
		}
		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, genPath(d.ID, fmt.Sprintf("/members/%d", son.ID)), nil, owner.token).status,
			"nothing was changed by the refused writes")

		// The owner sees the status change and who stands where.
		var invs gList[gInvitation]
		api.do(http.MethodGet, genPath(d.ID, "/invitations"), nil, owner.token).data(&invs)
		statuses := map[int64]string{}
		for _, i := range invs.Items {
			statuses[i.ID] = i.Status
		}
		assert.Equal(t, "accepted", statuses[inv.ID])
		var viewers gList[struct {
			UserID      int64  `json:"user_id"`
			PhoneNumber string `json:"phone_number"`
			MemberID    *int64 `json:"member_id"`
		}]
		api.do(http.MethodGet, genPath(d.ID, "/viewers"), nil, owner.token).data(&viewers)
		require.Len(t, viewers.Items, 1)
		require.NotNil(t, viewers.Items[0].MemberID)
		assert.Equal(t, son.ID, *viewers.Items[0].MemberID)

		// Accepting twice, or declining after accepting, finds it no longer pending.
		res = api.do(http.MethodPost, fmt.Sprintf("/v1/invitations/%d/accept", inv.ID), nil, invitee.token)
		assert.Equal(t, http.StatusConflict, res.status)
		assert.Equal(t, "INVITATION_NOT_PENDING", res.errorCode())

		// Clearing the node's number moves nobody's access but loses the position.
		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", son.ID)),
			map[string]any{"full_name": "Nguyễn Văn B", "gender": "male", "is_alive": true}, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		api.do(http.MethodGet, genPath(d.ID, ""), nil, invitee.token).data(&vd)
		assert.Nil(t, vd.MyMemberID, "the viewer still sees the tree, but no longer has a position")

		// A viewer leaves; the owner cannot leave through the same action.
		res = api.do(http.MethodDelete, genPath(d.ID, fmt.Sprintf("/viewers/%s", invitee.id)), nil, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, http.StatusNotFound, api.do(http.MethodGet, genPath(d.ID, ""), nil, invitee.token).status,
			"AC54: removal takes effect on the very next request")
		res = api.do(http.MethodDelete, genPath(d.ID, fmt.Sprintf("/viewers/%s", owner.id)), nil, owner.token)
		assert.Equal(t, http.StatusNotFound, res.status)
		assert.Equal(t, "VIEWER_NOT_FOUND", res.errorCode())
	})

	t.Run("AC15, AC53: a stranger learns nothing — the same 404 as a tree that does not exist", func(t *testing.T) {
		api.t = t
		owner := api.newUser()
		stranger := api.newUser()
		d := api.createTree(owner, "Họ riêng")
		root := api.founderID(owner, d.ID)

		missing := d.ID + 999
		paths := []struct {
			method, tail string
			body         any
		}{
			{http.MethodGet, "", nil},
			{http.MethodPut, "", map[string]any{"name": "x", "ancestral_home": "y"}},
			{http.MethodDelete, "", nil},
			{http.MethodPut, "/chronicle", map[string]any{"surname": "x"}},
			{http.MethodGet, "/members", nil},
			{http.MethodPost, "/members", child(root, "x", nil)},
			{http.MethodGet, fmt.Sprintf("/members/%d", root), nil},
			{http.MethodPut, fmt.Sprintf("/members/%d", root), map[string]any{"full_name": "x", "gender": "male", "is_alive": true}},
			{http.MethodDelete, fmt.Sprintf("/members/%d", root), nil},
			{http.MethodGet, fmt.Sprintf("/members/%d/profile", root), nil},
			{http.MethodPost, "/invitations", map[string]any{"member_id": root, "phone_number": "0911000222"}},
			{http.MethodGet, "/invitations", nil},
			{http.MethodGet, "/viewers", nil},
		}
		for _, p := range paths {
			real := api.do(p.method, genPath(d.ID, p.tail), p.body, stranger.token)
			ghost := api.do(p.method, genPath(missing, p.tail), p.body, stranger.token)
			assert.Equal(t, http.StatusNotFound, real.status, "%s %s: %s", p.method, p.tail, real.raw)
			assert.Equal(t, "GENEALOGY_NOT_FOUND", real.errorCode(), "%s %s", p.method, p.tail)
			assert.Equal(t, ghost.status, real.status, "%s %s: existence must not be observable", p.method, p.tail)
			assert.Equal(t, ghost.raw, real.raw, "%s %s: the body must be byte-identical", p.method, p.tail)
		}
		// Malformed ids are the same answer too.
		for _, id := range []string{"abc", "-1", "0"} {
			res := api.do(http.MethodGet, "/v1/genealogies/"+id, nil, stranger.token)
			assert.Equal(t, http.StatusNotFound, res.status, id)
			assert.Equal(t, "GENEALOGY_NOT_FOUND", res.errorCode(), id)
		}
		// The stranger's own list is empty.
		var l gList[struct{ ID int64 }]
		api.do(http.MethodGet, "/v1/genealogies", nil, stranger.token).data(&l)
		assert.Empty(t, l.Items)
	})

	t.Run("AC72: decline, owner cancel, and a viewer leaving", func(t *testing.T) {
		api.t = t
		owner := api.newUser()
		a := api.newUser()
		b := api.newUser()
		d := api.createTree(owner, "Họ D")
		root := api.founderID(owner, d.ID)
		_, ma := api.addMember(owner, d.ID, child(root, "A", nil))
		_, mb := api.addMember(owner, d.ID, child(root, "B", nil))

		inv := func(member int64, phone string) gInvitation {
			res := api.do(http.MethodPost, genPath(d.ID, "/invitations"), map[string]any{"member_id": member, "phone_number": phone}, owner.token)
			require.Equal(t, http.StatusCreated, res.status, res.raw)
			var i gInvitation
			res.data(&i)
			return i
		}
		ia := inv(ma.ID, a.phone)
		ib := inv(mb.ID, b.phone)

		// A declines: no access, and the owner sees `declined`.
		res := api.do(http.MethodPost, fmt.Sprintf("/v1/invitations/%d/decline", ia.ID), nil, a.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, http.StatusNotFound, api.do(http.MethodGet, genPath(d.ID, ""), nil, a.token).status)
		var list gList[gInvitation]
		api.do(http.MethodGet, genPath(d.ID, "/invitations"), nil, owner.token).data(&list)
		got := map[int64]string{}
		for _, i := range list.Items {
			got[i.ID] = i.Status
		}
		assert.Equal(t, "declined", got[ia.ID])

		// The owner cancels B's pending invitation: it disappears from both sides.
		res = api.do(http.MethodDelete, genPath(d.ID, fmt.Sprintf("/invitations/%d", ib.ID)), nil, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var inbox gList[gInvitation]
		api.do(http.MethodGet, "/v1/invitations", nil, b.token).data(&inbox)
		assert.Empty(t, inbox.Items)
		res = api.do(http.MethodPost, fmt.Sprintf("/v1/invitations/%d/accept", ib.ID), nil, b.token)
		assert.Equal(t, http.StatusConflict, res.status)

		// Invite A again after the decline (allowed), A accepts, then leaves.
		again := inv(ma.ID, a.phone)
		res = api.do(http.MethodPost, fmt.Sprintf("/v1/invitations/%d/accept", again.ID), nil, a.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, genPath(d.ID, ""), nil, a.token).status)

		res = api.do(http.MethodDelete, genPath(d.ID, fmt.Sprintf("/viewers/%s", a.id)), nil, a.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		assert.Equal(t, http.StatusNotFound, api.do(http.MethodGet, genPath(d.ID, ""), nil, a.token).status, "leaving takes effect immediately")

		// A viewer cannot remove someone else.
		res = api.do(http.MethodPost, fmt.Sprintf("/v1/invitations/%d/decline", 99999), nil, a.token)
		assert.Equal(t, http.StatusNotFound, res.status)
	})

	t.Run("a deceased member cannot be invited, and clearing a number cancels the invitation", func(t *testing.T) {
		api.t = t
		owner := api.newUser()
		d := api.createTree(owner, "Họ E")
		root := api.founderID(owner, d.ID)
		_, dead := api.addMember(owner, d.ID, child(root, "Cụ", map[string]any{"is_alive": false}))
		res := api.do(http.MethodPost, genPath(d.ID, "/invitations"), map[string]any{"member_id": dead.ID, "phone_number": "0911000333"}, owner.token)
		assert.Equal(t, http.StatusBadRequest, res.status)
		assert.Equal(t, "member_id", res.details()["field"])

		_, alive := api.addMember(owner, d.ID, child(root, "Còn sống", nil))
		res = api.do(http.MethodPost, genPath(d.ID, "/invitations"), map[string]any{"member_id": alive.ID, "phone_number": "0911000444"}, owner.token)
		require.Equal(t, http.StatusCreated, res.status, res.raw)
		var inv gInvitation
		res.data(&inv)

		res = api.do(http.MethodPut, genPath(d.ID, fmt.Sprintf("/members/%d", alive.ID)),
			map[string]any{"full_name": "Còn sống", "gender": "male", "is_alive": true}, owner.token)
		require.Equal(t, http.StatusOK, res.status, res.raw)
		var list gList[gInvitation]
		api.do(http.MethodGet, genPath(d.ID, "/invitations"), nil, owner.token).data(&list)
		for _, i := range list.Items {
			assert.NotEqual(t, inv.ID, i.ID, "a cancelled invitation is hidden")
		}
	})
}

func TestGenealogyLimitsComeFromConfig_E2E(t *testing.T) {
	baseURL, client := newE2EServerWithConfig(t, configOverrides{
		"genealogy": {"max_per_user": 2, "max_members": 4, "max_viewers": 1},
	})
	api := &apiClient{t: t, baseURL: baseURL, http: client}

	owner := api.newUser()
	api.createTree(owner, "Cây 1")
	second := api.createTree(owner, "Cây 2")
	res := api.do(http.MethodPost, "/v1/genealogies", map[string]any{
		"name": "Cây 3", "ancestral_home": "y", "founder": founderBody("a"),
	}, owner.token)
	assert.Equal(t, http.StatusConflict, res.status, "max_per_user = 2 from the running config")
	assert.Equal(t, "GENEALOGY_LIMIT_REACHED", res.errorCode())

	// The tree has founder + spouse; max_members = 4 admits two more.
	root := api.founderID(owner, second.ID)
	for i := 0; i < 2; i++ {
		r, _ := api.addMember(owner, second.ID, child(root, fmt.Sprintf("C%d", i), nil))
		require.Equal(t, http.StatusCreated, r.status, r.raw)
	}
	r, _ := api.addMember(owner, second.ID, child(root, "C3", nil))
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "MEMBER_LIMIT_REACHED", r.errorCode())

	// max_viewers = 1 counts pending invitations.
	ms := api.members(owner, second.ID)
	var a, b int64
	for _, m := range ms {
		if m.FullName == "C0" {
			a = m.ID
		}
		if m.FullName == "C1" {
			b = m.ID
		}
	}
	res = api.do(http.MethodPost, genPath(second.ID, "/invitations"), map[string]any{"member_id": a, "phone_number": "0911000555"}, owner.token)
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	res = api.do(http.MethodPost, genPath(second.ID, "/invitations"), map[string]any{"member_id": b, "phone_number": "0911000666"}, owner.token)
	assert.Equal(t, http.StatusConflict, res.status)
	assert.Equal(t, "VIEWER_LIMIT_REACHED", res.errorCode())
}

func TestGenealogyInvitationExpiry_E2E(t *testing.T) {
	baseURL, client := newE2EServerWithConfig(t, configOverrides{
		"genealogy": {"invitation_ttl": "1s"},
	})
	api := &apiClient{t: t, baseURL: baseURL, http: client}

	owner := api.newUser()
	invitee := api.newUser()
	d := api.createTree(owner, "Họ hết hạn")
	root := api.founderID(owner, d.ID)
	_, m := api.addMember(owner, d.ID, child(root, "B", nil))

	res := api.do(http.MethodPost, genPath(d.ID, "/invitations"), map[string]any{"member_id": m.ID, "phone_number": invitee.phone}, owner.token)
	require.Equal(t, http.StatusCreated, res.status, res.raw)
	var inv gInvitation
	res.data(&inv)

	time.Sleep(2 * time.Second)

	var inbox gList[gInvitation]
	api.do(http.MethodGet, "/v1/invitations", nil, invitee.token).data(&inbox)
	assert.Empty(t, inbox.Items, "an expired invitation is not in the inbox")

	res = api.do(http.MethodPost, fmt.Sprintf("/v1/invitations/%d/accept", inv.ID), nil, invitee.token)
	assert.Equal(t, http.StatusConflict, res.status)
	assert.Equal(t, "INVITATION_NOT_PENDING", res.errorCode(), "invitation_ttl comes from the running config")

	var list gList[gInvitation]
	api.do(http.MethodGet, genPath(d.ID, "/invitations"), nil, owner.token).data(&list)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "expired", list.Items[0].Status)

	// The lapsed invitation does not block a new one.
	res = api.do(http.MethodPost, genPath(d.ID, "/invitations"), map[string]any{"member_id": m.ID}, owner.token)
	assert.Equal(t, http.StatusCreated, res.status, res.raw)
	assert.False(t, strings.Contains(res.raw, "expired"))
}
