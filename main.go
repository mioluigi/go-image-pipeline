// Command go-image-pipeline converts a folder of JPEG and PNG photos into
// grayscale JPEGs that fit inside a square box, keeping each photo's shape,
// and reports how long each stage took. It can run without goroutines
// (sequential) or with them (pipeline, workers), so the times can be compared.
//
// With no arguments it processes the photos built into the program and writes
// to ./output. Run with -h for the options.
//
// Exit codes: 0 every image saved; 1 at least one image failed (the others
// were saved); 2 invalid usage, nothing processed.
//
// Replicates and extends CODEHEIM's pipeline example
// (github.com/code-heim/go_21_goroutines_pipeline, Apache-2.0).
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/mioluigi/go-image-pipeline/pipeline"
)

// Exit codes (spec FR-14a).
const (
	exitAllSaved     = 0
	exitImagesFailed = 1
	exitUsage        = 2
)

// Limits on numeric flags: wide enough for any real use, narrow enough to
// catch typing mistakes before any work starts.
const (
	maxSize    = 10_000
	maxWorkers = 1_024
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the whole program apart from os.Exit, so tests can call it.
func run(args []string, stdout, stderr io.Writer) int {
	settings, err := parseFlags(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return exitAllSaved
	}
	if err != nil {
		return exitUsage // the flag package has already shown the problem and the usage
	}
	job, err := prepare(settings)
	if err != nil {
		fmt.Fprintf(stderr, "go-image-pipeline: %v\nRun with -h for help.\n", err)
		return exitUsage
	}
	logger := newLogger(stderr, settings.verbose)
	logger.Info("starting", "mode", job.mode, "images", len(job.names), "input", job.inputLabel,
		"output", job.cfg.OutputDir, "size", job.cfg.Size, "workers", job.cfg.Workers)

	start := time.Now()
	results, err := pipeline.Run(job.mode, job.inputs, job.names, job.cfg)
	elapsed := time.Since(start)
	if err != nil {
		fmt.Fprintf(stderr, "go-image-pipeline: %v\n", err)
		return exitUsage
	}

	logResults(logger, results)
	failed := printReport(stdout, job, results, elapsed)
	logger.Info("finished", "saved", len(results)-failed, "failed", failed, "elapsed", elapsed)
	if failed > 0 {
		return exitImagesFailed
	}
	return exitAllSaved
}

// settings are the command-line flags as given, before validation.
type settings struct {
	mode, inputDir, outputDir string
	size, workers             int
	verbose                   bool
}

func parseFlags(args []string, stderr io.Writer) (settings, error) {
	var s settings
	flags := flag.NewFlagSet("go-image-pipeline", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&s.mode, "mode", "pipeline", "how to schedule the work: sequential (no goroutines), pipeline (one goroutine per stage), or workers (a pool of goroutines)")
	flags.StringVar(&s.inputDir, "in", "", "folder of JPEG and PNG photos (default: the photos built into the program)")
	flags.StringVar(&s.outputDir, "out", "output", "folder for the grayscale JPEGs; created if missing; must not be the input folder")
	flags.IntVar(&s.size, "size", 500, fmt.Sprintf("outputs fit inside size × size pixels, keeping their shape (1–%d)", maxSize))
	flags.IntVar(&s.workers, "workers", runtime.NumCPU(), fmt.Sprintf("goroutines in workers mode (1–%d; default: one per CPU)", maxWorkers))
	flags.BoolVar(&s.verbose, "verbose", false, "log every image's stage times to standard error")
	flags.Usage = func() {
		fmt.Fprint(stderr, "Converts photos to grayscale JPEGs, with or without goroutines, and times each stage.\n\n"+
			"Usage: go-image-pipeline [options]\n\nOptions:\n")
		flags.PrintDefaults()
		fmt.Fprint(stderr, "\nExamples:\n"+
			"  go-image-pipeline                                 the built-in photos, pipeline mode\n"+
			"  go-image-pipeline -mode sequential                the same without goroutines\n"+
			"  go-image-pipeline -mode workers -in ~/Pictures/trip -out ~/Desktop/gray\n")
	}
	if err := flags.Parse(args); err != nil {
		return settings{}, err
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected argument %q (use -in to choose a folder)\n", flags.Arg(0))
		flags.Usage()
		return settings{}, errors.New("unexpected argument")
	}
	return s, nil
}

// runPlan is a validated run: everything pipeline.Run needs, plus a label for
// the input that makes sense to a person.
type runPlan struct {
	mode       pipeline.Mode
	inputs     fs.FS
	inputLabel string
	names      []string
	cfg        pipeline.Config
}

// prepare checks every setting before any image is touched (spec FR-14).
func prepare(s settings) (runPlan, error) {
	mode, err := pipeline.ParseMode(s.mode)
	if err != nil {
		return runPlan{}, err
	}
	if s.size < 1 || s.size > maxSize {
		return runPlan{}, fmt.Errorf("-size must be between 1 and %d, got %d", maxSize, s.size)
	}
	if s.workers < 1 || s.workers > maxWorkers {
		return runPlan{}, fmt.Errorf("-workers must be between 1 and %d, got %d", maxWorkers, s.workers)
	}
	inputs, inputLabel, err := openInputs(s.inputDir)
	if err != nil {
		return runPlan{}, err
	}
	names, err := pipeline.ImageNames(inputs)
	if err != nil {
		return runPlan{}, fmt.Errorf("cannot read input folder %s: %w", inputLabel, err)
	}
	if len(names) == 0 {
		return runPlan{}, fmt.Errorf("no JPEG or PNG images in %s", inputLabel)
	}
	if s.inputDir != "" && sameFolder(s.inputDir, s.outputDir) {
		return runPlan{}, fmt.Errorf("output folder %s is the input folder; choose another so the originals are never overwritten", s.outputDir)
	}
	if err := checkOutputNames(names); err != nil {
		return runPlan{}, err
	}
	cfg := pipeline.Config{OutputDir: s.outputDir, Size: s.size, Workers: s.workers}
	return runPlan{mode: mode, inputs: inputs, inputLabel: inputLabel, names: names, cfg: cfg}, nil
}

// openInputs returns the built-in photos when inputDir is empty, otherwise
// the folder inputDir, after checking that it is a readable folder.
func openInputs(inputDir string) (fs.FS, string, error) {
	if inputDir == "" {
		inputs, err := bundledImages()
		return inputs, "the built-in photos", err
	}
	info, err := os.Stat(inputDir)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read input folder: %w", err)
	}
	if !info.IsDir() {
		return nil, "", fmt.Errorf("%s is not a folder", inputDir)
	}
	return os.DirFS(inputDir), inputDir, nil
}

