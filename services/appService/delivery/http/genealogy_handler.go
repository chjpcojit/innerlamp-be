package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Genealogy (gia phả) HTTP layer — docs/specs/features/genealogy/ v5.6.
//
// Every route is wrapped by the JWT auth middleware; the acting user is read
// from the token's claims (currentUserID), never from the path or body. Path
// ids that are not positive integers are passed to the use case as 0, which
// answers with the right "not found" without touching the database — and only
// after the caller's role was resolved, so a stranger learns nothing from a
// malformed id.

// pathID parses a path value. An id that is not a positive integer becomes 0.
func pathID(r *http.Request, name string) int64 {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id < 1 {
		return 0
	}
	return id
}

// --- shared wire shapes -----------------------------------------------------

// memberInfoRequest is the information about one person wherever a member is
// submitted: the founder, a spouse, POST and PUT of a member. Server-owned
// fields (id, kind, parent_id, generation...) are deliberately absent, so the
// strict decoder rejects them as unknown fields (AC11).
type memberInfoRequest struct {
	FullName            string  `json:"full_name"`
	Gender              *string `json:"gender"`
	IsAlive             *bool   `json:"is_alive"`
	BirthYear           *int    `json:"birth_year"`
	DeathYear           *int    `json:"death_year"`
	Residence           *string `json:"residence"`
	DharmaName          *string `json:"dharma_name"`
	TabletLocation      *string `json:"tablet_location"`
	BurialPlace         *string `json:"burial_place"`
	DeathMonth          *int    `json:"death_month"`
	DeathDay            *int    `json:"death_day"`
	DeathEnteredAsLunar *bool   `json:"death_entered_as_lunar"`
	PhoneNumber         *string `json:"phone_number"`
	Note                *string `json:"note"`
	IsClanHead          *bool   `json:"is_clan_head"`
	BirthOrder          *int    `json:"birth_order"`
}

func (r memberInfoRequest) toInfo() domain.MemberInfo {
	info := domain.MemberInfo{
		FullName: r.FullName, IsAlive: r.IsAlive, BirthYear: r.BirthYear, DeathYear: r.DeathYear,
		Residence: r.Residence, DharmaName: r.DharmaName, TabletLocation: r.TabletLocation,
		BurialPlace: r.BurialPlace, DeathMonth: r.DeathMonth, DeathDay: r.DeathDay,
		DeathEnteredAsLunar: r.DeathEnteredAsLunar, PhoneNumber: r.PhoneNumber, Note: r.Note,
		IsClanHead: r.IsClanHead, BirthOrder: r.BirthOrder,
	}
	if r.Gender != nil {
		g := domain.Gender(*r.Gender)
		info.Gender = &g
	}
	return info
}

type lunarDateResponse struct {
	Day   int  `json:"day"`
	Month int  `json:"month"`
	Leap  bool `json:"leap"`
}

