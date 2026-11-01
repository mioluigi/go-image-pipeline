// Package imageprocessing holds the four operations of the image pipeline,
// each applied to one image: decode, fit (resize keeping the aspect ratio),
// convert to grayscale, and encode as JPEG.
//
// The package reads from io.Reader and writes to io.Writer and never touches
// the file system, so callers decide where images come from and go, and tests
// run on bytes in memory. Adapted from CODEHEIM's image_processing package
// (github.com/code-heim/go_21_goroutines_pipeline, Apache-2.0): errors are
// returned instead of panicking, PNG input is accepted, and the fixed
// 500 × 500 resize became Fit, which keeps the shape of the image.
package imageprocessing

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder with image.Decode
	"io"
	"math"

	"github.com/nfnt/resize"
)

// ErrUnreadableImage is wrapped by every Decode error, so callers can tell a
// bad image file apart from a file-system error with errors.Is.
var ErrUnreadableImage = errors.New("unsupported or corrupt image (JPEG and PNG are accepted)")

// Decode reads one JPEG or PNG image.
func Decode(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadableImage, err)
	}
	return img, nil
}

// FitSize returns the largest size inside box with the same width-to-height
// ratio as source, rounded to whole pixels and never thinner than one pixel.
// Smaller sources are enlarged, so every output's longer side matches the box.
func FitSize(source, box image.Point) image.Point {
	if source.X <= 0 || source.Y <= 0 {
		return image.Point{}
	}
	scale := min(float64(box.X)/float64(source.X), float64(box.Y)/float64(source.Y))
	return image.Pt(scaledLength(source.X, scale), scaledLength(source.Y, scale))
}

// scaledLength multiplies a length in pixels by scale, rounding to the
// nearest pixel but never below one, so an extreme ratio cannot vanish.
func scaledLength(length int, scale float64) int {
	return max(1, int(math.Round(float64(length)*scale)))
}

// Fit resizes img to FitSize within a maxWidth × maxHeight box, using the
// Lanczos3 filter of CODEHEIM's original Resize.
func Fit(img image.Image, maxWidth, maxHeight int) image.Image {
	size := FitSize(img.Bounds().Size(), image.Pt(maxWidth, maxHeight))
	return resize.Resize(uint(size.X), uint(size.Y), img, resize.Lanczos3)
}

// Grayscale converts img pixel by pixel with color.GrayModel (ITU-R BT.601
// luma weights). The per-pixel At/Set loop is CODEHEIM's algorithm, kept
// unchanged because it is part of the workload the timings measure.
func Grayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray.Set(x, y, color.GrayModel.Convert(img.At(x, y)))
		}
	}
	return gray
}

// Encode writes img as a JPEG at the standard library's default quality, the
// same setting as CODEHEIM's original.
func Encode(w io.Writer, img image.Image) error {
	return jpeg.Encode(w, img, nil)
}
