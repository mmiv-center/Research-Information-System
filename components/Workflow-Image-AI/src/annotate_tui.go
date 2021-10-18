package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

type AnnotateTUI struct {
	dataSets                  map[string]map[string]SeriesInfo
	viewer                    *tview.TextView
	summary                   *tview.TreeView
	selection                 *tview.TreeView
	example1                  *tview.TextView
	example2                  *tview.TextView
	example3                  *tview.TextView
	example4                  *tview.TextView
	app                       *tview.Application
	flex                      *tview.Flex
	ast                       AST
	selectedDatasets          []dicom.Dataset
	currentImage              int
	selectedSeriesInformation SeriesInfo
	selectedSeriesInstanceUID string
	ontology                  interface{}
	annotations               map[string][]string
	config                    Config
}

/*func findSeriesInfo(dataSets map[string]map[string]SeriesInfo, SeriesInstanceUID string) (SeriesInfo, error) {
	for _, series := range dataSets {
		if _, ok := series[SeriesInstanceUID]; ok {
			return series[SeriesInstanceUID], nil
		}
	}
	return SeriesInfo{}, fmt.Errorf("SeriesInstanceUID %s not found", SeriesInstanceUID)
}*/

func (annotateTUI *AnnotateTUI) addDataset(dataset dicom.Dataset) {
	(*annotateTUI).selectedDatasets = append((*annotateTUI).selectedDatasets, dataset)
}

func (annotateTUI *AnnotateTUI) Init() {
	if annotateTUI.dataSets == nil || len(annotateTUI.dataSets) == 0 {
		fmt.Println("Warning: there are no datasets to visualize")
	}
	if len(annotateTUI.ast.Rules) == 0 {
		fmt.Println("Warning: there is no ast defined")
	}
	newPrimitive := func(text string) *tview.TextView {
		return tview.NewTextView().
			SetTextAlign(tview.AlignLeft).
			SetText(text)
	}
	annotateTUI.summary = tview.NewTreeView()
	annotateTUI.summary.SetBorder(true).SetTitle("Ontology")
	annotateTUI.viewer = newPrimitive("").SetDynamicColors(true)
	annotateTUI.selection = tview.NewTreeView()
	annotateTUI.selection.SetBorder(true)
	annotateTUI.selection.SetTitle("Selections")
	annotateTUI.viewer.SetBorder(true).SetTitle("DICOM")
	annotateTUI.example1 = newPrimitive("example 1").SetDynamicColors(true)
	annotateTUI.example1.SetBorder(true).SetTitle("DICOM")
	annotateTUI.example2 = newPrimitive("example 2").SetDynamicColors(true)
	annotateTUI.example2.SetBorder(true).SetTitle("DICOM")
	annotateTUI.example3 = newPrimitive("example 3").SetDynamicColors(true)
	annotateTUI.example3.SetBorder(true).SetTitle("DICOM")
	annotateTUI.example4 = newPrimitive("example 4").SetDynamicColors(true)
	annotateTUI.example4.SetBorder(true).SetTitle("DICOM")

	annotateTUI.annotations = make(map[string][]string)

	path_config := input_dir + "/.ror/config"
	conf, err := readConfig(path_config)
	if err == nil {
		var col tcell.Color
		// we set a text color only if the value is set (not equal to empty string)
		if conf.Viewer.TextColor != "" {
			col = tcell.GetColor(conf.Viewer.TextColor)
			annotateTUI.viewer.SetTextColor(col)
		}
	}
	annotateTUI.config = conf

	annotateTUI.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(annotateTUI.summary, 30, 1, false).
			AddItem(annotateTUI.viewer, 0, 1, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(annotateTUI.example1, 0, 30, false).
				AddItem(annotateTUI.example2, 0, 30, false).
				AddItem(annotateTUI.example3, 0, 30, false).
				AddItem(annotateTUI.example4, 0, 30, false), 30, 1, false), 0, 1, false).
		AddItem(annotateTUI.selection, 12, 1, false)

	// start with setting up the list of selected datasets
	selected, names := findMatchingSets(annotateTUI.ast, annotateTUI.dataSets)
	root := tview.NewTreeNode("Selections").SetReference("")
	annotateTUI.selection.SetRoot(root).SetCurrentNode(root)

	for idx, entry := range selected {
		firstSeries, err := findSeriesInfo(annotateTUI.dataSets, entry[0])
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
			node2 := tview.NewTreeNode(fmt.Sprintf("%s %s series %d %s %d image%s", names[idx][idx2], firstSeries.Modality, firstSeries.SeriesNumber, entry2, firstSeries.NumImages, s)).
				SetReference(entry2).
				SetSelectable(true)
			node.AddChild(node2)
		}
	}

	annotateTUI.selection.SetSelectedFunc(func(node *tview.TreeNode) {
		SeriesInstanceUID := node.GetReference().(string)
		if len(SeriesInstanceUID) == 0 {
			return
		}
		// the reference is the series instance uid, get a picture for that series
		series, err := findSeriesInfo(annotateTUI.dataSets, SeriesInstanceUID)
		if err != nil {
			fmt.Println("we got an error", err)
			return
		}
		// remember the series information
		annotateTUI.selectedSeriesInformation = series
		searchPath := series.Path
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			fmt.Println("warning: this search path could not be found.. we give up here")
			return
		}
		annotateTUI.selectedSeriesInstanceUID = SeriesInstanceUID
		annotateTUI.selectedDatasets = nil
		annotateTUI.currentImage = 0
		updateMarkers(annotateTUI)
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
					if SeriesInstanceUID != annotateTUI.selectedSeriesInstanceUID {
						return nil // ignore that file
					}
					_, err := dataset.FindElementByTag(tag.PixelData)
					if err != nil {
						return nil // ignore files that have no images
					}
					annotateTUI.addDataset(dataset)
					//fmt.Printf("add dataset to address: %p\n", statusTUI.selectedDatasets)
					//statusTUI.selectedDatasets = append(statusTUI.selectedDatasets, dataset)

					/* showDataset(dataset, 1, path, "", statusTUI.viewer)
					if statusTUI.app != nil {
						statusTUI.app.Draw()
					} */
					/* annotateTUI.summary.Clear()
					fmt.Fprintf(annotateTUI.summary, "images found: %d\n", len(annotateTUI.selectedDatasets)) */
					// return errors.New("found an image, stop the walk")
					// we have at least one image, so we can display the next one now

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

	var annotations []string
	if annotateTUI.ontology != nil {
		annotations = getAnnotations(annotateTUI, annotateTUI.ontology)
		printAnnotations(annotations, annotateTUI, annotateTUI.summary)
	}

	annotateTUI.Run(annotations)
}

