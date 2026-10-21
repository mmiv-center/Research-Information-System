package main

import (
	"os"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

// TestUpdateSummaryTracksChanges checks the per-image update of the
// "Current selection" panel: the panel must show the metadata of the image
// that is displayed, lines that differ from the previously displayed image
// must be marked with a leading '*', unchanged lines must not, and the scroll
// position of the panel must survive the re-render.
//
// The test needs one or two DICOM files of the same series (different slices),
// provided via ROR_TEST_DICOM and, optionally, ROR_TEST_DICOM2.
func TestUpdateSummaryTracksChanges(t *testing.T) {
	path := os.Getenv("ROR_TEST_DICOM")
	if path == "" {
		t.Skip("ROR_TEST_DICOM not set")
	}
	ds1, err := dicom.ParseFile(path, nil)
	if err != nil {
		t.Fatalf("could not parse %s: %v", path, err)
	}

	statusTUI := &StatusTUI{
		summary: tview.NewTextView().SetDynamicColors(true),
		app:     &tview.Application{},
		selectedSeriesInformation: SeriesInfo{
			SeriesDescription: "TEST SERIES",
			ClassifyTypes:     []string{"x", "y"},
		},
	}

	// first render of a series: the panel shows the image metadata and no
	// line is marked as changed
	updateSummary(statusTUI, ds1)
	text1 := statusTUI.summary.GetText(true) // true: strip the color tags
	if !strings.Contains(text1, "TEST SERIES") {
		t.Errorf("panel does not contain the series description:\n%s", text1)
	}
	if strings.Contains(text1, "*") {
		t.Errorf("first render must not contain '*' marks:\n%s", text1)
	}

	// the scroll position must survive a re-render of the same image
	row, col := 3, 5
	statusTUI.summary.ScrollTo(row, col)
	updateSummary(statusTUI, ds1)
	r, c := statusTUI.summary.GetScrollOffset()
	if r != row || c != col {
		t.Errorf("scroll position was reset: want (%d, %d), got (%d, %d)", row, col, r, c)
	}
	text2 := statusTUI.summary.GetText(true)
	if text2 != text1 {
		t.Errorf("re-rendering the same image changed the text:\n%s", text2)
	}
	if strings.Contains(text2, "*") {
		t.Errorf("re-rendering the same image must not mark lines:\n%s", text2)
	}

	// a second image of the series: exactly the lines whose content changed
	// are marked with '*'
	path2 := os.Getenv("ROR_TEST_DICOM2")
	if path2 == "" {
		t.Skip("ROR_TEST_DICOM2 not set, skipping change detection check")
	}
	ds2, err := dicom.ParseFile(path2, nil)
	if err != nil {
		t.Fatalf("could not parse %s: %v", path2, err)
	}
	updateSummary(statusTUI, ds2)
	text3 := statusTUI.summary.GetText(true)

	oldLines := strings.Split(text2, "\n")
	newLines := strings.Split(text3, "\n")
	if len(oldLines) != len(newLines) {
		t.Fatalf("line count changed: %d -> %d", len(oldLines), len(newLines))
	}
	// the two slices must differ in at least one line (and agree in at least
	// one) so both directions of the marking are actually exercised
	numChanged, numUnchanged := 0, 0
	for i, line := range newLines {
		if line == oldLines[i] {
			numUnchanged++
		} else {
			numChanged++
		}
	}
	if numChanged == 0 {
		t.Skip("the two DICOM files are identical, nothing to detect")
	}
	if numUnchanged == 0 {
		t.Skip("all lines differ between the two files, unchanged-line check is vacuous")
	}
	for i, line := range newLines {
		plain := strings.TrimPrefix(line, "*")
		marked := line != plain
		changed := plain != oldLines[i]
		if marked && !changed {
			t.Errorf("line %d is marked although unchanged: %q", i, line)
		}
		if !marked && changed {
			t.Errorf("line %d changed but is not marked: %q (was %q)", i, line, oldLines[i])
		}
	}
}

// TestShowImageNowClampsIndex checks the slice stepping used while the
// animation is stopped: the index must stay inside the series no matter how
// often the key is pressed, stepping must update the pane title, and a series
// without images must be a no-op instead of a panic.
func TestShowImageNowClampsIndex(t *testing.T) {
	path := os.Getenv("ROR_TEST_DICOM")
	if path == "" {
		t.Skip("ROR_TEST_DICOM not set")
	}
	ds, err := dicom.ParseFile(path, nil)
	if err != nil {
		t.Fatalf("could not parse %s: %v", path, err)
	}

	statusTUI := &StatusTUI{
		viewer:  tview.NewTextView().SetDynamicColors(true),
		summary: tview.NewTextView().SetDynamicColors(true),
		app:     tview.NewApplication(),
	}

	// no images selected yet: stepping must not panic
	showImageNow(statusTUI, 0)
	if statusTUI.currentImage != 0 {
		t.Errorf("currentImage changed although there is no image: %d", statusTUI.currentImage)
	}

	statusTUI.selectedDatasets = []dicom.Dataset{ds, ds}
	statusTUI.currentImage = 0

	showImageNow(statusTUI, -3)
	if statusTUI.currentImage != 0 {
		t.Errorf("stepping before the first slice: want currentImage 0, got %d", statusTUI.currentImage)
	}
	if title := statusTUI.viewer.GetTitle(); title != "DICOM image 1/2" {
		t.Errorf("title after clamping to the first slice: %q", title)
	}

	statusTUI.currentImage = 1
	showImageNow(statusTUI, 42)
	if statusTUI.currentImage != 1 {
		t.Errorf("stepping past the last slice: want currentImage 1, got %d", statusTUI.currentImage)
	}
	if title := statusTUI.viewer.GetTitle(); title != "DICOM image 2/2" {
		t.Errorf("title after clamping to the last slice: %q", title)
	}
}

// TestCaptureInputNavigatesSlices checks the key handling while the animation
// is stopped: with the DICOM pane focused the cursor keys step through the
// slices and are consumed (the handler returns nil, so tview redraws and does
// not forward the key); with another pane focused the keys pass through
// untouched; and 'c' toggles the animation.
func TestCaptureInputNavigatesSlices(t *testing.T) {
	path := os.Getenv("ROR_TEST_DICOM")
	if path == "" {
		t.Skip("ROR_TEST_DICOM not set")
	}
	ds, err := dicom.ParseFile(path, nil)
	if err != nil {
		t.Fatalf("could not parse %s: %v", path, err)
	}

	statusTUI := &StatusTUI{
		viewer:           tview.NewTextView().SetDynamicColors(true),
		summary:          tview.NewTextView().SetDynamicColors(true),
		app:              tview.NewApplication(),
		selectedDatasets: []dicom.Dataset{ds, ds},
		currentImage:     0,
		stopAnimation:    true,
	}
	statusTUI.app.SetFocus(statusTUI.viewer)

	key := func(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, 0, tcell.ModNone) }

	// down/right: next slice, and the key is consumed
	if got := statusTUI.captureInput(key(tcell.KeyDown)); got != nil {
		t.Errorf("KeyDown with the DICOM pane focused must be consumed, got %v", got)
	}
	if statusTUI.currentImage != 1 {
		t.Errorf("KeyDown did not advance to the next slice: %d", statusTUI.currentImage)
	}
	if got := statusTUI.captureInput(key(tcell.KeyRight)); got != nil {
		t.Errorf("KeyRight with the DICOM pane focused must be consumed, got %v", got)
	}
	if statusTUI.currentImage != 1 { // clamped at the last slice
		t.Errorf("KeyRight past the last slice: %d", statusTUI.currentImage)
	}

	// up/left: previous slice, and the key is consumed
	if got := statusTUI.captureInput(key(tcell.KeyUp)); got != nil {
		t.Errorf("KeyUp with the DICOM pane focused must be consumed, got %v", got)
	}
	if statusTUI.currentImage != 0 {
		t.Errorf("KeyUp did not go back to the previous slice: %d", statusTUI.currentImage)
	}
	if got := statusTUI.captureInput(key(tcell.KeyLeft)); got != nil {
		t.Errorf("KeyLeft with the DICOM pane focused must be consumed, got %v", got)
	}
	if statusTUI.currentImage != 0 { // clamped at the first slice
		t.Errorf("KeyLeft before the first slice: %d", statusTUI.currentImage)
	}

	// with another pane focused the cursor keys must pass through untouched
	statusTUI.app.SetFocus(tview.NewTextView())
	ev := key(tcell.KeyDown)
	if got := statusTUI.captureInput(ev); got != ev {
		t.Errorf("KeyDown with another pane focused must pass through, got %v", got)
	}
	if statusTUI.currentImage != 0 {
		t.Errorf("KeyDown with another pane focused must not step: %d", statusTUI.currentImage)
	}

	// 'c' toggles the animation and passes through
	evc := tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone)
	if got := statusTUI.captureInput(evc); got != evc {
		t.Errorf("the 'c' key must pass through, got %v", got)
	}
	if statusTUI.stopAnimation {
		t.Errorf("'c' did not stop the animation")
	}
	if got := statusTUI.captureInput(evc); got != evc {
		t.Errorf("the 'c' key must pass through, got %v", got)
	}
	if !statusTUI.stopAnimation {
		t.Errorf("'c' did not start the animation again")
	}
}

