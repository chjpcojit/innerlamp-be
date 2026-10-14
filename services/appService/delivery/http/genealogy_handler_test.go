package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

type genMux struct {
	mux      *http.ServeMux
	trees    *mockGenealogyUsecase
	members  *mockGenealogyMemberUsecase
	profiles *mockGenealogyProfileUsecase
	access   *mockGenealogyAccessUsecase
	invitee  *mockGenealogyInvitationUsecase
}

// newGenMux registers the routes exactly as cmd/main.go does (minus the auth
// middleware, whose claims the test requests carry directly).
func newGenMux() *genMux {
	g := &genMux{
		trees: new(mockGenealogyUsecase), members: new(mockGenealogyMemberUsecase),
		profiles: new(mockGenealogyProfileUsecase), access: new(mockGenealogyAccessUsecase),
		invitee: new(mockGenealogyInvitationUsecase),
	}
	th := NewGenealogyHandler(g.trees)
	mh := NewGenealogyMemberHandler(g.members, g.profiles)
	ah := NewGenealogyAccessHandler(g.access)
	ih := NewGenealogyInvitationHandler(g.invitee)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/genealogies", th.Create)
	mux.HandleFunc("GET /v1/genealogies", th.List)
	mux.HandleFunc("GET /v1/genealogies/{id}", th.Get)
	mux.HandleFunc("PUT /v1/genealogies/{id}", th.Update)
	mux.HandleFunc("DELETE /v1/genealogies/{id}", th.Delete)
	mux.HandleFunc("PUT /v1/genealogies/{id}/chronicle", th.UpdateChronicle)
	mux.HandleFunc("GET /v1/genealogies/{id}/members", mh.List)
	mux.HandleFunc("POST /v1/genealogies/{id}/members", mh.Add)
	mux.HandleFunc("GET /v1/genealogies/{id}/members/{memberId}", mh.Get)
	mux.HandleFunc("PUT /v1/genealogies/{id}/members/{memberId}", mh.Update)
	mux.HandleFunc("DELETE /v1/genealogies/{id}/members/{memberId}", mh.Delete)
	mux.HandleFunc("PUT /v1/genealogies/{id}/members/{memberId}/relation", mh.Move)
	mux.HandleFunc("GET /v1/genealogies/{id}/members/{memberId}/profile", mh.Profile)
	mux.HandleFunc("POST /v1/genealogies/{id}/invitations", ah.Invite)
	mux.HandleFunc("GET /v1/genealogies/{id}/invitations", ah.ListInvitations)
	mux.HandleFunc("DELETE /v1/genealogies/{id}/invitations/{invitationId}", ah.CancelInvitation)
	mux.HandleFunc("GET /v1/genealogies/{id}/viewers", ah.ListViewers)
	mux.HandleFunc("DELETE /v1/genealogies/{id}/viewers/{userId}", ah.RemoveViewer)
	mux.HandleFunc("GET /v1/invitations", ih.ListMine)
	mux.HandleFunc("POST /v1/invitations/{id}/accept", ih.Accept)
	mux.HandleFunc("POST /v1/invitations/{id}/decline", ih.Decline)
	g.mux = mux
	return g
}

func sampleDetail() *domain.GenealogyDetail {
	mid := int64(11)
	head := int64(3)
	return &domain.GenealogyDetail{
		Genealogy:        domain.Genealogy{ID: 42, Name: "Họ Nguyễn", AncestralHome: "Nam Định", CreatedAt: time.Unix(0, 0).UTC(), UpdatedAt: time.Unix(0, 0).UTC()},
		MyRole:           domain.RoleViewer,
		MyMemberID:       &mid,
		ClanHeadMemberID: &head,
		FounderName:      "Tổ",
		GenerationsCount: 3,
		MembersCount:     7,
		Chronicle: domain.Chronicle{
			ChapYear: intp(2024), ChapMonth: intp(2), ChapDay: intp(10),
			ChapLunar: &domain.LunarDate{Day: 1, Month: 1},
		},
	}
}

