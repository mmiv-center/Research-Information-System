package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
}

func findSeriesInfo(dataSets map[string]map[string]SeriesInfo, SeriesInstanceUID string) (SeriesInfo, error) {
	for _, series := range dataSets {
		if _, ok := series[SeriesInstanceUID]; ok {
			return series[SeriesInstanceUID], nil
		}
	}
	return SeriesInfo{}, fmt.Errorf("SeriesInstanceUID %s not found", SeriesInstanceUID)
}

func addDataset(statusTUI *StatusTUI, dataset dicom.Dataset) {
	if len((*statusTUI).selectedDatasets) == 0 {
		// this is the first time we add a dataset, show the
		// meta-data if we have an app
		if (*statusTUI).app != nil {
			var sAllInfo string
			removeBraces := regexp.MustCompile("(^{)|(}$)")
			for _, a := range (*statusTUI).selectedSeriesInformation.All {
				sAllInfo += " " + removeBraces.ReplaceAllString(fmt.Sprintf("%v", a), "") + "\n"
			}
			//fmt.Printf("IN FIRST DATASET! %s", sAllInfo)

			(*statusTUI).summary.Clear()
			fmt.Fprintf((*statusTUI).summary, "%s\n%s\n\n%s", (*statusTUI).selectedSeriesInformation.SeriesDescription, strings.Join((*statusTUI).selectedSeriesInformation.ClassifyTypes, ","),
				sAllInfo)
		}
	}

	(*statusTUI).selectedDatasets = append((*statusTUI).selectedDatasets, dataset)
}

func (statusTUI *StatusTUI) Init() {
	if statusTUI.dataSets == nil || len(statusTUI.dataSets) == 0 {
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
	statusTUI.summary = newPrimitive("")
	statusTUI.summary.SetBorder(true).SetTitle("Current selection")
	statusTUI.viewer = newPrimitive("").SetDynamicColors(true)
	statusTUI.selection = tview.NewTreeView()
	statusTUI.selection.SetBorder(true)
	statusTUI.selection.SetTitle("Selections")
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
	selected, names := findMatchingSets(statusTUI.ast, statusTUI.dataSets)
	root := tview.NewTreeNode("Selections").SetReference("")
	statusTUI.selection.SetRoot(root).SetCurrentNode(root)

	for idx, entry := range selected {
		firstSeries, err := findSeriesInfo(statusTUI.dataSets, entry[0])
		if err != nil {
			continue
		}
		node := tview.NewTreeNode(fmt.Sprintf("%d/%d %s-%s [yellow]%s", idx+1, len(selected), firstSeries.PatientID, firstSeries.PatientName, firstSeries.StudyDescription)).
			SetReference(entry).
			SetSelectable(false)
		root.AddChild(node)
		for idx2, entry2 := range entry {
			s := "s"
			if firstSeries.NumImages == 1 {
				s = ""
			}
			node2 := tview.NewTreeNode(fmt.Sprintf("%s series %d \"%s\" [gray]%s[-] %d image%s", names[idx][idx2], firstSeries.SeriesNumber, firstSeries.SeriesDescription, entry2, firstSeries.NumImages, s)).
				SetReference(entry2).
				SetSelectable(true)
			node.AddChild(node2)
		}
	}

	statusTUI.selection.SetSelectedFunc(func(node *tview.TreeNode) {
		SeriesInstanceUID := node.GetReference().(string)
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

func showImage(statusTUI *StatusTUI, idx int) {
	if idx >= len(statusTUI.selectedDatasets) {
		idx = len(statusTUI.selectedDatasets) - 1
	}
	if idx < 0 {
		idx = 0
	}
	statusTUI.currentImage = idx
	showDataset(statusTUI.selectedDatasets[idx], 1, "path", "", statusTUI.viewer, statusTUI.config.Viewer.Clip)
	if statusTUI.app != nil {
		statusTUI.app.Draw()
		statusTUI.viewer.SetTitle(fmt.Sprintf("DICOM image %d/%d", statusTUI.currentImage+1, len(statusTUI.selectedDatasets)))
	}

}

//nextImage displays one image from the currently selected image series in the viewer
func nextImage(statusTUI *StatusTUI, t time.Time) {
	if statusTUI.stopAnimation {
		return
	}
	//fmt.Printf("do something %p\n", &statusTUI.selectedDatasets)
	if len(statusTUI.selectedDatasets) == 0 {
		return
	}

	idx := ((statusTUI.currentImage) + 1) % len(statusTUI.selectedDatasets)
	showImage(statusTUI, idx)
}

func (statusTUI *StatusTUI) Run() {
	statusTUI.stopAnimation = false
	// start a timer to display an image, should be like very 500msec
	go doEvery(200*time.Millisecond, statusTUI, nextImage)

	statusTUI.app = tview.NewApplication()

	statusTUI.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		k := event.Key()
		prim := statusTUI.app.GetFocus()
		ch := rune(0)
		//if statusTUI.app.GetFocus() == statusTUI.viewer {
		if statusTUI.stopAnimation && prim == statusTUI.viewer {
			if k == tcell.KeyDown {
				// what window is active?
				//fmt.Println("Down KEY")
				statusTUI.currentImage++
				// this seems to crash the program
				//showImage(statusTUI, statusTUI.currentImage)

			} else if k == tcell.KeyUp {
				//fmt.Println("Up KEY")
				statusTUI.currentImage--
				//showImage(statusTUI, statusTUI.currentImage)
			}
		}
		//}
		if k == tcell.KeyRune {
			ch = event.Rune()
			if ch == rune('c') {
				statusTUI.stopAnimation = !statusTUI.stopAnimation
			}
		}
		return event
	})

	if err := statusTUI.app.SetRoot(statusTUI.flex, true).SetFocus(statusTUI.selection).EnableMouse(true).Run(); err != nil {
		fmt.Println("Error: The --tui mode is only available in a propper terminal.")
		panic(err)
	}
	defer statusTUI.app.Stop()
}

func (statusTUI StatusTUI) Stop() {
	statusTUI.app.Stop()
}