// TestLeftPanelWidth checks the panel width of both display modes: mode 1 is
// the narrow default, mode 2 doubles it.
func TestLeftPanelWidth(t *testing.T) {
	if leftPanelWidth(1) != 30 {
		t.Errorf("mode 1: want width 30, got %d", leftPanelWidth(1))
	}
	if leftPanelWidth(2) != 60 {
		t.Errorf("mode 2: want width 60, got %d", leftPanelWidth(2))
	}
}

// TestSummaryTitle checks that the "DICOM Meta Data" title hints the shortcut
// to the *other* mode by graying out that mode's number: in mode 1 the '2' is
// gray (press '2' to switch) and in mode 2 the '1' is gray (press '1' to
// switch back). The active mode's number stays un-grayed.
func TestSummaryTitle(t *testing.T) {
	// The default (mode 0, before it is set to 1) behaves like mode 1.
	for _, mode := range []int{0, 1} {
		title := summaryTitle(mode)
		if !strings.Contains(title, "DICOM Meta Data") {
			t.Errorf("mode %d: title should name the panel, got %q", mode, title)
		}
		if !strings.Contains(title, "[gray]2") {
			t.Errorf("mode %d: the '2' shortcut must be gray, got %q", mode, title)
		}
		if strings.Contains(title, "[gray]1") {
			t.Errorf("mode %d: the active '1' must not be gray, got %q", mode, title)
		}
	}

	title := summaryTitle(2)
	if !strings.Contains(title, "DICOM Meta Data") {
		t.Errorf("mode 2: title should name the panel, got %q", title)
	}
	if !strings.Contains(title, "[gray]1") {
		t.Errorf("mode 2: the '1' shortcut must be gray, got %q", title)
	}
	if strings.Contains(title, "[gray]2") {
		t.Errorf("mode 2: the active '2' must not be gray, got %q", title)
	}
}

