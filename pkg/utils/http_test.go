package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequest is a sample struct for testing JSON decoding
type TestRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Age      int    `json:"age"`
	Password string `json:"password"`
}

func TestDecodeJSONBody_ValidRequest(t *testing.T) {
	body := `{"email": "test@example.com", "name": "John Doe", "age": 30}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dst.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", dst.Email)
	}

	if dst.Name != "John Doe" {
		t.Errorf("expected name 'John Doe', got '%s'", dst.Name)
	}

	if dst.Age != 30 {
		t.Errorf("expected age 30, got %d", dst.Age)
	}
}

func TestDecodeJSONBody_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}

	if !IsDecodeError(err, ErrEmptyBody) {
		t.Errorf("expected ErrEmptyBody, got %v", err)
	}
}

func TestDecodeJSONBody_MalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing closing brace", `{"email": "test@example.com"`},
		{"invalid syntax", `{email: "test@example.com"}`},
		{"trailing comma", `{"email": "test@example.com",}`},
		{"double comma", `{"email": "test@example.com",, "name": "John"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			var dst TestRequest
			err := DecodeJSONBody(req, &dst, nil)

			if err == nil {
				t.Fatal("expected error for malformed JSON, got nil")
			}

			if !IsDecodeError(err, ErrInvalidJSON) {
				t.Errorf("expected ErrInvalidJSON, got %v", err)
			}
		})
	}
}

