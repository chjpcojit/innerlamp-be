package validation

import (
	"strings"
	"testing"
)

func TestValidateMexicanPhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"Valid 10 digits", "5512345678", false},
		{"Valid with +52", "+525512345678", false},
		{"Empty", "", true},
		{"Too short", "12345", true},
		{"Too long", "123456789012", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMexicanPhone(tt.phone)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMexicanPhone() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeMexicanPhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{"Clean number", "5512345678", "5512345678"},
		{"With +52", "+525512345678", "5512345678"},
		{"With spaces", "55 1234 5678", "5512345678"},
		{"With dashes", "55-1234-5678", "5512345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMexicanPhone(tt.phone)
			if got != tt.want {
				t.Errorf("NormalizeMexicanPhone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"Valid simple", "test@example.com", false},
		{"Valid with subdomain", "user@mail.example.com", false},
		{"Valid with plus", "user+tag@example.com", false},
		{"Valid with dots", "first.last@example.com", false},
		{"Empty", "", true},
		{"No @", "testexample.com", true},
		{"No domain", "test@", true},
		{"No local part", "@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeEmail(tt.email)
			if got != tt.want {
				t.Errorf("NormalizeEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCURP(t *testing.T) {
	tests := []struct {
		name    string
		curp    string
		wantErr bool
	}{
		{"Valid male", "HEGG560427MVZRRL04", false},
		{"Valid female", "MACE920715MDFRRL09", false},
		{"Valid lowercase", "hegg560427mvzrrl04", false},
		{"Empty", "", true},
		{"Too short", "HEGG560427MVZ", true},
		{"Too long", "HEGG560427MVZRRL0400", true},
		{"Invalid format", "1234567890123456", true},
		{"Invalid sex", "HEGG560427XVZRRL04", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCURP(tt.curp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCURP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeCURP(t *testing.T) {
	tests := []struct {
		name string
		curp string
		want string
	}{
		{"Lowercase", "hegg560427mvzrrl04", "HEGG560427MVZRRL04"},
		{"With spaces", "  HEGG560427MVZRRL04  ", "HEGG560427MVZRRL04"},
		{"Mixed", "  hegg560427MVZRRL04  ", "HEGG560427MVZRRL04"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeCURP(tt.curp)
			if got != tt.want {
				t.Errorf("NormalizeCURP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateNoXSS(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Clean text", "Hello World", false},
		{"Script tag", "<script>alert('xss')</script>", true},
		{"Iframe", "<iframe src='evil.com'></iframe>", true},
		{"JavaScript protocol", "javascript:alert('xss')", true},
		{"Onclick event", "<div onclick='alert()'>test</div>", true},
		{"Embed tag", "<embed src='evil.swf'>", true},
		{"Object tag", "<object data='evil.swf'>", true},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNoXSS(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNoXSS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"HTML tags", "<b>Hello</b>", "&lt;b&gt;Hello&lt;/b&gt;"},
		{"Script tag", "<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"Quotes", `"Hello"`, "&#34;Hello&#34;"},
		{"Ampersand", "A & B", "A &amp; B"},
		{"Null bytes", "Hello\x00World", "HelloWorld"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{"Trims whitespace", "  Hello  ", func(s string) bool { return s == "Hello" }},
		{"Removes control chars", "Hello\x01\x02World", func(s string) bool { return !strings.Contains(s, "\x01") }},
		{"Preserves newlines", "Hello\nWorld", func(s string) bool { return strings.Contains(s, "\n") }},
		{"Removes null bytes", "Hello\x00World", func(s string) bool { return !strings.Contains(s, "\x00") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeInput(tt.input)
			if !tt.check(got) {
				t.Errorf("SanitizeInput() = %v, failed check", got)
			}
		})
	}
}

func TestValidateStringLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		min     int
		max     int
		wantErr bool
	}{
		{"Within range", "Hello", 3, 10, false},
		{"Too short", "Hi", 5, 10, true},
		{"Too long", "Hello World!", 3, 10, true},
		{"Exact min", "Hello", 5, 10, false},
		{"Exact max", "HelloWorld", 5, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringLength(tt.value, "field", tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStringLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"Valid", "Hello", false},
		{"Empty", "", true},
		{"Whitespace only", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.value, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequired() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAlphanumeric(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"Valid", "Hello123", false},
		{"With space", "Hello 123", true},
		{"With special char", "Hello@123", true},
		{"Letters only", "Hello", false},
		{"Numbers only", "123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAlphanumeric(tt.value, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAlphanumeric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAlpha(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"Valid", "Hello", false},
		{"With space", "Hello World", false},
		{"With number", "Hello1", true},
		{"With special char", "Hello!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAlpha(tt.value, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAlpha() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNumeric(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"Valid", "12345", false},
		{"With letter", "123a5", true},
		{"With space", "123 45", true},
		{"With special char", "123-45", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNumeric(tt.value, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNumeric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePostalCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"Valid", "03940", false},
		{"Too short", "0394", true},
		{"Too long", "039400", true},
		{"With letters", "0394A", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePostalCode(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePostalCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateINEKey(t *testing.T) {
	tests := []struct {
		name    string
		ineKey  string
		wantErr bool
	}{
		{"Valid 13 digits", "1234567890123", false},
		{"Valid 18 digits", "123456789012345678", false},
		{"Too short", "12345", true},
		{"Invalid length", "12345678901234", true},
		{"With letters", "12345678901A3", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateINEKey(tt.ineKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateINEKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator(t *testing.T) {
	v := NewValidator()

	v.ValidateField(ValidateRequired("", "name"))
	v.ValidateField(ValidateEmail("invalid-email"))
	v.ValidateField(ValidateCURP("invalid"))

	if !v.HasErrors() {
		t.Error("Expected validator to have errors")
	}

	if len(v.Errors()) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(v.Errors()))
	}

	err := v.Error()
	if err == nil {
		t.Error("Expected error from validator")
	}
}

func TestValidatorNoErrors(t *testing.T) {
	v := NewValidator()

	v.ValidateField(ValidateRequired("value", "name"))
	v.ValidateField(ValidateEmail("test@example.com"))

	if v.HasErrors() {
		t.Error("Expected validator to have no errors")
	}

	err := v.Error()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
