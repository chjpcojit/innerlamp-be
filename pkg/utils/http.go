package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
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

// isJSONContentType reports whether the Content-Type header names a JSON
// media type. The media type is normalised through mime.ParseMediaType, so
// casing and parameters (charset, ...) do not matter: "APPLICATION/JSON" and
// "application/json; charset=utf-8" both pass. A media type whose subtype
// ends in "+json" (application/vnd.api+json, ...) passes too.
//
// An empty header returns true: a request with no Content-Type at all stays
// lenient, which is the behaviour TestDecodeJSONBody_NoContentType pins. A
// header that fails to parse is treated as unsupported.
func isJSONContentType(header string) bool {
	if header == "" {
		return true
	}

	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil {
		return false
	}

	if mediaType == "application/json" {
		return true
	}

	return strings.HasSuffix(mediaType, "+json")
}

// unknownFieldName extracts the field name from the message of an
// encoding/json "unknown field" error, which is an *errors.errorString
// shaped `json: unknown field "x"` — not a *json.UnmarshalTypeError, so the
// field name is only available in the message text.
func unknownFieldName(msg string) string {
	_, after, found := strings.Cut(msg, `unknown field "`)
	if !found {
		return ""
	}
	name, _, found := strings.Cut(after, `"`)
	if !found {
		return ""
	}
	return name
}

// DecodeJSONBody decodes JSON request body with error handling.
//
// The gates run in a fixed order, and that order is part of the contract:
// reversing two of them changes the error returned for the same request.
//
//  1. Content-Type — non-JSON media type → ErrInvalidJSON ("unsupported
//     content type"); a missing header passes.
//  2. Read the body, capped at MaxBodySize+1 bytes (MaxBodySize <= 0 means
//     no limit).
//  3. Over the cap → ErrBodyTooLarge, even when the bytes read happen to
//     end on a valid JSON boundary.
//  4. Empty / whitespace-only → ErrEmptyBody.
//  5. Decode → ErrUnknownField (Field set), ErrInvalidFieldType (Field set)
//     or ErrInvalidJSON.
//  6. Anything but whitespace after the first JSON value → ErrInvalidJSON.
func DecodeJSONBody(r *http.Request, dst any, opts *DecodeOptions) error {
	if opts == nil {
		opts = DefaultDecodeOptions()
	}

	// Gate 1: Content-Type
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		return &DecodeError{
			Message: "unsupported content type",
			Err:     ErrInvalidJSON,
		}
	}

	// Gate 2: read body, capped one byte past the limit so gate 3 can tell
	// "exactly at the limit" from "over it". io.LimitReader truncates
	// silently, so reading the extra byte is the only signal available
	// without http.MaxBytesReader (which would close the connection).
	var reader io.Reader = r.Body
	if opts.MaxBodySize > 0 {
		reader = io.LimitReader(r.Body, opts.MaxBodySize+1)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return &DecodeError{
			Message: "failed to read request body",
			Err:     err,
		}
	}

	// Gate 3: over the limit
	if opts.MaxBodySize > 0 && int64(len(data)) > opts.MaxBodySize {
		return &DecodeError{
			Message: ErrBodyTooLarge.Error(),
			Err:     ErrBodyTooLarge,
		}
	}

	// Gate 4: empty / whitespace-only
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return &DecodeError{
			Message: ErrEmptyBody.Error(),
			Err:     ErrEmptyBody,
		}
	}

	// Gate 5: decode
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	if opts.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(dst); err != nil {
		// Check for specific error types
		if strings.Contains(err.Error(), "unknown field") {
			return &DecodeError{
				Field:   unknownFieldName(err.Error()),
				Message: ErrUnknownField.Error(),
				Err:     ErrUnknownField,
			}
		}

		if strings.Contains(err.Error(), "cannot unmarshal") {
			var field string
			var ute *json.UnmarshalTypeError
			if errors.As(err, &ute) {
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

	// Gate 6: trailing content. A second Decode into an empty struct is
	// io.EOF when only whitespace remains (valid) and nil or a syntax error
	// when a further value follows. decoder.More() is not usable here: it is
	// defined for elements inside an array/object, not for a top-level
	// stream.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
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
	var de *DecodeError
	if errors.As(err, &de) {
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

	// The status line is already sent, so an encode failure (a dropped
	// connection, or a data value that cannot marshal) cannot be answered
	// with a different response; appending a plain-text body would only
	// corrupt the JSON the client is reading.
	_ = json.NewEncoder(w).Encode(response)
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

	// The status line is already sent, so an encode failure (a dropped
	// connection, or a data value that cannot marshal) cannot be answered
	// with a different response; appending a plain-text body would only
	// corrupt the JSON the client is reading.
	_ = json.NewEncoder(w).Encode(response)
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
