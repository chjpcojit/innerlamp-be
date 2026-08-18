# Validation Package

Comprehensive input validation and sanitization package for Mexican onboarding data with XSS prevention and format validation.

## Features

- **Mexican Phone Number Validation** - 10-digit format with +52 country code support
- **Email Validation** - RFC-compliant email format validation
- **CURP Validation** - Mexican citizen ID format validation (18 characters)
- **XSS Prevention** - Detection and sanitization of cross-site scripting attacks
- **Postal Code Validation** - Mexican 5-digit postal code format
- **INE Key Validation** - Mexican credential number (13 or 18 digits)
- **General Validators** - Required, length, alphanumeric, alpha, numeric validations
- **Fluent API** - Builder pattern for accumulating multiple validation errors

## Installation

```go
import "base_project/pkg/validation"
```

## Quick Start

### Mexican Phone Number Validation

```go
// Validate phone number
err := validation.ValidateMexicanPhone("5512345678")
if err != nil {
    // Handle error
}

// Also accepts +52 format
err = validation.ValidateMexicanPhone("+525512345678")

// Normalize phone number to 10 digits
normalized := validation.NormalizeMexicanPhone("55-1234-5678")
// Result: "5512345678"
```

### Email Validation

```go
// Validate email format
err := validation.ValidateEmail("user@example.com")
if err != nil {
    // Handle error
}

// Normalize email (lowercase, trim)
normalized := validation.NormalizeEmail("  USER@EXAMPLE.COM  ")
// Result: "user@example.com"
```

### CURP Validation

```go
// Validate CURP format (18 characters)
err := validation.ValidateCURP("HEGG560427MVZRRL04")
if err != nil {
    // Handle error
}

// Normalize CURP (uppercase, trim)
normalized := validation.NormalizeCURP("  hegg560427mvzrrl04  ")
// Result: "HEGG560427MVZRRL04"
```

### XSS Prevention

```go
// Check for XSS patterns
err := validation.ValidateNoXSS("<script>alert('xss')</script>")
if err != nil {
    // Contains XSS - reject input
}

// Sanitize HTML in strings
sanitized := validation.SanitizeString("<b>Hello</b>")
// Result: "&lt;b&gt;Hello&lt;/b&gt;"

// Comprehensive sanitization
clean := validation.SanitizeInput("  <script>alert()</script>  ")
// Result: Trimmed, HTML-escaped, control chars removed
```

### Postal Code Validation

```go
// Validate Mexican postal code (5 digits)
err := validation.ValidatePostalCode("03940")
if err != nil {
    // Handle error
}
```

### INE Key Validation

```go
// Validate INE credential number (13 or 18 digits)
err := validation.ValidateINEKey("1234567890123")
if err != nil {
    // Handle error
}
```

### General Validators

```go
// Required field
err := validation.ValidateRequired("value", "fieldName")

// String length
err := validation.ValidateStringLength("value", "fieldName", 3, 10)

// Alphanumeric only
err := validation.ValidateAlphanumeric("ABC123", "fieldName")

// Letters only (allows spaces)
err := validation.ValidateAlpha("Hello World", "fieldName")

// Numbers only
err := validation.ValidateNumeric("12345", "fieldName")
```

### Fluent Validator API

```go
// Create validator
v := validation.NewValidator()

// Add multiple validations
v.ValidateField(validation.ValidateRequired(name, "name"))
v.ValidateField(validation.ValidateEmail(email))
v.ValidateField(validation.ValidateCURP(curp))
v.ValidateField(validation.ValidateMexicanPhone(phone))

// Check if any errors occurred
if v.HasErrors() {
    // Get all errors
    errors := v.Errors()
    for _, err := range errors {
        fmt.Printf("Field: %s, Message: %s\n", err.Field, err.Message)
    }

    // Or get combined error
    return v.Error()
}
```

## Usage Examples

### User Registration Validation

```go
func ValidateUserRegistration(req *RegistrationRequest) error {
    v := validation.NewValidator()

    // Required fields
    v.ValidateField(validation.ValidateRequired(req.Email, "email"))
    v.ValidateField(validation.ValidateRequired(req.Phone, "phone"))
    v.ValidateField(validation.ValidateRequired(req.CURP, "curp"))

    // Format validation
    v.ValidateField(validation.ValidateEmail(req.Email))
    v.ValidateField(validation.ValidateMexicanPhone(req.Phone))
    v.ValidateField(validation.ValidateCURP(req.CURP))

    // XSS prevention on text fields
    v.ValidateField(validation.ValidateNoXSS(req.FirstName))
    v.ValidateField(validation.ValidateNoXSS(req.LastName))

    if v.HasErrors() {
        return v.Error()
    }

    return nil
}
```

