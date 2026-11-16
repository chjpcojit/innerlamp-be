// Package photo decodes an uploaded memorial photo and re-encodes it as a
// plain JPEG before it is stored.
//
// The HTTP layer only sniffs the client's bytes, and a client can put anything
// in a file whose first bytes look like an image. Decoding the real format and
// re-encoding is what makes the stored object trustworthy: the new JPEG has no
// EXIF, no GPS and no trailing payload, because the encoder writes only image
// data.
package photo

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"

	"github.com/disintegration/imaging"
	"golang.org/x/image/webp"
)

const (
	// MaxEdge is the longest side of a stored photo, in pixels.
	MaxEdge = 2048
	// JPEGQuality is the quality of the re-encoded JPEG.
	JPEGQuality = 85
	// MaxPixels caps the decoded bitmap. A 5 MiB file can otherwise expand to
	// a bitmap large enough to exhaust the process memory.
	MaxPixels int64 = 50 << 20
)

var (
	// ErrUnsupported means the bytes are not a JPEG, PNG or WebP image, or the
	// image is corrupt.
	ErrUnsupported = errors.New("photo: unsupported image")

	// ErrTooLarge means the image decoded, but its pixel count is over
	// MaxPixels.
	ErrTooLarge = errors.New("photo: image too large")
)

// Normalize decodes data, applies its EXIF orientation, composites any alpha
// over white, shrinks it so the longest side is at most MaxEdge and returns a
// JPEG at JPEGQuality.
//
// It returns ErrUnsupported for anything that is not a decodable JPEG, PNG or
// WebP, and ErrTooLarge for an image over MaxPixels. The returned bytes are
// never derived from the input's metadata, so EXIF/GPS cannot survive.
func Normalize(data []byte) ([]byte, error) {
	format, width, height, err := inspect(data)
	if err != nil {
		return nil, err
	}
	if int64(width)*int64(height) > MaxPixels {
		return nil, ErrTooLarge
	}

	var img image.Image
	switch format {
	case "jpeg", "png":
		// AutoOrientation reads the EXIF orientation tag and rotates the
		// pixels, so viewers no longer have to.
		img, err = imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	case "webp":
		img, err = webp.Decode(bytes.NewReader(data))
	default:
		return nil, ErrUnsupported
	}
	if err != nil {
		return nil, ErrUnsupported
	}
	if img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
		return nil, ErrUnsupported
	}

	img = flattenAlpha(img)
	if img.Bounds().Dx() > MaxEdge || img.Bounds().Dy() > MaxEdge {
		img = imaging.Fit(img, MaxEdge, MaxEdge, imaging.Lanczos)
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, ErrUnsupported
	}
	return out.Bytes(), nil
}

// inspect reads the header only: the format name as image.DecodeConfig sees it
// and the pixel dimensions, without allocating the full bitmap.
func inspect(data []byte) (format string, width, height int, err error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", 0, 0, ErrUnsupported
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return "", 0, 0, ErrUnsupported
	}
	return format, cfg.Width, cfg.Height, nil
}

// flattenAlpha draws img over an opaque white background and returns a bitmap
// with no alpha channel, so a transparent PNG becomes white instead of black
// once JPEG drops the alpha channel.
func flattenAlpha(img image.Image) *image.NRGBA {
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Over)
	return dst
}
