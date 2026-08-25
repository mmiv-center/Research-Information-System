package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

type StatusTUI struct {
	dataSets  map[string]map[string]SeriesInfo
	viewer    *tview.TextView
	summary   *tview.TextView
	selection *tview.TreeView
	app       *tview.Application
	flex      *tview.Flex
	// topSection is the inner flex that lays the summary (left) and the
	// viewer (right) out side by side; it is kept as a field so its layout can
	// be re-sized when the display mode changes.
	topSection                *tview.Flex
	ast                       AST
	selectedDatasets          []dicom.Dataset
	currentImage              int
	selectedSeriesInformation SeriesInfo
	config                    Config
	stopAnimation             bool
	lastSelectedSeries        string
	summaryLines              []string
	// tagVaries reports, for the currently selected series, which tags have a
	// value that is not constant across the series (i.e. the tag changes at
	// some point). Such tags get a gray star in the "DICOM Meta Data" panel so
	// the interesting (variable) tags stand out even while their value is
	// currently stable; a nil map simply means none are marked.
	tagVaries map[tag.Tag]bool
	// mode selects how the "DICOM Meta Data" panel is rendered:
	//   1 (default) tag shown as "(group,tag)", panel 30 wide
	//   2           tag shown as its camel-case name, panel 60 wide
	// Toggled with the keys '1' and '2'.
	mode int
}

// leftPanelWidth is the width of the "DICOM Meta Data" panel for a given mode.
func leftPanelWidth(mode int) int {
	if mode == 2 {
		return 60
	}
	return 30
}

// summaryTitle builds the "DICOM Meta Data" panel title. Besides naming the
// panel it carries a hint of the shortcut that switches to the *other* display
// mode: the inactive mode's number is shown in gray so it reads as "press this
// key to switch" (press '2' while in mode 1, '1' while in mode 2). The gray is
// applied with tview style tags, which the border-title renderer honors; "[-]"
// resets back to the title's own color. Any mode value other than 2 is treated
// as mode 1 (the default).
func summaryTitle(mode int) string {
	if mode == 2 {
		return "DICOM Meta Data [[gray]1[-]|2]"
	}
	return "DICOM Meta Data [1|[gray]2[-]]"
}

// maxTagKeywordLen is the maximum width of the human-readable tag keyword
// shown in mode 2. Real DICOM tag names in a CT series run up to ~36 chars
// (e.g. "ContrastBolusIngredientConcentration"); most are under 28. Truncating
// at 30 keeps the vast majority of names intact while guaranteeing that, in
// the 60-wide panel, there is still room for the tag's value on the same line.
const maxTagKeywordLen = 30

// tagKeyword returns the human-readable DICOM keyword of a tag in camel-case
// notation without spaces (e.g. "InstanceNumber"), truncated to
// maxTagKeywordLen. If the tag is not part of the DICOM dictionary (e.g. a
// private/vendor tag), it falls back to the "(gggg,tttt)" form so the line is
// still identifiable.
func tagKeyword(t tag.Tag) string {
	info, err := tag.Find(t)
	if err != nil {
		return t.String()
	}
	kw := info.Keyword
	if len(kw) > maxTagKeywordLen {
		kw = kw[:maxTagKeywordLen]
	}
	return kw
}

func findSeriesInfo(dataSets map[string]map[string]SeriesInfo, SeriesInstanceUID string) (SeriesInfo, error) {
	for _, series := range dataSets {
		if _, ok := series[SeriesInstanceUID]; ok {
			return series[SeriesInstanceUID], nil
		}
	}
	return SeriesInfo{}, fmt.Errorf("SeriesInstanceUID %s not found", SeriesInstanceUID)
}

// typeToColor maps the type of a variable (TagAndValue) to a tview color tag
// for the "Current selection" panel. The type is shown as a color of the line
// instead of as trailing text. Unknown or empty types keep the default color.
func typeToColor(t string) string {
	switch t {
	case "numeric":
		return "[aqua]"
	case "categorical":
		return "[lime]"
	case "time":
		return "[yellow]"
	case "datetime":
		return "[olive]"
	case "uid":
		return "[silver]"
	default:
		return ""
	}
}