// TestTagKeyword checks the camel-case tag names used in mode 2: known tags
// resolve to their DICOM keyword, keywords wider than maxTagKeywordLen are
// truncated, and tags that are unknown to the DICOM dictionary (e.g. private
// vendor tags) fall back to the "(gggg,tttt)" form.
func TestTagKeyword(t *testing.T) {
	// a short, well-known tag is shown in full
	if got := tagKeyword(tag.InstanceNumber); got != "InstanceNumber" {
		t.Errorf("InstanceNumber: want %q, got %q", "InstanceNumber", got)
	}

	// a keyword wider than the maximum is truncated to it
	long := tagKeyword(tag.ContrastBolusIngredientConcentration)
	if len(long) != maxTagKeywordLen {
		t.Errorf("long keyword: want length %d, got %d (%q)", maxTagKeywordLen, len(long), long)
	}
	if long != "ContrastBolusIngredientConcentration"[:maxTagKeywordLen] {
		t.Errorf("truncation: want %q, got %q", "ContrastBolusIngredientConcentration"[:maxTagKeywordLen], long)
	}

	// a tag outside the dictionary falls back to the (group,tag) form
	private := tag.Tag{Group: 0x0009, Element: 0x0010} // odd group => private
	if got := tagKeyword(private); got != "(0009,0010)" {
		t.Errorf("private tag: want %q, got %q", "(0009,0010)", got)
	}
}

