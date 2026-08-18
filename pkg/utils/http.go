package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Error types for decode operations
const (
	ErrorTypeValidation = "validation"
	ErrorTypeBusiness   = "business"
	ErrorTypeSystem     = "system"
)

// Decode error constants
var (
	ErrEmptyBody        = errors.New("request body is empty")
	ErrInvalidJSON      = errors.New("invalid JSON")
	ErrInvalidFieldType = errors.New("invalid field type")
	ErrUnknownField     = errors.New("unknown field")
	ErrBodyTooLarge     = errors.New("request body too large")
)

// DecodeOptions configuration for JSON decoding
type DecodeOptions struct {
	MaxBodySize           int64
	DisallowUnknownFields bool
}

// DecodeError represents a JSON decoding error with details
type DecodeError struct {
	Field   string
	Message string
	Err     error
}

func (e *DecodeError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func (e *DecodeError) Unwrap() error {
	return e.Err
}

// ErrorDetail represents error information in API response
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ApiResponse standard API response format
type ApiResponse struct {
	Success bool           `json:"success"`
	Data    any            `json:"data,omitempty"`
	Error   *ErrorDetail   `json:"error,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// DefaultDecodeOptions returns default decode options
func DefaultDecodeOptions() *DecodeOptions {
	return &DecodeOptions{
		MaxBodySize:           1 << 20, // 1MB
		DisallowUnknownFields: false,
	}
}

// DecodeJSONBody decodes JSON request body with error handling
func DecodeJSONBody(r *http.Request, dst any, opts *DecodeOptions) error {
	if opts == nil {
		opts = DefaultDecodeOptions()
	}

	// Read body
	var reader io.Reader = r.Body
	if opts.MaxBodySize > 0 {
		reader = io.LimitReader(r.Body, opts.MaxBodySize)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return &DecodeError{
			Message: "failed to read request body",
			Err:     err,
		}
	}

	// Check if body is empty
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return &DecodeError{
			Message: ErrEmptyBody.Error(),
			Err:     ErrEmptyBody,
		}
	}

	// Decode JSON
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	if opts.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(dst); err != nil {
		// Check for specific error types
		if strings.Contains(err.Error(), "unknown field") {
			// Extract field name from error message
			var field string
			if ue, ok := err.(*json.UnmarshalTypeError); ok {
				field = ue.Field
			}
			return &DecodeError{
				Field:   field,
				Message: ErrUnknownField.Error(),
				Err:     ErrUnknownField,
			}
		}

		if strings.Contains(err.Error(), "cannot unmarshal") {
			var field string
			if ute, ok := err.(*json.UnmarshalTypeError); ok {
				field = ute.Field
			}
			return &DecodeError{
				Field:   field,
				Message: ErrInvalidFieldType.Error(),
				Err:     ErrInvalidFieldType,
			}
		}

		return &DecodeError{
			Message: ErrInvalidJSON.Error(),
			Err:     ErrInvalidJSON,
		}
	}

	return nil
}

// DecodeJSONBodyStrict decodes JSON with strict mode (disallow unknown fields)
func DecodeJSONBodyStrict(r *http.Request, dst any) error {
	opts := &DecodeOptions{
		MaxBodySize:           1 << 20,
		DisallowUnknownFields: true,
	}
	return DecodeJSONBody(r, dst, opts)
}

// IsDecodeError checks if error is a DecodeError with optional target check
func IsDecodeError(err error, target error) bool {
	if err == nil {
		return false
	}

	de, ok := err.(*DecodeError)
	if !ok {
		return false
	}

	if target == nil {
		return true
	}

	return errors.Is(de.Err, target)
}

// GetDecodeErrorDetails extracts field name and message from DecodeError
func GetDecodeErrorDetails(err error) (field string, message string) {
	if de, ok := err.(*DecodeError); ok {
		return de.Field, de.Message
	}
	return "", ""
}

// WriteSuccessResponse writes a success response with data (200 OK)
func WriteSuccessResponse(w http.ResponseWriter, data any) {
	WriteSuccessResponseWithStatus(w, http.StatusOK, data)
}

// WriteSuccessResponseWithStatus writes a success response with custom status code
func WriteSuccessResponseWithStatus(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ApiResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// WriteErrorResponse writes an error response
func WriteErrorResponse(w http.ResponseWriter, statusCode int, errorType, code, message string, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ApiResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
			Type:    errorType,
		},
		Details: details,
	}

	json.NewEncoder(w).Encode(response)
}

// WriteValidationError writes a validation error response (400 Bad Request)
func WriteValidationError(w http.ResponseWriter, code, message string, details map[string]any) {
	WriteErrorResponse(w, http.StatusBadRequest, ErrorTypeValidation, code, message, details)
}

// WriteBusinessError writes a business logic error response
func WriteBusinessError(w http.ResponseWriter, statusCode int, code, message string, details map[string]any) {
	WriteErrorResponse(w, statusCode, ErrorTypeBusiness, code, message, details)
}

// WriteSystemError writes a system error response (500 Internal Server Error)
func WriteSystemError(w http.ResponseWriter, code, message string, details map[string]any) {
	WriteErrorResponse(w, http.StatusInternalServerError, ErrorTypeSystem, code, message, details)
}