func addDataset(statusTUI *StatusTUI, dataset dicom.Dataset) {
	(*statusTUI).selectedDatasets = append((*statusTUI).selectedDatasets, dataset)
}

// updateSummary re-renders the "Current selection" panel with the metadata of
// the displayed image, so the panel follows the animation instead of only
// showing the series information. Each tag line can carry a leading star: a
// bright '*' when the value differs from the previously displayed image, or a
// gray '*' when the value is unchanged right now but the tag changes at some
// point in the series (see tagVaries), so the interesting tags stay visible.
// Replacing the text of the TextView preserves its scroll position (tview
// keeps the line offset and the track-end state), so the panel does not jump
// back to the top with every new image.
func updateSummary(statusTUI *StatusTUI, dataset dicom.Dataset) {
	if statusTUI.app == nil {
		return
	}
	series := statusTUI.selectedSeriesInformation
	all := allTagAndValues(dataset)

	// the plain-text lines of the panel; they are remembered (in
	// summaryLines) so the next image can be compared line by line. The tag is
	// kept too so the "varies across the series" star can be looked up per tag.
	type panelLine struct {
		tag  tag.Tag
		text string
		typ  string
	}
	lines := make([]panelLine, 0, len(all)+3)
	lines = append(lines, panelLine{text: series.SeriesDescription})
	lines = append(lines, panelLine{text: strings.Join(series.ClassifyTypes, ",")})
	lines = append(lines, panelLine{text: ""})
	for _, a := range all {
		// mode 1 shows the tag as "(group,tag)"; mode 2 shows its human-readable
		// camel-case name (see tagKeyword)
		label := a.Tag.String()
		if statusTUI.mode == 2 {
			label = tagKeyword(a.Tag)
		}
		lines = append(lines, panelLine{tag: a.Tag, text: label + " " + fmt.Sprintf("%v", a.Value), typ: a.Type})
	}

	plain := make([]string, len(lines))
	var out strings.Builder
	for i, l := range lines {
		plain[i] = l.text
		// A line carries a single star, colored by what is special about it:
		//   bright '*'  the value differs from the previously displayed image
		//                (the current behaviour; takes priority)
		//   gray   '*'  the value is unchanged right now but this tag changes
		//                at some point in the series, so the interesting tags
		//                stay visible. It is set per series (tagVaries) and is
		//                independent of the first-render check below.
		// Header lines have no tag, so tagVaries looks them up as false and
		// they only ever get the bright star.
		changed := statusTUI.summaryLines != nil &&
			(i >= len(statusTUI.summaryLines) || statusTUI.summaryLines[i] != l.text)
		var mark string
		switch {
		case changed:
			mark = "*" // default (bright) color
		case statusTUI.tagVaries[l.tag]:
			mark = "[gray]*[-]"
		default:
			mark = " "
		}
		content := l.text
		if color := typeToColor(l.typ); color != "" {
			content = color + l.text + "[-]"
		}
		out.WriteString(mark + content)
		out.WriteByte('\n')
	}

	statusTUI.summary.Clear()
	fmt.Fprint(statusTUI.summary, out.String())
	statusTUI.summaryLines = plain
}

// sequenceNameOf returns the Sequence Name (0018,0050) of a single image, or
// the empty string when the tag is absent or empty. It is the primary sort key,
// so images belonging to different sequences are grouped together before the
// per-sequence (Slice Location / Instance Number) ordering is applied.
func sequenceNameOf(dataset dicom.Dataset) string {
	if v, err := dataset.FindElementByTag(tag.SequenceName); err == nil {
		if strs := dicom.MustGetStrings(v.Value); len(strs) > 0 {
			return strs[0]
		}
	}
	return ""
}