func TestGenealogyHandler_TreeRoutes(t *testing.T) {
	t.Run("create returns 201 with the overview", func(t *testing.T) {
		g := newGenMux()
		d := sampleDetail()
		d.MyRole = domain.RoleOwner
		g.trees.On("Create", mock.Anything, int64(7), mock.MatchedBy(func(in domain.CreateGenealogyInput) bool {
			return in.Name == "Họ Nguyễn" && in.AncestralHome == "Nam Định" &&
				in.Founder.FullName == "Tổ" && in.Founder.Gender != nil && *in.Founder.Gender == domain.GenderMale &&
				len(in.Spouses) == 1 && in.Spouses[0].FullName == "Vợ"
		})).Return(d, nil)

		rec := serve(g.mux, memorialReq("POST", "/v1/genealogies", `{
			"name":"Họ Nguyễn","ancestral_home":"Nam Định",
			"founder":{"full_name":"Tổ","gender":"male","is_alive":true},
			"spouses":[{"full_name":"Vợ","is_alive":true}]}`, "7"))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, "Họ Nguyễn", data["name"])
		assert.Equal(t, "owner", data["my_role"])
		assert.EqualValues(t, 11, data["my_member_id"])
		assert.EqualValues(t, 3, data["clan_head_member_id"])
		ch := data["chronicle"].(map[string]any)
		assert.Equal(t, "Tổ", ch["founder_name"])
		assert.EqualValues(t, 7, ch["members_count"])
		assert.EqualValues(t, 1, ch["chap_lunar_day"], "the server-derived lunar date is returned")
		assert.Nil(t, ch["surname"], "absent optional values are null, not omitted")
	})

	t.Run("unknown and server-owned fields are rejected", func(t *testing.T) {
		g := newGenMux()
		for _, body := range []string{
			`{"name":"x","ancestral_home":"y","founder":{"full_name":"a","gender":"male","is_alive":true},"owner_user_id":1}`,
			`{"name":"x","ancestral_home":"y","founder":{"full_name":"a","gender":"male","is_alive":true,"generation":1}}`,
			`{"name":"x","ancestral_home":"y","founder":{"full_name":"a","gender":"male","is_alive":true,"kind":"blood"}}`,
			``,
			`not json`,
		} {
			rec := serve(g.mux, memorialReq("POST", "/v1/genealogies", body, "7"))
			assert.Equal(t, http.StatusBadRequest, rec.Code, body)
		}
		g.trees.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a validation error carries the field", func(t *testing.T) {
		g := newGenMux()
		g.trees.On("Create", mock.Anything, int64(7), mock.Anything).
			Return(nil, &domain.GenealogyValidationError{Field: "founder.gender", Message: "the founder must be male"})
		rec := serve(g.mux, memorialReq("POST", "/v1/genealogies", `{"name":"x","ancestral_home":"y","founder":{"full_name":"a","gender":"female","is_alive":true}}`, "7"))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		body := decode(t, rec)
		assert.Equal(t, "VALIDATION_ERROR", body["error"].(map[string]any)["code"])
		assert.Equal(t, "founder.gender", body["details"].(map[string]any)["field"])
	})

	t.Run("no access token is 401", func(t *testing.T) {
		g := newGenMux()
		rec := serve(g.mux, memorialReq("GET", "/v1/genealogies", "", ""))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		g.trees.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
	})

	t.Run("list is never null", func(t *testing.T) {
		g := newGenMux()
		g.trees.On("List", mock.Anything, int64(7)).Return(nil, nil)
		rec := serve(g.mux, memorialReq("GET", "/v1/genealogies", "", "7"))
		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, []any{}, data["items"])
		assert.EqualValues(t, 0, data["total"])
	})

	t.Run("get passes the numeric id; a malformed id reaches the use case as 0", func(t *testing.T) {
		g := newGenMux()
		g.trees.On("Get", mock.Anything, int64(7), int64(42)).Return(sampleDetail(), nil)
		g.trees.On("Get", mock.Anything, int64(7), int64(0)).Return(nil, domain.ErrGenealogyNotFound)

		assert.Equal(t, http.StatusOK, serve(g.mux, memorialReq("GET", "/v1/genealogies/42", "", "7")).Code)
		for _, id := range []string{"abc", "-3", "0", "1.5"} {
			rec := serve(g.mux, memorialReq("GET", "/v1/genealogies/"+id, "", "7"))
			assert.Equal(t, http.StatusNotFound, rec.Code, id)
			assert.Equal(t, "GENEALOGY_NOT_FOUND", errCode(t, rec), id)
		}
	})

	t.Run("a viewer writing is 403 GENEALOGY_FORBIDDEN", func(t *testing.T) {
		g := newGenMux()
		g.trees.On("UpdateInfo", mock.Anything, int64(7), int64(42), "x", "y").Return(nil, domain.ErrGenealogyForbidden)
		g.trees.On("Delete", mock.Anything, int64(7), int64(42)).Return(domain.ErrGenealogyForbidden)
		for _, tt := range []struct{ method, target, body string }{
			{"PUT", "/v1/genealogies/42", `{"name":"x","ancestral_home":"y"}`},
			{"DELETE", "/v1/genealogies/42", ""},
		} {
			rec := serve(g.mux, memorialReq(tt.method, tt.target, tt.body, "7"))
			assert.Equal(t, http.StatusForbidden, rec.Code, tt.method)
			assert.Equal(t, "GENEALOGY_FORBIDDEN", errCode(t, rec))
		}
	})

	t.Run("create at the limit is 409", func(t *testing.T) {
		g := newGenMux()
		g.trees.On("Create", mock.Anything, int64(7), mock.Anything).Return(nil, domain.ErrGenealogyLimitReached)
		rec := serve(g.mux, memorialReq("POST", "/v1/genealogies", `{"name":"x","ancestral_home":"y","founder":{"full_name":"a","gender":"male","is_alive":true}}`, "7"))
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "GENEALOGY_LIMIT_REACHED", errCode(t, rec))
	})

	t.Run("delete answers success without data", func(t *testing.T) {
		g := newGenMux()
		g.trees.On("Delete", mock.Anything, int64(7), int64(42)).Return(nil)
		rec := serve(g.mux, memorialReq("DELETE", "/v1/genealogies/42", "", "7"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, true, decode(t, rec)["success"])
	})

	t.Run("chronicle: writable fields in, derived fields out and refused in", func(t *testing.T) {
		g := newGenMux()
		g.trees.On("UpdateChronicle", mock.Anything, int64(7), int64(42), mock.MatchedBy(func(c domain.Chronicle) bool {
			return c.Surname != nil && *c.Surname == "Nguyễn" && c.ChapYear != nil && *c.ChapYear == 2026 && c.ChapEnteredAsLunar &&
				c.CoupletUpperReading != nil
		})).Return(sampleDetail(), nil)

		rec := serve(g.mux, memorialReq("PUT", "/v1/genealogies/42/chronicle",
			`{"surname":"Nguyễn","chap_year":2026,"chap_month":1,"chap_day":31,"chap_entered_as_lunar":true,"couplet_upper_reading":"thượng liên"}`, "7"))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 1, data["chap_lunar_month"])

		for _, derived := range []string{`{"founder_name":"x"}`, `{"members_count":3}`, `{"chap_lunar_day":1}`} {
			rec := serve(g.mux, memorialReq("PUT", "/v1/genealogies/42/chronicle", derived, "7"))
			assert.Equal(t, http.StatusBadRequest, rec.Code, derived)
		}
	})
}

