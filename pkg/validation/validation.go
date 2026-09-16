// Package validation wraps github.com/go-playground/validator/v10 behind a
// single type, so the library instance is created in exactly one place
// (the composition root) instead of being re-created per layer.
//
// The package ships one example rule in phone.go and no other; it does not
// decide the wire format of error field names. Callers register the custom
// tags they need via Register and pick the field-name tag via
// SetFieldNameTag. Tags the library already provides (required, email, min,
// max, e164, oneof, ...) are used directly as struct tags and are deliberately
// not re-wrapped here.
package validation

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator holds the underlying library instance. It is the value passed
// between layers. The inner field is unexported on purpose: every operation
// goes through Struct/Var/Register/SetFieldNameTag, so the library surface
// touches the repo through one door instead of leaking a *validator.Validate
// into every consumer.
//
// The library documents its instance as thread-safe and intended to be used
// as a singleton (it caches parsed tags/structs behind its own lock), so each
// service builds one Validator and shares it. Configuring it is the exception:
// the library's Register* methods (including RegisterTagNameFunc) are NOT
// thread-safe and must all run before the first validation, which is why
// Register and SetFieldNameTag belong at the composition root.
type Validator struct {
	validate *validator.Validate
}

// New builds a Validator. It registers no custom tag and picks no wire
// format — both are the caller's decision. Until the composition root calls
// Register/SetFieldNameTag, only built-in tags work and validation errors
// carry the Go field name.
func New() *Validator {
	return &Validator{validate: validator.New()}
}

// Register binds a struct-tag name to the function that implements it, for
// rules the library does not provide:
//
//	v.Register(validation.PhoneTag, validation.PhoneRule)
//	// struct tag: `validate:"required,phone"`
//
// The function belongs to the caller; this package does not keep a list of
// "tags a service should have".
//
// Three ways a bad registration shows up:
//
//   - empty tag, or nil fn: the library returns an error ("function Key
//     cannot be empty" / "function cannot be empty")
//   - a tag that collides with a built-in (required, dive, omitempty, ...)
//     or contains a reserved character (.[],|=+()`~!@#$%^&*"/?<>{}): the
//     library PANICS instead of returning an error
//   - a tag already registered: no error, no panic — it silently overwrites
//     the previous function
//
// All three are programming errors that surface at startup, so keep Register
// calls at the composition root. Any tag used in a struct tag but never
// registered makes the library panic ("Undefined validation function") on the
// first validation rather than skipping the rule.
func (v *Validator) Register(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}

// SetFieldNameTag makes validation errors report fields by the given struct
// tag instead of the Go field name:
//
//	v.SetFieldNameTag("json")
//	// PhoneNumber string `json:"phone_number"` -> error says "phone_number"
//
// The library does NOT read the json tag on its own. By default it reports the
// Go field name (PhoneNumber); json:"..." only means something to
// encoding/json, and no package understands another package's tags. Callers
// that never call this get Go names.
//
// Which tag to pass is the delivery layer's decision: "json", "form", "xml"...
// The name is taken up to the first comma, so json:"nickname,omitempty" yields
// "nickname". When the tag is absent, empty (json:",omitempty") or json:"-",
// the Go field name is kept so the response always has a key.
//
// Like Register, this is not thread-safe, and the library caches field names
// on the first struct validation — call it at the composition root, before
// validating anything.
func (v *Validator) SetFieldNameTag(tag string) {
	v.validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get(tag), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// Struct validates s against its `validate:"..."` struct tags.
//
// The returned error is the library's own — validator.ValidationErrors when a
// field fails — so callers still use errors.As to find which field/tag broke.
func (v *Validator) Struct(s any) error {
	return v.validate.Struct(s)
}

// Var validates a single value that is not part of a struct (path/query
// params), against the given tag string.
func (v *Validator) Var(field any, tag string) error {
	return v.validate.Var(field, tag)
}

// NormalizeEmail lowercases and trims whitespace. Every email comparison or
// lookup MUST go through this so that User@Example.com and user@example.com
// are the same account. It does not check whether the email is well formed —
// that is the email tag's job at the delivery layer.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
