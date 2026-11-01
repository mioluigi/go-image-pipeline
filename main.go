package main

import (
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/mioluigi/go-image-pipeline/imageprocessing"
)

// readImage and writeImage keep CODEHEIM's original behaviour (panic on any
// error) on top of the new reader/writer API until T10 replaces this program.
func readImage(path string) image.Image {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	img, err := imageprocessing.Decode(file)
	if err != nil {
		panic(err)
	}
	return img
}

func writeImage(path string, img image.Image) {
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if err := imageprocessing.Encode(file, img); err != nil {
		panic(err)
	}
}

type Job struct {
	InputPath string
	Image     image.Image
	OutPath   string
}

func loadImage(paths []string) <-chan Job {
	out := make(chan Job)
	go func() {
		// For each input path create a job and add it to
		// the out channel
		for _, p := range paths {
			job := Job{InputPath: p,
				OutPath: strings.Replace(p, "images/", "images/output/", 1)}
			job.Image = readImage(p)
			out <- job
		}
		close(out)
	}()
	return out
}

func resize(input <-chan Job) <-chan Job {
	out := make(chan Job)
	go func() {
		// For each input job, create a new job after resize and add it to
		// the out channel
		for job := range input { // Read from the channel
			job.Image = imageprocessing.Fit(job.Image, 500, 500)
			out <- job
		}
		close(out)
	}()
	return out
}

func convertToGrayscale(input <-chan Job) <-chan Job {
	out := make(chan Job)
	go func() {
		for job := range input { // Read from the channel
			job.Image = imageprocessing.Grayscale(job.Image)
			out <- job
		}
		close(out)
	}()
	return out
}

func saveImage(input <-chan Job) <-chan bool {
	out := make(chan bool)
	go func() {
		for job := range input { // Read from the channel
			writeImage(job.OutPath, job.Image)
			out <- true
		}
		close(out)
	}()
	return out
}

func main() {

	imagePaths := []string{"images/image1.jpeg",
		"images/image2.jpeg",
		"images/image3.jpeg",
		"images/image4.jpeg",
	}

	channel1 := loadImage(imagePaths)
	channel2 := resize(channel1)
	channel3 := convertToGrayscale(channel2)
	writeResults := saveImage(channel3)

	for success := range writeResults {
		if success {
			fmt.Println("Success!")
		} else {
			fmt.Println("Failed!")
		}
	}
}