// sliceSortKey extracts the two per-sequence ordering keys from a single image:
// the Instance Number (0020,0013, VR IS) and the Slice Location (0020,1041, VR
// DS). The dicom library stores both as strings (they map to VRStringList), so
// we parse them here. hasSliceLoc reports whether a Slice Location was present
// and could be parsed as a number.
func sliceSortKey(dataset dicom.Dataset) (instanceNumber int, sliceLocation float64, hasSliceLoc bool) {
	if v, err := dataset.FindElementByTag(tag.InstanceNumber); err == nil {
		if strs := dicom.MustGetStrings(v.Value); len(strs) > 0 {
			instanceNumber, _ = strconv.Atoi(strs[0])
		}
	}
	if v, err := dataset.FindElementByTag(tag.SliceLocation); err == nil {
		if strs := dicom.MustGetStrings(v.Value); len(strs) > 0 {
			if loc, err := strconv.ParseFloat(strs[0], 64); err == nil {
				sliceLocation = loc
				hasSliceLoc = true
			}
		}
	}
	return
}

// sortDatasets orders the images of a series for the animation. The primary
// key is the Sequence Name (0018,0050), so images of different sequences are
// grouped together in name order. Within a sequence, when it carries more than
// one distinct Slice Location the anatomical order is used (Slice Location
// ascending, with Instance Number as a tie-breaker); otherwise it falls back
// to acquisition order (Instance Number ascending). This mirrors how DICOM
// viewers order slices for display, and keeps the animation a clean sweep
// through each sequence's anatomy.
func sortDatasets(datasets []dicom.Dataset) {
	// Pair each dataset with its sort keys so the keys move together while
	// sorting. (Sorting the datasets with a comparator that reads a separate,
	// unsorted key slice is wrong: the sort only permutes the datasets, so the
	// keys fall out of step with them and the result is an arbitrary order.)
	type item struct {
		dataset        dicom.Dataset
		sequenceName   string
		instanceNumber int
		sliceLocation  float64
		hasSliceLoc    bool
	}
	items := make([]item, len(datasets))
	for i, d := range datasets {
		in, sl, has := sliceSortKey(d)
		items[i] = item{d, sequenceNameOf(d), in, sl, has}
	}
	// Within a Sequence Name, Slice Location is the ordering key only when the
	// sequence carries more than one distinct value; a single (or absent) Slice
	// Location carries no order, so Instance Number drives the sequence instead.
	distinct := map[string]map[float64]struct{}{}
	for _, it := range items {
		if !it.hasSliceLoc {
			continue
		}
		set := distinct[it.sequenceName]
		if set == nil {
			set = map[float64]struct{}{}
			distinct[it.sequenceName] = set
		}
		set[it.sliceLocation] = struct{}{}
	}
	useSliceLoc := func(name string) bool { return len(distinct[name]) > 1 }
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.sequenceName != b.sequenceName {
			return a.sequenceName < b.sequenceName
		}
		if useSliceLoc(a.sequenceName) && a.sliceLocation != b.sliceLocation {
			return a.sliceLocation < b.sliceLocation
		}
		return a.instanceNumber < b.instanceNumber
	})
	for i := range items {
		datasets[i] = items[i].dataset
	}
}

// computeTagVaries examines every image of a series and reports the tags whose
// value is not constant across it — i.e. the tags that "change at some point".
// A tag is present in the returned map only if it takes more than one distinct
// value, so a lookup is a simple boolean test (and a missing tag, or a lookup
// against a nil map, reads as false). The value strings are compared the same
// way the panel renders them (fmt.Sprintf("%v", value)), so this matches what
// the user sees change.
func computeTagVaries(datasets []dicom.Dataset) map[tag.Tag]bool {
	values := make(map[tag.Tag]map[string]struct{})
	for _, d := range datasets {
		for _, a := range allTagAndValues(d) {
			v := fmt.Sprintf("%v", a.Value)
			set := values[a.Tag]
			if set == nil {
				set = make(map[string]struct{})
				values[a.Tag] = set
			}
			set[v] = struct{}{}
		}
	}
	varies := make(map[tag.Tag]bool)
	for t, set := range values {
		if len(set) > 1 {
			varies[t] = true
		}
	}
	return varies
}