func TestDecodeJSONBody_InvalidFieldType(t *testing.T) {
	body := `{"email": "test@example.com", "age": "not a number"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err == nil {
		t.Fatal("expected error for invalid field type, got nil")
	}

	if !IsDecodeError(err, ErrInvalidFieldType) {
		t.Errorf("expected ErrInvalidFieldType, got %v", err)
	}

	field, _ := GetDecodeErrorDetails(err)
	if field != "age" {
		t.Errorf("expected field 'age', got '%s'", field)
	}
}

func TestDecodeJSONBody_UnknownField_Allowed(t *testing.T) {
	body := `{"email": "test@example.com", "unknown_field": "value"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	opts := &DecodeOptions{
		MaxBodySize:           1 << 20,
		DisallowUnknownFields: false,
	}
	err := DecodeJSONBody(req, &dst, opts)

	if err != nil {
		t.Fatalf("expected no error when unknown fields are allowed, got %v", err)
	}
}

func TestDecodeJSONBody_UnknownField_Disallowed(t *testing.T) {
	body := `{"email": "test@example.com", "unknown_field": "value"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	opts := &DecodeOptions{
		MaxBodySize:           1 << 20,
		DisallowUnknownFields: true,
	}
	err := DecodeJSONBody(req, &dst, opts)

	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}

	if !IsDecodeError(err, ErrUnknownField) {
		t.Errorf("expected ErrUnknownField, got %v", err)
	}

	field, _ := GetDecodeErrorDetails(err)
	if field != "unknown_field" {
		t.Errorf("expected field 'unknown_field', got '%s'", field)
	}
}

func TestDecodeJSONBody_BodyTooLarge(t *testing.T) {
	// Create a body larger than the limit
	largeBody := `{"email": "` + strings.Repeat("a", 1000) + `@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	opts := &DecodeOptions{
		MaxBodySize: 100, // Very small limit
	}
	err := DecodeJSONBody(req, &dst, opts)

	if err == nil {
		t.Fatal("expected error for body too large, got nil")
	}

	// The error could be either body too large or invalid JSON (truncated)
	if !IsDecodeError(err, ErrBodyTooLarge) && !IsDecodeError(err, ErrInvalidJSON) {
		t.Errorf("expected ErrBodyTooLarge or ErrInvalidJSON, got %v", err)
	}
}

func TestDecodeJSONBody_MultipleJSONObjects(t *testing.T) {
	body := `{"email": "test@example.com"}{"email": "another@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err == nil {
		t.Fatal("expected error for multiple JSON objects, got nil")
	}

	if !IsDecodeError(err, ErrInvalidJSON) {
		t.Errorf("expected ErrInvalidJSON, got %v", err)
	}
}

func TestDecodeJSONBody_WrongContentType(t *testing.T) {
	body := `{"email": "test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	var dst TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err == nil {
		t.Fatal("expected error for wrong content type, got nil")
	}

	if !IsDecodeError(err, ErrInvalidJSON) {
		t.Errorf("expected ErrInvalidJSON for wrong content type, got %v", err)
	}
}

func TestDecodeJSONBody_NoContentType(t *testing.T) {
	// Should work without Content-Type header (lenient)
	body := `{"email": "test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	// No Content-Type header set

	var dst TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err != nil {
		t.Fatalf("expected no error when Content-Type is missing, got %v", err)
	}

	if dst.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", dst.Email)
	}
}

func TestDecodeJSONBody_ContentTypeWithCharset(t *testing.T) {
	body := `{"email": "test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	var dst TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err != nil {
		t.Fatalf("expected no error for Content-Type with charset, got %v", err)
	}
}

func TestDecodeJSONBodyStrict(t *testing.T) {
	// Test that strict mode disallows unknown fields
	body := `{"email": "test@example.com", "unknown": "value"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	err := DecodeJSONBodyStrict(req, &dst)

	if err == nil {
		t.Fatal("expected error for unknown field in strict mode, got nil")
	}

	if !IsDecodeError(err, ErrUnknownField) {
		t.Errorf("expected ErrUnknownField, got %v", err)
	}
}

func TestDecodeJSONBodyStrict_ValidRequest(t *testing.T) {
	body := `{"email": "test@example.com", "name": "John"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	err := DecodeJSONBodyStrict(req, &dst)

	if err != nil {
		t.Fatalf("expected no error for valid request in strict mode, got %v", err)
	}
}

func TestDefaultDecodeOptions(t *testing.T) {
	opts := DefaultDecodeOptions()

	if opts.MaxBodySize != 1<<20 {
		t.Errorf("expected MaxBodySize 1MB, got %d", opts.MaxBodySize)
	}

	if opts.DisallowUnknownFields != false {
		t.Error("expected DisallowUnknownFields to be false by default")
	}
}

func TestIsDecodeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "matching DecodeError",
			err:      &DecodeError{Message: "test", Err: ErrEmptyBody},
			target:   ErrEmptyBody,
			expected: true,
		},
		{
			name:     "DecodeError with nil target",
			err:      &DecodeError{Message: "test", Err: ErrEmptyBody},
			target:   nil,
			expected: true,
		},
		{
			name:     "non-matching DecodeError",
			err:      &DecodeError{Message: "test", Err: ErrInvalidJSON},
			target:   ErrEmptyBody,
			expected: false,
		},
		{
			name:     "non-DecodeError",
			err:      errors.New("some other error"),
			target:   ErrEmptyBody,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			target:   ErrEmptyBody,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDecodeError(tt.err, tt.target)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetDecodeErrorDetails(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedField   string
		expectedMessage string
	}{
		{
			name:            "DecodeError with field",
			err:             &DecodeError{Field: "email", Message: "invalid email"},
			expectedField:   "email",
			expectedMessage: "invalid email",
		},
		{
			name:            "DecodeError without field",
			err:             &DecodeError{Message: "request body is empty"},
			expectedField:   "",
			expectedMessage: "request body is empty",
		},
		{
			name:            "non-DecodeError",
			err:             errors.New("some error"),
			expectedField:   "",
			expectedMessage: "",
		},
		{
			name:            "nil error",
			err:             nil,
			expectedField:   "",
			expectedMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, message := GetDecodeErrorDetails(tt.err)
			if field != tt.expectedField {
				t.Errorf("expected field '%s', got '%s'", tt.expectedField, field)
			}
			if message != tt.expectedMessage {
				t.Errorf("expected message '%s', got '%s'", tt.expectedMessage, message)
			}
		})
	}
}

