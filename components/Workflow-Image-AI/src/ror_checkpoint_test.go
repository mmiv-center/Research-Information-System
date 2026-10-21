package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestCopyDatasetsIndependence verifies that copyDatasets produces a deep copy
// (no shared maps or backing arrays), so the background writer can marshal it
// while the import loop keeps mutating the original.
func TestCopyDatasetsIndependence(t *testing.T) {
	orig := map[string]map[string]SeriesInfo{
		"study1": {
			"series1": {
				Modality:        "CT",
				NumImages:       5,
				ClassifyTypes:   []string{"axial", "coronal"},
				All:             []TagAndValue{{Value: []string{"1", "2"}, Type: "numeric"}},
				SOPInstanceUIDs: []string{"sop1"},
			},
		},
	}
	cp := copyDatasets(orig)

	// new map entries on the original must not leak into the copy
	orig["study1"]["series2"] = SeriesInfo{Modality: "MR"}
	orig["study2"] = map[string]SeriesInfo{"seriesX": {Modality: "PET"}}
	if _, ok := cp["study2"]; ok {
		t.Fatal("copy should not contain the study added after copying")
	}
	if _, ok := cp["study1"]["series2"]; ok {
		t.Fatal("copy should not contain the series added after copying")
	}

	// writing through the copy's slices must not bleed into the original,
	// which proves the copies do not share backing arrays
	cv := cp["study1"]["series1"]
	cv.ClassifyTypes[0] = "CHANGED"
	if ov := orig["study1"]["series1"]; ov.ClassifyTypes[0] == "CHANGED" {
		t.Fatal("copy shares the ClassifyTypes backing array with the original")
	}
	cv.All[0].Value[0] = "CHANGED"
	if ov := orig["study1"]["series1"]; ov.All[0].Value[0] == "CHANGED" {
		t.Fatal("copy shares the All[0].Value backing array with the original")
	}
	cv.SOPInstanceUIDs[0] = "CHANGED"
	if ov := orig["study1"]["series1"]; ov.SOPInstanceUIDs[0] == "CHANGED" {
		t.Fatal("copy shares the SOPInstanceUIDs backing array with the original")
	}
}

// TestCheckpointWriterRace simulates the import loop (a single goroutine
// mutating the live datasets and handing deep copies to the writer) running
// alongside the background writer goroutine. Run with -race to confirm the
// snapshot + handoff is race-free.
func TestCheckpointWriterRace(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ror"), 0755); err != nil {
		t.Fatal(err)
	}
	oldInputDir := input_dir
	input_dir = dir
	defer func() { input_dir = oldInputDir }()

	datasets := map[string]map[string]SeriesInfo{}
	w := newConfigCheckpointWriter()
	w.start()

	// main goroutine: mutate the live dataset and request a checkpoint, exactly
	// as dataSets does at each %200 boundary. The writer concurrently marshals
	// + writes its own captured copies.
	for i := 0; i < 200; i++ {
		datasets["study"] = map[string]SeriesInfo{}
		datasets["study"]["series"] = SeriesInfo{
			NumImages:       i,
			ClassifyTypes:   []string{"axial", "coronal"},
			All:             []TagAndValue{{Value: []string{"1", "2"}, Type: "numeric"}},
			SOPInstanceUIDs: []string{"sop0", "sop1"},
		}
		cfg := Config{Data: DataInfo{Path: "x"}}
		cfg.Data.DataInfo = copyDatasets(datasets)
		w.request(cfg)
	}

	// pre-build a batch of independent snapshots, then fire concurrent request()
	// calls to stress the writer's mutex / latest-wins coalescing
	prebuilt := make([]Config, 0, 40)
	for i := 0; i < 40; i++ {
		c := Config{Data: DataInfo{Path: "x"}}
		c.Data.DataInfo = copyDatasets(datasets)
		prebuilt = append(prebuilt, c)
	}
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range prebuilt {
				w.request(prebuilt[i])
			}
		}()
	}
	wg.Wait()

	w.stop() // flushes the final snapshot and waits for the writer to exit

	if _, err := os.Stat(filepath.Join(dir, ".ror", "config")); err != nil {
		t.Fatalf("expected the config file to have been written atomically: %v", err)
	}
}