func (statusTUI *StatusTUI) Init() {
	if len(statusTUI.dataSets) == 0 {
		fmt.Println("Warning: there are no datasets to visualize")
	}
	if len(statusTUI.ast.Rules) == 0 {
		fmt.Println("Warning: there is no ast defined")
	}
	newPrimitive := func(text string) *tview.TextView {
		return tview.NewTextView().
			SetTextAlign(tview.AlignLeft).
			SetText(text)
	}
	statusTUI.summary = newPrimitive("").SetDynamicColors(true)
	// keep lines on a single row: the window is narrow, so wrapping makes
	// the text hard to read. Long lines are clipped at the border instead
	// and can be scrolled sideways (left/right arrow or h/l) while vertical
	// scrolling works as before (up/down arrow or j/k).
	statusTUI.summary.SetWrap(false)
	statusTUI.summary.SetBorder(true).SetTitle(summaryTitle(statusTUI.mode))
	statusTUI.viewer = newPrimitive("").SetDynamicColors(true)
	statusTUI.selection = tview.NewTreeView()
	statusTUI.selection.SetBorder(true)
	statusTUI.selection.SetTitle("Series by job")
	statusTUI.viewer.SetBorder(true).SetTitle("DICOM")

	path_config := input_dir + "/.ror/config"
	conf, err := readConfig(path_config)
	if err == nil {
		var col tcell.Color
		// we set a text color only if the value is set (not equal to empty string)
		if conf.Viewer.TextColor != "" {
			col = tcell.GetColor(conf.Viewer.TextColor)
			statusTUI.viewer.SetTextColor(col)
		}
	}
	statusTUI.config = conf

	// default display mode: tag shown as "(group,tag)", panel 30 wide
	statusTUI.mode = 1
	// the inner flex lays the summary (left) and the viewer (right) out side by
	// side; keep a reference so the summary's width can be re-sized on a mode
	// change (mode 2 doubles it). The outer flex stacks that row above the
	// selection tree.
	statusTUI.topSection = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(statusTUI.summary, leftPanelWidth(statusTUI.mode), 1, false).
		AddItem(statusTUI.viewer, 0, 1, true)
	statusTUI.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(statusTUI.topSection, 0, 1, false).
		AddItem(statusTUI.selection, 12, 1, false)

	// start with setting up the list of selected datasets
	selected, _ := findMatchingSets(statusTUI.ast, statusTUI.dataSets)
	root := tview.NewTreeNode("Selections").SetReference("")
	statusTUI.selection.SetRoot(root).SetCurrentNode(root)

	for idx, entry := range selected {
		firstSeries, err := findSeriesInfo(statusTUI.dataSets, entry[0].SeriesInstanceUID)
		if err != nil {
			continue
		}
		node := tview.NewTreeNode(fmt.Sprintf("%d/%d %s-%s [yellow]%s", idx+1, len(selected), firstSeries.PatientID, firstSeries.PatientName, firstSeries.StudyDescription)).
			SetSelectable(false)
		root.AddChild(node)
		s := "s"
		if firstSeries.NumImages == 1 {
			s = ""
		}
		// sort the entries in the list by SeriesNumber + SeriesDescription
		type OneStudy struct {
			SeriesNumber      int
			SeriesDescription string
			SeriesInstanceUID string
			NumImages         int
			Name              string
			SequenceName      string
		}
		var AllSeries []OneStudy = make([]OneStudy, 0)
		for _, entry2 := range entry {
			firstSeries, err := findSeriesInfo(statusTUI.dataSets, entry2.SeriesInstanceUID)
			if err != nil {
				continue
			}
			AllSeries = append(AllSeries, OneStudy{
				SeriesNumber:      firstSeries.SeriesNumber,
				SeriesDescription: firstSeries.SeriesDescription,
				SeriesInstanceUID: entry2.SeriesInstanceUID,
				NumImages:         firstSeries.NumImages,
				Name:              entry2.Name,
				SequenceName:      firstSeries.SequenceName,
			})
		}
		// sort AllSeries now
		sort.Slice(AllSeries[:], func(i, j int) bool {
			if AllSeries[i].SeriesNumber < AllSeries[j].SeriesNumber {
				return true
			}
			if (AllSeries[i].SeriesNumber == AllSeries[j].SeriesNumber) && (AllSeries[i].SeriesDescription < AllSeries[j].SeriesDescription) {
				return true
			}
			return false
		})
		// what is an appropriate number of decimal places for the SeriesNumber to line up?
		for _, entry2 := range AllSeries {
			ss := entry2.SequenceName
			if len(entry2.SequenceName) > 0 {
				ss = " \"" + ss + "\""
			}
			node2 := tview.NewTreeNode(fmt.Sprintf("%s series %03d%s [blue]\"%s\"[-] [gray]%s[-] %d image%s", entry2.Name, entry2.SeriesNumber, ss, entry2.SeriesDescription, entry2.SeriesInstanceUID, entry2.NumImages, s)).
				SetReference(entry2.SeriesInstanceUID).
				SetSelectable(true)
			node.AddChild(node2)
		}
	}

	statusTUI.selection.SetSelectedFunc(func(node *tview.TreeNode) {
		// calling this function twice for the same node should disable the function again
		SeriesInstanceUID := node.GetReference().(string)
		// is the SeriesInstanceUID the same as the one before? Switch off in that case?
		if statusTUI.lastSelectedSeries == SeriesInstanceUID {
			statusTUI.stopAnimation = true
			statusTUI.lastSelectedSeries = "" // set to empty so we can start it again after the next select
			return
		}
		statusTUI.stopAnimation = false
		statusTUI.lastSelectedSeries = SeriesInstanceUID

		if len(SeriesInstanceUID) == 0 {
			return
		}
		// the reference is the series instance uid, get a picture for that series
		series, err := findSeriesInfo(statusTUI.dataSets, SeriesInstanceUID)
		if err != nil {
			fmt.Println("we got an error", err)
			return
		}
		// remember the series information
		statusTUI.selectedSeriesInformation = series
		searchPath := series.Path
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			if statusTUI.app != nil {
				fmt.Fprintf(statusTUI.viewer, "The path %s could not be found. Maybe a drive was disconnected?\n", searchPath)
			} else {
				fmt.Println("warning: search path could not be found. Maybe a drive was disconnected?")
			}
			return
		}
		SelectedSeriesInstanceUID := SeriesInstanceUID
		statusTUI.selectedDatasets = nil
		// the first image of the new series is not marked as changed, only
		// the differences between the images of the series get the bright '*'
		statusTUI.summaryLines = nil
		// the gray "varies in series" stars are recomputed for the new series
		// below (after the images are loaded); clear the previous series' set
		statusTUI.tagVaries = nil
		statusTUI.currentImage = 0
		statusTUI.stopAnimation = false
		filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if err != nil {
				return err
			}

			dataset, err := dicom.ParseFile(path, nil) // See also: dicom.Parse which has a generic io.Reader API.
			if err == nil {
				SeriesInstanceUIDVal, err := dataset.FindElementByTag(tag.SeriesInstanceUID)
				if err == nil {
					var SeriesInstanceUID string = dicom.MustGetStrings(SeriesInstanceUIDVal.Value)[0]
					if SeriesInstanceUID != SelectedSeriesInstanceUID {
						return nil // ignore that file
					}
					_, err := dataset.FindElementByTag(tag.PixelData)
					if err != nil {
						return nil // ignore files that have no images
					}
					addDataset(statusTUI, dataset)
				}
			}
			return nil
		})
		// all images of the series are now in memory; order them before the
		// animation starts so it sweeps through the anatomy in a sensible order
		// instead of following the file-system walk order
		sortDatasets(statusTUI.selectedDatasets)
		// work out which tags vary across the series so the panel can gray-star
		// them; this is independent of the per-frame bright-star comparison
		statusTUI.tagVaries = computeTagVaries(statusTUI.selectedDatasets)

		children := node.GetChildren()
		if len(children) == 0 {
			// Load and show files in this directory.
			//path := reference.(string)
			//add(node, path)
		} else {
			// Collapse if visible, expand if collapsed.
			node.SetExpanded(!node.IsExpanded())
		}
	})

	statusTUI.Run()
}

