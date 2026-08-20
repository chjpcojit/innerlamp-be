package domain

// ProfileUpdateInput is the use-case input for US-004 AC2/AC5. Both fields
// are always written as given: an empty string clears the corresponding
// column (AC5 — leaving a field blank and saving removes previously saved
// data), so there is no separate "unchanged" sentinel to worry about.
type ProfileUpdateInput struct {
	DharmaName      string
	OrdinationPlace string
}
