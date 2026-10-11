package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// templeSuggestionUsecase is the subset of *usecase.TempleSuggestionUsecase the
// handler needs, declared here (the consumer) so it can be tested with a mock.
type templeSuggestionUsecase interface {
	List(ctx context.Context, status string, page int) (*domain.SuggestionPage, error)
	Get(ctx context.Context, id int64) (*domain.TempleSuggestion, error)
	CreateTemple(ctx context.Context, suggestionID, adminID int64, role string, in domain.NewTemple) (*domain.CreatedTemple, error)
}

// TempleSuggestionHandler serves the review routes. Authorization is the route
// wrapper's (auth middleware + RequireActiveSession + RequireAnyRole); the
// handler only reads the caller's identity from the claims.
type TempleSuggestionHandler struct {
	usecase templeSuggestionUsecase
	// imageBase is the origin root-relative image paths resolve against; empty =
	// the request's own origin (see config.ServerConfig.PublicBaseURL).
	imageBase string
}

// NewTempleSuggestionHandler wires the handler.
func NewTempleSuggestionHandler(usecase templeSuggestionUsecase, imageBase string) *TempleSuggestionHandler {
	return &TempleSuggestionHandler{usecase: usecase, imageBase: strings.TrimRight(imageBase, "/")}
}

type suggestionItemResponse struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	ProvinceCode *string   `json:"province_code"`
	Province     string    `json:"province"`
	WardCode     *string   `json:"ward_code"`
	Ward         *string   `json:"ward"`
	Address      string    `json:"address"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type suggestionDetailResponse struct {
	suggestionItemResponse
	TempleID     *string    `json:"temple_id"`
	ReviewedBy   *int64     `json:"reviewed_by"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	RejectReason *string    `json:"reject_reason"`
}

type suggestionPageResponse struct {
	Items      []suggestionItemResponse `json:"items"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	Total      int                      `json:"total"`
	TotalPages int                      `json:"total_pages"`
}

func toSuggestionItem(s domain.TempleSuggestion) suggestionItemResponse {
	return suggestionItemResponse{
		ID: s.ID, Name: s.Name, ProvinceCode: s.ProvinceCode, Province: s.Province,
		WardCode: s.WardCode, Ward: s.Ward, Address: s.Address, Status: s.Status, CreatedAt: s.CreatedAt,
	}
}

func toSuggestionDetail(s domain.TempleSuggestion) suggestionDetailResponse {
	return suggestionDetailResponse{
		suggestionItemResponse: toSuggestionItem(s),
		TempleID:               s.TempleID, ReviewedBy: s.ReviewedBy, ReviewedAt: s.ReviewedAt, RejectReason: s.RejectReason,
	}
}

// List serves GET /v1/admin/temple-suggestions?status=&page=.
func (h *TempleSuggestionHandler) List(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil {
			writeDomainError(w, domain.ErrInvalidPage)
			return
		}
		page = p
	}
	result, err := h.usecase.List(r.Context(), r.URL.Query().Get("status"), page)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	items := make([]suggestionItemResponse, 0, len(result.Items))
	for _, s := range result.Items {
		items = append(items, toSuggestionItem(s))
	}
	utils.WriteSuccessResponse(w, suggestionPageResponse{
		Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total, TotalPages: result.TotalPages,
	})
}

// Get serves GET /v1/admin/temple-suggestions/{id}.
func (h *TempleSuggestionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := suggestionID(w, r)
	if !ok {
		return
	}
	s, err := h.usecase.Get(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, toSuggestionDetail(*s))
}

// suggestionID reads {id}; a non-positive or non-numeric id is the same 404 as
// an unknown one (AC14, AC23). It writes the response and reports false then.
func suggestionID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeDomainError(w, domain.ErrSuggestionNotFound)
		return 0, false
	}
	return id, true
}

type amenitiesRequest struct {
	ParkingCar        bool `json:"parking_car"`
	ParkingMotorcycle bool `json:"parking_motorcycle"`
	VegetarianMeals   bool `json:"vegetarian_meals"`
	SummerRetreat     bool `json:"summer_retreat"`
}

// createTempleRequest is the form the admin submits. Numbers are pointers so an
// absent latitude/longitude/established_year is told apart from a zero (AC18).
type createTempleRequest struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Region          string   `json:"region"`
	ImageURL        string   `json:"image_url"`
	ThumbnailURL    string   `json:"thumbnail_url"`
	Aliases         []string `json:"aliases"`
	Sect            string   `json:"sect"`
	Founder         string   `json:"founder"`
	CurrentAbbot    string   `json:"current_abbot"`
	EstablishedYear *int     `json:"established_year"`
	HeritageRank    string   `json:"heritage_rank"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	// Address is the part below ward level; province_code/ward_code carry the rest.
	Address        string           `json:"address"`
	ProvinceCode   string           `json:"province_code"`
	WardCode       string           `json:"ward_code"`
	OperatingHours string           `json:"operating_hours"`
	Amenities      amenitiesRequest `json:"amenities"`

	Introduction         string   `json:"introduction"`
	ArchitectureHeritage string   `json:"architecture_heritage"`
	VisitingNotes        []string `json:"visiting_notes"`
}

type createdSuggestionResponse struct {
	ID       int64   `json:"id"`
	Status   string  `json:"status"`
	TempleID *string `json:"temple_id"`
}

type createdTempleResponse struct {
	Suggestion createdSuggestionResponse `json:"suggestion"`
	Temple     templeDetailResponse      `json:"temple"`
}

