package http

import (
	"context"
	"net/http"
	"strconv"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// templeUsecase is the subset of *usecase.TempleUsecase this handler depends
// on, declared here (the consumer) so the handler is tested with a mock.
type templeUsecase interface {
	Top(ctx context.Context) ([]domain.Temple, error)
	List(ctx context.Context, page int) (*domain.TemplePage, error)
	Get(ctx context.Context, id string) (*domain.Temple, error)
}

// TempleHandler serves the read-only temple catalogue. All three routes are
// public: browsing needs no account.
type TempleHandler struct {
	usecase templeUsecase
	imageHost
}

// NewTempleHandler wires the handler.
func NewTempleHandler(usecase templeUsecase, opts ...ImageOption) *TempleHandler {
	return &TempleHandler{usecase: usecase, imageHost: newImageHost(opts)}
}

type locationResponse struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

// templeSummaryResponse is the card shape shared by top and list. image_url and
// thumbnail_url are null (not "") while the catalogue has no image for a temple,
// so a client can use plain Optional handling.
type templeSummaryResponse struct {
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
}

type amenitiesResponse struct {
	ParkingCar        bool `json:"parking_car"`
	ParkingMotorcycle bool `json:"parking_motorcycle"`
	VegetarianMeals   bool `json:"vegetarian_meals"`
	SummerRetreat     bool `json:"summer_retreat"`
}

// templeDetailResponse is the summary plus everything the detail screen shows.
type templeDetailResponse struct {
	templeSummaryResponse
	Aliases        []string          `json:"aliases"`
	Founder        string            `json:"founder"`
	CurrentAbbot   string            `json:"current_abbot"`
	OperatingHours string            `json:"operating_hours"`
	Amenities      amenitiesResponse `json:"amenities"`

	Introduction         string   `json:"introduction"`
	ArchitectureHeritage string   `json:"architecture_heritage"`
	VisitingNotes        []string `json:"visiting_notes"`
}

type templeTopResponse struct {
	Items []templeSummaryResponse `json:"items"`
}

type templeListResponse struct {
	Items      []templeSummaryResponse `json:"items"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	Total      int                     `json:"total"`
	TotalPages int                     `json:"total_pages"`
}

func newTempleSummary(h imageHost, r *http.Request, t domain.Temple) templeSummaryResponse {
	return templeSummaryResponse{
		ID: t.ID, Name: t.Name, Type: t.Type, Region: t.Region,
		ThumbnailURL: h.ptr(r, t.Thumbnail), ImageURL: h.ptr(r, t.ImageURL),
		Sect: t.Sect, HeritageRank: t.Heritage, EstablishedYear: t.Established,
		Location: locationResponse{Lat: t.Latitude, Lng: t.Longitude, Address: t.Address},
	}
}

func newTempleSummaries(h imageHost, r *http.Request, ts []domain.Temple) []templeSummaryResponse {
	out := make([]templeSummaryResponse, 0, len(ts))
	for _, t := range ts {
		out = append(out, newTempleSummary(h, r, t))
	}
	return out
}

// Top serves GET /v1/temples/top.
func (h *TempleHandler) Top(w http.ResponseWriter, r *http.Request) {
	items, err := h.usecase.Top(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, templeTopResponse{Items: newTempleSummaries(h.imageHost, r, items)})
}

// List serves GET /v1/temples?page=N. page defaults to 1; anything that is not
// a positive integer is 400 INVALID_PAGE.
func (h *TempleHandler) List(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil {
			writeDomainError(w, domain.ErrInvalidPage)
			return
		}
		page = p
	}

	result, err := h.usecase.List(r.Context(), page)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	utils.WriteSuccessResponse(w, templeListResponse{
		Items: newTempleSummaries(h.imageHost, r, result.Items), Page: result.Page, PageSize: result.PageSize,
		Total: result.Total, TotalPages: result.TotalPages,
	})
}

// Get serves GET /v1/temples/{id}.
func (h *TempleHandler) Get(w http.ResponseWriter, r *http.Request) {
	t, err := h.usecase.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	aliases := t.Aliases
	if aliases == nil {
		aliases = []string{}
	}
	notes := t.VisitingNotes
	if notes == nil {
		notes = []string{}
	}
	utils.WriteSuccessResponse(w, templeDetailResponse{
		templeSummaryResponse: newTempleSummary(h.imageHost, r, *t),
		Aliases:               aliases, Founder: t.Founder, CurrentAbbot: t.Abbot, OperatingHours: t.OperatingHours,
		Amenities: amenitiesResponse{
			ParkingCar: t.Amenities.ParkingCar, ParkingMotorcycle: t.Amenities.ParkingMotorcycle,
			VegetarianMeals: t.Amenities.VegetarianMeals, SummerRetreat: t.Amenities.SummerRetreat,
		},
		Introduction: t.Introduction, ArchitectureHeritage: t.ArchitectureHeritage, VisitingNotes: notes,
	})
}
