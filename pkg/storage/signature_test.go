package storage

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsExpiredSignatureError(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{"expired v4 token", errors.New("googleapi: Error 400: ExpiredToken"), true},
		{"signature mismatch", errors.New("googleapi: Error 400: SignatureDoesNotMatch"), true},
		{"wrapped", fmt.Errorf("download %q: %w", "k", errors.New("ExpiredToken")), true},
		{"not found", ErrObjectNotFound, false},
		{"access denied", ErrAccessDenied, false},
		{"nil", nil, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsExpiredSignatureError(tt.err))
		})
	}
}