### Address Validation

```go
func ValidateAddress(addr *Address) error {
    v := validation.NewValidator()

    // Required fields
    v.ValidateField(validation.ValidateRequired(addr.Street, "street"))
    v.ValidateField(validation.ValidateRequired(addr.City, "city"))
    v.ValidateField(validation.ValidateRequired(addr.PostalCode, "postalCode"))

    // Postal code format
    v.ValidateField(validation.ValidatePostalCode(addr.PostalCode))

    // XSS prevention
    v.ValidateField(validation.ValidateNoXSS(addr.Street))
    v.ValidateField(validation.ValidateNoXSS(addr.City))

    return v.Error()
}
```

### Sanitization Before Storage

```go
func SanitizeUserInput(user *User) {
    // Normalize data
    user.Email = validation.NormalizeEmail(user.Email)
    user.Phone = validation.NormalizeMexicanPhone(user.Phone)
    user.CURP = validation.NormalizeCURP(user.CURP)

    // Sanitize text fields
    user.FirstName = validation.SanitizeInput(user.FirstName)
    user.LastName = validation.SanitizeInput(user.LastName)
    user.Address = validation.SanitizeInput(user.Address)
}
```

## XSS Attack Patterns Detected

The package detects the following XSS attack vectors:

1. **Script Tags**: `<script>`, `</script>`
2. **Iframes**: `<iframe`, `</iframe>`
3. **JavaScript Protocol**: `javascript:`
4. **Event Handlers**: `onclick=`, `onload=`, `onerror=`, etc.
5. **Embed Tags**: `<embed`, `</embed>`
6. **Object Tags**: `<object`, `</object>`

## Mexican Format Specifications

### Phone Number Format

- **10 digits**: `5512345678`
- **With country code**: `+525512345678`
- **Accepts formatting**: Spaces, dashes, parentheses (normalized to 10 digits)

### CURP Format

- **18 characters**: `AAAA######HMCNNN##`
- **First 4**: Letters (apellido paterno, apellido materno, nombre)
- **Next 6**: Date of birth (YYMMDD)
- **Next 1**: Sex indicator (H=Male, M=Female)
- **Next 5**: Letters (state + internal consonants)
- **Last 2**: Check digits

### Postal Code Format

- **5 digits**: `03940`

### INE Key Format

- **13 digits**: Credential number
- **18 digits**: Full credential number with check digits

## Error Types

### ValidationError

```go
type ValidationError struct {
    Field   string // Field name that failed validation
    Message string // Human-readable error message
}
```

Example errors:

- `"email: invalid email format"`
- `"phone: invalid Mexican phone number format"`
- `"curp: invalid CURP format (must be 18 characters)"`
- `"name: contains potentially malicious content (XSS detected)"`

## Security Best Practices

1. **Always validate input** before processing or storing data
2. **Use XSS prevention** on all user-generated text fields
3. **Normalize data** before storage for consistency
4. **Sanitize output** when displaying user data in HTML context
5. **Combine validation** with encryption for PII data (use with `pkg/encryption`)
6. **Log validation failures** for security monitoring (use with `pkg/audit`)

## Integration with Other Packages

### With Encryption Package

```go
import (
    "base_project/pkg/validation"
    "base_project/pkg/encryption"
)

// Validate then encrypt PII
v := validation.NewValidator()
v.ValidateField(validation.ValidateCURP(curp))

if v.HasErrors() {
    return v.Error()
}

normalizedCURP := validation.NormalizeCURP(curp)
encryptedCURP, err := encryptionService.Encrypt(normalizedCURP)
```

### With Audit Package

```go
import (
    "base_project/pkg/validation"
    "base_project/pkg/audit"
)

// Log validation failures
if err := validation.ValidateEmail(email); err != nil {
    audit.LogFromContext(ctx).
        WithEventType(audit.EventTypeValidationError).
        WithFields(audit.Fields{
            "field": "email",
            "error": err.Error(),
        }).
        Log()
    return err
}
```

## Performance Considerations

- **Regex Compilation**: All regex patterns are pre-compiled at package initialization
- **Normalization**: Lightweight string operations with minimal allocations
- **Sanitization**: HTML escaping is standard library implementation (fast)
- **Validation**: O(1) regex matching for format validation

## Testing

Run tests with coverage:

```bash
go test ./pkg/validation/... -v -cover
```

## Examples

See usage examples in tests: [validation_test.go](validation_test.go)

## Related Packages

- [pkg/encryption](../encryption/README.md) - AES-256-GCM encryption for PII data
- [pkg/audit](../audit/README.md) - Compliance audit logging
- [pkg/password](../password/README.md) - Bcrypt password hashing

## License

Part of base_project microservices architecture.