// memberResponse is the Member wire shape. Optional values are JSON null, never
// absent or empty strings.
type memberResponse struct {
	ID                  int64     `json:"id"`
	Kind                string    `json:"kind"`
	FullName            string    `json:"full_name"`
	Gender              string    `json:"gender"`
	BirthYear           *int      `json:"birth_year"`
	IsAlive             bool      `json:"is_alive"`
	DeathYear           *int      `json:"death_year"`
	Residence           *string   `json:"residence"`
	DharmaName          *string   `json:"dharma_name"`
	TabletLocation      *string   `json:"tablet_location"`
	BurialPlace         *string   `json:"burial_place"`
	DeathMonth          *int      `json:"death_month"`
	DeathDay            *int      `json:"death_day"`
	DeathEnteredAsLunar *bool     `json:"death_entered_as_lunar"`
	DeathLunarDay       *int      `json:"death_lunar_day"`
	DeathLunarMonth     *int      `json:"death_lunar_month"`
	DeathLunarLeap      *bool     `json:"death_lunar_leap"`
	PhoneNumber         *string   `json:"phone_number"`
	Note                *string   `json:"note"`
	IsClanHead          bool      `json:"is_clan_head"`
	ParentID            *int64    `json:"parent_id"`
	OtherParentID       *int64    `json:"other_parent_id"`
	BirthOrder          *int      `json:"birth_order"`
	SpouseOfID          *int64    `json:"spouse_of_id"`
	Generation          int       `json:"generation"`
	InLawLabel          *string   `json:"in_law_label"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func newMemberResponse(v *domain.MemberView) memberResponse {
	m := &v.GenealogyMember
	r := memberResponse{
		ID: m.ID, Kind: string(m.Kind), FullName: m.FullName, Gender: string(m.Gender),
		BirthYear: m.BirthYear, IsAlive: m.IsAlive, DeathYear: m.DeathYear, Residence: m.Residence,
		DharmaName: m.DharmaName, TabletLocation: m.TabletLocation, BurialPlace: m.BurialPlace,
		DeathMonth: m.DeathMonth, DeathDay: m.DeathDay, DeathEnteredAsLunar: m.DeathEnteredAsLunar,
		PhoneNumber: m.PhoneNumber, Note: m.Note, IsClanHead: m.IsClanHead,
		ParentID: m.ParentID, OtherParentID: m.OtherParentID, BirthOrder: m.BirthOrder,
		SpouseOfID: m.SpouseOfID, Generation: m.Generation, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
	if m.DeathLunar != nil {
		day, month, leap := m.DeathLunar.Day, m.DeathLunar.Month, m.DeathLunar.Leap
		r.DeathLunarDay, r.DeathLunarMonth, r.DeathLunarLeap = &day, &month, &leap
	}
	if v.InLawLabel != nil {
		l := string(*v.InLawLabel)
		r.InLawLabel = &l
	}
	return r
}

type warningResponse struct {
	Code            string `json:"code"`
	MemberID        int64  `json:"member_id"`
	RelatedMemberID int64  `json:"related_member_id"`
}

// memberWriteResponse is a Member plus the advisory warnings of the write that
// produced it (AC27). GET never carries warnings.
type memberWriteResponse struct {
	memberResponse
	Warnings []warningResponse `json:"warnings"`
}

func newMemberWriteResponse(res *domain.MemberResult) memberWriteResponse {
	w := make([]warningResponse, 0, len(res.Warnings))
	for _, x := range res.Warnings {
		w = append(w, warningResponse{Code: x.Code, MemberID: x.MemberID, RelatedMemberID: x.RelatedMemberID})
	}
	return memberWriteResponse{memberResponse: newMemberResponse(&res.MemberView), Warnings: w}
}

type listResponse[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

func newList[T any](items []T) listResponse[T] {
	if items == nil {
		items = []T{}
	}
	return listResponse[T]{Items: items, Total: len(items)}
}

// --- trees and chronicle ----------------------------------------------------

// genealogyUsecase is the subset of *usecase.GenealogyUsecase this handler uses.
type genealogyUsecase interface {
	Create(ctx context.Context, userID int64, in domain.CreateGenealogyInput) (*domain.GenealogyDetail, error)
	List(ctx context.Context, userID int64) ([]domain.GenealogySummary, error)
	Get(ctx context.Context, userID, genealogyID int64) (*domain.GenealogyDetail, error)
	UpdateInfo(ctx context.Context, userID, genealogyID int64, name, ancestralHome string) (*domain.GenealogyDetail, error)
	Delete(ctx context.Context, userID, genealogyID int64) error
	UpdateChronicle(ctx context.Context, userID, genealogyID int64, in domain.Chronicle) (*domain.GenealogyDetail, error)
}

// GenealogyHandler serves the tree itself and its chronicle.
type GenealogyHandler struct {
	usecase genealogyUsecase
}

// NewGenealogyHandler wires the handler.
func NewGenealogyHandler(usecase genealogyUsecase) *GenealogyHandler {
	return &GenealogyHandler{usecase: usecase}
}

type createGenealogyRequest struct {
	Name          string              `json:"name"`
	AncestralHome string              `json:"ancestral_home"`
	Founder       memberInfoRequest   `json:"founder"`
	Spouses       []memberInfoRequest `json:"spouses"`
}

type updateGenealogyRequest struct {
	Name          string `json:"name"`
	AncestralHome string `json:"ancestral_home"`
}

// chronicleRequest carries only the writable chronicle fields. The derived ones
// (founder_name, counts, *_lunar_*) are absent, so sending them is rejected.
type chronicleRequest struct {
	Surname             *string `json:"surname"`
	CurrentLocation     *string `json:"current_location"`
	SinceYear           *int    `json:"since_year"`
	Origin              *string `json:"origin"`
	History             *string `json:"history"`
	HallName            *string `json:"hall_name"`
	HallAddress         *string `json:"hall_address"`
	ChapYear            *int    `json:"chap_year"`
	ChapMonth           *int    `json:"chap_month"`
	ChapDay             *int    `json:"chap_day"`
	ChapEnteredAsLunar  bool    `json:"chap_entered_as_lunar"`
	HoanhPhi            *string `json:"hoanh_phi"`
	HoanhPhiReading     *string `json:"hoanh_phi_reading"`
	HoanhPhiMeaning     *string `json:"hoanh_phi_meaning"`
	CoupletUpper        *string `json:"couplet_upper"`
	CoupletUpperReading *string `json:"couplet_upper_reading"`
	CoupletUpperMeaning *string `json:"couplet_upper_meaning"`
	CoupletLower        *string `json:"couplet_lower"`
	CoupletLowerReading *string `json:"couplet_lower_reading"`
	CoupletLowerMeaning *string `json:"couplet_lower_meaning"`
	Teaching            *string `json:"teaching"`
}

func (r chronicleRequest) toChronicle() domain.Chronicle {
	return domain.Chronicle{
		Surname: r.Surname, CurrentLocation: r.CurrentLocation, SinceYear: r.SinceYear, Origin: r.Origin,
		History: r.History, HallName: r.HallName, HallAddress: r.HallAddress,
		ChapYear: r.ChapYear, ChapMonth: r.ChapMonth, ChapDay: r.ChapDay, ChapEnteredAsLunar: r.ChapEnteredAsLunar,
		HoanhPhi: r.HoanhPhi, HoanhPhiReading: r.HoanhPhiReading, HoanhPhiMeaning: r.HoanhPhiMeaning,
		CoupletUpper: r.CoupletUpper, CoupletUpperReading: r.CoupletUpperReading, CoupletUpperMeaning: r.CoupletUpperMeaning,
		CoupletLower: r.CoupletLower, CoupletLowerReading: r.CoupletLowerReading, CoupletLowerMeaning: r.CoupletLowerMeaning,
		Teaching: r.Teaching,
	}
}

// chronicleResponse is the Chronicle wire shape: the writable fields, the
// derived lunar chạp date, and the derived founder name and counts (AC40, AC41).
type chronicleResponse struct {
	Surname             *string `json:"surname"`
	CurrentLocation     *string `json:"current_location"`
	SinceYear           *int    `json:"since_year"`
	Origin              *string `json:"origin"`
	History             *string `json:"history"`
	HallName            *string `json:"hall_name"`
	HallAddress         *string `json:"hall_address"`
	ChapYear            *int    `json:"chap_year"`
	ChapMonth           *int    `json:"chap_month"`
	ChapDay             *int    `json:"chap_day"`
	ChapEnteredAsLunar  bool    `json:"chap_entered_as_lunar"`
	ChapLunarDay        *int    `json:"chap_lunar_day"`
	ChapLunarMonth      *int    `json:"chap_lunar_month"`
	ChapLunarLeap       *bool   `json:"chap_lunar_leap"`
	HoanhPhi            *string `json:"hoanh_phi"`
	HoanhPhiReading     *string `json:"hoanh_phi_reading"`
	HoanhPhiMeaning     *string `json:"hoanh_phi_meaning"`
	CoupletUpper        *string `json:"couplet_upper"`
	CoupletUpperReading *string `json:"couplet_upper_reading"`
	CoupletUpperMeaning *string `json:"couplet_upper_meaning"`
	CoupletLower        *string `json:"couplet_lower"`
	CoupletLowerReading *string `json:"couplet_lower_reading"`
	CoupletLowerMeaning *string `json:"couplet_lower_meaning"`
	Teaching            *string `json:"teaching"`
	FounderName         string  `json:"founder_name"`
	GenerationsCount    int     `json:"generations_count"`
	MembersCount        int     `json:"members_count"`
}

func newChronicleResponse(d *domain.GenealogyDetail) chronicleResponse {
	c := d.Chronicle
	r := chronicleResponse{
		Surname: c.Surname, CurrentLocation: c.CurrentLocation, SinceYear: c.SinceYear, Origin: c.Origin,
		History: c.History, HallName: c.HallName, HallAddress: c.HallAddress,
		ChapYear: c.ChapYear, ChapMonth: c.ChapMonth, ChapDay: c.ChapDay, ChapEnteredAsLunar: c.ChapEnteredAsLunar,
		HoanhPhi: c.HoanhPhi, HoanhPhiReading: c.HoanhPhiReading, HoanhPhiMeaning: c.HoanhPhiMeaning,
		CoupletUpper: c.CoupletUpper, CoupletUpperReading: c.CoupletUpperReading, CoupletUpperMeaning: c.CoupletUpperMeaning,
		CoupletLower: c.CoupletLower, CoupletLowerReading: c.CoupletLowerReading, CoupletLowerMeaning: c.CoupletLowerMeaning,
		Teaching: c.Teaching, FounderName: d.FounderName, GenerationsCount: d.GenerationsCount, MembersCount: d.MembersCount,
	}
	if c.ChapLunar != nil {
		day, month, leap := c.ChapLunar.Day, c.ChapLunar.Month, c.ChapLunar.Leap
		r.ChapLunarDay, r.ChapLunarMonth, r.ChapLunarLeap = &day, &month, &leap
	}
	return r
}

// genealogyDetailResponse is the GenealogyDetail wire shape (AC14).
type genealogyDetailResponse struct {
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	AncestralHome    string            `json:"ancestral_home"`
	MyRole           string            `json:"my_role"`
	MyMemberID       *int64            `json:"my_member_id"`
	ClanHeadMemberID *int64            `json:"clan_head_member_id"`
	Chronicle        chronicleResponse `json:"chronicle"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func newGenealogyDetailResponse(d *domain.GenealogyDetail) genealogyDetailResponse {
	return genealogyDetailResponse{
		ID: d.ID, Name: d.Name, AncestralHome: d.AncestralHome, MyRole: string(d.MyRole),
		MyMemberID: d.MyMemberID, ClanHeadMemberID: d.ClanHeadMemberID,
		Chronicle: newChronicleResponse(d), CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type genealogySummaryResponse struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	AncestralHome string    `json:"ancestral_home"`
	MyRole        string    `json:"my_role"`
	CreatedAt     time.Time `json:"created_at"`
}

// Create serves POST /v1/genealogies.
func (h *GenealogyHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req createGenealogyRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	in := domain.CreateGenealogyInput{
		Name: req.Name, AncestralHome: req.AncestralHome, Founder: req.Founder.toInfo(),
	}
	for _, s := range req.Spouses {
		in.Spouses = append(in.Spouses, s.toInfo())
	}
	d, err := h.usecase.Create(r.Context(), userID, in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, newGenealogyDetailResponse(d))
}

// List serves GET /v1/genealogies.
func (h *GenealogyHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	items, err := h.usecase.List(r.Context(), userID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]genealogySummaryResponse, 0, len(items))
	for _, g := range items {
		out = append(out, genealogySummaryResponse{
			ID: g.ID, Name: g.Name, AncestralHome: g.AncestralHome, MyRole: string(g.MyRole), CreatedAt: g.CreatedAt,
		})
	}
	utils.WriteSuccessResponse(w, newList(out))
}

// Get serves GET /v1/genealogies/{id}.
func (h *GenealogyHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	d, err := h.usecase.Get(r.Context(), userID, pathID(r, "id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newGenealogyDetailResponse(d))
}

// Update serves PUT /v1/genealogies/{id}: rename (owner only).
func (h *GenealogyHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req updateGenealogyRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	d, err := h.usecase.UpdateInfo(r.Context(), userID, pathID(r, "id"), req.Name, req.AncestralHome)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newGenealogyDetailResponse(d))
}

// Delete serves DELETE /v1/genealogies/{id} (owner only).
func (h *GenealogyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	if err := h.usecase.Delete(r.Context(), userID, pathID(r, "id")); err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, nil)
}

// UpdateChronicle serves PUT /v1/genealogies/{id}/chronicle: replaces every
// writable field (owner only).
func (h *GenealogyHandler) UpdateChronicle(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req chronicleRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	d, err := h.usecase.UpdateChronicle(r.Context(), userID, pathID(r, "id"), req.toChronicle())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, newChronicleResponse(d))
}
