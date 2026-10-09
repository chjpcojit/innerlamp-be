package domain

import (
	"context"
	"errors"
)

// Gallery image types as stored in gallery_images.type and the source data.
const (
	GalleryChua      = "CHUA"
	GalleryTuongPhat = "TUONG_PHAT"
	GalleryThien     = "THIEN"
	GalleryLeHoi     = "LE_HOI"
)

// GalleryTypes lists every valid gallery type in the order the seeder interleaves them.
var GalleryTypes = []string{GalleryChua, GalleryTuongPhat, GalleryThien, GalleryLeHoi}

// IsValidGalleryType reports whether t is one of GalleryTypes.
func IsValidGalleryType(t string) bool {
	for _, v := range GalleryTypes {
		if t == v {
			return true
		}
	}
	return false
}

// GalleryImage is one beautiful Buddhist photo. It maps the gallery_images table.
//
// Plain data: no json/db tags — the wire shape is delivery's, the SQL shape is
// repository's. Author, License and SourceURL carry the attribution the image's
// Creative Commons licence requires.
type GalleryImage struct {
	ID           string
	Type         string
	SortOrder    int
	Title        string
	Location     string
	ThumbnailURL string
	ImageURL     string
	Author       string
	License      string
	SourceURL    string
}

// GalleryPage is one page of a gallery list plus what a client needs to know
// whether another page exists.
type GalleryPage struct {
	Items      []GalleryImage
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

var (
	// ErrGalleryImageNotFound — no gallery image has the requested id.
	ErrGalleryImageNotFound = errors.New("gallery image not found")

	// ErrInvalidGalleryType — the type filter is not one of GalleryTypes.
	ErrInvalidGalleryType = errors.New("invalid gallery type")
)

// GalleryRepository is the port the gallery use case reads through, and the
// seeder writes through.
//
// Ordering is part of the contract: every read returns images by
// sort_order ASC, id ASC — deterministic, so paging never repeats or skips a row.
type GalleryRepository interface {
	// ListTop returns the first limit images of any type in order.
	ListTop(ctx context.Context, limit int) ([]GalleryImage, error)

	// List returns limit images starting at offset, and the total number of
	// matching images. imageType "" means every type. An offset past the end
	// returns an empty slice, not an error.
	List(ctx context.Context, imageType string, offset, limit int) ([]GalleryImage, int, error)

	// FindByID returns the image or ErrGalleryImageNotFound.
	FindByID(ctx context.Context, id string) (*GalleryImage, error)

	// Upsert inserts or updates images keyed by id, idempotently, atomically.
	Upsert(ctx context.Context, images []GalleryImage) error
}
