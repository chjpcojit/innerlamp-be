package validation

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

// dto exercises the tags the wrapper is expected to pass straight through to
// the library, plus the field-name tag fallbacks.
type dto struct {
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"required,phone"`
	Nickname string `json:"nickname,omitempty" validate:"omitempty,min=2"`
	Untagged string `json:"untagged"`
	Hidden   string `json:"-" validate:"omitempty,min=2"`
	NoName   string `json:",omitempty" validate:"omitempty,min=2"`
}

// newTestValidator mirrors what cmd/main.go does: build, pick the wire format,
// register the service's custom tags — all before the first validate.
func newTestValidator(t *testing.T) *Validator {
	t.Helper()

	v := New()
	v.SetFieldNameTag("json")
	if err := v.Register(PhoneTag, PhoneRule); err != nil {
		t.Fatalf("Register(%q) error = %v", PhoneTag, err)
	}
	return v
}

func TestNewRegistersNothing(t *testing.T) {
	// A fresh Validator must not know `phone` — the tag only exists once the
	// caller registers it. The library panics on an unregistered tag used in a
	// struct, which is how this surfaces (loudly, at the first request).
	v := New()

	defer func() {
		if recover() == nil {
			t.Error("expected panic validating a struct that uses an unregistered tag")
		}
	}()

	_ = v.Struct(struct {
		Phone string `validate:"phone"`
	}{Phone: "5512345678"})
}

func TestStructUsesJSONFieldNames(t *testing.T) {
	v := newTestValidator(t)

	err := v.Struct(dto{})
	if err == nil {
		t.Fatal("expected validation errors for an empty dto")
	}

	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		t.Fatalf("error = %v, want validator.ValidationErrors", err)
	}

	// Empty dto fails on email and phone; both must be reported under their
	// JSON names, not the Go field names.
	got := map[string]string{}
	for _, fe := range fieldErrs {
		got[fe.Field()] = fe.Tag()
	}

	if tag, ok := got["email"]; !ok || tag != "required" {
		t.Errorf("errors = %v, want an entry email=required", got)
	}
	if tag, ok := got["phone"]; !ok || tag != "required" {
		t.Errorf("errors = %v, want an entry phone=required", got)
	}
}

func TestFieldNameFallsBackToGoName(t *testing.T) {
	v := newTestValidator(t)

	// Hidden and NoName both carry `json:"-"` / `json:",omitempty"`, which
	// resolve to no usable name → the library must fall back to the Go field
	// name so the error is never keyed by an empty string.
	err := v.Struct(dto{Hidden: "x", NoName: "y"})
	if err == nil {
		t.Fatal("expected validation errors")
	}

	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		t.Fatalf("error = %v, want validator.ValidationErrors", err)
	}

	got := map[string]bool{}
	for _, fe := range fieldErrs {
		got[fe.Field()] = true
	}

	if !got["Hidden"] {
		t.Errorf("errors = %v, want field name Hidden", got)
	}
	if !got["NoName"] {
		t.Errorf("errors = %v, want field name NoName", got)
	}
}

func TestStructWithoutFieldNameTagUsesGoNames(t *testing.T) {
	// No SetFieldNameTag call → errors carry Go field names.
	v := New()

	err := v.Struct(struct {
		Email string `json:"email" validate:"required"`
	}{})
	if err == nil {
		t.Fatal("expected a validation error")
	}

	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		t.Fatalf("error = %v, want validator.ValidationErrors", err)
	}
	if fieldErrs[0].Field() != "Email" {
		t.Errorf("field = %q, want %q", fieldErrs[0].Field(), "Email")
	}
}

func TestStructAccepts(t *testing.T) {
	v := newTestValidator(t)

	err := v.Struct(dto{
		Email:    "user@example.com",
		Phone:    "+84 912 345 678",
		Nickname: "an",
	})
	if err != nil {
		t.Errorf("Struct() error = %v, want nil", err)
	}
}

func TestVar(t *testing.T) {
	v := newTestValidator(t)

	if err := v.Var("+84912345678", PhoneTag); err != nil {
		t.Errorf("Var(%q) error = %v, want nil", "+84912345678", err)
	}
	if err := v.Var("not-a-phone", PhoneTag); err == nil {
		t.Error("Var(\"not-a-phone\") error = nil, want an error")
	}
}

func TestRegisterRejectsEmptyTag(t *testing.T) {
	v := New()

	if err := v.Register("", PhoneRule); err == nil {
		t.Error("Register(\"\") error = nil, want an error")
	}
}

func TestRegisterRejectsNilFunc(t *testing.T) {
	v := New()

	if err := v.Register("custom", nil); err == nil {
		t.Error("Register with a nil rule error = nil, want an error")
	}
}

func TestRegisterPanicsOnBuiltinTag(t *testing.T) {
	v := New()

	defer func() {
		if recover() == nil {
			t.Error("expected panic registering a tag that shadows a built-in")
		}
	}()

	_ = v.Register("required", PhoneRule)
}

func TestRegisterPanicsOnReservedCharacters(t *testing.T) {
	v := New()

	defer func() {
		if recover() == nil {
			t.Error("expected panic registering a tag with reserved characters")
		}
	}()

	_ = v.Register("pho|ne", PhoneRule)
}

func TestRegisterOverwritesSilently(t *testing.T) {
	v := New()

	alwaysTrue := func(validator.FieldLevel) bool { return true }
	alwaysFalse := func(validator.FieldLevel) bool { return false }

	if err := v.Register("custom", alwaysTrue); err != nil {
		t.Fatalf("first Register error = %v", err)
	}
	// The library does not report a duplicate registration — it overwrites.
	if err := v.Register("custom", alwaysFalse); err != nil {
		t.Fatalf("second Register error = %v, want nil (silent overwrite)", err)
	}

	err := v.Var("anything", "custom")
	if err == nil {
		t.Error("Var() error = nil, want the second rule to have taken effect")
	}
}

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{"Uppercase", "TEST@EXAMPLE.COM", "test@example.com"},
		{"Mixed case", "TeSt@ExAmPlE.CoM", "test@example.com"},
		{"With spaces", "  test@example.com  ", "test@example.com"},
		{"Already normalized", "test@example.com", "test@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := NormalizeEmail(tt.email); got != tt.want {
				t.Errorf("NormalizeEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPhoneRule(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		{"Plain digits", "84912345678", true},
		{"With country code", "+84912345678", true},
		{"Spaces", "+84 912 345 678", true},
		{"Dashes and parens", "(912) 345-678", true},
		{"Dots", "912.345.678", true},
		{"Minimum digits", "12345678", true},
		{"Maximum digits", "123456789012345", true},
		{"Trimmed", "  +84912345678  ", true},

		{"Empty", "", false},
		{"Whitespace only", "   ", false},
		{"Too few digits", "1234567", false},
		{"Too many digits", "1234567890123456", false},
		{"Letters", "8491234abcd", false},
		{"Second plus", "+84+912345678", false},
		{"Plus alone", "+", false},
		{"Plus with separators", "+ - . ( )", false},
		{"Non-ASCII digits", "+84٩١٢٣٤٥٦٧٨", false},
		{"Control character", "+8491234\x005678", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPhone(tt.phone); got != tt.want {
				t.Errorf("IsValidPhone(%q) = %v, want %v", tt.phone, got, tt.want)
			}
		})
	}
}

func TestPhoneTagMatchesRule(t *testing.T) {
	if PhoneTag == "" {
		t.Fatal("PhoneTag must not be empty")
	}
	if strings.ContainsAny(PhoneTag, `.[],|=+()~!@#$%^&*"/?<>{}`) {
		t.Errorf("PhoneTag %q contains characters reserved by the library", PhoneTag)
	}
}
