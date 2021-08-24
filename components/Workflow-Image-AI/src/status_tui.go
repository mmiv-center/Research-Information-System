package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	ast       AST
}

func findSeriesInfo(dataSets map[string]map[string]SeriesInfo, SeriesInstanceUID string) (SeriesInfo, error) {
	for _, series := range dataSets {
		if _, ok := series[SeriesInstanceUID]; ok {
			return series[SeriesInstanceUID], nil
		}
	}
	return SeriesInfo{}, fmt.Errorf("SeriesInstanceUID %s not found", SeriesInstanceUID)
}

func (statusTUI StatusTUI) Init() {
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
	statusTUI.summary.SetBorder(true).SetTitle("Database")
	statusTUI.viewer = newPrimitive("")
	statusTUI.selection = tview.NewTreeView()
	statusTUI.selection.SetBorder(true)
	statusTUI.selection.SetTitle("Selections")
	statusTUI.viewer.SetBorder(true).SetTitle("DICOM")

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
			node2 := tview.NewTreeNode(fmt.Sprintf("%s series %d %s %d image%s", names[idx][idx2], firstSeries.SeriesNumber, entry2, firstSeries.NumImages, s)).
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
		searchPath := series.Path
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			fmt.Println("warning: this search path could not be found.. we give up here")
			return
		}
		SelectedSeriesInstanceUID := SeriesInstanceUID
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

					showDataset(dataset, 1, path, "", statusTUI.viewer)
					if statusTUI.app != nil {
						statusTUI.app.Draw()
					}
					return errors.New("found an image, stop the walk")
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

func (statusTUI StatusTUI) Run() {
	statusTUI.app = tview.NewApplication()
	if err := statusTUI.app.SetRoot(statusTUI.flex, true).SetFocus(statusTUI.selection).EnableMouse(true).Run(); err != nil {
		fmt.Println("Error: The --tui mode is only available in a propper terminal.")
		panic(err)
	}
	defer statusTUI.app.Stop()
}

func (statusTUI StatusTUI) Stop() {
	statusTUI.app.Stop()
}