func TestGenealogyMemberHandler(t *testing.T) {
	label := domain.InLawLabelDau
	view := func(id int64) domain.MemberView {
		return domain.MemberView{GenealogyMember: domain.GenealogyMember{
			ID: id, Kind: domain.MemberKindInLaw, Gender: domain.GenderFemale, FullName: "Vợ", IsAlive: true,
			SpouseOfID: i64p(3), Generation: 2, DeathLunar: &domain.LunarDate{Day: 15, Month: 7, Leap: true},
		}, InLawLabel: &label}
	}

	t.Run("list is the whole tree, flat, with derived fields", func(t *testing.T) {
		g := newGenMux()
		v := view(9)
		g.members.On("List", mock.Anything, int64(7), int64(42)).Return([]domain.MemberView{v}, nil)
		rec := serve(g.mux, memorialReq("GET", "/v1/genealogies/42/members", "", "7"))
		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 1, data["total"])
		m := data["items"].([]any)[0].(map[string]any)
		assert.Equal(t, "in_law", m["kind"])
		assert.Equal(t, "DAU", m["in_law_label"])
		assert.EqualValues(t, 15, m["death_lunar_day"])
		assert.Equal(t, true, m["death_lunar_leap"])
		assert.Nil(t, m["parent_id"])
		assert.EqualValues(t, 3, m["spouse_of_id"])
		_, hasWarnings := m["warnings"]
		assert.False(t, hasWarnings, "GET never carries warnings")
	})

	t.Run("add returns 201 with warnings and decodes the flat body", func(t *testing.T) {
		g := newGenMux()
		res := &domain.MemberResult{
			MemberView: domain.MemberView{GenealogyMember: domain.GenealogyMember{ID: 50, Kind: domain.MemberKindBlood, Gender: domain.GenderMale, FullName: "Con", IsAlive: true, Generation: 3}},
			Warnings:   []domain.MemberWarning{{Code: domain.WarningChildBornBeforeParent, MemberID: 50, RelatedMemberID: 3}},
		}
		g.members.On("Add", mock.Anything, int64(7), int64(42), mock.MatchedBy(func(in domain.AddMemberInput) bool {
			return in.Relation == "child" && in.RelatedMemberID == 3 && in.OtherParentID != nil && *in.OtherParentID == 6 &&
				in.Member.FullName == "Con" && in.Member.Gender != nil && *in.Member.Gender == domain.GenderMale &&
				in.Member.BirthYear != nil && *in.Member.BirthYear == 1945
		})).Return(res, nil)

		rec := serve(g.mux, memorialReq("POST", "/v1/genealogies/42/members",
			`{"relation":"child","related_member_id":3,"other_parent_id":6,"full_name":"Con","gender":"male","is_alive":true,"birth_year":1945}`, "7"))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 50, data["id"])
		w := data["warnings"].([]any)
		require.Len(t, w, 1)
		assert.Equal(t, "CHILD_BORN_BEFORE_PARENT", w[0].(map[string]any)["code"])
	})

	t.Run("add: server-owned fields are not accepted", func(t *testing.T) {
		g := newGenMux()
		for _, body := range []string{
			`{"relation":"child","related_member_id":3,"full_name":"x","gender":"male","is_alive":true,"parent_id":3}`,
			`{"relation":"child","related_member_id":3,"full_name":"x","gender":"male","is_alive":true,"generation":9}`,
			`{"relation":"child","related_member_id":3,"full_name":"x","gender":"male","is_alive":true,"kind":"in_law"}`,
			`{"relation":"child","related_member_id":3,"full_name":"x","gender":"male","is_alive":true,"spouse_of_id":3}`,
		} {
			rec := serve(g.mux, memorialReq("POST", "/v1/genealogies/42/members", body, "7"))
			assert.Equal(t, http.StatusBadRequest, rec.Code, body)
		}
	})

	t.Run("update refuses relationship fields (they have their own route)", func(t *testing.T) {
		g := newGenMux()
		for _, body := range []string{
			`{"full_name":"x","gender":"male","is_alive":true,"relation":"child"}`,
			`{"full_name":"x","gender":"male","is_alive":true,"related_member_id":3}`,
			`{"full_name":"x","gender":"male","is_alive":true,"parent_id":3}`,
			`{"full_name":"x","gender":"male","is_alive":true,"other_parent_id":3}`,
		} {
			rec := serve(g.mux, memorialReq("PUT", "/v1/genealogies/42/members/7", body, "7"))
			assert.Equal(t, http.StatusBadRequest, rec.Code, body)
		}
	})

	t.Run("move uses the relation route", func(t *testing.T) {
		g := newGenMux()
		res := &domain.MemberResult{MemberView: domain.MemberView{GenealogyMember: domain.GenealogyMember{ID: 7, Kind: domain.MemberKindBlood, Gender: domain.GenderMale, Generation: 4}}}
		g.members.On("Move", mock.Anything, int64(7), int64(42), int64(7), domain.MoveMemberInput{
			Relation: "child", RelatedMemberID: 4, BirthOrder: intp(2),
		}).Return(res, nil)
		rec := serve(g.mux, memorialReq("PUT", "/v1/genealogies/42/members/7/relation",
			`{"relation":"child","related_member_id":4,"birth_order":2}`, "7"))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.EqualValues(t, 4, decode(t, rec)["data"].(map[string]any)["generation"])
	})

	t.Run("errors map to the contract codes", func(t *testing.T) {
		cases := []struct {
			name     string
			err      error
			wantCode int
			wantErr  string
		}{
			{"member not found", domain.ErrMemberNotFound, http.StatusNotFound, "MEMBER_NOT_FOUND"},
			{"descendants", domain.ErrMemberHasDescendants, http.StatusConflict, "MEMBER_HAS_DESCENDANTS"},
			{"founder delete", domain.ErrFounderNotDeletable, http.StatusConflict, "FOUNDER_NOT_DELETABLE"},
			{"founder move", domain.ErrFounderNotMovable, http.StatusConflict, "FOUNDER_NOT_MOVABLE"},
			{"member cap", domain.ErrMemberLimitReached, http.StatusConflict, "MEMBER_LIMIT_REACHED"},
			{"phone used", domain.ErrMemberPhoneInUse, http.StatusConflict, "MEMBER_PHONE_ALREADY_USED"},
			{"stranger", domain.ErrGenealogyNotFound, http.StatusNotFound, "GENEALOGY_NOT_FOUND"},
			{"viewer", domain.ErrGenealogyForbidden, http.StatusForbidden, "GENEALOGY_FORBIDDEN"},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				g := newGenMux()
				g.members.On("Delete", mock.Anything, int64(7), int64(42), int64(7)).Return(tt.err)
				rec := serve(g.mux, memorialReq("DELETE", "/v1/genealogies/42/members/7", "", "7"))
				assert.Equal(t, tt.wantCode, rec.Code)
				assert.Equal(t, tt.wantErr, errCode(t, rec))
			})
		}
	})

	t.Run("profile renders the computed fields and null label for UNKNOWN", func(t *testing.T) {
		g := newGenMux()
		mid := int64(11)
		age := 36
		mu := 37
		p := &domain.MemberProfile{
			Member:     domain.MemberView{GenealogyMember: domain.GenealogyMember{ID: 14, Kind: domain.MemberKindBlood, Gender: domain.GenderMale, FullName: "Z", IsAlive: true, Generation: 3}},
			MyMemberID: &mid, Age: &age, TraditionalAge: &mu,
			Father:       &domain.MemberRef{ID: 3, FullName: "Lộc", Gender: domain.GenderMale, Generation: 2, IsAlive: true},
			Siblings:     []domain.SiblingRef{{MemberRef: domain.MemberRef{ID: 7, FullName: "A"}, IsHalfSibling: true}},
			Lineage:      []domain.LineageItem{{ID: 1, FullName: "Tổ", Generation: 1}},
			Chi:          &domain.ChiItem{Ancestor: domain.MemberRef{ID: 14, FullName: "Z"}, Rank: 1, IsEldest: true},
			ChiPath:      []domain.ChiItem{{Ancestor: domain.MemberRef{ID: 14, FullName: "Z"}, Rank: 1, IsEldest: true}},
			RelationToMe: &domain.Kinship{Code: domain.KinshipUnknown},
		}
		g.profiles.On("GetProfile", mock.Anything, int64(7), int64(42), int64(14)).Return(p, nil)
		rec := serve(g.mux, memorialReq("GET", "/v1/genealogies/42/members/14/profile", "", "7"))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 36, data["age"])
		assert.EqualValues(t, 37, data["traditional_age"], "tuổi mụ is returned beside the Western age")
		assert.Nil(t, data["mother"])
		assert.Equal(t, []any{}, data["children"])
		assert.Equal(t, []any{}, data["spouses"])
		rel := data["relation_to_me"].(map[string]any)
		assert.Equal(t, "UNKNOWN", rel["code"])
		assert.Nil(t, rel["label"])
		sib := data["siblings"].([]any)[0].(map[string]any)
		assert.Equal(t, true, sib["is_half_sibling"])
		assert.Len(t, data["chi_path"].([]any), 1)
	})
}

