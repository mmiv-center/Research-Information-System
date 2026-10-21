package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

// testDataset builds a minimal dataset carrying an InstanceNumber and a
// SliceLocation, both as DICOM strings, the same way the parsed real files do.
func testDataset(instanceNumber, sliceLocation string) dicom.Dataset {
	return testDatasetSeq("", instanceNumber, sliceLocation)
}

// testDatasetSeq is testDataset plus a Sequence Name (0018,0050), used to
// exercise the primary sort key. An empty sequenceName yields a dataset
// identical to testDataset's, so the two stay in agreement.
func testDatasetSeq(sequenceName, instanceNumber, sliceLocation string) dicom.Dataset {
	mustStr := func(s string) dicom.Value {
		v, err := dicom.NewValue([]string{s})
		if err != nil {
			panic(err)
		}
		return v
	}
	return dicom.Dataset{Elements: []*dicom.Element{
		{Tag: tag.SequenceName, Value: mustStr(sequenceName)},
		{Tag: tag.InstanceNumber, Value: mustStr(instanceNumber)},
		{Tag: tag.SliceLocation, Value: mustStr(sliceLocation)},
	}}
}

// TestSortDatasetsBySliceLocation is the regression test for the cursor-key
// "random location" bug. The slices are intentionally given in an order that
// is neither the InstanceNumber order nor the SliceLocation order, so a
// correct sort (SliceLocation ascending, InstanceNumber tie-break) yields a
// specific, checkable result. With the buggy comparator (keys not permuted
// with the datasets) the output order was arbitrary.
func TestSortDatasetsBySliceLocation(t *testing.T) {
	// (instance, sliceLocation) pairs in a deliberately scrambled walk order.
	datasets := []dicom.Dataset{
		testDataset("10", "50.0"),
		testDataset("1", "-30.5"),
		testDataset("3", "10.0"),
		testDataset("7", "50.0"),  // same location as instance 10 -> tie-break by instance
		testDataset("2", "-30.5"), // same location as instance 1 -> tie-break by instance
	}

	sortDatasets(datasets)

	// SliceLocation ascending, ties broken by InstanceNumber ascending:
	//   -30.5 -> instance 1 then 2 ;  10.0 -> instance 3 ;  50.0 -> instance 7 then 10
	want := []struct {
		instance int
		location float64
	}{
		{1, -30.5},
		{2, -30.5},
		{3, 10.0},
		{7, 50.0},
		{10, 50.0},
	}
	for i, d := range datasets {
		in, loc, _ := sliceSortKey(d)
		if in != want[i].instance || loc != want[i].location {
			t.Fatalf("position %d: want instance=%d loc=%.2f, got instance=%d loc=%.2f",
				i, want[i].instance, want[i].location, in, loc)
		}
	}
}

// TestSortDatasetsFallsBackToInstanceNumber checks the fallback ordering when
// no image carries a (distinct) SliceLocation: InstanceNumber ascending.
func TestSortDatasetsFallsBackToInstanceNumber(t *testing.T) {
	// Only one distinct slice location => useSliceLocation is false, so the
	// InstanceNumber must drive the order.
	datasets := []dicom.Dataset{
		testDataset("30", "100.0"),
		testDataset("15", "100.0"),
		testDataset("5", "100.0"),
	}

	sortDatasets(datasets)

	for i, d := range datasets {
		in, _, _ := sliceSortKey(d)
		if want := []int{5, 15, 30}[i]; in != want {
			t.Fatalf("position %d: want instance=%d, got %d", i, want, in)
		}
	}
}

// TestSortDatasetsGroupsBySequenceName is the regression test for "sort by
// SequenceName first". The InstanceNumbers and SliceLocations deliberately
// interleave across two sequences so that any ordering which does NOT group by
// SequenceName first comes out wrong. The expected result is sequence A's
// images (in anatomical order) before sequence B's.
func TestSortDatasetsGroupsBySequenceName(t *testing.T) {
	// (sequenceName, instance, sliceLocation) in a deliberately scrambled
	// order: the instance/location order is 1,2,8,9 but the sequence order is
	// A,A,B,B, so only a SequenceName-first sort yields the `want` below.
	datasets := []dicom.Dataset{
		testDatasetSeq("B", "1", "10.0"),
		testDatasetSeq("A", "9", "90.0"),
		testDatasetSeq("B", "2", "20.0"),
		testDatasetSeq("A", "8", "80.0"),
	}

	sortDatasets(datasets)

	// Sequence A (locations 80.0, 90.0) first, then sequence B (10.0, 20.0).
	want := []struct {
		sequence string
		instance int
		location float64
	}{
		{"A", 8, 80.0},
		{"A", 9, 90.0},
		{"B", 1, 10.0},
		{"B", 2, 20.0},
	}
	for i, d := range datasets {
		seq := sequenceNameOf(d)
		in, loc, _ := sliceSortKey(d)
		if seq != want[i].sequence || in != want[i].instance || loc != want[i].location {
			t.Fatalf("position %d: want seq=%s instance=%d loc=%.2f, got seq=%s instance=%d loc=%.2f",
				i, want[i].sequence, want[i].instance, want[i].location, seq, in, loc)
		}
	}
}

