package main

import (
	_ "embed"

	"github.com/suyashkumar/dicom"
)

//go:embed templates/classifyRules.json
var classifyRules string

type Rules struct {
	Path     string
	DataInfo map[string]map[string]SeriesInfo
}

func ClassifyDICOM(dataset dicom.Dataset) []string {

	// parse the classifyRules using its structure

	return []string{}
}
