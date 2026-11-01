package pipeline_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/mioluigi/go-image-pipeline/imageprocessing"
	"github.com/mioluigi/go-image-pipeline/pipeline"
)

var allModes = []pipeline.Mode{pipeline.Sequential, pipeline.Pipeline, pipeline.Workers}

// noiseImage returns a width × height image of random colours from a fixed
// seed: realistic work for the encoder, and the same bytes on every run.
func noiseImage(width, height int, seed uint64) *image.RGBA {
	random := rand.New(rand.NewPCG(seed, seed))
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := range img.Pix {
		img.Pix[i] = uint8(random.UintN(256))
	}
	return img
}

func jpegBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

// syntheticImages returns an in-memory folder of landscape, portrait, square
// and PNG images, and their names in input order.
func syntheticImages(t *testing.T) (fstest.MapFS, []string) {
	t.Helper()
	inputs := fstest.MapFS{
		"landscape.jpg": {Data: jpegBytes(t, noiseImage(160, 90, 1))},
		"portrait.jpeg": {Data: jpegBytes(t, noiseImage(90, 160, 2))},
		"square.jpg":    {Data: jpegBytes(t, noiseImage(120, 120, 3))},
		"drawing.png":   {Data: pngBytes(t, noiseImage(100, 50, 4))},
		"tiny.jpg":      {Data: jpegBytes(t, noiseImage(10, 20, 5))},
	}
	return inputs, []string{"landscape.jpg", "portrait.jpeg", "square.jpg", "drawing.png", "tiny.jpg"}
}