func doEvery(d time.Duration, statusTUI *StatusTUI, f func(*StatusTUI, time.Time)) {
	for x := range time.Tick(d) {
		f(statusTUI, x)
	}
}

// showImageNow renders image idx into the DICOM pane and the "Current
// selection" panel and remembers it as the current image. It deliberately
// does not request a redraw: it must run inside tview's event loop (from the
// input capture or a queued update), where tview draws right after the handler
// returns — and where calling app.Draw() would deadlock the application,
// because app.Draw() queues the draw on that very loop and blocks until the
// loop processes it.
func showImageNow(statusTUI *StatusTUI, idx int) {
	if len(statusTUI.selectedDatasets) == 0 {
		return
	}
	if idx >= len(statusTUI.selectedDatasets) {
		idx = len(statusTUI.selectedDatasets) - 1
	}
	if idx < 0 {
		idx = 0
	}
	statusTUI.currentImage = idx
	showDataset(statusTUI.selectedDatasets[idx], 1, "path", "", statusTUI.viewer, statusTUI.config.Viewer.Clip)
	if statusTUI.app != nil {
		// update the "Current selection" panel for the new image before the
		// redraw so both are visible in the same frame
		updateSummary(statusTUI, statusTUI.selectedDatasets[idx])
		statusTUI.viewer.SetTitle(fmt.Sprintf("DICOM image %d/%d [c-ine, ▲⏷-scroll]", statusTUI.currentImage+1, len(statusTUI.selectedDatasets)))
	}
}