func TestGenealogyAccessHandlers(t *testing.T) {
	t.Run("invite returns 201 with a pending invitation", func(t *testing.T) {
		g := newGenMux()
		g.access.On("Invite", mock.Anything, int64(7), int64(42), domain.InviteInput{MemberID: 11, PhoneNumber: "0912345678"}).
			Return(&domain.Invitation{ID: 5, MemberID: 11, MemberFullName: "A", PhoneNumber: "+84912345678", Status: domain.InvitationPending}, nil)
		rec := serve(g.mux, memorialReq("POST", "/v1/genealogies/42/invitations", `{"member_id":11,"phone_number":"0912345678"}`, "7"))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		data := decode(t, rec)["data"].(map[string]any)
		assert.Equal(t, "pending", data["status"])
		assert.EqualValues(t, 11, data["member_id"])
		assert.Nil(t, data["responded_at"])
	})

	t.Run("invite without a number passes an empty number", func(t *testing.T) {
		g := newGenMux()
		g.access.On("Invite", mock.Anything, int64(7), int64(42), domain.InviteInput{MemberID: 11}).
			Return(&domain.Invitation{ID: 5, Status: domain.InvitationPending}, nil)
		rec := serve(g.mux, memorialReq("POST", "/v1/genealogies/42/invitations", `{"member_id":11}`, "7"))
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("invite conflicts", func(t *testing.T) {
		for _, tt := range []struct {
			err      error
			status   int
			wantCode string
		}{
			{domain.ErrViewerExists, http.StatusConflict, "VIEWER_ALREADY_EXISTS"},
			{domain.ErrInvitationAlreadyPending, http.StatusConflict, "INVITATION_ALREADY_PENDING"},
			{domain.ErrViewerLimitReached, http.StatusConflict, "VIEWER_LIMIT_REACHED"},
			{domain.ErrMemberPhoneInUse, http.StatusConflict, "MEMBER_PHONE_ALREADY_USED"},
			{domain.ErrMemberNotFound, http.StatusNotFound, "MEMBER_NOT_FOUND"},
		} {
			g := newGenMux()
			g.access.On("Invite", mock.Anything, int64(7), int64(42), mock.Anything).Return(nil, tt.err)
			rec := serve(g.mux, memorialReq("POST", "/v1/genealogies/42/invitations", `{"member_id":11,"phone_number":"0912345678"}`, "7"))
			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, tt.wantCode, errCode(t, rec))
		}
	})

	t.Run("invitation list and cancel", func(t *testing.T) {
		g := newGenMux()
		now := time.Now().UTC()
		g.access.On("ListInvitations", mock.Anything, int64(7), int64(42)).Return([]domain.Invitation{
			{ID: 5, MemberID: 11, MemberFullName: "A", PhoneNumber: "+84912345678", Status: domain.InvitationAccepted, CreatedAt: now, ExpiresAt: now, RespondedAt: &now},
		}, nil)
		g.access.On("CancelInvitation", mock.Anything, int64(7), int64(42), int64(5)).Return(nil)
		g.access.On("CancelInvitation", mock.Anything, int64(7), int64(42), int64(0)).Return(domain.ErrInvitationNotFound)

		rec := serve(g.mux, memorialReq("GET", "/v1/genealogies/42/invitations", "", "7"))
		require.Equal(t, http.StatusOK, rec.Code)
		item := decode(t, rec)["data"].(map[string]any)["items"].([]any)[0].(map[string]any)
		assert.Equal(t, "accepted", item["status"])

		assert.Equal(t, http.StatusOK, serve(g.mux, memorialReq("DELETE", "/v1/genealogies/42/invitations/5", "", "7")).Code)
		rec = serve(g.mux, memorialReq("DELETE", "/v1/genealogies/42/invitations/abc", "", "7"))
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "INVITATION_NOT_FOUND", errCode(t, rec))
	})

	t.Run("viewers: list shows who stands where, remove, leave", func(t *testing.T) {
		g := newGenMux()
		g.access.On("ListViewers", mock.Anything, int64(7), int64(42)).Return([]domain.Viewer{
			{UserID: 9, PhoneNumber: "+84900000009", MemberID: i64p(11), CreatedAt: time.Now()},
			{UserID: 10, PhoneNumber: "+84900000010", CreatedAt: time.Now()},
		}, nil)
		g.access.On("RemoveViewer", mock.Anything, int64(7), int64(42), int64(9)).Return(nil)
		g.access.On("RemoveViewer", mock.Anything, int64(7), int64(42), int64(0)).Return(domain.ErrViewerNotFound)

		rec := serve(g.mux, memorialReq("GET", "/v1/genealogies/42/viewers", "", "7"))
		require.Equal(t, http.StatusOK, rec.Code)
		items := decode(t, rec)["data"].(map[string]any)["items"].([]any)
		assert.EqualValues(t, 11, items[0].(map[string]any)["member_id"])
		assert.Nil(t, items[1].(map[string]any)["member_id"])

		assert.Equal(t, http.StatusOK, serve(g.mux, memorialReq("DELETE", "/v1/genealogies/42/viewers/9", "", "7")).Code)
		rec = serve(g.mux, memorialReq("DELETE", "/v1/genealogies/42/viewers/zzz", "", "7"))
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "VIEWER_NOT_FOUND", errCode(t, rec))
	})

	t.Run("a viewer calling an owner-only route is 403", func(t *testing.T) {
		g := newGenMux()
		g.access.On("ListViewers", mock.Anything, int64(7), int64(42)).Return(nil, domain.ErrGenealogyForbidden)
		rec := serve(g.mux, memorialReq("GET", "/v1/genealogies/42/viewers", "", "7"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Equal(t, "GENEALOGY_FORBIDDEN", errCode(t, rec))
	})
}

func TestGenealogyInvitationHandler(t *testing.T) {
	t.Run("inbox shows only the tree name, the member and the inviter", func(t *testing.T) {
		g := newGenMux()
		now := time.Now().UTC()
		g.invitee.On("ListMine", mock.Anything, int64(7)).Return([]domain.Invitation{{
			ID: 5, GenealogyName: "Họ Nguyễn", MemberFullName: "Nguyễn Văn B", InvitedByPhone: "+84900000001",
			MemberID: 11, GenealogyID: 42, PhoneNumber: "+84900000007", CreatedAt: now, ExpiresAt: now,
		}}, nil)
		rec := serve(g.mux, memorialReq("GET", "/v1/invitations", "", "7"))
		require.Equal(t, http.StatusOK, rec.Code)
		item := decode(t, rec)["data"].(map[string]any)["items"].([]any)[0].(map[string]any)
		assert.Equal(t, "Họ Nguyễn", item["genealogy_name"])
		assert.Equal(t, "Nguyễn Văn B", item["member_full_name"])
		assert.Equal(t, "+84900000001", item["invited_by_phone"])
		for _, leaked := range []string{"genealogy_id", "member_id", "phone_number", "invited_by"} {
			_, present := item[leaked]
			assert.False(t, present, "%s must not appear in the inbox before acceptance", leaked)
		}
	})

	t.Run("accept returns the tree, the member and the viewer role", func(t *testing.T) {
		g := newGenMux()
		g.invitee.On("Accept", mock.Anything, int64(7), int64(5)).Return(&domain.AcceptResult{GenealogyID: 42, MemberID: 11, Role: domain.RoleViewer}, nil)
		rec := serve(g.mux, memorialReq("POST", "/v1/invitations/5/accept", "", "7"))
		require.Equal(t, http.StatusOK, rec.Code)
		data := decode(t, rec)["data"].(map[string]any)
		assert.EqualValues(t, 42, data["genealogy_id"])
		assert.EqualValues(t, 11, data["member_id"])
		assert.Equal(t, "viewer", data["role"])
	})

	t.Run("accept and decline error codes", func(t *testing.T) {
		for _, tt := range []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrInvitationNotFound, http.StatusNotFound, "INVITATION_NOT_FOUND"},
			{domain.ErrInvitationNotPending, http.StatusConflict, "INVITATION_NOT_PENDING"},
			{domain.ErrViewerExists, http.StatusConflict, "VIEWER_ALREADY_EXISTS"},
			{domain.ErrViewerLimitReached, http.StatusConflict, "VIEWER_LIMIT_REACHED"},
		} {
			g := newGenMux()
			g.invitee.On("Accept", mock.Anything, int64(7), int64(5)).Return(nil, tt.err)
			g.invitee.On("Decline", mock.Anything, int64(7), int64(5)).Return(tt.err)
			rec := serve(g.mux, memorialReq("POST", "/v1/invitations/5/accept", "", "7"))
			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, tt.code, errCode(t, rec))
			rec = serve(g.mux, memorialReq("POST", "/v1/invitations/5/decline", "", "7"))
			assert.Equal(t, tt.status, rec.Code)
		}
	})

	t.Run("decline answers success; a malformed id is not found", func(t *testing.T) {
		g := newGenMux()
		g.invitee.On("Decline", mock.Anything, int64(7), int64(5)).Return(nil)
		g.invitee.On("Accept", mock.Anything, int64(7), int64(0)).Return(nil, domain.ErrInvitationNotFound)
		assert.Equal(t, http.StatusOK, serve(g.mux, memorialReq("POST", "/v1/invitations/5/decline", "", "7")).Code)
		rec := serve(g.mux, memorialReq("POST", "/v1/invitations/x/accept", "", "7"))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func i64p(v int64) *int64 { return &v }

func intp(v int) *int { return &v }

func TestGenealogyRoutes_WrongMethodIs405(t *testing.T) {
	g := newGenMux()
	for _, tt := range []struct{ method, target string }{
		{"PATCH", "/v1/genealogies/42"},
		{"POST", "/v1/genealogies/42/chronicle"},
		{"GET", "/v1/genealogies/42/members/7/relation"},
		{"PUT", "/v1/genealogies/42/viewers"},
		{"DELETE", "/v1/invitations"},
		{"GET", "/v1/invitations/5/accept"},
	} {
		rec := serve(g.mux, memorialReq(tt.method, tt.target, "", "7"))
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code, "%s %s", tt.method, tt.target)
	}
}