func TestDecodeError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DecodeError
		expected string
	}{
		{
			name:     "with field",
			err:      &DecodeError{Field: "email", Message: "invalid format"},
			expected: "email: invalid format",
		},
		{
			name:     "without field",
			err:      &DecodeError{Message: "request body is empty"},
			expected: "request body is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDecodeError_Unwrap(t *testing.T) {
	underlying := ErrEmptyBody
	err := &DecodeError{Message: "test", Err: underlying}

	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Errorf("expected underlying error %v, got %v", underlying, unwrapped)
	}
}

func TestDecodeJSONBody_NilOptions(t *testing.T) {
	body := `{"email": "test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	// Passing nil options should use defaults
	err := DecodeJSONBody(req, &dst, nil)

	if err != nil {
		t.Fatalf("expected no error with nil options, got %v", err)
	}
}

func TestDecodeJSONBody_ZeroMaxBodySize(t *testing.T) {
	// Large body with zero max size (no limit)
	largeBody := `{"email": "` + strings.Repeat("a", 10000) + `@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	opts := &DecodeOptions{
		MaxBodySize: 0, // No limit
	}
	err := DecodeJSONBody(req, &dst, opts)

	if err != nil {
		t.Fatalf("expected no error with zero max body size, got %v", err)
	}
}

