package photo

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newJPEG encodes a solid-colour JPEG of the given size. The colour differs per
// pixel so a bug that returns a blank image is visible.
func newJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 40, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}))
	return buf.Bytes()
}

// exifAPP1 builds an APP1 segment holding a minimal EXIF block: IFD0 with the
// orientation tag plus a GPS IFD pointer, and one GPS IFD entry (GPSLatitudeRef).
// That is enough for a JPEG to carry both an EXIF and a GPS marker.
func exifAPP1(t *testing.T, orientation uint16) []byte {
	t.Helper()
	var tiff bytes.Buffer
	put := func(v any) { require.NoError(t, binary.Write(&tiff, binary.LittleEndian, v)) }

	tiff.WriteString("II")
	put(uint16(0x2A))   // TIFF magic
	put(uint32(8))      // IFD0 offset
	put(uint16(2))      // two entries
	put(uint16(0x0112)) // Orientation
	put(uint16(3))      // SHORT
	put(uint32(1))
	put(orientation)
	put(uint16(0))
	put(uint16(0x8825)) // GPSInfoIFDPointer
	put(uint16(4))      // LONG
	put(uint32(1))
	put(uint32(38)) // GPS IFD starts right after IFD0
	put(uint32(0))  // no next IFD

	// GPS IFD at offset 38.
	put(uint16(1))
	put(uint16(0x0001)) // GPSLatitudeRef
	put(uint16(2))      // ASCII
	put(uint32(2))
	tiff.Write([]byte{'N', 0, 0, 0})
	put(uint32(0))

	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

// withExif inserts the APP1 segment directly after the SOI marker.
func withExif(t *testing.T, jpg, app1 []byte) []byte {
	t.Helper()
	out := append([]byte{}, jpg[:2]...)
	out = append(out, app1...)
	return append(out, jpg[2:]...)
}

// hasAPP1 walks the JPEG segments and reports whether any APP1 (EXIF) segment
// survives. Go's encoder writes APP0/JFIF only, so a true result means metadata
// was carried over from the upload.
func hasAPP1(t *testing.T, jpg []byte) bool {
	t.Helper()
	require.GreaterOrEqual(t, len(jpg), 2)
	require.Equal(t, byte(0xFF), jpg[0])
	require.Equal(t, byte(0xD8), jpg[1])
	i := 2
	for i+3 < len(jpg) {
		if jpg[i] != 0xFF {
			return false
		}
		marker := jpg[i+1]
		if marker == 0xD8 || marker == 0xD9 {
			i += 2
			continue
		}
		length := int(binary.BigEndian.Uint16(jpg[i+2 : i+4]))
		if marker == 0xE1 {
			return true
		}
		if length < 2 {
			return false
		}
		i += 2 + length
	}
	return false
}

func TestNormalize_StripsExifAndGps(t *testing.T) {
	jpg := newJPEG(t, 200, 100)
	withMeta := withExif(t, jpg, exifAPP1(t, 1))
	require.True(t, hasAPP1(t, withMeta), "fixture must carry an APP1 segment")

	out, err := Normalize(withMeta)
	require.NoError(t, err)

	assert.NotContains(t, string(out), "Exif")
	assert.NotContains(t, string(out), "GPSLatitudeRef")
	assert.False(t, hasAPP1(t, out), "stored bytes must not carry an EXIF APP1 segment")

	cfg, format, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
	assert.Equal(t, 200, cfg.Width)
	assert.Equal(t, 100, cfg.Height)
}

func TestNormalize_AppliesExifOrientation(t *testing.T) {
	// Orientation 6 means "rotate 90° clockwise": a 200x100 file is displayed
	// as 100x200, so the stored pixels must be rotated to match.
	jpg := withExif(t, newJPEG(t, 200, 100), exifAPP1(t, 6))

	out, err := Normalize(jpg)
	require.NoError(t, err)

	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, 100, cfg.Width, "orientation 6 must swap the axes")
	assert.Equal(t, 200, cfg.Height)
}

func TestNormalize_DownscalesLongestEdge(t *testing.T) {
	out, err := Normalize(newJPEG(t, 4000, 3000))
	require.NoError(t, err)

	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, MaxEdge, cfg.Width)
	assert.Equal(t, 1536, cfg.Height)
}

func TestNormalize_KeepsSmallImage(t *testing.T) {
	out, err := Normalize(newJPEG(t, 320, 240))
	require.NoError(t, err)

	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, 320, cfg.Width)
	assert.Equal(t, 240, cfg.Height)
}

func TestNormalize_RejectsGarbage(t *testing.T) {
	for _, tt := range []struct {
		name string
		data []byte
	}{
		{"text", []byte("this is not an image at all")},
		{"empty", nil},
		{"truncated png", []byte("\x89PNG\r\n\x1a\n")},
		{"gif", []byte("GIF89a")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := Normalize(tt.data)
			assert.ErrorIs(t, err, ErrUnsupported)
			assert.Nil(t, out)
		})
	}
}

func TestNormalize_PngAlphaBecomesWhite(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if x < 10 {
				img.Set(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: 0}) // fully transparent
			} else {
				img.Set(x, y, color.NRGBA{R: 200, G: 30, B: 30, A: 255})
			}
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	out, err := Normalize(buf.Bytes())
	require.NoError(t, err)

	decoded, err := jpeg.Decode(bytes.NewReader(out))
	require.NoError(t, err)

	r, g, b, _ := decoded.At(2, 5).RGBA()
	assert.InDelta(t, 255, r>>8, 6, "transparent pixel must become white")
	assert.InDelta(t, 255, g>>8, 6)
	assert.InDelta(t, 255, b>>8, 6)

	r, g, b, _ = decoded.At(17, 5).RGBA()
	assert.Greater(t, r>>8, uint32(150), "opaque red must survive")
	assert.Less(t, g>>8, uint32(90))
}
