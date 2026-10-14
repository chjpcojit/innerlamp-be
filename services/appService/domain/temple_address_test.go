package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTemple_FullAddress pins AC 33 of docs/specs/features/temple (v1.7): the full
// address is the non-empty parts address, ward, province joined by ", " when the
// temple has a ward, and the stored address unchanged when it has none.
func TestTemple_FullAddress(t *testing.T) {
	cases := []struct {
		name string
		in   Temple
		want string
	}{
		{"all three parts", Temple{Address: "73 Mai Thị Lựu", Ward: "Phường Tân Định", Province: "Hồ Chí Minh"},
			"73 Mai Thị Lựu, Phường Tân Định, Hồ Chí Minh"},
		{"nothing below ward level", Temple{Address: "", Ward: "Xã Hương Sơn", Province: "Hà Nội"},
			"Xã Hương Sơn, Hà Nội"},
		{"blank address is not joined", Temple{Address: "   ", Ward: "Xã Hương Sơn", Province: "Hà Nội"},
			"Xã Hương Sơn, Hà Nội"},
		{"legacy row keeps its stored full address", Temple{Address: "Xã Ninh Hải, Huyện Hoa Lư, Tỉnh Ninh Bình", Province: "Ninh Bình"},
			"Xã Ninh Hải, Huyện Hoa Lư, Tỉnh Ninh Bình"},
		{"legacy row with empty address", Temple{Province: "Ninh Bình"}, ""},
		{"ward without province", Temple{Address: "Thôn 1", Ward: "Xã Hạ Long"}, "Thôn 1, Xã Hạ Long"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.in.FullAddress())
		})
	}
}
