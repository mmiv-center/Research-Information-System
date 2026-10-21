package main

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
	"golang.org/x/image/draw"
)

// BenchmarkTUIRender measures the per-frame cost of the TUI image path of
// showDataset (decode + scale + half-block rendering) plus the summary panel
// data collection, using a real sample file. The output tells us how much
// work one scroll/step event currently does inside the event loop.
func BenchmarkTUIRender(b *testing.B) {
	var samples []string
	filepath.Walk("../data", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".dcm") {
			return nil
		}
		if len(samples) < 6 {
			samples = append(samples, path)
		}
		if len(samples) >= 6 {
			return filepath.SkipAll
		}
		return nil
	})
	if len(samples) == 0 {
		b.Skip("no sample data found")
	}

	// Simulate a 140x40 viewer cell grid (fitImageGrid output for a wide
	// terminal, as in the TUI).
	vw, printableHeight := 140, 40
	twidth, cells := fitImageGrid(vw, printableHeight)
	theight := cells * 2
	b.Logf("target grid: %dx%d px (%d half-block rows)", twidth, theight, cells)

	for _, sample := range samples {
		dataset, err := dicom.ParseFile(sample, nil)
		if err != nil {
			continue
		}
		pixelDataElement, err := dataset.FindElementByTag(tag.PixelData)
		if err != nil {
			continue
		}
		pixelDataInfo := dicom.MustGetPixelDataInfo(pixelDataElement.Value)
		if len(pixelDataInfo.Frames) == 0 {
			continue
		}
		fr := pixelDataInfo.Frames[0]

		name := filepath.Base(sample)
		// full pipeline per frame, as showDataset does in TUI mode
		b.Run("full/"+name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				native, err := fr.GetNativeFrame()
				if err != nil {
					b.Fatal(err)
				}
				img, err := native.GetImage()
				if err != nil {
					b.Fatal(err)
				}
				if img == nil {
					return
				}
				newImage := image.NewGray16(image.Rect(0, 0, twidth, theight))
				draw.CatmullRom.Scale(newImage, newImage.Bounds(), img, img.Bounds(), draw.Over, nil)
				printImage2RunesHalfBlocks(newImage, nil, 32768)
			}
		})
	}
}