// sameFolder reports whether two paths name the same existing folder,
// following symbolic links and ignoring letter case where the file system
// does (as on macOS). An output folder that does not exist yet cannot be the
// input folder.
func sameFolder(first, second string) bool {
	firstInfo, err := os.Stat(first)
	if err != nil {
		return false
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		return false
	}
	return os.SameFile(firstInfo, secondInfo)
}

// checkOutputNames rejects inputs that would overwrite each other's output,
// such as a.jpg and a.png, which are both saved as a.jpg.
func checkOutputNames(names []string) error {
	inputFor := make(map[string]string, len(names))
	for _, name := range names {
		outputName := pipeline.OutputName(name)
		if earlier, taken := inputFor[outputName]; taken {
			return fmt.Errorf("%s and %s would both be saved as %s; rename one", earlier, name, outputName)
		}
		inputFor[outputName] = name
	}
	return nil
}

func newLogger(stderr io.Writer, verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))
}

func logResults(logger *slog.Logger, results []pipeline.Result) {
	for _, result := range results {
		times := result.StageTimes
		logger.Debug("image done", "image", result.InputName, "saved", result.Err == nil,
			"load", times.Load, "resize", times.Resize, "grayscale", times.Grayscale, "save", times.Save, "error", result.Err)
	}
}