// config writes into a fresh temporary folder; 64-pixel outputs keep tests fast.
func config(t *testing.T) pipeline.Config {
	return pipeline.Config{OutputDir: filepath.Join(t.TempDir(), "out"), Size: 64, Workers: 3}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestModesProduceIdenticalImages is spec FR-9 / Constitution 1.1: a
// concurrent mode is only trusted if its files equal the sequential ones.
func TestModesProduceIdenticalImages(t *testing.T) {
	inputs, names := syntheticImages(t)
	reference, err := pipeline.Run(pipeline.Sequential, inputs, names, config(t))
	if err != nil {
		t.Fatal(err)
	}
	wantSizes := []image.Point{{64, 36}, {36, 64}, {64, 64}, {64, 32}, {32, 64}}
	for i, result := range reference {
		if result.Err != nil {
			t.Fatalf("sequential %s: %v", result.InputName, result.Err)
		}
		if result.OutputSize != wantSizes[i] {
			t.Errorf("%s output size = %v, want %v", result.InputName, result.OutputSize, wantSizes[i])
		}
	}
	for _, mode := range allModes[1:] {
		t.Run(mode.String(), func(t *testing.T) {
			results, err := pipeline.Run(mode, inputs, names, config(t))
			if err != nil {
				t.Fatal(err)
			}
			for i, result := range results {
				if result.Err != nil {
					t.Fatalf("%s: %v", result.InputName, result.Err)
				}
				if !bytes.Equal(readFile(t, result.OutputPath), readFile(t, reference[i].OutputPath)) {
					t.Errorf("%s differs from the sequential output", result.InputName)
				}
			}
		})
	}
}

func TestResultsDescribeEachImage(t *testing.T) {
	inputs, names := syntheticImages(t)
	cfg := config(t)
	results, err := pipeline.Run(pipeline.Sequential, inputs, names, cfg)
	if err != nil {
		t.Fatal(err)
	}
	first := results[0]
	if first.InputName != "landscape.jpg" || first.OriginalSize != image.Pt(160, 90) {
		t.Errorf("first result = %q %v, want landscape.jpg (160,90)", first.InputName, first.OriginalSize)
	}
	if want := filepath.Join(cfg.OutputDir, "drawing.jpg"); results[3].OutputPath != want {
		t.Errorf("PNG output path = %q, want %q", results[3].OutputPath, want)
	}
	decoded, err := imageprocessing.Decode(bytes.NewReader(readFile(t, results[3].OutputPath)))
	if err != nil {
		t.Fatalf("output is not a readable image: %v", err)
	}
	if decoded.ColorModel() != color.GrayModel {
		t.Errorf("output colour model = %v, want grayscale", decoded.ColorModel())
	}
}

// TestFailingImagesDoNotStopTheOthers is spec FR-6 and FR-6b, in every mode.
func TestFailingImagesDoNotStopTheOthers(t *testing.T) {
	inputs, _ := syntheticImages(t)
	inputs["broken.jpg"] = &fstest.MapFile{Data: []byte("not a jpeg at all")}
	names := []string{"landscape.jpg", "broken.jpg", "missing.jpg", "drawing.png", "square.jpg"}
	for _, mode := range allModes {
		t.Run(mode.String(), func(t *testing.T) {
			results, err := pipeline.Run(mode, inputs, names, config(t))
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != len(names) {
				t.Fatalf("%d results, want %d", len(results), len(names))
			}
			for i, result := range results {
				if result.InputName != names[i] {
					t.Fatalf("result %d is %q, want %q (input order)", i, result.InputName, names[i])
				}
			}
			if !errors.Is(results[1].Err, imageprocessing.ErrUnreadableImage) {
				t.Errorf("broken.jpg error = %v, want ErrUnreadableImage", results[1].Err)
			}
			if !errors.Is(results[2].Err, fs.ErrNotExist) {
				t.Errorf("missing.jpg error = %v, want fs.ErrNotExist", results[2].Err)
			}
			for _, i := range []int{0, 3, 4} {
				if results[i].Err != nil {
					t.Errorf("%s failed: %v", names[i], results[i].Err)
				}
				if _, err := os.Stat(results[i].OutputPath); err != nil {
					t.Errorf("%s was not saved: %v", names[i], err)
				}
			}
		})
	}
}

// TestFailedImageSkipsLaterStages checks the pass-through rule: a stage does
// nothing (and takes no time) once an earlier stage has failed.
func TestFailedImageSkipsLaterStages(t *testing.T) {
	inputs := fstest.MapFS{"broken.jpg": {Data: []byte("garbage")}}
	results, err := pipeline.Run(pipeline.Pipeline, inputs, []string{"broken.jpg"}, config(t))
	if err != nil {
		t.Fatal(err)
	}
	times := results[0].StageTimes
	if times.Resize != 0 || times.Grayscale != 0 || times.Save != 0 {
		t.Errorf("stages after a failed load ran: %+v", times)
	}
	if results[0].OutputPath != "" {
		t.Errorf("failed image reports output path %q", results[0].OutputPath)
	}
}

func TestStageTimesAreRecorded(t *testing.T) {
	inputs, names := syntheticImages(t)
	for _, mode := range allModes {
		results, err := pipeline.Run(mode, inputs, names, config(t))
		if err != nil {
			t.Fatal(err)
		}
		times := results[0].StageTimes
		if times.Load <= 0 || times.Resize <= 0 || times.Grayscale <= 0 || times.Save <= 0 {
			t.Errorf("%s: missing stage time in %+v", mode, times)
		}
		if times.Total() != times.Load+times.Resize+times.Grayscale+times.Save {
			t.Errorf("%s: Total() = %v, want the sum of the stages", mode, times.Total())
		}
	}
}

func TestSaveErrorIsReportedPerImage(t *testing.T) {
	inputs, names := syntheticImages(t)
	for _, mode := range allModes {
		t.Run(mode.String(), func(t *testing.T) {
			cfg := config(t)
			// A folder where square.jpg should be written makes that one save fail,
			// even for the root user, unlike a read-only folder.
			if err := os.MkdirAll(filepath.Join(cfg.OutputDir, "square.jpg"), 0o755); err != nil {
				t.Fatal(err)
			}
			results, err := pipeline.Run(mode, inputs, names, cfg)
			if err != nil {
				t.Fatal(err)
			}
			if results[2].Err == nil {
				t.Error("saving square.jpg over a folder did not fail")
			}
			if results[0].Err != nil || results[4].Err != nil {
				t.Errorf("other images failed: %v, %v", results[0].Err, results[4].Err)
			}
		})
	}
}

// TestNoGoroutineIsLeftBehind is spec FR-6a: every goroutine a run starts has
// ended when Run returns, including runs where images fail.
func TestNoGoroutineIsLeftBehind(t *testing.T) {
	inputs, _ := syntheticImages(t)
	inputs["broken.jpg"] = &fstest.MapFile{Data: []byte("garbage")}
	names := []string{"broken.jpg", "landscape.jpg", "missing.jpg", "square.jpg", "broken.jpg"}
	for _, mode := range allModes {
		t.Run(mode.String(), func(t *testing.T) {
			before := runtime.NumGoroutine()
			if _, err := pipeline.Run(mode, inputs, names, config(t)); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(time.Second)
			for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
			}
			if after := runtime.NumGoroutine(); after > before {
				t.Fatalf("%d goroutines before the run, %d after", before, after)
			}
		})
	}
}