// TestUpdateSummaryMode checks the "DICOM Meta Data" panel in both display
// modes: mode 1 shows the tags as "(group,tag)", mode 2 shows them as their
// camel-case names.
//
// The test needs one DICOM file of the series, provided via ROR_TEST_DICOM.
func TestUpdateSummaryMode(t *testing.T) {
	path := os.Getenv("ROR_TEST_DICOM")
	if path == "" {
		t.Skip("ROR_TEST_DICOM not set")
	}
	ds, err := dicom.ParseFile(path, nil)
	if err != nil {
		t.Fatalf("could not parse %s: %v", path, err)
	}

	newTUI := func(mode int) *StatusTUI {
		return &StatusTUI{
			summary: tview.NewTextView().SetDynamicColors(true),
			app:     &tview.Application{},
			mode:    mode,
			selectedSeriesInformation: SeriesInfo{
				SeriesDescription: "TEST SERIES",
				ClassifyTypes:     []string{"x", "y"},
			},
		}
	}

	// mode 1: the "(group,tag)" notation is present and no camel-case keyword
	// of the same tag leaks in
	m1 := newTUI(1)
	updateSummary(m1, ds)
	text1 := m1.summary.GetText(true)
	if !strings.Contains(text1, "(00") {
		t.Errorf("mode 1 panel must contain '(gggg,tttt)' tags:\n%s", text1)
	}

	// mode 2: the camel-case name of a tag that is in the panel replaces the
	// "(group,tag)" form of that tag
	m2 := newTUI(2)
	updateSummary(m2, ds)
	text2 := m2.summary.GetText(true)
	// take the first tag of the panel that is known to the DICOM dictionary,
	// so both modes render it and we can compare the two labels
	found := false
	for _, a := range allTagAndValues(ds) {
		info, err := tag.Find(a.Tag)
		if err != nil || len(info.Keyword) == 0 {
			continue
		}
		found = true
		want := info.Keyword
		if len(want) > maxTagKeywordLen {
			want = want[:maxTagKeywordLen]
		}
		hex := a.Tag.String()
		if !strings.Contains(text2, want) {
			t.Errorf("mode 2 panel is missing the camel-case name %q for %s:\n%s", want, hex, text2)
		}
		if strings.Contains(text2, hex) {
			t.Errorf("mode 2 panel still shows the (group,tag) form %s for %s:\n%s", hex, want, text2)
		}
		break
	}
	if !found {
		t.Skip("no dictionary-known tag in the panel, nothing to compare")
	}
}

// panelDataset builds a minimal dataset with two string-valued tags, in a fixed
// order (SliceLocation first, PatientID second) so the panel lines are
// predictable: the three header lines, then "(0020,1041) …" then "(0010,0020) …".
func panelDataset(sliceLoc, patientID string) dicom.Dataset {
	mustStr := func(s string) dicom.Value {
		v, err := dicom.NewValue([]string{s})
		if err != nil {
			panic(err)
		}
		return v
	}
	return dicom.Dataset{Elements: []*dicom.Element{
		{Tag: tag.SliceLocation, Value: mustStr(sliceLoc)},
		{Tag: tag.PatientID, Value: mustStr(patientID)},
	}}
}

// lineContaining returns the raw (tag-inclusive) panel line that contains sub.
func lineContaining(t *testing.T, raw, sub string) string {
	t.Helper()
	for _, line := range strings.Split(raw, "\n") {
		if strings.Contains(line, sub) {
			return line
		}
	}
	t.Fatalf("no panel line contains %q in:\n%s", sub, raw)
	return ""
}

