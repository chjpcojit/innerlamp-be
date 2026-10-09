package domain

import "errors"

// CalendarImage is one Buddha/Bodhisattva photo of the lunar-calendar block,
// with the attribution its licence (CC BY / CC BY-SA, mostly) requires.
type CalendarImage struct {
	// ID is the file name without its extension, e.g. "buddha_001".
	ID string
	// File is the file name under assets/images/home-calendar, e.g. "buddha_001.jpg".
	File       string
	Author     string
	License    string
	LicenseURL string
	SourceURL  string
}

// ErrInvalidDate — the date query is not a real YYYY-MM-DD date within the supported years.
var ErrInvalidDate = errors.New("date must be a valid YYYY-MM-DD between 1900 and 2100")