func TestDecodeJSONBody_NestedStruct(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	body := `{"name": "John", "address": {"street": "123 Main St", "city": "NYC"}}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst Person
	err := DecodeJSONBody(req, &dst, nil)

	if err != nil {
		t.Fatalf("expected no error for nested struct, got %v", err)
	}

	if dst.Name != "John" {
		t.Errorf("expected name 'John', got '%s'", dst.Name)
	}
	if dst.Address.Street != "123 Main St" {
		t.Errorf("expected street '123 Main St', got '%s'", dst.Address.Street)
	}
	if dst.Address.City != "NYC" {
		t.Errorf("expected city 'NYC', got '%s'", dst.Address.City)
	}
}

func TestDecodeJSONBody_Array(t *testing.T) {
	body := `[{"email": "test1@example.com"}, {"email": "test2@example.com"}]`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst []TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err != nil {
		t.Fatalf("expected no error for array, got %v", err)
	}

	if len(dst) != 2 {
		t.Fatalf("expected 2 items, got %d", len(dst))
	}

	if dst[0].Email != "test1@example.com" {
		t.Errorf("expected first email 'test1@example.com', got '%s'", dst[0].Email)
	}
	if dst[1].Email != "test2@example.com" {
		t.Errorf("expected second email 'test2@example.com', got '%s'", dst[1].Email)
	}
}

func TestDecodeJSONBody_WhitespaceBody(t *testing.T) {
	body := `   `
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err == nil {
		t.Fatal("expected error for whitespace-only body, got nil")
	}

	if !IsDecodeError(err, ErrEmptyBody) {
		t.Errorf("expected ErrEmptyBody, got %v", err)
	}
}

func TestDecodeJSONBody_NullValue(t *testing.T) {
	body := `null`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst *TestRequest
	err := DecodeJSONBody(req, &dst, nil)

	if err != nil {
		t.Fatalf("expected no error for null value, got %v", err)
	}

	if dst != nil {
		t.Errorf("expected nil, got %+v", dst)
	}
}

func TestDecodeJSONBody_Map(t *testing.T) {
	body := `{"key1": "value1", "key2": "value2"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var dst map[string]string
	err := DecodeJSONBody(req, &dst, nil)

	if err != nil {
		t.Fatalf("expected no error for map, got %v", err)
	}

	if dst["key1"] != "value1" {
		t.Errorf("expected key1='value1', got '%s'", dst["key1"])
	}
	if dst["key2"] != "value2" {
		t.Errorf("expected key2='value2', got '%s'", dst["key2"])
	}
}

func BenchmarkDecodeJSONBody(b *testing.B) {
	body := `{"email": "test@example.com", "name": "John Doe", "age": 30, "password": "secret123"}`
	bodyBytes := []byte(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		var dst TestRequest
		_ = DecodeJSONBody(req, &dst, nil)
	}
}

// ============================================================================
// Standard API Response Format Tests (ADR-002)
// ============================================================================

func TestWriteSuccessResponse(t *testing.T) {
	tests := []struct {
		name         string
		data         interface{}
		expectedCode int
	}{
		{
			name: "simple map data",
			data: map[string]string{
				"user_id": "123",
				"email":   "user@example.com",
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "struct data",
			data: TestRequest{
				Email: "test@example.com",
				Name:  "John Doe",
				Age:   30,
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "nil data",
			data:         nil,
			expectedCode: http.StatusOK,
		},
		{
			name:         "array data",
			data:         []string{"item1", "item2", "item3"},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteSuccessResponse(w, tt.data)

			// Verify HTTP status code
			if w.Code != tt.expectedCode {
				t.Errorf("expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			// Verify Content-Type header
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
			}

			// Parse response body
			var response ApiResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Verify response structure
			if !response.Success {
				t.Error("expected success=true, got false")
			}

			if response.Error != nil {
				t.Errorf("expected error=nil, got %+v", response.Error)
			}

			// Verify data is present (except for nil data test)
			if tt.data != nil && response.Data == nil {
				t.Error("expected data to be present, got nil")
			}
		})
	}
}

func TestWriteErrorResponse(t *testing.T) {
	tests := []struct {
		name               string
		statusCode         int
		errorType          string
		code               string
		message            string
		details            map[string]interface{}
		expectedStatusCode int
	}{
		{
			name:               "validation error",
			statusCode:         http.StatusBadRequest,
			errorType:          ErrorTypeValidation,
			code:               "EMAIL_REQUIRED",
			message:            "Email là bắt buộc",
			details:            map[string]interface{}{"field": "email"},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "business error - conflict",
			statusCode:         http.StatusConflict,
			errorType:          ErrorTypeBusiness,
			code:               "EMAIL_EXISTS",
			message:            "Email đã được sử dụng",
			details:            nil,
			expectedStatusCode: http.StatusConflict,
		},
		{
			name:               "authentication error",
			statusCode:         http.StatusUnauthorized,
			errorType:          ErrorTypeBusiness,
			code:               "INVALID_PASSWORD",
			message:            "Mật khẩu không chính xác",
			details:            nil,
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "system error",
			statusCode:         http.StatusInternalServerError,
			errorType:          ErrorTypeSystem,
			code:               "INTERNAL_ERROR",
			message:            "Lỗi hệ thống",
			details:            nil,
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "not found error",
			statusCode:         http.StatusNotFound,
			errorType:          ErrorTypeBusiness,
			code:               "USER_NOT_FOUND",
			message:            "Tài khoản không tồn tại",
			details:            map[string]interface{}{"user_id": "123"},
			expectedStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteErrorResponse(w, tt.statusCode, tt.errorType, tt.code, tt.message, tt.details)

			// Verify HTTP status code
			if w.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			// Verify Content-Type header
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
			}

			// Parse response body
			var response ApiResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Verify response structure
			if response.Success {
				t.Error("expected success=false, got true")
			}

			if response.Data != nil {
				t.Errorf("expected data=nil, got %+v", response.Data)
			}

			if response.Error == nil {
				t.Fatal("expected error to be present, got nil")
			}

			// Verify error details
			if response.Error.Code != tt.code {
				t.Errorf("expected code '%s', got '%s'", tt.code, response.Error.Code)
			}

			if response.Error.Message != tt.message {
				t.Errorf("expected message '%s', got '%s'", tt.message, response.Error.Message)
			}

			if response.Error.Type != tt.errorType {
				t.Errorf("expected type '%s', got '%s'", tt.errorType, response.Error.Type)
			}

			// Verify details if provided
			if tt.details != nil && response.Details == nil {
				t.Error("expected details to be present, got nil")
			}
		})
	}
}

func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteValidationError(w, "EMAIL_REQUIRED", "Email là bắt buộc",
		map[string]interface{}{"field": "email"})

	// Verify HTTP status code
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Parse response
	var response ApiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify it's a validation error
	if response.Error.Type != ErrorTypeValidation {
		t.Errorf("expected error type '%s', got '%s'", ErrorTypeValidation, response.Error.Type)
	}

	if response.Error.Code != "EMAIL_REQUIRED" {
		t.Errorf("expected code 'EMAIL_REQUIRED', got '%s'", response.Error.Code)
	}
}