// nextImage displays one image from the currently selected image series in the viewer
func nextImage(statusTUI *StatusTUI, t time.Time) {
	if statusTUI.stopAnimation {
		return
	}
	//fmt.Printf("do something %p\n", &statusTUI.selectedDatasets)
	if len(statusTUI.selectedDatasets) == 0 {
		return
	}

	idx := ((statusTUI.currentImage) + 1) % len(statusTUI.selectedDatasets)
	// QueueUpdateDraw runs the render inside tview's event loop, so it can
	// never interleave with the key handler (which runs in the same loop) or
	// with a draw. This is the safe way to update the UI from this goroutine.
	if statusTUI.app != nil {
		statusTUI.app.QueueUpdateDraw(func() {
			showImageNow(statusTUI, idx)
		})
	} else {
		showImageNow(statusTUI, idx)
	}
}

// isSeriesNode reports whether node represents an actual image series, i.e. its
// reference is a non-empty SeriesInstanceUID. The "Selections" root and the
// job/group nodes are not series (their reference is empty or nil) and are
// excluded so scrolling never lands the highlight on them.
func isSeriesNode(node *tview.TreeNode) bool {
	ref, ok := node.GetReference().(string)
	return ok && len(ref) > 0
}

// flattenSelection returns the nodes of the selection tree in the same
// top-to-bottom order tview renders them: a preorder walk from the root that
// only descends into expanded nodes. This mirrors the TreeView's internal row
// list (its top level is 0), so a row index here lines up with a visible row
// there.
func flattenSelection(root *tview.TreeNode) []*tview.TreeNode {
	var nodes []*tview.TreeNode
	var walk func(n *tview.TreeNode)
	walk = func(n *tview.TreeNode) {
		nodes = append(nodes, n)
		if n.IsExpanded() {
			for _, c := range n.GetChildren() {
				walk(c)
			}
		}
	}
	if root != nil {
		walk(root)
	}
	return nodes
}

