package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileValidationError_MatchesSentinel(t *testing.T) {
	err := error(&ProfileValidationError{Field: "upload_id", Message: "must be a UUID"})

	assert.True(t, errors.Is(err, ErrProfileInvalid))
	assert.Equal(t, "upload_id: must be a UUID", err.Error())
}

func TestProfilePatch_IsEmpty(t *testing.T) {
	name := "Thanh Liên"
	tests := []struct {
		name  string
		patch ProfilePatch
		empty bool
	}{
		{"nothing present", ProfilePatch{}, true},
		{"dharma name set", ProfilePatch{DharmaName: OptionalString{Set: true, Value: &name}}, false},
		{"dharma name cleared", ProfilePatch{DharmaName: OptionalString{Set: true}}, false},
		{"temple cleared", ProfilePatch{OrdinationTempleID: OptionalString{Set: true}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.empty, tt.patch.IsEmpty())
		})
	}
}
