package imageprocessing_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/mioluigi/go-image-pipeline/imageprocessing"
)

// solidImage returns a width × height image filled with one colour, so the
// expected result of every operation on it is known in advance.
func solidImage(width, height int, fill color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, fill)
		}
	}
	return img
}

// TestGrayscaleKnownValues checks grey levels computed independently from the
// ITU-R BT.601 luma weights Go's color.GrayModel documents (0.299 R + 0.587 G
// + 0.114 B, in 16-bit fixed point), not values copied from the code.
func TestGrayscaleKnownValues(t *testing.T) {
	cases := []struct {
		name     string
		fill     color.RGBA
		wantGray uint8
	}{
		{"red", color.RGBA{255, 0, 0, 255}, 76},
		{"green", color.RGBA{0, 255, 0, 255}, 150},
		{"blue", color.RGBA{0, 0, 255, 255}, 29},
		{"white", color.RGBA{255, 255, 255, 255}, 255},
		{"black", color.RGBA{0, 0, 0, 255}, 0},
		{"mid grey", color.RGBA{128, 128, 128, 255}, 128},
		{"orange-brown", color.RGBA{200, 100, 50, 255}, 124},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gray := imageprocessing.Grayscale(solidImage(3, 2, tc.fill))
			for y := 0; y < 2; y++ {
				for x := 0; x < 3; x++ {
					if got := gray.GrayAt(x, y).Y; got != tc.wantGray {
						t.Fatalf("pixel (%d,%d) = %d, want %d", x, y, got, tc.wantGray)
					}
				}
			}
		})
	}
}

// TestGrayscaleKeepsBounds uses a sub-image whose bounds do not start at
// (0,0), a common source of off-by-origin bugs.
func TestGrayscaleKeepsBounds(t *testing.T) {
	source := solidImage(10, 10, color.RGBA{255, 0, 0, 255}).SubImage(image.Rect(2, 3, 7, 9))
	gray := imageprocessing.Grayscale(source)
	if gray.Bounds() != source.Bounds() {
		t.Fatalf("bounds = %v, want %v", gray.Bounds(), source.Bounds())
	}
	if got := gray.GrayAt(2, 3).Y; got != 76 {
		t.Fatalf("corner pixel = %d, want 76", got)
	}
}

func TestFitSize(t *testing.T) {
	cases := []struct {
		name                string
		source, box, wanted image.Point
	}{
		{"square stays square (CODEHEIM's case)", image.Pt(1536, 1536), image.Pt(500, 500), image.Pt(500, 500)},
		{"landscape 4:3", image.Pt(4032, 3024), image.Pt(500, 500), image.Pt(500, 375)},
		{"portrait 3:4", image.Pt(3024, 4032), image.Pt(500, 500), image.Pt(375, 500)},
		{"landscape 2:1", image.Pt(1536, 768), image.Pt(500, 500), image.Pt(500, 250)},
		{"smaller than the box is enlarged", image.Pt(100, 50), image.Pt(500, 500), image.Pt(500, 250)},
		{"rounds to the nearest pixel", image.Pt(1000, 333), image.Pt(500, 500), image.Pt(500, 167)},
		{"never narrower than one pixel", image.Pt(1, 2000), image.Pt(500, 500), image.Pt(1, 500)},
		{"never shorter than one pixel", image.Pt(2000, 1), image.Pt(500, 500), image.Pt(500, 1)},
		{"box that is not square", image.Pt(1000, 1000), image.Pt(500, 300), image.Pt(300, 300)},
		{"empty source stays empty", image.Pt(0, 0), image.Pt(500, 500), image.Pt(0, 0)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := imageprocessing.FitSize(tc.source, tc.box); got != tc.wanted {
				t.Fatalf("FitSize(%v, %v) = %v, want %v", tc.source, tc.box, got, tc.wanted)
			}
		})
	}
}

func TestFitProducesFitSize(t *testing.T) {
	source := solidImage(300, 120, color.RGBA{10, 200, 30, 255})
	fitted := imageprocessing.Fit(source, 100, 100)
	if got, want := fitted.Bounds().Size(), image.Pt(100, 40); got != want {
		t.Fatalf("Fit size = %v, want %v", got, want)
	}
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestDecodeAcceptsJPEGAndPNG(t *testing.T) {
	source := solidImage(40, 30, color.RGBA{1, 2, 3, 255})
	for name, data := range map[string][]byte{"jpeg": encodeJPEG(t, source), "png": encodePNG(t, source)} {
		t.Run(name, func(t *testing.T) {
			img, err := imageprocessing.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if got := img.Bounds().Size(); got != image.Pt(40, 30) {
				t.Fatalf("size = %v, want 40×30", got)
			}
		})
	}
}

// oneByOneGIF is a valid GIF written out by hand: importing image/gif here
// would register its decoder and hide the "unsupported format" case.
var oneByOneGIF = []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00" +
	"!\xf9\x04\x00\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")

func TestDecodeRejectsUnreadableImages(t *testing.T) {
	validJPEG := encodeJPEG(t, solidImage(64, 64, color.RGBA{9, 9, 9, 255}))
	cases := map[string][]byte{
		"empty file":       {},
		"not an image":     []byte("not a jpeg at all"),
		"unsupported GIF":  oneByOneGIF,
		"truncated JPEG":   validJPEG[:len(validJPEG)/2],
		"JPEG header only": validJPEG[:2],
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := imageprocessing.Decode(bytes.NewReader(data))
			if !errors.Is(err, imageprocessing.ErrUnreadableImage) {
				t.Fatalf("error = %v, want ErrUnreadableImage", err)
			}
		})
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	gray := imageprocessing.Grayscale(solidImage(20, 10, color.RGBA{128, 128, 128, 255}))
	var buffer bytes.Buffer
	if err := imageprocessing.Encode(&buffer, gray); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := jpeg.Decode(&buffer)
	if err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
	if got := decoded.Bounds().Size(); got != image.Pt(20, 10) {
		t.Fatalf("size = %v, want 20×10", got)
	}
}

// failingWriter stands in for a full disk or a closed file.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestEncodeReportsWriteErrors(t *testing.T) {
	gray := imageprocessing.Grayscale(solidImage(20, 10, color.Black))
	if err := imageprocessing.Encode(failingWriter{}, gray); err == nil {
		t.Fatal("Encode to a failing writer returned nil error")
	}
}