// captureMouse is the global mouse handler, invoked by tview before it routes a
// mouse action to the focused primitive. When the user scrolls over the
// selection tree (the bottom window), the touch-pad scroll moves the visible
// rows but leaves the highlighted series where it was. Here we move the
// highlight to the series at the edge the scroll is heading toward — the
// topmost visible one for a scroll up, the bottommost for a scroll down — so
// the highlight follows the scroll. The event is returned unchanged so tview
// still performs its own one-row scroll; we only re-point the highlight. Other
// actions, and scrolls over other windows, pass through untouched.
func (statusTUI *StatusTUI) captureMouse(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
	if statusTUI.selection == nil ||
		(action != tview.MouseScrollUp && action != tview.MouseScrollDown) {
		return event, action
	}
	x, y := event.Position()
	if !statusTUI.selection.InRect(x, y) {
		return event, action // scroll is not over the series tree; leave it alone
	}

	// tview applies the scroll on the next draw as (offset += ±1, clamped), so
	// the new first visible row is at most one away from the current offset.
	offset := statusTUI.selection.GetScrollOffset()
	rowCount := statusTUI.selection.GetRowCount()
	_, _, _, height := statusTUI.selection.GetInnerRect()
	if height <= 0 || rowCount <= 0 {
		return event, action
	}
	switch action {
	case tview.MouseScrollUp:
		offset--
	case tview.MouseScrollDown:
		offset++
	}
	if offset < 0 {
		offset = 0
	}
	if maxOffset := rowCount - height; maxOffset < 0 {
		return event, action // fewer rows than the window; nothing to highlight
	} else if offset > maxOffset {
		offset = maxOffset
	}

	// The visible rows after this scroll are [offset, offset+height). Pick the
	// series at the edge the scroll is moving toward.
	target := selectionScrollTarget(flattenSelection(statusTUI.selection.GetRoot()), offset, height, action)
	if target != nil && target != statusTUI.selection.GetCurrentNode() {
		statusTUI.selection.SetCurrentNode(target)
	}
	return event, action
}

// selectionScrollTarget returns the series node to highlight after scrolling in
// the given direction, within the visible rows [offset, offset+height) of the
// flattened tree: the first series for a scroll up, the last for a scroll
// down. It returns nil when that range holds no series node.
func selectionScrollTarget(nodes []*tview.TreeNode, offset, height int, action tview.MouseAction) *tview.TreeNode {
	end := offset + height
	if end > len(nodes) {
		end = len(nodes)
	}
	if end <= offset {
		return nil
	}
	switch action {
	case tview.MouseScrollUp:
		for _, n := range nodes[offset:end] {
			if isSeriesNode(n) {
				return n
			}
		}
	case tview.MouseScrollDown:
		for i := end - 1; i >= offset; i-- {
			if isSeriesNode(nodes[i]) {
				return nodes[i]
			}
		}
	}
	return nil
}