func TestWorkerCounts(t *testing.T) {
	inputs, names := syntheticImages(t)
	for _, workers := range []int{0, 1, 2, 64} { // 0 means one per CPU
		cfg := config(t)
		cfg.Workers = workers
		results, err := pipeline.Run(pipeline.Workers, inputs, names, cfg)
		if err != nil {
			t.Fatalf("workers=%d: %v", workers, err)
		}
		for _, result := range results {
			if result.Err != nil {
				t.Errorf("workers=%d %s: %v", workers, result.InputName, result.Err)
			}
		}
	}
}

func TestEmptyInputListIsNotAnError(t *testing.T) {
	for _, mode := range allModes {
		results, err := pipeline.Run(mode, fstest.MapFS{}, nil, config(t))
		if err != nil || len(results) != 0 {
			t.Errorf("%s: results %v, error %v; want none", mode, results, err)
		}
	}
}

func TestRunRejectsBadSetup(t *testing.T) {
	inputs, names := syntheticImages(t)
	cfg := config(t)
	if _, err := pipeline.Run(pipeline.Mode(99), inputs, names, cfg); err == nil {
		t.Error("unknown mode accepted")
	}
	blocker := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.OutputDir = blocker
	if _, err := pipeline.Run(pipeline.Sequential, inputs, names, cfg); err == nil {
		t.Error("output folder that is a file accepted")
	}
	cfg.OutputDir = t.TempDir()
	cfg.Size = 0
	if _, err := pipeline.Run(pipeline.Sequential, inputs, names, cfg); err == nil {
		t.Error("size 0 accepted")
	}
}

func TestParseMode(t *testing.T) {
	for _, mode := range allModes {
		parsed, err := pipeline.ParseMode(mode.String())
		if err != nil || parsed != mode {
			t.Errorf("ParseMode(%q) = %v, %v", mode.String(), parsed, err)
		}
	}
	if _, err := pipeline.ParseMode("fast"); err == nil {
		t.Error(`ParseMode("fast") accepted`)
	}
	if got := pipeline.Mode(99).String(); got != "Mode(99)" {
		t.Errorf("String of an unknown mode = %q", got)
	}
}

func TestOutputName(t *testing.T) {
	cases := map[string]string{
		"a.jpeg":       "a.jpg",
		"photos/b.PNG": "b.jpg",
		"c":            "c.jpg",
		"d.tar.png":    "d.tar.jpg",
		"already.jpg":  "already.jpg",
	}
	for input, want := range cases {
		if got := pipeline.OutputName(input); got != want {
			t.Errorf("OutputName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestImageNames(t *testing.T) {
	folder := fstest.MapFS{
		"b.JPG":            {Data: []byte("x")},
		"a.png":            {Data: []byte("x")},
		"c.jpeg":           {Data: []byte("x")},
		"notes.txt":        {Data: []byte("x")},
		"folder.jpg/x.jpg": {Data: []byte("x")},
	}
	names, err := pipeline.ImageNames(folder)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(names, ","); got != "a.png,b.JPG,c.jpeg" {
		t.Errorf("ImageNames = %s, want a.png,b.JPG,c.jpeg", got)
	}
	if _, err := pipeline.ImageNames(os.DirFS(filepath.Join(t.TempDir(), "missing"))); err == nil {
		t.Error("a missing folder was listed without error")
	}
}
