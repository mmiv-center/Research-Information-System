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
	dataSets                  map[string]map[string]SeriesInfo
	viewer                    *tview.TextView
	summary                   *tview.TextView
	selection                 *tview.TreeView
	app                       *tview.Application
	flex                      *tview.Flex
	ast                       AST
	selectedDatasets          []dicom.Dataset
	currentImage              int
	selectedSeriesInformation SeriesInfo
	config                    Config
	stopAnimation             bool
	lastSelectedSeries        string
	summaryLines              []string
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
// showing the series information. Lines whose content differs from the
// previously displayed image are marked with a leading '*' so the changes
// stand out. Replacing the text of the TextView preserves its scroll
// position (tview keeps the line offset and the track-end state), so the
// panel does not jump back to the top with every new image.
func updateSummary(statusTUI *StatusTUI, dataset dicom.Dataset) {
	if statusTUI.app == nil {
		return
	}
	series := statusTUI.selectedSeriesInformation
	all := allTagAndValues(dataset)

	// the plain-text lines of the panel; they are remembered (in
	// summaryLines) so the next image can be compared line by line
	type panelLine struct {
		text string
		typ  string
	}
	lines := make([]panelLine, 0, len(all)+3)
	lines = append(lines, panelLine{text: series.SeriesDescription})
	lines = append(lines, panelLine{text: strings.Join(series.ClassifyTypes, ",")})
	lines = append(lines, panelLine{text: ""})
	for _, a := range all {
		lines = append(lines, panelLine{text: a.Tag.String() + " " + fmt.Sprintf("%v", a.Value), typ: a.Type})
	}

	plain := make([]string, len(lines))
	var out strings.Builder
	for i, l := range lines {
		plain[i] = l.text
		// mark lines that changed compared to the previously displayed
		// image; the first rendering of a series (summaryLines is nil) is
		// not marked
		mark := " "
		if statusTUI.summaryLines != nil && (i >= len(statusTUI.summaryLines) || statusTUI.summaryLines[i] != l.text) {
			mark = "*"
		}
		line := mark + l.text
		if color := typeToColor(l.typ); color != "" {
			line = mark + color + l.text + "[-]"
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}

	statusTUI.summary.Clear()
	fmt.Fprint(statusTUI.summary, out.String())
	statusTUI.summaryLines = plain
}

// sliceSortKey extracts the two ordering keys from a single image: the
// Instance Number (0020,0013, VR IS) and the Slice Location (0020,1041, VR DS).
// The dicom library stores both as strings (they map to VRStringList), so we
// parse them here. hasSliceLoc reports whether a Slice Location was present and
// could be parsed as a number.
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

// sortDatasets orders the images of a series for the animation. When every
// image carries a Slice Location and those values actually differ across the
// series, the anatomical order is used (Slice Location ascending, with Instance
// Number as a tie-breaker). Otherwise we fall back to acquisition order
// (Instance Number ascending). This mirrors how DICOM viewers order slices for
// display, and keeps the animation a clean sweep through the anatomy.
func sortDatasets(datasets []dicom.Dataset) {
	type key struct {
		instanceNumber int
		sliceLocation  float64
		hasSliceLoc    bool
	}
	keys := make([]key, len(datasets))
	numWithSlice := 0
	distinct := map[float64]struct{}{}
	for i, d := range datasets {
		in, sl, has := sliceSortKey(d)
		keys[i] = key{in, sl, has}
		if has {
			numWithSlice++
			distinct[sl] = struct{}{}
		}
	}
	useSliceLocation := numWithSlice == len(datasets) && len(distinct) > 1
	sort.SliceStable(datasets, func(i, j int) bool {
		if useSliceLocation {
			if keys[i].sliceLocation != keys[j].sliceLocation {
				return keys[i].sliceLocation < keys[j].sliceLocation
			}
			return keys[i].instanceNumber < keys[j].instanceNumber
		}
		return keys[i].instanceNumber < keys[j].instanceNumber
	})
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
	statusTUI.summary.SetBorder(true).SetTitle("DICOM Meta Data")
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

	statusTUI.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(statusTUI.summary, 30, 1, false).
			AddItem(statusTUI.viewer, 0, 1, true), 0, 1, false).
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
		// the differences between the images of the series get the '*'
		statusTUI.summaryLines = nil
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
		statusTUI.viewer.SetTitle(fmt.Sprintf("DICOM image %d/%d", statusTUI.currentImage+1, len(statusTUI.selectedDatasets)))
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
	if k == tcell.KeyRune && event.Rune() == rune('c') {
		statusTUI.stopAnimation = !statusTUI.stopAnimation
	}
	return event
}

func (statusTUI *StatusTUI) Run() {
	statusTUI.stopAnimation = false
	// start a timer to display an image, should be like very 500msec
	go doEvery(200*time.Millisecond, statusTUI, nextImage)

	statusTUI.app = tview.NewApplication()

	statusTUI.app.SetInputCapture(statusTUI.captureInput)

	if err := statusTUI.app.SetRoot(statusTUI.flex, true).SetFocus(statusTUI.selection).EnableMouse(true).Run(); err != nil {
		fmt.Println("Error: The --tui mode is only available in a propper terminal.")
		panic(err)
	}
	defer statusTUI.app.Stop()
}

func (statusTUI StatusTUI) Stop() {
	statusTUI.app.Stop()
}