// TestSortDatasetsSequenceFallbackToInstanceNumber checks that the
// InstanceNumber fallback is applied within each SequenceName group, not
// globally: every image here shares one SliceLocation, so each sequence must
// order by InstanceNumber, with sequence A's group still coming first.
func TestSortDatasetsSequenceFallbackToInstanceNumber(t *testing.T) {
	datasets := []dicom.Dataset{
		testDatasetSeq("A", "5", "100.0"),
		testDatasetSeq("A", "3", "100.0"),
		testDatasetSeq("A", "7", "100.0"),
		testDatasetSeq("B", "1", "100.0"),
		testDatasetSeq("B", "2", "100.0"),
	}

	sortDatasets(datasets)

	// Sequence A in InstanceNumber order (3,5,7), then sequence B (1,2).
	want := []struct {
		sequence string
		instance int
	}{
		{"A", 3},
		{"A", 5},
		{"A", 7},
		{"B", 1},
		{"B", 2},
	}
	for i, d := range datasets {
		seq := sequenceNameOf(d)
		in, _, _ := sliceSortKey(d)
		if seq != want[i].sequence || in != want[i].instance {
			t.Fatalf("position %d: want seq=%s instance=%d, got seq=%s instance=%d",
				i, want[i].sequence, want[i].instance, seq, in)
		}
	}
}

// TestSortDatasetsRealSeries is an optional end-to-end check against a real
// on-disk series (e.g. a 900-frame acquisition). Point ROR_TEST_SERIES_DIR at
// a directory holding a single series of .dcm files and it verifies the
// sorted order is monotonic — the property the cursor-key navigation relies on
// to step through slices in anatomical order. Skipped when the variable is unset.
func TestSortDatasetsRealSeries(t *testing.T) {
	dir := os.Getenv("ROR_TEST_SERIES_DIR")
	if dir == "" {
		t.Skip("ROR_TEST_SERIES_DIR not set")
	}

	var seriesUID string
	var datasets []dicom.Dataset
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ds, err := dicom.ParseFile(path, nil)
		if err != nil {
			return nil
		}
		v, err := ds.FindElementByTag(tag.SeriesInstanceUID)
		if err != nil {
			return nil
		}
		uid := dicom.MustGetStrings(v.Value)[0]
		if _, err := ds.FindElementByTag(tag.PixelData); err != nil {
			return nil
		}
		if seriesUID == "" {
			seriesUID = uid
		}
		if uid != seriesUID {
			return nil
		}
		datasets = append(datasets, ds)
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(datasets) < 10 {
		t.Skipf("only %d images found in %s, not a real series", len(datasets), dir)
	}

	sortDatasets(datasets)

	// sortDatasets orders by SliceLocation (ascending, InstanceNumber
	// tie-break) when more than one distinct location is present, else by
	// InstanceNumber. Verify the order is monotonic under whichever rule applies.
	distinct := map[float64]bool{}
	locations := make([]float64, len(datasets))
	instances := make([]int, len(datasets))
	for i, d := range datasets {
		in, loc, _ := sliceSortKey(d)
		locations[i] = loc
		instances[i] = in
		distinct[loc] = true
	}
	if len(distinct) > 1 {
		for i := 1; i < len(locations); i++ {
			if locations[i] < locations[i-1] {
				t.Fatalf("slice locations not ascending at index %d: %v > %v", i, locations[i-1], locations[i])
			}
		}
	} else {
		for i := 1; i < len(instances); i++ {
			if instances[i] < instances[i-1] {
				t.Fatalf("instance numbers not ascending at index %d: %d > %d", i, instances[i-1], instances[i])
			}
		}
	}
	t.Logf("verified monotonic order over %d images of %s", len(datasets), seriesUID)
}
