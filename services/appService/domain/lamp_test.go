package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLampType_Valid(t *testing.T) {
	for _, tt := range AllLampTypes {
		assert.True(t, tt.Valid(), string(tt))
	}
	for _, bad := range []LampType{"", "oil_lamp", "CANDLE", "OIL_LAMP "} {
		assert.False(t, bad.Valid(), string(bad))
	}
	assert.Len(t, AllLampTypes, 3)
}

func TestIntention_Valid(t *testing.T) {
	for _, ok := range []Intention{IntentionPeace, IntentionSalvation, IntentionMeritDedication, IntentionGratitude} {
		assert.True(t, ok.Valid(), string(ok))
	}
	for _, bad := range []Intention{"", "peace", "WEALTH"} {
		assert.False(t, bad.Valid(), string(bad))
	}
}

func TestDisplayName(t *testing.T) {
	name := "Thích Minh An"
	empty := ""
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"dharma name", &name, "Thích Minh An"},
		{"nil is anonymous", nil, AnonymousLighterName},
		{"empty is anonymous", &empty, AnonymousLighterName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Lamp{LighterName: tt.in}.DisplayName())
			assert.Equal(t, tt.want, GardenLamp{LighterName: tt.in}.DisplayName())
		})
	}
	assert.Equal(t, "Vô danh", AnonymousLighterName)
}

func TestLampValidationError_MatchesSentinel(t *testing.T) {
	err := error(&LampValidationError{Field: "recipient", Message: "required"})
	assert.ErrorIs(t, err, ErrLampInvalid)
	var ve *LampValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Equal(t, "recipient: required", err.Error())
}
