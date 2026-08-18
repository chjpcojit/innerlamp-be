package utils

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the validator.Validate with singleton pattern
type Validator struct {
	validate *validator.Validate
}

var defaultValidator *Validator

func init() {
	v := &Validator{
		validate: validator.New(),
	}
	// Register custom validators if needed
	defaultValidator = v
}

// GetValidator returns the singleton validator instance
func GetValidator() *Validator {
	return defaultValidator
}

// ValidationError represents a structured validation error with field details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag"`
	Value   any    `json:"value,omitempty"`
}

// ValidationErrors is a collection of validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}
	msgs := make([]string, len(ve.Errors))
	for i, e := range ve.Errors {
		msgs[i] = fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return strings.Join(msgs, "; ")
}

// Validate validates a struct using the validator library
// If custom validation is needed, pass customValidators map with field names as keys
// and validation functions that return error message (nil = valid, "" = use tag default msg)
func (v *Validator) Validate(data any, customValidators map[string]func(any) string) error {
	var errs []ValidationError

	// Run standard validator
	if err := v.validate.Struct(data); err != nil {
		for _, fe := range err.(validator.ValidationErrors) {
			errMsg := getValidationErrorMessage(fe)
			errs = append(errs, ValidationError{
				Field:   fe.Field(),
				Message: errMsg,
				Tag:     fe.Tag(),
				Value:   fe.Value(),
			})
		}
	}

	// Run custom validators if provided
	if customValidators != nil {
		// Get field values using reflection
		if len(customValidators) > 0 {
			customErrs := runCustomValidators(data, customValidators)
			errs = append(errs, customErrs...)
		}
	}

	if len(errs) > 0 {
		return ValidationErrors{Errors: errs}
	}

	return nil
}

// getValidationErrorMessage returns a human-readable error message for validation errors
func getValidationErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", fe.Field(), fe.Param())
	case "minmax":
		return fmt.Sprintf("%s must be between %s and %s characters", fe.Field(), fe.Param(), "")
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fe.Field(), fe.Param())
	case "numeric":
		return fmt.Sprintf("%s must be numeric", fe.Field())
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", fe.Field())
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", fe.Field())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", fe.Field())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", fe.Field())
	case "ip":
		return fmt.Sprintf("%s must be a valid IP address", fe.Field())
	case "custom":
		return fmt.Sprintf("%s validation failed", fe.Field())
	default:
		return fmt.Sprintf("%s failed %s validation", fe.Field(), fe.Tag())
	}
}

// runCustomValidators runs custom validation functions and returns validation errors
func runCustomValidators(data any, customValidators map[string]func(any) string) []ValidationError {
	var errs []ValidationError

	// Note: For production use, implement reflection-based field extraction
	// This is a simplified version - you may need to extract field values properly
	for field, validatorFunc := range customValidators {
		// This is a placeholder - actual implementation would use reflection to get field values
		// See examples in handler code for proper usage
		if validatorFunc != nil {
			// Custom validation is called from handler with extracted field value
			_ = validatorFunc
		}
		_ = field
	}

	return errs
}

// IsValidationError checks if error is a ValidationErrors
func IsValidationError(err error) bool {
	_, ok := err.(ValidationErrors)
	return ok
}

// GetValidationErrorDetails extracts validation errors as map for API response
func GetValidationErrorDetails(err error) map[string]any {
	if ve, ok := err.(ValidationErrors); ok {
		return map[string]any{
			"validation_errors": ve.Errors,
		}
	}
	return nil
}
