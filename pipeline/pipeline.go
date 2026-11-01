// Package pipeline runs the four image stages — load, resize, grayscale,
// save — over many images in one of three modes that differ only in
// scheduling:
//
//   - Sequential: one image at a time, all four stages, no goroutines.
//   - Pipeline: CODEHEIM's design, one goroutine per stage joined by channels,
//     so different images are in different stages at the same time.
//   - Workers: a pool of goroutines, each running all four stages on one image.
//
// All modes call the same stage functions (Constitution 4.2), so a timing
// difference between modes measures concurrency and nothing else.
//
// A failure travels with its image: the stage that fails records the error in
// the job, and later stages pass that job through untouched. No stage ever
// stops reading early, so every goroutine ends when its input channel closes
// and none is left blocked (spec FR-6, FR-6a). Results come back in input
// order whatever order the images finish in (FR-6b).
package pipeline

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mioluigi/go-image-pipeline/imageprocessing"
)

// Mode selects how the stages are scheduled.
type Mode int

const (
	Sequential Mode = iota // no goroutines
	Pipeline               // one goroutine per stage (CODEHEIM's design)
	Workers                // a pool of goroutines, each processing whole images
)

var modeNames = [...]string{Sequential: "sequential", Pipeline: "pipeline", Workers: "workers"}

// String returns the mode's command-line name.
func (m Mode) String() string {
	if m < 0 || int(m) >= len(modeNames) {
		return fmt.Sprintf("Mode(%d)", int(m))
	}
	return modeNames[m]
}

// ParseMode returns the Mode with the given command-line name.
func ParseMode(name string) (Mode, error) {
	for mode, modeName := range modeNames {
		if name == modeName {
			return Mode(mode), nil
		}
	}
	return 0, fmt.Errorf("unknown mode %q (choose %s)", name, strings.Join(modeNames[:], ", "))
}

// Config holds the settings shared by every image in a run.
type Config struct {
	OutputDir string // created if missing
	Size      int    // outputs fit inside Size × Size, keeping their shape
	Workers   int    // goroutines in Workers mode; below 1 means one per CPU
}

// StageTimes is the time one image spent in each stage. A stage skipped
// because an earlier one failed takes zero time.
type StageTimes struct {
	Load, Resize, Grayscale, Save time.Duration
}

// Total is the time the image spent in all four stages.
func (t StageTimes) Total() time.Duration {
	return t.Load + t.Resize + t.Grayscale + t.Save
}

// Result reports what happened to one input image.
type Result struct {
	InputName    string
	OutputPath   string      // empty if the image was not saved
	OriginalSize image.Point // zero if the image could not be loaded
	OutputSize   image.Point // zero if the image was not saved
	StageTimes   StageTimes
	Err          error // nil if the image was saved
}

// OutputName is the file name an input is saved under: its base name with a
// .jpg extension, because every output is a JPEG ("photos/b.png" → "b.jpg").
func OutputName(inputName string) string {
	base := path.Base(inputName)
	return strings.TrimSuffix(base, path.Ext(base)) + ".jpg"
}

// ImageNames lists the JPEG and PNG files at the top level of folder, in
// lexical order; subfolders and other files are ignored.
func ImageNames(folder fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(folder, ".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch strings.ToLower(path.Ext(entry.Name())) {
		case ".jpg", ".jpeg", ".png":
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// Run processes the named files of inputs in the given mode and returns one
// Result per name, in the same order. The error is non-nil only when the run
// cannot start (unknown mode, invalid size, output folder cannot be created);
// a failure of one image is reported in its Result and the others continue.
func Run(mode Mode, inputs fs.FS, names []string, cfg Config) ([]Result, error) {
	scheduler, err := schedulerFor(mode)
	if err != nil {
		return nil, err
	}
	if cfg.Size < 1 {
		return nil, fmt.Errorf("output size must be at least 1 pixel, got %d", cfg.Size)
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output folder: %w", err)
	}
	results := make([]Result, len(names))
	collect := func(finished job) { results[finished.index] = finished.result() }
	scheduler(stages{inputs: inputs, cfg: cfg}, names, collect)
	return results, nil
}

// A scheduler runs every named image through the stages and hands each
// finished job to collect, from a single goroutine (the caller's), so collect
// needs no locking.
type scheduler func(s stages, names []string, collect func(job))

func schedulerFor(mode Mode) (scheduler, error) {
	switch mode {
	case Sequential:
		return runSequential, nil
	case Pipeline:
		return runPipeline, nil
	case Workers:
		return runWorkers, nil
	}
	return nil, fmt.Errorf("unknown mode %v", mode)
}

// job is one image travelling through the stages.
type job struct {
	index        int // position in the input list, so results keep input order
	inputName    string
	image        image.Image
	originalSize image.Point
	outputPath   string
	outputSize   image.Point
	times        StageTimes
	err          error // set by the first stage that fails
}

func newJob(index int, inputName string) job {
	return job{index: index, inputName: inputName}
}

func (j job) result() Result {
	return Result{
		InputName:    j.inputName,
		OutputPath:   j.outputPath,
		OriginalSize: j.originalSize,
		OutputSize:   j.outputSize,
		StageTimes:   j.times,
		Err:          j.err,
	}
}

// stages holds what the four stage functions need besides the job itself.
type stages struct {
	inputs fs.FS
	cfg    Config
}

// process runs all four stages on one job: the unit of work of the
// Sequential and Workers modes.
func (s stages) process(j job) job {
	return s.save(s.grayscale(s.resize(s.load(j))))
}

func (s stages) load(j job) job {
	start := time.Now()
	j.image, j.err = decodeFile(s.inputs, j.inputName)
	if j.err == nil {
		j.originalSize = j.image.Bounds().Size()
	}
	j.times.Load = time.Since(start)
	return j
}

func (s stages) resize(j job) job {
	if j.err != nil {
		return j
	}
	start := time.Now()
	j.image = imageprocessing.Fit(j.image, s.cfg.Size, s.cfg.Size)
	j.times.Resize = time.Since(start)
	return j
}

func (s stages) grayscale(j job) job {
	if j.err != nil {
		return j
	}
	start := time.Now()
	j.image = imageprocessing.Grayscale(j.image)
	j.times.Grayscale = time.Since(start)
	return j
}

func (s stages) save(j job) job {
	if j.err != nil {
		return j
	}
	start := time.Now()
	outputPath := filepath.Join(s.cfg.OutputDir, OutputName(j.inputName))
	j.err = encodeFile(outputPath, j.image)
	if j.err == nil {
		j.outputPath = outputPath
		j.outputSize = j.image.Bounds().Size()
	}
	j.image = nil // the pixels are no longer needed; let the memory go
	j.times.Save = time.Since(start)
	return j
}

func decodeFile(inputs fs.FS, name string) (image.Image, error) {
	file, err := inputs.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close() // read-only: a close error cannot lose data
	return imageprocessing.Decode(file)
}

// encodeFile writes img as a JPEG. A failed Close is reported too, because
// for a written file it can mean the data never reached the disk.
func encodeFile(outputPath string, img image.Image) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	encodeErr := imageprocessing.Encode(file, img)
	return errors.Join(encodeErr, file.Close())
}