func printAnnotations(annotations []string, annotateTUI *AnnotateTUI, viewer *tview.TreeView) {
	var an []string
	an = append(an, annotations...)
	sort.Strings(an)
	// fmt.Fprintf(viewer, "Ontology\n")

	root := tview.NewTreeNode("Ontology").SetReference("")
	annotateTUI.summary.SetRoot(root)

	for key := 1; key < len(an); key++ {
		//fmt.Fprintf(viewer, "%d: %s\n", key, an[key])
		// TODO: we should check if the annotation contains a mark for this node already

		node := tview.NewTreeNode(fmt.Sprintf("[ ] %s", an[key])).
			SetReference(an[key]).
			SetSelectable(true)
		root.AddChild(node)
	}

	annotateTUI.summary.SetSelectedFunc(func(node *tview.TreeNode) {
		marker := node.GetReference().(string)
		if annotateTUI.selectedSeriesInstanceUID != "" {
			annotateTUI.markImage(annotateTUI.selectedSeriesInformation, annotateTUI.selectedSeriesInstanceUID, marker)
			// update the markers to indicate the new state
			updateMarkers(annotateTUI)
		}
	})
}

func updateMarkers(annotateTUI *AnnotateTUI) {
	//  disable all markers first
	for _, node := range annotateTUI.summary.GetRoot().GetChildren() {
		node.SetText(fmt.Sprintf("[ [] %s", node.GetReference()))
	}

	for marker, seriesInstanceUIDs := range annotateTUI.annotations {
		for _, seriesInstanceUID := range seriesInstanceUIDs {
			if seriesInstanceUID == annotateTUI.selectedSeriesInstanceUID {
				// for this marker and the current series there should be an [x]
				for _, node := range annotateTUI.summary.GetRoot().GetChildren() {
					if node.GetReference() == marker {
						node.SetText(fmt.Sprintf("[x[] %s", marker))
					}
				}
			}
		}
	}
}

func getAnnotations(annotateTUI *AnnotateTUI, ontology interface{}) []string {
	var annotations []string
	for key := range ontology.(map[string]interface{}) {
		//fmt.Fprintf(annotateTUI.example1, "found a key %s and value %s\n", key, entry)
		annotations = append(annotations, key)
	}
	return annotations
}

func doEveryAnnotate(d time.Duration, annotateTUI *AnnotateTUI, f func(*AnnotateTUI, time.Time)) {
	for x := range time.Tick(d) {
		f(annotateTUI, x)
	}
}

//nextImage displays one image from the currently selected image series in the viewer
func nextImageAnnotate(annotateTUI *AnnotateTUI, t time.Time) {
	//fmt.Printf("do something %p\n", &statusTUI.selectedDatasets)
	if len(annotateTUI.selectedDatasets) == 0 {
		return
	}

	idx := (annotateTUI.currentImage + 1) % len(annotateTUI.selectedDatasets)
	if idx >= len(annotateTUI.selectedDatasets) {
		idx = len(annotateTUI.selectedDatasets) - 1
	}
	if idx < 0 {
		idx = 0
	}
	annotateTUI.currentImage = idx
	showDataset(annotateTUI.selectedDatasets[idx], 1, "path", "", annotateTUI.viewer, annotateTUI.config.Viewer.Clip)
	if annotateTUI.app != nil {
		annotateTUI.app.Draw()
	}
	var sAllInfo string
	for _, a := range annotateTUI.selectedSeriesInformation.All {
		sAllInfo += fmt.Sprintf(" %v\n", a)
	}

	/*annotateTUI.summary.Clear()
	fmt.Fprintf(annotateTUI.summary, "image %d/%d\n%s\n%s\n\n%s", annotateTUI.currentImage+1, len(annotateTUI.selectedDatasets),
		annotateTUI.selectedSeriesInformation.SeriesDescription, strings.Join(annotateTUI.selectedSeriesInformation.ClassifyTypes, ","),
		sAllInfo)
	annotateTUI.summary.ScrollToBeginning() */
}

