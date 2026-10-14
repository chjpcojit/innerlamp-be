package http

import (
	"context"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// genealogyMemberUsecase is the subset of *usecase.GenealogyMemberUsecase this
// handler uses.
type genealogyMemberUsecase interface {
	List(ctx context.Context, userID, genealogyID int64) ([]domain.MemberView, error)
	Get(ctx context.Context, userID, genealogyID, memberID int64) (*domain.MemberView, error)
	Add(ctx context.Context, userID, genealogyID int64, in domain.AddMemberInput) (*domain.MemberResult, error)
	Update(ctx context.Context, userID, genealogyID, memberID int64, info domain.MemberInfo) (*domain.MemberResult, error)
	Delete(ctx context.Context, userID, genealogyID, memberID int64) error
	Move(ctx context.Context, userID, genealogyID, memberID int64, in domain.MoveMemberInput) (*domain.MemberResult, error)
}

// genealogyProfileUsecase is the subset of *usecase.GenealogyProfileUsecase.
type genealogyProfileUsecase interface {
	GetProfile(ctx context.Context, userID, genealogyID, memberID int64) (*domain.MemberProfile, error)
}

// GenealogyMemberHandler serves the people of a tree and their computed profile.
type GenealogyMemberHandler struct {
	members  genealogyMemberUsecase
	profiles genealogyProfileUsecase
}

// NewGenealogyMemberHandler wires the handler.
func NewGenealogyMemberHandler(members genealogyMemberUsecase, profiles genealogyProfileUsecase) *GenealogyMemberHandler {
	return &GenealogyMemberHandler{members: members, profiles: profiles}
}

// addMemberRequest is POST .../members: how the person attaches, plus the
// person's own information at the top level.
type addMemberRequest struct {
	Relation        string `json:"relation"`
	RelatedMemberID int64  `json:"related_member_id"`
	OtherParentID   *int64 `json:"other_parent_id"`
	memberInfoRequest
}

// moveMemberRequest is PUT .../members/{memberId}/relation.
type moveMemberRequest struct {
	Relation        string `json:"relation"`
	RelatedMemberID int64  `json:"related_member_id"`
	OtherParentID   *int64 `json:"other_parent_id"`
	BirthOrder      *int   `json:"birth_order"`
}

// List serves GET /v1/genealogies/{id}/members: the whole tree, flat.
func (h *GenealogyMemberHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	items, err := h.members.List(r.Context(), userID, pathID(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]memberResponse, 0, len(items))
	for i := range items {
		out = append(out, newMemberResponse(&items[i]))
	}
	utils.WriteSuccessResponse(w, newList(out))
}

// Get serves GET /v1/genealogies/{id}/members/{memberId}.
func (h *GenealogyMemberHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	m, err := h.members.Get(r.Context(), userID, pathID(r, "id"), pathID(r, "memberId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newMemberResponse(m))
}

// Add serves POST /v1/genealogies/{id}/members (owner only).
func (h *GenealogyMemberHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req addMemberRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	res, err := h.members.Add(r.Context(), userID, pathID(r, "id"), domain.AddMemberInput{
		Relation: req.Relation, RelatedMemberID: req.RelatedMemberID, OtherParentID: req.OtherParentID,
		Member: req.memberInfoRequest.toInfo(),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, newMemberWriteResponse(res))
}

// Update serves PUT /v1/genealogies/{id}/members/{memberId}: editable fields
// only; relationships have their own route (owner only).
func (h *GenealogyMemberHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req memberInfoRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	res, err := h.members.Update(r.Context(), userID, pathID(r, "id"), pathID(r, "memberId"), req.toInfo())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newMemberWriteResponse(res))
}

// Delete serves DELETE /v1/genealogies/{id}/members/{memberId} (owner only).
func (h *GenealogyMemberHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	if err := h.members.Delete(r.Context(), userID, pathID(r, "id"), pathID(r, "memberId")); err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, nil)
}

// Move serves PUT /v1/genealogies/{id}/members/{memberId}/relation (owner only).
func (h *GenealogyMemberHandler) Move(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req moveMemberRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	res, err := h.members.Move(r.Context(), userID, pathID(r, "id"), pathID(r, "memberId"), domain.MoveMemberInput{
		Relation: req.Relation, RelatedMemberID: req.RelatedMemberID,
		OtherParentID: req.OtherParentID, BirthOrder: req.BirthOrder,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newMemberWriteResponse(res))
}

// --- profile ----------------------------------------------------------------

type memberRefResponse struct {
	ID         int64  `json:"id"`
	FullName   string `json:"full_name"`
	Gender     string `json:"gender"`
	Generation int    `json:"generation"`
	IsAlive    bool   `json:"is_alive"`
}

type siblingResponse struct {
	memberRefResponse
	IsHalfSibling bool `json:"is_half_sibling"`
}

type lineageItemResponse struct {
	ID         int64  `json:"id"`
	FullName   string `json:"full_name"`
	Generation int    `json:"generation"`
}

type chiItemResponse struct {
	Ancestor memberRefResponse `json:"ancestor"`
	Rank     int               `json:"rank"`
	IsEldest bool              `json:"is_eldest"`
}

type kinshipResponse struct {
	Code  string  `json:"code"`
	Label *string `json:"label"`
}

type memberProfileResponse struct {
	Member         memberResponse        `json:"member"`
	MyMemberID     *int64                `json:"my_member_id"`
	Age            *int                  `json:"age"`
	TraditionalAge *int                  `json:"traditional_age"`
	Father         *memberRefResponse    `json:"father"`
	Mother         *memberRefResponse    `json:"mother"`
	Spouses        []memberRefResponse   `json:"spouses"`
	Siblings       []siblingResponse     `json:"siblings"`
	Children       []memberRefResponse   `json:"children"`
	Lineage        []lineageItemResponse `json:"lineage"`
	Chi            *chiItemResponse      `json:"chi"`
	ChiPath        []chiItemResponse     `json:"chi_path"`
	RelationToMe   *kinshipResponse      `json:"relation_to_me"`
}

func newMemberRef(m domain.MemberRef) memberRefResponse {
	return memberRefResponse{ID: m.ID, FullName: m.FullName, Gender: string(m.Gender), Generation: m.Generation, IsAlive: m.IsAlive}
}

func newMemberRefPtr(m *domain.MemberRef) *memberRefResponse {
	if m == nil {
		return nil
	}
	r := newMemberRef(*m)
	return &r
}

func newChiItem(c domain.ChiItem) chiItemResponse {
	return chiItemResponse{Ancestor: newMemberRef(c.Ancestor), Rank: c.Rank, IsEldest: c.IsEldest}
}

func newMemberProfileResponse(p *domain.MemberProfile) memberProfileResponse {
	out := memberProfileResponse{
		Member: newMemberResponse(&p.Member), MyMemberID: p.MyMemberID, Age: p.Age, TraditionalAge: p.TraditionalAge,
		Father: newMemberRefPtr(p.Father), Mother: newMemberRefPtr(p.Mother),
		Spouses:  make([]memberRefResponse, 0, len(p.Spouses)),
		Siblings: make([]siblingResponse, 0, len(p.Siblings)),
		Children: make([]memberRefResponse, 0, len(p.Children)),
		Lineage:  make([]lineageItemResponse, 0, len(p.Lineage)),
		ChiPath:  make([]chiItemResponse, 0, len(p.ChiPath)),
	}
	for _, s := range p.Spouses {
		out.Spouses = append(out.Spouses, newMemberRef(s))
	}
	for _, s := range p.Siblings {
		out.Siblings = append(out.Siblings, siblingResponse{memberRefResponse: newMemberRef(s.MemberRef), IsHalfSibling: s.IsHalfSibling})
	}
	for _, c := range p.Children {
		out.Children = append(out.Children, newMemberRef(c))
	}
	for _, l := range p.Lineage {
		out.Lineage = append(out.Lineage, lineageItemResponse{ID: l.ID, FullName: l.FullName, Generation: l.Generation})
	}
	if p.Chi != nil {
		c := newChiItem(*p.Chi)
		out.Chi = &c
	}
	for _, c := range p.ChiPath {
		out.ChiPath = append(out.ChiPath, newChiItem(c))
	}
	if p.RelationToMe != nil {
		k := kinshipResponse{Code: string(p.RelationToMe.Code)}
		if p.RelationToMe.Label != "" {
			l := p.RelationToMe.Label
			k.Label = &l
		}
		out.RelationToMe = &k
	}
	return out
}

// Profile serves GET /v1/genealogies/{id}/members/{memberId}/profile.
func (h *GenealogyMemberHandler) Profile(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	p, err := h.profiles.GetProfile(r.Context(), userID, pathID(r, "id"), pathID(r, "memberId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newMemberProfileResponse(p))
}