func TestWriteBusinessError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		code       string
		message    string
	}{
		{
			name:       "conflict error",
			statusCode: http.StatusConflict,
			code:       "EMAIL_EXISTS",
			message:    "Email đã được sử dụng",
		},
		{
			name:       "unauthorized error",
			statusCode: http.StatusUnauthorized,
			code:       "INVALID_PASSWORD",
			message:    "Mật khẩu không chính xác",
		},
		{
			name:       "forbidden error",
			statusCode: http.StatusForbidden,
			code:       "ACCOUNT_BLOCKED",
			message:    "Tài khoản đã bị khóa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteBusinessError(w, tt.statusCode, tt.code, tt.message, nil)

			// Verify HTTP status code
			if w.Code != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, w.Code)
			}

			// Parse response
			var response ApiResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Verify it's a business error
			if response.Error.Type != ErrorTypeBusiness {
				t.Errorf("expected error type '%s', got '%s'", ErrorTypeBusiness, response.Error.Type)
			}

			if response.Error.Code != tt.code {
				t.Errorf("expected code '%s', got '%s'", tt.code, response.Error.Code)
			}
		})
	}
}

func TestWriteSystemError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteSystemError(w, "DATABASE_ERROR", "Lỗi kết nối cơ sở dữ liệu", nil)

	// Verify HTTP status code
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}

	// Parse response
	var response ApiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify it's a system error
	if response.Error.Type != ErrorTypeSystem {
		t.Errorf("expected error type '%s', got '%s'", ErrorTypeSystem, response.Error.Type)
	}

	if response.Error.Code != "DATABASE_ERROR" {
		t.Errorf("expected code 'DATABASE_ERROR', got '%s'", response.Error.Code)
	}
}

func TestApiResponse_SuccessStructure(t *testing.T) {
	// Test that ApiResponse has the correct JSON structure
	response := ApiResponse{
		Success: true,
		Data: map[string]string{
			"id":    "123",
			"email": "test@example.com",
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// Verify JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check required fields
	if success, ok := parsed["success"].(bool); !ok || !success {
		t.Error("expected success field to be true")
	}

	if _, ok := parsed["data"]; !ok {
		t.Error("expected data field to be present")
	}

	// Error should not be in successful response
	if _, ok := parsed["error"]; ok {
		t.Error("expected error field to be omitted in success response")
	}
}

func TestApiResponse_ErrorStructure(t *testing.T) {
	// Test that ApiResponse error has the correct JSON structure
	response := ApiResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    "EMAIL_REQUIRED",
			Message: "Email là bắt buộc",
			Type:    ErrorTypeValidation,
		},
		Details: map[string]interface{}{
			"field": "email",
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// Verify JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check required fields
	if success, ok := parsed["success"].(bool); !ok || success {
		t.Error("expected success field to be false")
	}

	// Data should not be in error response
	if _, ok := parsed["data"]; ok {
		t.Error("expected data field to be omitted in error response")
	}

	// Error should be present
	errObj, ok := parsed["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error field to be present")
	}

	if code, ok := errObj["code"].(string); !ok || code != "EMAIL_REQUIRED" {
		t.Errorf("expected error code 'EMAIL_REQUIRED', got '%v'", errObj["code"])
	}

	if typ, ok := errObj["type"].(string); !ok || typ != ErrorTypeValidation {
		t.Errorf("expected error type '%s', got '%v'", ErrorTypeValidation, errObj["type"])
	}

	// Details should be present
	if _, ok := parsed["details"]; !ok {
		t.Error("expected details field to be present")
	}
}