// CreateTemple serves POST /v1/admin/temple-suggestions/{id}/create-temple. The
// admin id and role come from the token, never from the body.
func (h *TempleSuggestionHandler) CreateTemple(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtcontext.GetClaims(r.Context())
	if !ok {
		WriteAuthError(w, r, errors.New("no claims in context"))
		return
	}
	adminID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		WriteAuthError(w, r, err)
		return
	}
	id, ok := suggestionID(w, r)
	if !ok {
		return
	}

	var req createTempleRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	// Required numbers: absent is "missing", not zero.
	for _, f := range []struct {
		name    string
		missing bool
	}{
		{"established_year", req.EstablishedYear == nil},
		{"latitude", req.Latitude == nil},
		{"longitude", req.Longitude == nil},
	} {
		if f.missing {
			writeDomainError(w, &domain.TempleInputError{Field: f.name, Message: f.name + " is required"})
			return
		}
	}

	res, err := h.usecase.CreateTemple(r.Context(), id, adminID, claims.Role, domain.NewTemple{
		Name: req.Name, Type: req.Type, Region: req.Region, ImageURL: req.ImageURL, ThumbnailURL: req.ThumbnailURL,
		Aliases: req.Aliases, Sect: req.Sect, Founder: req.Founder, CurrentAbbot: req.CurrentAbbot,
		EstablishedYear: *req.EstablishedYear, HeritageRank: req.HeritageRank,
		Latitude: *req.Latitude, Longitude: *req.Longitude,
		Address: req.Address, ProvinceCode: req.ProvinceCode, WardCode: req.WardCode,
		OperatingHours: req.OperatingHours,
		Amenities: domain.TempleAmenities{
			ParkingCar: req.Amenities.ParkingCar, ParkingMotorcycle: req.Amenities.ParkingMotorcycle,
			VegetarianMeals: req.Amenities.VegetarianMeals, SummerRetreat: req.Amenities.SummerRetreat,
		},
		Introduction: req.Introduction, ArchitectureHeritage: req.ArchitectureHeritage, VisitingNotes: req.VisitingNotes,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponseWithStatus(w, http.StatusCreated, createdTempleResponse{
		Suggestion: createdSuggestionResponse{ID: res.Suggestion.ID, Status: res.Suggestion.Status, TempleID: res.Suggestion.TempleID},
		Temple:     h.toTempleDetail(r, res.Temple),
	})
}

// --- temple detail: same shape as appService's GET /v1/temples/{id} (AC16) ---

type locationResponse struct {
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	Address      string  `json:"address"`
	Province     string  `json:"province"`
	ProvinceCode *string `json:"province_code"`
	WardCode     *string `json:"ward_code"`
	Ward         *string `json:"ward"`
	FullAddress  string  `json:"full_address"`
}

type amenitiesResponse struct {
	ParkingCar        bool `json:"parking_car"`
	ParkingMotorcycle bool `json:"parking_motorcycle"`
	VegetarianMeals   bool `json:"vegetarian_meals"`
	SummerRetreat     bool `json:"summer_retreat"`
}

type templeDetailResponse struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Type            string           `json:"type"`
	Region          string           `json:"region"`
	ThumbnailURL    *string          `json:"thumbnail_url"`
	ImageURL        *string          `json:"image_url"`
	Sect            string           `json:"sect"`
	HeritageRank    string           `json:"heritage_rank"`
	EstablishedYear int              `json:"established_year"`
	Location        locationResponse `json:"location"`

	Aliases        []string          `json:"aliases"`
	Founder        string            `json:"founder"`
	CurrentAbbot   string            `json:"current_abbot"`
	OperatingHours string            `json:"operating_hours"`
	Amenities      amenitiesResponse `json:"amenities"`

	Introduction         string   `json:"introduction"`
	ArchitectureHeritage string   `json:"architecture_heritage"`
	VisitingNotes        []string `json:"visiting_notes"`
}

func (h *TempleSuggestionHandler) toTempleDetail(r *http.Request, t domain.Temple) templeDetailResponse {
	aliases, notes := t.Aliases, t.VisitingNotes
	if aliases == nil {
		aliases = []string{}
	}
	if notes == nil {
		notes = []string{}
	}
	return templeDetailResponse{
		ID: t.ID, Name: t.Name, Type: t.Type, Region: t.Region,
		ThumbnailURL: h.imageURL(r, t.ThumbnailURL), ImageURL: h.imageURL(r, t.ImageURL),
		Sect: t.Sect, HeritageRank: t.HeritageRank, EstablishedYear: t.EstablishedYear,
		Location: locationResponse{
			Lat: t.Latitude, Lng: t.Longitude, Address: t.Address, Province: t.Province,
			ProvinceCode: nullIfEmpty(t.ProvinceCode), WardCode: nullIfEmpty(t.WardCode), Ward: nullIfEmpty(t.Ward),
			FullAddress: t.FullAddress(),
		},
		Aliases: aliases, Founder: t.Founder, CurrentAbbot: t.CurrentAbbot, OperatingHours: t.OperatingHours,
		Amenities: amenitiesResponse{
			ParkingCar: t.Amenities.ParkingCar, ParkingMotorcycle: t.Amenities.ParkingMotorcycle,
			VegetarianMeals: t.Amenities.VegetarianMeals, SummerRetreat: t.Amenities.SummerRetreat,
		},
		Introduction: t.Introduction, ArchitectureHeritage: t.ArchitectureHeritage, VisitingNotes: notes,
	}
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// imageURL resolves a stored image location the way appService does: "" is
// null, a root-relative path gets the public origin, anything else is returned
// as is.
func (h *TempleSuggestionHandler) imageURL(r *http.Request, raw string) *string {
	if raw == "" {
		return nil
	}
	if !strings.HasPrefix(raw, "/") {
		return &raw
	}
	base := h.imageBase
	if base == "" {
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}
	u := base + raw
	return &u
}
