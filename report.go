package main

import (
	"fmt"
	"image"
	"io"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/mioluigi/go-image-pipeline/pipeline"
)

// printReport writes one line per image, in input order, then a summary, and
// returns the number of images that failed. Plain ASCII, so it reads the same
// in a Windows console.
func printReport(out io.Writer, plan runPlan, results []pipeline.Result, elapsed time.Duration) int {
	table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	var failed int
	var stageTotals pipeline.StageTimes
	for _, result := range results {
		addStageTimes(&stageTotals, result.StageTimes)
		if result.Err != nil {
			failed++
			fmt.Fprintf(table, "FAILED\t%s\t%v\n", result.InputName, result.Err)
			continue
		}
		fmt.Fprintf(table, "saved\t%s\t%s -> %s\t%s\n", result.InputName,
			dimensions(result.OriginalSize), dimensions(result.OutputSize), result.OutputPath)
	}
	table.Flush()

	fmt.Fprintf(out, "\n%d saved, %d failed in %s (%s)\n", len(results)-failed, failed, seconds(elapsed), modeDescription(plan))
	fmt.Fprintf(out, "Time in each stage, summed over images: load %s, resize %s, grayscale %s, save %s\n",
		seconds(stageTotals.Load), seconds(stageTotals.Resize), seconds(stageTotals.Grayscale), seconds(stageTotals.Save))
	if plan.mode != pipeline.Sequential {
		fmt.Fprintln(out, "(Stages overlap when goroutines are used, so these sums can exceed the total time.)")
	}
	fmt.Fprintf(out, "Output folder: %s\n", absolute(plan.cfg.OutputDir))
	return failed
}

func addStageTimes(total *pipeline.StageTimes, add pipeline.StageTimes) {
	total.Load += add.Load
	total.Resize += add.Resize
	total.Grayscale += add.Grayscale
	total.Save += add.Save
}

func modeDescription(plan runPlan) string {
	if plan.mode == pipeline.Workers {
		return fmt.Sprintf("mode workers, %d workers", plan.cfg.Workers)
	}
	return "mode " + plan.mode.String()
}

func dimensions(size image.Point) string {
	return fmt.Sprintf("%dx%d", size.X, size.Y)
}

func seconds(d time.Duration) string {
	return fmt.Sprintf("%.2f s", d.Seconds())
}

// absolute shows where the files went even when the program was started by
// double-clicking, where the working folder is not obvious.
func absolute(folder string) string {
	if abs, err := filepath.Abs(folder); err == nil {
		return abs
	}
	return folder
}
