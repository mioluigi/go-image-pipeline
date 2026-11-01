package pipeline_test

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"io/fs"
	"os"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/mioluigi/go-image-pipeline/pipeline"
)

// benchmarkSize is the output box, as in the program's default (and CODEHEIM's 500 × 500).
const benchmarkSize = 500

// syntheticPhotos are four distinct 1536 × 1024 JPEGs (fixed seeds), made once:
// photo-sized, so the stages do realistic amounts of work, and identical on
// every machine, so results can be compared between machines.
var syntheticPhotos = sync.OnceValue(func() [][]byte {
	photos := make([][]byte, 4)
	for i := range photos {
		var buffer bytes.Buffer
		if err := jpeg.Encode(&buffer, noiseImage(1536, 1024, uint64(100+i)), nil); err != nil {
			panic(err) // cannot happen when writing to memory
		}
		photos[i] = buffer.Bytes()
	}
	return photos
})

// syntheticFolder returns an in-memory folder of count images cycling through
// the four synthetic photos, so the files are not read from disk while timed.
func syntheticFolder(count int) (fstest.MapFS, []string) {
	folder := fstest.MapFS{}
	names := make([]string, count)
	for i := range names {
		names[i] = fmt.Sprintf("photo%03d.jpg", i)
		folder[names[i]] = &fstest.MapFile{Data: syntheticPhotos()[i%4]}
	}
	return folder, names
}

// benchmarkRun times complete runs and reports throughput in images per
// second next to Go's time and allocations per run.
func benchmarkRun(b *testing.B, mode pipeline.Mode, inputs fs.FS, names []string) {
	cfg := pipeline.Config{OutputDir: b.TempDir(), Size: benchmarkSize} // Workers: one per CPU
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		results, err := pipeline.Run(mode, inputs, names, cfg)
		if err != nil {
			b.Fatal(err)
		}
		for _, result := range results {
			if result.Err != nil {
				b.Fatalf("%s: %v", result.InputName, result.Err)
			}
		}
	}
	b.ReportMetric(float64(len(names)*b.N)/b.Elapsed().Seconds(), "images/s")
}

// BenchmarkRun compares the three modes on 4, 16 and 64 synthetic photos.
// Run with: go test -bench BenchmarkRun -benchmem ./pipeline/
func BenchmarkRun(b *testing.B) {
	for _, mode := range allModes {
		for _, count := range []int{4, 16, 64} {
			inputs, names := syntheticFolder(count)
			b.Run(fmt.Sprintf("mode=%s/images=%d", mode, count), func(b *testing.B) {
				benchmarkRun(b, mode, inputs, names)
			})
		}
	}
}

// BenchmarkRunBundled compares the three modes on the photos in ../images,
// the ones the program ships with.
func BenchmarkRunBundled(b *testing.B) {
	inputs := os.DirFS("../images")
	names, err := pipeline.ImageNames(inputs)
	if err != nil || len(names) == 0 {
		b.Skipf("no bundled images in ../images (%v)", err)
	}
	for _, mode := range allModes {
		b.Run(fmt.Sprintf("mode=%s/images=%d", mode, len(names)), func(b *testing.B) {
			benchmarkRun(b, mode, inputs, names)
		})
	}
}
