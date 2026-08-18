package validation

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"
)

// Mexican phone number patterns
var (
	// MexicanPhoneRegex matches Mexican phone numbers (10 digits)
	MexicanPhoneRegex = regexp.MustCompile(`^[0-9]{10}$`)

	// MexicanPhoneWithCountryCodeRegex matches +52 prefix
	MexicanPhoneWithCountryCodeRegex = regexp.MustCompile(`^\+?52[0-9]{10}$`)

	// EmailRegex matches valid email addresses
	EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	// CURPRegex matches Mexican CURP format (18 characters)
	CURPRegex = regexp.MustCompile(`^[A-Z]{4}[0-9]{6}[HM][A-Z]{5}[0-9A-Z][0-9]$`)

	// XSSPatterns detect common XSS attack vectors
	XSSPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`),
		regexp.MustCompile(`(?i)<embed[^>]*>`),
		regexp.MustCompile(`(?i)<object[^>]*>`),
	}
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ValidateMexicanPhone validates a Mexican phone number
func ValidateMexicanPhone(phone string) error {
	if phone == "" {
		return NewValidationError("phone", "phone number is required")
	}

	// Remove common separators
	cleaned := strings.ReplaceAll(phone, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")

	// Check if it matches 10-digit format
	if MexicanPhoneRegex.MatchString(cleaned) {
		return nil
	}

	// Check if it matches +52 format
	if MexicanPhoneWithCountryCodeRegex.MatchString(cleaned) {
		return nil
	}

	return NewValidationError("phone", "invalid Mexican phone number format (expected 10 digits or +52 followed by 10 digits)")
}

// NormalizeMexicanPhone normalizes a Mexican phone number to 10 digits
func NormalizeMexicanPhone(phone string) string {
	// Remove all non-digit characters except +
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == '+' {
			return r
		}
		return -1
	}, phone)

	// Remove +52 prefix if present
	cleaned = strings.TrimPrefix(cleaned, "+52")
	cleaned = strings.TrimPrefix(cleaned, "52")

	// Take last 10 digits
	if len(cleaned) > 10 {
		cleaned = cleaned[len(cleaned)-10:]
	}

	return cleaned
}

// ValidateEmail validates an email address
func ValidateEmail(email string) error {
	if email == "" {
		return NewValidationError("email", "email is required")
	}

	// Check length
	if len(email) > 254 {
		return NewValidationError("email", "email is too long (max 254 characters)")
	}

	// Check format
	if !EmailRegex.MatchString(email) {
		return NewValidationError("email", "invalid email format")
	}

	// Additional checks
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return NewValidationError("email", "invalid email format")
	}

	localPart := parts[0]
	domainPart := parts[1]

	// Local part should not start or end with a dot
	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return NewValidationError("email", "invalid email format")
	}

	// Domain should have at least one dot
	if !strings.Contains(domainPart, ".") {
		return NewValidationError("email", "invalid email domain")
	}

	return nil
}

// NormalizeEmail normalizes an email address to lowercase
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateCURP validates a Mexican CURP (Clave Única de Registro de Población)
func ValidateCURP(curp string) error {
	if curp == "" {
		return NewValidationError("curp", "CURP is required")
	}

	// Convert to uppercase for validation
	curp = strings.ToUpper(strings.TrimSpace(curp))

	// Check length
	if len(curp) != 18 {
		return NewValidationError("curp", "CURP must be exactly 18 characters")
	}

	// Check format
	if !CURPRegex.MatchString(curp) {
		return NewValidationError("curp", "invalid CURP format")
	}

	// Validate sex character (position 10)
	sex := string(curp[10])
	if sex != "H" && sex != "M" {
		return NewValidationError("curp", "invalid CURP sex indicator (must be H or M)")
	}

	return nil
}

// NormalizeCURP normalizes a CURP to uppercase without spaces
func NormalizeCURP(curp string) string {
	return strings.ToUpper(strings.TrimSpace(curp))
}

// SanitizeString removes potential XSS vectors from a string
func SanitizeString(input string) string {
	if input == "" {
		return input
	}

	// HTML escape
	sanitized := html.EscapeString(input)

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	return sanitized
}

// ValidateNoXSS checks if a string contains XSS attack vectors
func ValidateNoXSS(input string) error {
	if input == "" {
		return nil
	}

	for _, pattern := range XSSPatterns {
		if pattern.MatchString(input) {
			return NewValidationError("input", "potentially malicious content detected")
		}
	}

	return nil
}

// SanitizeInput performs comprehensive sanitization
func SanitizeInput(input string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(input)

	// HTML escape
	sanitized = html.EscapeString(sanitized)

	// Remove control characters except newline and tab
	sanitized = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, sanitized)

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	return sanitized
}

// ValidateStringLength validates string length within min/max bounds
func ValidateStringLength(value, field string, min, max int) error {
	length := len(value)

	if min > 0 && length < min {
		return NewValidationError(field, fmt.Sprintf("must be at least %d characters", min))
	}

	if max > 0 && length > max {
		return NewValidationError(field, fmt.Sprintf("must not exceed %d characters", max))
	}

	return nil
}

// ValidateRequired checks if a value is not empty
func ValidateRequired(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return NewValidationError(field, "is required")
	}
	return nil
}

// ValidateAlphanumeric checks if a string contains only alphanumeric characters
func ValidateAlphanumeric(value, field string) error {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return NewValidationError(field, "must contain only letters and numbers")
		}
	}
	return nil
}

// ValidateAlpha checks if a string contains only letters
func ValidateAlpha(value, field string) error {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) {
			return NewValidationError(field, "must contain only letters")
		}
	}
	return nil
}

// ValidateNumeric checks if a string contains only digits
func ValidateNumeric(value, field string) error {
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return NewValidationError(field, "must contain only numbers")
		}
	}
	return nil
}

// ValidatePostalCode validates a Mexican postal code (5 digits)
func ValidatePostalCode(postalCode string) error {
	if postalCode == "" {
		return NewValidationError("postal_code", "postal code is required")
	}

	if len(postalCode) != 5 {
		return NewValidationError("postal_code", "postal code must be 5 digits")
	}

	if err := ValidateNumeric(postalCode, "postal_code"); err != nil {
		return err
	}

	return nil
}

// ValidateINEKey validates Mexican INE/IFE credential number
func ValidateINEKey(ineKey string) error {
	if ineKey == "" {
		return NewValidationError("ine_key", "INE key is required")
	}

	// Remove spaces
	ineKey = strings.ReplaceAll(ineKey, " ", "")

	// INE key is typically 13 or 18 digits
	if len(ineKey) != 13 && len(ineKey) != 18 {
		return NewValidationError("ine_key", "INE key must be 13 or 18 digits")
	}

	if err := ValidateNumeric(ineKey, "ine_key"); err != nil {
		return err
	}

	return nil
}

// Validator provides a fluent interface for validation
type Validator struct {
	errors []*ValidationError
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make([]*ValidationError, 0),
	}
}

// ValidateField adds a validation check
func (v *Validator) ValidateField(err error) *Validator {
	if err != nil {
		if validationErr, ok := err.(*ValidationError); ok {
			v.errors = append(v.errors, validationErr)
		}
	}
	return v
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all validation errors
func (v *Validator) Errors() []*ValidationError {
	return v.errors
}

// Error returns a formatted error message with all validation errors
func (v *Validator) Error() error {
	if !v.HasErrors() {
		return nil
	}

	messages := make([]string, len(v.errors))
	for i, err := range v.errors {
		messages[i] = err.Error()
	}

	return fmt.Errorf("validation failed: %s", strings.Join(messages, "; "))
}