// markImage keeps track of all series that are marked by a given tag a
func (annotateTUI *AnnotateTUI) markImage(data SeriesInfo, selectedSeriesInstanceUID string, a string) {
	//fmt.Printf("marking this image series with %s", a)
	//annotateTUI.summary.SetTitle(a)
	// does that entry exist already
	for s, n := range annotateTUI.annotations[a] {
		if selectedSeriesInstanceUID == n {
			// remove the entry instead of adding it (toggle the mark)
			annotateTUI.annotations[a] = append(annotateTUI.annotations[a][:s], annotateTUI.annotations[a][s+1:]...)
			return
		}
	}
	if len(annotateTUI.annotations[a]) > 0 {
		var idx1 = rand.Intn((len(annotateTUI.annotations[a]) - 0) + 0)
		var idx2 = rand.Intn((len(annotateTUI.annotations[a]) - 0) + 0)
		var idx3 = rand.Intn((len(annotateTUI.annotations[a]) - 0) + 0)
		var idx4 = rand.Intn((len(annotateTUI.annotations[a]) - 0) + 0)
		var example1 string = annotateTUI.annotations[a][ idx1 ]
		annotateTUI.displayOneImage(example1, annotateTUI.example1)
		var example2 string = annotateTUI.annotations[a][ idx2 ]
		annotateTUI.displayOneImage(example2, annotateTUI.example2)
		var example3 string = annotateTUI.annotations[a][ idx3 ]
		annotateTUI.displayOneImage(example3, annotateTUI.example3)
		var example4 string = annotateTUI.annotations[a][ idx4 ]
		annotateTUI.displayOneImage(example4, annotateTUI.example4)
		//annotateTUI.example1.SetTitle(fmt.Sprintf("%d/%d", idx1, len(annotateTUI.annotations[a])))
		//annotateTUI.example2.SetTitle(fmt.Sprintf("%d", idx2))
		//annotateTUI.example3.SetTitle(fmt.Sprintf("%d", idx3))
		//annotateTUI.example4.SetTitle(fmt.Sprintf("%d", idx4))
	}
	annotateTUI.annotations[a] = append(annotateTUI.annotations[a], selectedSeriesInstanceUID)
	// if we have added a marker we can also show other images with the same marker
}

func (annotateTUI *AnnotateTUI) displayOneImage(SeriesInstanceUID string, viewer *tview.TextView) {
	// find that image, load as DICOM and display
	series, err := findSeriesInfo(annotateTUI.dataSets, SeriesInstanceUID)
	if err != nil {
		fmt.Println("we got an error", err)
		return
	}
	searchPath := series.Path
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		fmt.Println("warning: search path could not be found. Maybe a drive was disconnected?")
		return
	}
	var SelectedSeriesInstanceUID string = SeriesInstanceUID
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
				// now display that image and stop doing more
				showDataset(dataset, 1, "path", "", viewer, annotateTUI.config.Viewer.Clip)
				return errors.New("stop")
			}
		}
		return nil
	})
}

func (annotateTUI *AnnotateTUI) Run(annotations []string) {
	// start a timer to display an image, should be like very 500msec
	go doEveryAnnotate(200*time.Millisecond, annotateTUI, nextImageAnnotate)

	annotateTUI.app = tview.NewApplication()

	// add a handler for the keyboard events to assign
	// we like to limit the list of ontological terms to the once that start with the printed character
	annotateTUI.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		k := event.Key()
		ch := rune(0)
		if k == tcell.KeyRune {
			ch = event.Rune()
			root := annotateTUI.summary.GetRoot()
			for _, c := range root.GetChildren() {
				aa := strings.ToLower(c.GetReference().(string))
				if rune(aa[0]) == ch {
					annotateTUI.summary.SetCurrentNode(c)
					return nil
				}
			}
			/*
				for k, a := range annotations {
					if ch == rune(fmt.Sprintf("%d", k+1)[0]) {
						// we received this button click
						//fmt.Fprintf(annotateTUI.summary, "clicked on %s\n", a)
						annotateTUI.markImage(annotateTUI.selectedSeriesInformation, annotateTUI.selectedSeriesInstanceUID, a)
						return nil4321`
					}
				} */
		}
		return event
	})

	if err := annotateTUI.app.SetRoot(annotateTUI.flex, true).SetFocus(annotateTUI.selection).EnableMouse(true).Run(); err != nil {
		fmt.Println("Error: The --tui mode is only available in a propper terminal.")
		panic(err)
	}
	defer annotateTUI.app.Stop()
}

func (annotateTUI AnnotateTUI) Stop() {
	annotateTUI.app.Stop()
}
