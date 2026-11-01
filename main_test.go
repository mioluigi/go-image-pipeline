package main

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runProgram runs the program in-process, the way main does, and returns its
// exit code, standard output and standard error.
func runProgram(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// writeJPEG writes a small solid-colour JPEG with the given size.
func writeJPEG(t *testing.T, filePath string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := range img.Pix {
		img.Pix[i] = 180
	}
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// folderWithImages returns a new folder holding the named files: names ending
// in .jpg are real images, any other name gets the given junk content.
func folderWithImages(t *testing.T, names ...string) string {
	t.Helper()
	folder := t.TempDir()
	for _, name := range names {
		filePath := filepath.Join(folder, name)
		if strings.HasSuffix(name, ".jpg") {
			writeJPEG(t, filePath, 80, 60)
			continue
		}
		if err := os.WriteFile(filePath, []byte("not an image"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return folder
}

// changeToTempDir runs the rest of the test in an empty folder, so the
// default output folder ("output") lands somewhere disposable.
func changeToTempDir(t *testing.T) string {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	folder := t.TempDir()
	if err := os.Chdir(folder); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	return folder
}

func countFiles(t *testing.T, folder string) int {
	t.Helper()
	entries, err := os.ReadDir(folder)
	if err != nil {
		return 0
	}
	return len(entries)
}

// TestNoArguments is spec FR-14 and AC-7: the bundled images, the default
// mode, the default output folder, exit code 0.
func TestNoArguments(t *testing.T) {
	folder := changeToTempDir(t)
	code, stdout, stderr := runProgram(t)
	if code != 0 {
		t.Fatalf("exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	bundled, err := bundledImageNames()
	if err != nil {
		t.Fatal(err)
	}
	if got := countFiles(t, filepath.Join(folder, "output")); got != len(bundled) {
		t.Errorf("%d files in output/, want %d", got, len(bundled))
	}
	for _, want := range []string{"saved, 0 failed", "mode pipeline", "Output folder:", "resize"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "level=") {
		t.Errorf("log lines leaked into standard output:\n%s", stdout)
	}
}

func TestEveryModeRuns(t *testing.T) {
	input := folderWithImages(t, "a.jpg", "b.jpg", "c.jpg")
	for _, mode := range []string{"sequential", "pipeline", "workers"} {
		t.Run(mode, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "out")
			code, stdout, stderr := runProgram(t, "-mode", mode, "-in", input, "-out", output, "-size", "40", "-workers", "2")
			if code != 0 {
				t.Fatalf("exit code %d\n%s\n%s", code, stdout, stderr)
			}
			if !strings.Contains(stdout, "3 saved, 0 failed") || !strings.Contains(stdout, "40x30") {
				t.Errorf("unexpected output:\n%s", stdout)
			}
			if mode == "workers" && !strings.Contains(stdout, "2 workers") {
				t.Errorf("worker count not shown:\n%s", stdout)
			}
		})
	}
}

// TestInvalidUsage covers every exit-2 case of spec FR-14 and FR-14a: a clear
// message on standard error, and nothing processed.
func TestInvalidUsage(t *testing.T) {
	images := folderWithImages(t, "a.jpg")
	textOnly := folderWithImages(t, "notes.txt")
	collision := folderWithImages(t, "a.jpg", "a.png")
	aFile := filepath.Join(images, "a.jpg")
	linkToImages := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(images, linkToImages); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{"unknown mode", []string{"-mode", "fast"}, `unknown mode "fast"`},
		{"size zero", []string{"-size", "0"}, "-size must be between 1 and 10000"},
		{"size too large", []string{"-size", "10001"}, "-size must be between 1 and 10000"},
		{"size not a number", []string{"-size", "big"}, "invalid value"},
		{"zero workers", []string{"-workers", "0"}, "-workers must be between 1 and 1024"},
		{"too many workers", []string{"-workers", "2000"}, "-workers must be between 1 and 1024"},
		{"unknown flag", []string{"-colour"}, "flag provided but not defined"},
		{"extra argument", []string{"photos"}, `unexpected argument "photos"`},
		{"input folder missing", []string{"-in", filepath.Join(images, "nope")}, "cannot read input folder"},
		{"input is a file", []string{"-in", aFile}, "is not a folder"},
		{"no images in folder", []string{"-in", textOnly}, "no JPEG or PNG images"},
		{"output is the input", []string{"-in", images, "-out", images}, "is the input folder"},
		{"output links to the input", []string{"-in", images, "-out", linkToImages}, "is the input folder"},
		{"two inputs, one output name", []string{"-in", collision}, "would both be saved as a.jpg"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			folder := changeToTempDir(t)
			code, stdout, stderr := runProgram(t, tc.args...)
			if code != 2 {
				t.Fatalf("exit code %d, want 2\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
			}
			if !strings.Contains(stderr, tc.wantMessage) {
				t.Errorf("standard error lacks %q:\n%s", tc.wantMessage, stderr)
			}
			if countFiles(t, filepath.Join(folder, "output")) != 0 {
				t.Error("images were processed despite invalid usage")
			}
		})
	}
}

func TestHelpExitsZero(t *testing.T) {
	code, _, stderr := runProgram(t, "-h")
	if code != 0 || !strings.Contains(stderr, "-mode") {
		t.Fatalf("exit code %d, usage:\n%s", code, stderr)
	}
}

// TestOneBadImage is spec FR-6 and FR-14a: exit 1, the failure named, and the
// good images still saved.
func TestOneBadImage(t *testing.T) {
	input := folderWithImages(t, "good1.jpg", "broken.png", "good2.jpg")
	output := t.TempDir()
	code, stdout, stderr := runProgram(t, "-in", input, "-out", output)
	if code != 1 {
		t.Fatalf("exit code %d, want 1\n%s\n%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "FAILED") || !strings.Contains(stdout, "broken.png") || !strings.Contains(stdout, "2 saved, 1 failed") {
		t.Errorf("failure not reported clearly:\n%s", stdout)
	}
	if got := countFiles(t, output); got != 2 {
		t.Errorf("%d files saved, want 2", got)
	}
}

// TestInputsAreNeverChanged is spec FR-7, the bug Spike 001 found in the
// original: the input files' contents are identical after a run.
func TestInputsAreNeverChanged(t *testing.T) {
	input := folderWithImages(t, "a.jpg", "b.jpg")
	before := folderChecksums(t, input)
	if code, stdout, stderr := runProgram(t, "-in", input, "-out", filepath.Join(input, "results")); code != 0 {
		t.Fatalf("exit code %d\n%s\n%s", code, stdout, stderr)
	}
	after := folderChecksums(t, input)
	for name, sum := range before {
		if after[name] != sum {
			t.Errorf("%s was changed", name)
		}
	}
}

func folderChecksums(t *testing.T, folder string) map[string][32]byte {
	t.Helper()
	sums := map[string][32]byte{}
	entries, err := os.ReadDir(folder)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(folder, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		sums[entry.Name()] = sha256.Sum256(data)
	}
	return sums
}

func TestVerboseLogsEachImage(t *testing.T) {
	input := folderWithImages(t, "a.jpg")
	code, _, stderr := runProgram(t, "-in", input, "-out", t.TempDir(), "-verbose")
	if code != 0 || !strings.Contains(stderr, "image done") || !strings.Contains(stderr, "a.jpg") {
		t.Fatalf("exit code %d, logs:\n%s", code, stderr)
	}
}

func TestUnwritableOutputFolder(t *testing.T) {
	input := folderWithImages(t, "a.jpg")
	blocker := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runProgram(t, "-in", input, "-out", filepath.Join(blocker, "out"))
	if code != 2 || !strings.Contains(stderr, "creating output folder") {
		t.Fatalf("exit code %d, stderr:\n%s", code, stderr)
	}
}
