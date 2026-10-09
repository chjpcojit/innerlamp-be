package main

import (
	"encoding/json"
	"fmt"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// sourceImage mirrors one object of assets/data/buddhist_gallery.json.
type sourceImage struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	Location     string `json:"location"`
	ThumbnailURL string `json:"thumbnail_url"`
	ImageURL     string `json:"image_url"`
	Author       string `json:"author"`
	License      string `json:"license"`
	SourceURL    string `json:"source_url"`
}

// parseGallery turns the source JSON into domain images, validating what the
// database would otherwise reject one row at a time: unique ids, a known type,
// a title and both image URLs.
//
// SortOrder interleaves the types round-robin (see migrations/0017): the file
// groups images by type, so its raw order would make "top" all one type. Within
// a type the file order is kept.
func parseGallery(src []byte) ([]domain.GalleryImage, error) {
	var raw []sourceImage
	if err := json.Unmarshal(src, &raw); err != nil {
		return nil, fmt.Errorf("decode gallery: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("no gallery images in source")
	}

	seen := make(map[string]bool, len(raw))
	byType := make(map[string][]domain.GalleryImage, len(domain.GalleryTypes))
	for i, s := range raw {
		if s.ID == "" {
			return nil, fmt.Errorf("entry %d: empty id", i)
		}
		if seen[s.ID] {
			return nil, fmt.Errorf("duplicate id %q", s.ID)
		}
		seen[s.ID] = true
		if !domain.IsValidGalleryType(s.Type) {
			return nil, fmt.Errorf("image %q: unknown type %q", s.ID, s.Type)
		}
		if s.Title == "" {
			return nil, fmt.Errorf("image %q: empty title", s.ID)
		}
		if s.ImageURL == "" || s.ThumbnailURL == "" {
			return nil, fmt.Errorf("image %q: empty image_url or thumbnail_url", s.ID)
		}
		byType[s.Type] = append(byType[s.Type], domain.GalleryImage{
			ID: s.ID, Type: s.Type, Title: s.Title, Location: s.Location,
			ThumbnailURL: s.ThumbnailURL, ImageURL: s.ImageURL,
			Author: s.Author, License: s.License, SourceURL: s.SourceURL,
		})
	}

	out := make([]domain.GalleryImage, 0, len(raw))
	for round := 0; len(out) < len(raw); round++ {
		for _, t := range domain.GalleryTypes {
			if round < len(byType[t]) {
				img := byType[t][round]
				img.SortOrder = len(out)
				out = append(out, img)
			}
		}
	}
	return out, nil
}
