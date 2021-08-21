package main

import "github.com/rivo/tview"

type StatusTUI struct {
	dataSets map[string]map[string]SeriesInfo 
	viewer *tview.TextView
	summary *tview.TextView
	selection *tview.TextView
	app *tview.Application
	flex *tview.Flex
}

func (statusTUI StatusTUI) InitTUI() {
	newPrimitive := func(text string) *tview.TextView {
		return tview.NewTextView().
			SetTextAlign(tview.AlignLeft).
			SetText(text)
	}
	statusTUI.summary = newPrimitive("")
	statusTUI.summary.SetBorder(true).SetTitle("Database")
	statusTUI.viewer = newPrimitive("")
	statusTUI.selection = newPrimitive("")
	statusTUI.selection.SetBorder(true)
	statusTUI.selection.SetTitle("File")
	statusTUI.viewer.SetBorder(true).SetTitle("DICOM")

	statusTUI.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(statusTUI.summary, 30, 1, false).
			AddItem(statusTUI.viewer, 0, 1, true), 0, 1, false).
		AddItem(statusTUI.selection, 6, 1, false)

	go statusTUI.Run()
}

func (statusTUI StatusTUI) Run() {
	statusTUI.app = tview.NewApplication()
	if err := app.SetRoot(statusTUI.flex, true).EnableMouse(false).Run(); err != nil {
		panic(err)
	}
	statusTUI.app.Stop()
}

func (statusTUI StatusTUI) Stop() {
	statusTUI.app.Stop()
}
