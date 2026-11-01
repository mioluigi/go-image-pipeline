package main

import (
	"embed"
	"io/fs"

	"github.com/mioluigi/go-image-pipeline/pipeline"
)

// embeddedFiles holds the images/ folder, built into the program so it runs
// with no arguments from anywhere: a downloaded .exe or a double-clicked .app.
//
//go:embed images
var embeddedFiles embed.FS

// bundledImages returns the built-in photos as a folder.
func bundledImages() (fs.FS, error) {
	return fs.Sub(embeddedFiles, "images")
}

// bundledImageNames lists the built-in photos (used by the tests).
func bundledImageNames() ([]string, error) {
	images, err := bundledImages()
	if err != nil {
		return nil, err
	}
	return pipeline.ImageNames(images)
}