// TestUpdateSummaryStars checks the star markers in the "DICOM Meta Data"
// panel. A tag whose value differs from the previous image gets the bright
// '*' (the original behaviour); a tag that is unchanged right now but varies at
// some point in the series (tagVaries) gets a gray '*'; a constant tag gets
// none. The bright star takes priority when both apply.
func TestUpdateSummaryStars(t *testing.T) {
	// tagVaries: SliceLocation varies across the series, PatientID is constant.
	tagVaries := map[tag.Tag]bool{tag.SliceLocation: true}
	newTUI := func() *StatusTUI {
		return &StatusTUI{
			summary: tview.NewTextView().SetDynamicColors(true),
			app:     &tview.Application{},
			mode:    1,
			tagVaries: tagVaries,
			selectedSeriesInformation: SeriesInfo{
				SeriesDescription: "MY SERIES",
				ClassifyTypes:     []string{"a", "b"},
			},
		}
	}
	sliceLocLine := "(0020,1041)"
	patientLine := "(0010,0020)"

	t.Run("first frame: variable tag gray, constant tag none", func(t *testing.T) {
		u := newTUI()
		u.summaryLines = nil // first render of the series
		updateSummary(u, panelDataset("10.0", "XYZ"))
		raw := u.summary.GetText(false) // keep the style tags

		lo := lineContaining(t, raw, sliceLocLine)
		if !strings.HasPrefix(lo, "[gray]*[-]") {
			t.Errorf("variable tag should have a gray star, got %q", lo)
		}
		po := lineContaining(t, raw, patientLine)
		if strings.Contains(po, "*") {
			t.Errorf("constant tag should have no star, got %q", po)
		}
		if strings.Contains(raw, "*[aqua]") {
			t.Errorf("first frame must not show a bright star, got:\n%s", raw)
		}
	})

	t.Run("variable tag changed vs previous: bright star wins", func(t *testing.T) {
		u := newTUI()
		// previous frame: SliceLocation was 5.0, PatientID was XYZ
		u.summaryLines = []string{"MY SERIES", "a,b", "", "(0020,1041) [5.0]", "(0010,0020) [XYZ]"}
		updateSummary(u, panelDataset("10.0", "XYZ"))
		raw := u.summary.GetText(false)

		lo := lineContaining(t, raw, sliceLocLine)
		if !strings.HasPrefix(lo, "*") || strings.HasPrefix(lo, "[gray]*") {
			t.Errorf("changed variable tag should have a bright (not gray) star, got %q", lo)
		}
		po := lineContaining(t, raw, patientLine)
		if strings.Contains(po, "*") {
			t.Errorf("unchanged constant tag should have no star, got %q", po)
		}
	})

	t.Run("variable tag unchanged vs previous: gray star stays", func(t *testing.T) {
		u := newTUI()
		// previous frame had the same values as the current one
		u.summaryLines = []string{"MY SERIES", "a,b", "", "(0020,1041) [10.0]", "(0010,0020) [XYZ]"}
		updateSummary(u, panelDataset("10.0", "XYZ"))
		raw := u.summary.GetText(false)

		lo := lineContaining(t, raw, sliceLocLine)
		if !strings.HasPrefix(lo, "[gray]*[-]") {
			t.Errorf("unchanged-but-variable tag should keep its gray star, got %q", lo)
		}
		po := lineContaining(t, raw, patientLine)
		if strings.Contains(po, "*") {
			t.Errorf("constant tag should have no star, got %q", po)
		}
	})

	t.Run("nil tagVaries: no gray stars (backwards compatible)", func(t *testing.T) {
		u := newTUI()
		u.tagVaries = nil // e.g. a series never loaded
		u.summaryLines = nil
		updateSummary(u, panelDataset("10.0", "XYZ"))
		raw := u.summary.GetText(false)
		if strings.Contains(raw, "[gray]*") {
			t.Errorf("with a nil tagVaries no gray star may appear, got:\n%s", raw)
		}
	})
}

// TestComputeTagVaries checks that a tag is reported as varying exactly when it
// takes more than one distinct value across the series, and that a tag with a
// single constant value (or absent) is not.
func TestComputeTagVaries(t *testing.T) {
	// SliceLocation differs between the two images; PatientID is identical.
	datasets := []dicom.Dataset{
		panelDataset("5.0", "XYZ"),
		panelDataset("10.0", "XYZ"),
	}
	varies := computeTagVaries(datasets)

	if !varies[tag.SliceLocation] {
		t.Errorf("SliceLocation takes two values, it should be marked as varying: %v", varies)
	}
	if varies[tag.PatientID] {
		t.Errorf("PatientID is constant, it should not be marked as varying: %v", varies)
	}

	// A single image: nothing varies.
	single := computeTagVaries([]dicom.Dataset{panelDataset("5.0", "XYZ")})
	if len(single) != 0 {
		t.Errorf("a single image has no varying tags, got: %v", single)
	}

	// An empty series: no varying tags, and no panic.
	empty := computeTagVaries(nil)
	if len(empty) != 0 {
		t.Errorf("an empty series has no varying tags, got: %v", empty)
	}
}