// captureInput is the global key handler. It runs inside tview's event loop,
// so it must not call app.Draw() — app.Draw() queues the draw on that very
// loop and blocks until the loop processes it, which would make the loop wait
// for itself and deadlock the application (the reason the earlier slice
// navigation attempt crashed). Instead: render the new image, return nil to
// consume the key, and let tview redraw right after the handler returns.
func (statusTUI *StatusTUI) captureInput(event *tcell.EventKey) *tcell.EventKey {
	k := event.Key()
	// with the animation stopped, the cursor keys step through the slices of
	// the selected series while the DICOM pane has the focus: down/right
	// shows the next slice, up/left the previous one. The index is clamped in
	// showImageNow, so holding a key at the end of the series is harmless.
	if statusTUI.stopAnimation && statusTUI.app.GetFocus() == statusTUI.viewer && len(statusTUI.selectedDatasets) > 0 {
		switch k {
		case tcell.KeyDown, tcell.KeyRight:
			showImageNow(statusTUI, statusTUI.currentImage+1)
			return nil
		case tcell.KeyUp, tcell.KeyLeft:
			showImageNow(statusTUI, statusTUI.currentImage-1)
			return nil
		}
	}
	if k == tcell.KeyRune {
		switch event.Rune() {
		case rune('1'):
			// switch to mode 1: tag as "(group,tag)", panel 30 wide
			statusTUI.setMode(1)
			return nil // consume the key; tview redraws with the new layout
		case rune('2'):
			// switch to mode 2: tag as its camel-case name, panel 60 wide
			statusTUI.setMode(2)
			return nil // consume the key; tview redraws with the new layout
		case rune('q'):
			statusTUI.stopAnimation = !statusTUI.stopAnimation
			if app := statusTUI.app; app != nil {
				app.Stop()
			}
			return nil // quit the application
		case rune('c'):
			statusTUI.stopAnimation = !statusTUI.stopAnimation
		}
	}
	return event
}

// setMode switches the "DICOM Meta Data" panel between mode 1 (default: tag as
// "(group,tag)", panel 30 wide) and mode 2 (tag as its camel-case name, panel
// 60 wide). It runs inside tview's event loop, so it must not call app.Draw();
// it does the resize and re-render synchronously, and tview redraws right
// after the key handler returns.
func (statusTUI *StatusTUI) setMode(mode int) {
	if mode == statusTUI.mode {
		return // already in this mode
	}
	oldMode := statusTUI.mode
	statusTUI.mode = mode

	// (a) update the panel title so it hints the shortcut to switch back: the
	// now-inactive mode's number is shown in gray.
	statusTUI.summary.SetTitle(summaryTitle(mode))

	// (b) re-size the left panel. The viewer sits to its right in the same
	// row, so it absorbs the width change on the next layout pass.
	statusTUI.topSection.ResizeItem(statusTUI.summary, leftPanelWidth(mode), 1)

	// The viewer's on-screen rect is only updated when the flex is drawn, but
	// we want to re-render the image at its new width right now (and we are in
	// the event loop, so we cannot request a draw). The row's total width is
	// unchanged, so the viewer simply shifts by the summary's width delta.
	if x, y, w, h := statusTUI.viewer.GetRect(); w > 0 {
		delta := leftPanelWidth(mode) - leftPanelWidth(oldMode)
		statusTUI.viewer.SetRect(x+delta, y, w-delta, h)
	}

	// (c) re-render the current image and the panel in the new format. The
	// first render after the switch is not marked as changed (like a newly
	// selected series).
	statusTUI.summaryLines = nil
	if len(statusTUI.selectedDatasets) > 0 {
		showImageNow(statusTUI, statusTUI.currentImage)
	}
}

func (statusTUI *StatusTUI) Run() {
	statusTUI.stopAnimation = false
	// start a timer to display an image, should be like very 500msec
	go doEvery(200*time.Millisecond, statusTUI, nextImage)

	statusTUI.app = tview.NewApplication()

	statusTUI.app.SetInputCapture(statusTUI.captureInput)
	// route mouse actions through our handler too, so a touch-pad scroll over
	// the series tree can move the highlight (see captureMouse)
	statusTUI.app.SetMouseCapture(statusTUI.captureMouse)

	if err := statusTUI.app.SetRoot(statusTUI.flex, true).SetFocus(statusTUI.selection).EnableMouse(true).Run(); err != nil {
		fmt.Println("Error: The --tui mode is only available in a propper terminal.")
		panic(err)
	}
	defer statusTUI.app.Stop()
}

func (statusTUI StatusTUI) Stop() {
	statusTUI.app.Stop()
}
