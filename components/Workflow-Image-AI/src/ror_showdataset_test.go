package main

import (
	"os"
	"testing"

	"github.com/rivo/tview"
	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

// TestShowDatasetRendersAreStable is a regression test for the TUI animation
// bug where the same series rendered inverted on one loop, correct on the next
// loop, inverted again, ... The cause was that showDataset() transformed the
// frame's raw pixel buffer in place (v -> 32768-v), and that map is its own
// inverse: every render of the same cached dataset toggled between two images.
//
// The test needs a real unsigned 16-bit DICOM file, provided via ROR_TEST_DICOM.
func TestShowDatasetRendersAreStable(t *testing.T) {
	path := os.Getenv("ROR_TEST_DICOM")
	if path == "" {
		t.Skip("ROR_TEST_DICOM not set")
	}

	// parse once — this is what the TUI does: the dataset is cached and the
	// same instance is re-rendered for every animation frame
	ds, err := dicom.ParseFile(path, nil)
	if err != nil {
		t.Fatalf("could not parse %s: %v", path, err)
	}
	pixelDataElement, err := ds.FindElementByTag(tag.PixelData)
	if err != nil {
		t.Skip("file has no pixel data")
	}
	frames := dicom.MustGetPixelDataInfo(pixelDataElement.Value).Frames
	if len(frames) == 0 {
		t.Skip("file has no frames")
	}
	native, err := frames[0].GetNativeFrame()
	if err != nil {
		t.Skip("frame has no native data")
	}
	raw, ok := native.RawDataSlice().([]uint16)
	if !ok {
		t.Skip("pixel data is not []uint16")
	}
	first := raw[0]
	last := raw[len(raw)-1]

	// a viewer with a non-trivial size so the TUI render path produces real output
	viewer := tview.NewTextView()
	viewer.Box.SetRect(0, 0, 60, 30)

	w1, h1 := showDataset(ds, 1, "path", "", viewer, nil)
	if w1 == 0 || h1 == 0 {
		t.Fatal("showDataset did not render an image")
	}
	out1 := viewer.GetText(true)
	if out1 == "" {
		t.Fatal("viewer is empty after first render")
	}

	// second render of the same cached dataset — animation loop 2
	w2, h2 := showDataset(ds, 2, "path", "", viewer, nil)
	out2 := viewer.GetText(true)

	if w2 != w1 || h2 != h1 {
		t.Fatalf("dimensions changed between renders: %dx%d -> %dx%d", w1, h1, w2, h2)
	}
	if out1 != out2 {
		t.Fatalf("render output is not stable across renders (inverted-image bug):\nfirst: %q...\nsecond: %q...",
			out1[:min(len(out1), 200)], out2[:min(len(out2), 200)])
	}

	// and the frame's raw data must not have been mutated by the display
	if raw[0] != first || raw[len(raw)-1] != last {
		t.Fatalf("showDataset mutated the frame's raw pixel data: first %d -> %d, last %d -> %d",
			first, raw[0], last, raw[len(raw)-1])
	}
}
