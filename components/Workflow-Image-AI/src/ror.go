// Code written 2021 by Hauke Bartsch.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mkmik/argsort"
	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"golang.org/x/image/draw"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"image/color"
	_ "image/jpeg"

	"github.com/rivo/tview"
)

const version string = "0.0.3"

// The string below will be replaced during build time using
// -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S"`"
var compileDate string = ".unknown"

var own_name string = "ror"

// will store the path to the config file
var input_dir string

//go:generate /Users/hauke/go/bin/goyacc -o select_group.go select_group.y

//go:embed templates/README.md
var readme string

//go:embed templates/python/stub.py
var stub_py string

//go:embed templates/notebook/stub.ipynb
var stub_ipynb string

//go:embed templates/bash/stub.sh
var stub_sh string

//go:embed templates/python/requirements.txt
var requirements string

//go:embed templates/python/requirements.yml
var requirements_yml string

//go:embed templates/python/Dockerfile
var dockerfile string

//go:embed templates/bash/Dockerfile_bash
var dockerfile_bash string

//go:embed templates/.dockerignore
var dockerignore string

//go:embed templates/webapp/index.html
var webapp_index string

//go:embed templates/webapp/js/all.js
var webapp_js_all string

//go:embed templates/webapp/js/bootstrap.min.js
var webapp_js_boostrap string

//go:embed templates/webapp/js/colorbrewer.js
var webapp_js_colorbrewer string

//go:embed templates/webapp/js/jquery-3.2.1.min.js
var webapp_js_jquery string

//go:embed templates/webapp/js/popper.min.js
var webapp_js_popper string

//go:embed templates/webapp/css/style.css
var webapp_css_style string

//go:embed templates/webapp/css/bootstrap.min.css
var webapp_css_bootstrap string

//go:embed templates/webapp/Dockerfile_webapp
var webapp_dockerfile string

var structure *tview.TextView
var viewer *tview.TextView
var footer *tview.TextView
var globalHeight int
var globalWidth int

func exitGracefully(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func check(e error) {
	if e != nil {
		exitGracefully(e)
	}
}

type AuthorInfo struct {
	Name, Email string
}

type DataInfo struct {
	Path     string
	DataInfo map[string]map[string]SeriesInfo
}

type Viewer struct {
	TextColor string
}

type Annotate struct {
	Ontology interface{}
}

type Config struct {
	Date             string
	Data             DataInfo
	SeriesFilter     string
	SeriesFilterType string
	Author           AuthorInfo
	TempDirectory    string
	CallString       string
	ProjectName      string
	SortDICOM        bool
	ProjectType      string
	ProjectToken     string
	LastDataFolder   string
	Viewer           Viewer
	Annotate         Annotate
}

type TagAndValue struct {
	Tag   tag.Tag  `json:"tag"`
	Value []string `json:"value"`
}

type SeriesInfo struct {
	SeriesDescription     string
	NumImages             int
	SeriesNumber          int
	SequenceName          string
	Modality              string
	StudyDescription      string
	Manufacturer          string
	ManufacturerModelName string
	Path                  string
	PatientID             string
	PatientName           string
	ClassifyTypes         []string
	All                   []TagAndValue
}

// readConfig parses a provided config file as JSON.
// It returns the parsed code as a marshaled structure.
// @return
func readConfig(path_string string) (Config, error) {
	// todo: check directories up as well
	if _, err := os.Stat(path_string); err != nil && os.IsNotExist(err) {
		return Config{}, fmt.Errorf("file %s does not exist", path_string)
	}
	// we need to check if the config file has the correct permissions,
	// produce a warning if it does not!
	if fileInfo, err := os.Stat(path_string); err == nil {
		mode := fileInfo.Mode()
		mode_str := fmt.Sprintf("%s", mode)
		if mode_str != "-rw-------" && runtime.GOOS != "windows" {
			fmt.Println("Warning: Your config file is not secure. Change the permissions by 'chmod 0600 .ror/config'. Now: ", mode)
		}
	} else {
		fmt.Println(err)
	}

	// var buf bytes.Buffer
	fi, err := os.Open(path_string)
	if err != nil {
		return Config{}, err
	}
	defer fi.Close()

	gzreader, err := gzip.NewReader(fi)
	if err != nil {
		log.Fatal(err)
	}
	byteValue, err := io.ReadAll(gzreader)
	if err != nil {
		log.Fatal(err)
	}

	var config Config

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &config)

	if err := gzreader.Close(); err != nil {
		log.Fatal(err)
	}

	return config, nil
}

//writeConfig writes the provided config to the given path
func (config Config) writeConfig() bool {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	dir_path := input_dir + "/.ror/config"

	// Setting the Header fields is optional.
	zw.Name = dir_path
	zw.Comment = "see github.com/MMIV-Center/Research-Information-System/"
	zw.ModTime = time.Now()

	file, _ := json.MarshalIndent(config, "", " ")
	_, err := zw.Write(file)

	if err != nil {
		log.Fatal(err)
	}

	if err := zw.Close(); err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(dir_path, buf.Bytes(), 0600)
	if err != nil {
		log.Fatal(err)
	}

	return true
}

type Description struct {
	NameFromSelect    string
	SeriesInstanceUID string
	SeriesDescription string
	StudyInstanceUID  string
	NumFiles          int
	Modality          string
	PatientID         string
	PatientName       string
	SequenceName      string
	StudyDate         string
	StudyTime         string
	SeriesTime        string
	SeriesNumber      string
	ProcessDataPath   string
	ClassifyTypes     []string
}

// img.At(x, y).RGBA() returns four uint32 values; we want a Pixel
/*func rgbaToPixel(r uint32, g uint32, b uint32, a uint32) Pixel {
	return Pixel{int(r / 257), int(g / 257), int(b / 257), int(a / 257)}
}*/

// Pixel struct example
type Pixel struct {
	R int
	G int
	B int
	A int
}

var ASCIISTR = "MND8OZ$7I?+=~:,.."

// from http://paulbourke.net/dataformats/asciiart/
var ASCIISTR2 = "$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\\|()1{}[]?-_+~<>i!lI;:,\"^`'."
var ASCIISTR3 = " .:-=+*#%@"
var ASCIISTR4 []rune = []rune{'\u2580', '\u2581', '\u2582', '\u2583', '\u2584', '\u2585', '\u2586', '\u2587', '\u2588'}
var ASCIISTR5 []rune = []rune{'\u25CC', '\u25CB', '\u25CE', '\u25CD', '\u25CF'}

// reverse reverses the argument and returns the result
func reverse(s string) string {
	o := make([]rune, utf8.RuneCountInString(s))
	i := len(o)
	for _, c := range s {
		i--
		o[i] = c
	}
	return string(o)
}

func reverseRunes(s []rune) []rune {
	o := make([]rune, len(s))
	i := len(o)
	for _, c := range s {
		i--
		o[i] = c
	}
	return o
}

// complement2 computes the 2-complement of a number
func complement2(x uint16) int16 {
	return int16(^x) + 1
}

// printImage2ASCII prints the image as ASCII art
func printImage2Runes(img image.Image, PhotometricInterpretation string, PixelPaddingValue int) string {
	//table := []byte(reverse(ASCIISTR))
	//table := []byte(reverse(ASCIISTR2))
	table := reverseRunes(ASCIISTR5)
	if PhotometricInterpretation == "MONOCHROME1" { // only valid if samples per pixel is 1
		table = []rune(ASCIISTR5)
	}
	//table := []byte(ASCIISTR3)
	buf := new(bytes.Buffer)
	w := img.Bounds().Max.X
	h := img.Bounds().Max.Y

	firstSet := false
	var minVal int64
	var maxVal int64
	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			// this might be wrong if we have 8bit data - we interpret them as 16bit here which shifts them up
			g := color.Gray16Model.Convert(img.At(j, i))
			//g := img.At(j, i)
			y := int64(reflect.ValueOf(g).FieldByName("Y").Uint())
			if PixelPaddingValue != 32768 && y == int64(PixelPaddingValue) {
				continue
			}
			if !firstSet {
				maxVal = y
				minVal = maxVal
				firstSet = true
			}
			if y > maxVal {
				maxVal = y
				//fmt.Println(y, g)
			}
			if y < minVal {
				minVal = y
			}
		}
	}
	// todo: better to use a histogram to scale at 2%...99.9% per image
	var histogram [1024]int64
	bins := len(histogram)

	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			g := color.Gray16Model.Convert(img.At(j, i))
			//g := img.At(j, i)
			y := int64(reflect.ValueOf(g).FieldByName("Y").Uint())
			if PixelPaddingValue != 32768 && y == int64(PixelPaddingValue) {
				continue
			}
			//if math.IsInf(float64(y), 0) || math.IsNaN(float64(y)) {
			//	continue
			//}
			idx := int(math.Round((float64(y) - float64(minVal)) / float64(maxVal-minVal) * float64(bins-1)))
			idx = int(math.Min(float64(bins)-1, math.Max(0, float64(idx))))
			if idx != 0 && idx != bins-1 {
				histogram[idx] += 1
			}
		}
	}
	//fmt.Println(histogram)
	// compute the 2%, 99% borders in the cumulative density
	// for now we can remove the lowest and highest intensity value found
	sum := histogram[1]
	for i := 2; i < bins-1; i++ {
		sum += histogram[i]
	}
	var min2 int64 = minVal
	s := histogram[0]
	for i := 1; i < bins; i++ {
		if float32(s) >= (float32(sum) * 10.0 / 100.0) { // sum / 100 = ? / 2
			min2 = minVal + int64(float32(i)/float32(bins)*float32(maxVal-minVal))
			break
		}
		s += histogram[i]
	}
	var max99 int64 = maxVal
	s = histogram[0]
	for i := 1; i < bins; i++ {
		if float32(s) >= (float32(sum) * 95.0 / 100.0) { // sum / 100 = ? / 2
			max99 = minVal + int64(float32(i)/float32(bins)*float32(maxVal-minVal))
			break
		}
		s += histogram[i]
	}
	// fmt.Println("min2:", min2, "max99:", max99, "true min:", minVal, "true max:", maxVal)

	// some pixel are very dark and we need more contast
	//fmt.Println("max ", maxVal, "min", minVal)
	// denom := maxVal - minVal
	denom := max99 - min2
	if denom == 0 {
		denom = 1
	}
	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			g := color.Gray16Model.Convert(img.At(j, i))
			//g := img.At(j, i)
			y := int64(reflect.ValueOf(g).FieldByName("Y").Uint())
			if PixelPaddingValue != 32768 && y == int64(PixelPaddingValue) {
				_, err := buf.WriteRune(' ')
				if err != nil {
					fmt.Println("Error: writing space")
				}
				continue
			}
			//fmt.Println("got a number: ", img.At(j, i))
			pos := int((float32(y) - float32(min2)) * float32(len(table)-1) / float32(denom))
			pos = int(math.Min(float64(len(table)-1), math.Max(0, float64(pos))))
			_, err := buf.WriteRune(table[pos])
			if err != nil {
				fmt.Println("Error: writing rune")
			}
		}
		_ = buf.WriteByte('\n')
	}
	return buf.String()
}

// printImage2ASCII prints the image as ASCII art
func printImage2ASCII(img image.Image, PhotometricInterpretation string, PixelPaddingValue int) []byte {
	//table := []byte(reverse(ASCIISTR))
	table := []byte(reverse(ASCIISTR2))
	//table := ASCIISTR4
	if PhotometricInterpretation == "MONOCHROME1" { // only valid if samples per pixel is 1
		table = []byte(ASCIISTR2)
	}
	//table := []byte(ASCIISTR3)
	buf := new(bytes.Buffer)
	w := img.Bounds().Max.X
	h := img.Bounds().Max.Y

	firstSet := false
	var minVal int64
	var maxVal int64
	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			// this might be wrong if we have 8bit data - we interpret them as 16bit here which shifts them up
			g := color.Gray16Model.Convert(img.At(j, i))
			//g := img.At(j, i)
			y := int64(reflect.ValueOf(g).FieldByName("Y").Uint())
			if PixelPaddingValue != 32768 && y == int64(PixelPaddingValue) {
				continue
			}
			if !firstSet {
				maxVal = y
				minVal = maxVal
				firstSet = true
			}
			if y > maxVal {
				maxVal = y
				//fmt.Println(y, g)
			}
			if y < minVal {
				minVal = y
			}
		}
	}
	// todo: better to use a histogram to scale at 2%...99.9% per image
	var histogram [1024]int64
	bins := len(histogram)

	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			g := color.Gray16Model.Convert(img.At(j, i))
			//g := img.At(j, i)
			y := int64(reflect.ValueOf(g).FieldByName("Y").Uint())
			if PixelPaddingValue != 32768 && y == int64(PixelPaddingValue) {
				continue
			}
			//if math.IsInf(float64(y), 0) || math.IsNaN(float64(y)) {
			//	continue
			//}
			idx := int(math.Round((float64(y) - float64(minVal)) / float64(maxVal-minVal) * float64(bins-1)))
			idx = int(math.Min(float64(bins)-1, math.Max(0, float64(idx))))
			if idx != 0 && idx != bins-1 {
				histogram[idx] += 1
			}
		}
	}
	//fmt.Println(histogram)
	// compute the 2%, 99% borders in the cumulative density
	// for now we can remove the lowest and highest intensity value found
	sum := histogram[1]
	for i := 2; i < bins-1; i++ {
		sum += histogram[i]
	}
	var min2 int64 = minVal
	s := histogram[0]
	for i := 1; i < bins; i++ {
		if float32(s) >= (float32(sum) * 10.0 / 100.0) { // sum / 100 = ? / 2
			min2 = minVal + int64(float32(i)/float32(bins)*float32(maxVal-minVal))
			break
		}
		s += histogram[i]
	}
	var max99 int64 = maxVal
	s = histogram[0]
	for i := 1; i < bins; i++ {
		if float32(s) >= (float32(sum) * 95.0 / 100.0) { // sum / 100 = ? / 2
			max99 = minVal + int64(float32(i)/float32(bins)*float32(maxVal-minVal))
			break
		}
		s += histogram[i]
	}
	// fmt.Println("min2:", min2, "max99:", max99, "true min:", minVal, "true max:", maxVal)

	// some pixel are very dark and we need more contast
	//fmt.Println("max ", maxVal, "min", minVal)
	// denom := maxVal - minVal
	denom := max99 - min2
	if denom == 0 {
		denom = 1
	}
	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			g := color.Gray16Model.Convert(img.At(j, i))
			//g := img.At(j, i)
			y := int64(reflect.ValueOf(g).FieldByName("Y").Uint())
			if PixelPaddingValue != 32768 && y == int64(PixelPaddingValue) {
				_ = buf.WriteByte(' ')
				continue
			}
			//fmt.Println("got a number: ", img.At(j, i))
			pos := int((float32(y) - float32(min2)) * float32(len(table)-1) / float32(denom))
			pos = int(math.Min(float64(len(table)-1), math.Max(0, float64(pos))))
			_ = buf.WriteByte(table[pos])
		}
		_ = buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// Scale uses a different package for rescaling the image
func Scale(src image.Image, rect image.Rectangle, scale draw.Scaler) image.Image {
	dst := image.NewRGBA(rect)
	scale.Scale(dst, rect, src, src.Bounds(), draw.Over, nil)
	return dst
}

// showDataset is a helper function to display the dataset
func showDataset(dataset dicom.Dataset, counter int, path string, info string, viewer *tview.TextView) (int, int) {
	pixelDataElement, err := dataset.FindElementByTag(tag.PixelData)
	if err != nil {
		return 0, 0
	}
	var PixelRepresentation int = 0
	PixelRepresentationVal, err := dataset.FindElementByTag(tag.PixelRepresentation)
	if err == nil {
		PixelRepresentation = dicom.MustGetInts(PixelRepresentationVal.Value)[0]
	}
	var PhotometricInterpretation string = "MONOCHROME2"
	PhotometricInterpretationVal, err := dataset.FindElementByTag(tag.PhotometricInterpretation)
	if err == nil {
		PhotometricInterpretation = dicom.MustGetStrings(PhotometricInterpretationVal.Value)[0]
	}
	// This value seems to be defined in the original data format (before complement-2)
	var PixelPaddingValue int = 0
	PixelPaddingValueVal, err := dataset.FindElementByTag(tag.PixelPaddingValue)
	if err == nil {
		PixelPaddingValue = dicom.MustGetInts(PixelPaddingValueVal.Value)[0]
	}

	//langFmt := message.NewPrinter(language.English)

	pixelDataInfo := dicom.MustGetPixelDataInfo(pixelDataElement.Value)
	for _, fr := range pixelDataInfo.Frames {
		fmt.Printf("\033[0;0f") // go to top of the screen

		// we can try to convert the image here based on the pixel representation
		var img image.Image
		var convertHere bool = true
		if convertHere && PixelRepresentation == 1 {
			native_img, _ := fr.GetNativeFrame()
			if PixelPaddingValue != 0 { // this is for modality CT
				// if we have such a value we cannot assume it will actually work,
				// GE is an example where they used other values
				currValue := uint16(native_img.Data[0][0])
				currValue2 := complement2(currValue)
				PixelPaddingValue = int(32768) + int(currValue2)
			} else {
				PixelPaddingValue += int(32768)
			}
			for i := 0; i < native_img.Rows; i++ {
				for j := 0; j < native_img.Cols; j++ {
					currValue := uint16(native_img.Data[i*native_img.Cols+j][0])
					currValue2 := complement2(currValue)
					// the GetImage function will convert everything to uint16 later
					// so any values we might have here that are negative will be gone
					// lets shift into the positive range here (dah)
					native_img.Data[i*native_img.Cols+j][0] = 32768 + int(currValue2)
				}
			}
			img, err = native_img.GetImage()
			if err != nil {
				fmt.Println(err)
			}
		} else {
			img, _ = fr.GetImage() // The Go image.Image for this frame
		}

		//twidth = 196.0/2.0
		/*if app != nil {
			app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
				w, h := screen.Size()
				globalHeight = h
				globalWidth = w
				return false
			})
			// this does not work, we need access to the screen to be able to call Size()
			// _, _, _, theight = viewer.GetInnerRect()
		} else {
			globalWidth = 192 / 2
		} */
		if viewer != nil {
			_, _, globalWidth, globalHeight = viewer.GetInnerRect()
		} else {
			globalWidth = 192 / 2
			globalHeight = 192 / 2
		}
		twidth := globalWidth
		theight := int(math.Round(float64(globalWidth) / (80.0 / 30.0)))

		origbounds := img.Bounds()
		orig_width, orig_height := origbounds.Max.X, origbounds.Max.Y
		newImage := image.NewGray16(image.Rect(0, 0, twidth, theight))

		draw.ApproxBiLinear.Scale(newImage, image.Rect(0, 0, twidth, theight), img, origbounds, draw.Over, nil)

		//bounds := newImage.Bounds()
		// width, height := bounds.Max.X, bounds.Max.Y
		//p := printImage2ASCII(newImage, PhotometricInterpretation, PixelPaddingValue)
		p := printImage2Runes(newImage, PhotometricInterpretation, PixelPaddingValue)

		//fmt.Printf("%s", string(p))
		if viewer != nil {
			viewer.Clear()
			//app.SetFocus(viewer)
			//footer.Clear()
			//structure.Clear()
			fmt.Fprintf(viewer, "%s", string(p))
			return orig_width, orig_height
			// langFmt.Printf("\033[2K[%d] %s (%dx%d)\n", counter+1, path, orig_width, orig_height)
			//  fmt.Fprintf(footer, langFmt.Sprintf("\033[2K[%d] %s (%dx%d)\n", counter+1, path, orig_width, orig_height))
			//fmt.Fprintf(footer, langFmt.Sprintf("[%d] %s (%dx%d)\n", counter+1, path, orig_width, orig_height))
			//if len(info) > 0 {
			//	//fmt.Fprintf(structure, langFmt.Sprintf("\033[2K%s\n%d", info, theight))
			//	fmt.Fprintf(structure, langFmt.Sprintf("%s", info))
			//}
			//app.Draw()
		} else {
			fmt.Printf("%s", string(p))
			//langFmt.Printf("\033[2K[%d] %s (%dx%d)\n", counter+1, path, orig_width, orig_height)
			return orig_width, orig_height
		}
	}
	return 0, 0
}

// copyFiles will copy all DICOM files that fit the string to the dest_path directory.
// we could display those images as well on the command line - just to impress
func copyFiles(SelectedSeriesInstanceUID string, source_path string, dest_path string, sort_dicom bool, classifyTypes []string) (int, Description) {

	destination_path := dest_path + "/input"

	if _, err := os.Stat(destination_path); os.IsNotExist(err) {
		err := os.Mkdir(destination_path, 0755)
		if err != nil {
			exitGracefully(errors.New("could not create data directory"))
		}
	}
	var description Description
	description.SeriesInstanceUID = SelectedSeriesInstanceUID
	description.ProcessDataPath = dest_path
	description.ClassifyTypes = classifyTypes
	counter := 0
	fmt.Printf("\033[2J\n") // clear the screen

	var input_path_list []string
	if _, err := os.Stat(source_path); err != nil && os.IsNotExist(err) {
		// could be list of paths if we have a glob string
		input_path_list, err = filepath.Glob(source_path)
		if err != nil || len(input_path_list) < 1 {
			exitGracefully(errors.New("data path does not exist or is empty"))
		}
	} else {
		input_path_list = append(input_path_list, source_path)
	}

	for p := range input_path_list {
		err := filepath.Walk(input_path_list[p], func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if err != nil {
				return err
			}
			//fmt.Println("look at file: ", path)
			//fmt.Printf("\033[2J\n")

			dataset, err := dicom.ParseFile(path, nil) // See also: dicom.Parse which has a generic io.Reader API.
			if err == nil {
				SeriesInstanceUIDVal, err := dataset.FindElementByTag(tag.SeriesInstanceUID)
				if err == nil {
					var SeriesInstanceUID string = dicom.MustGetStrings(SeriesInstanceUIDVal.Value)[0]
					if SeriesInstanceUID != SelectedSeriesInstanceUID {
						return nil // ignore that file
					}

					// we can get a version of the image, scale it and print out on the command line
					// for a trigger call this has to work without the tui interface
					showImage := true
					if showImage {
						if app != nil {
							footer.Clear()
							structure.Clear()
						}
						info := ""
						langFmt := message.NewPrinter(language.English)
						if app == nil {
							viewer = nil
						}
						orig_width, orig_height := showDataset(dataset, counter+1, path, info, viewer)
						if app != nil {
							fmt.Fprintf(footer, langFmt.Sprintf("[%d] %s (%dx%d)\n", counter+1, path, orig_width, orig_height))
						} else {
							fmt.Println(langFmt.Sprintf("[%d] %s (%dx%d)\n", counter+1, path, orig_width, orig_height))
						}
						if len(info) > 0 {
							//fmt.Fprintf(structure, langFmt.Sprintf("\033[2K%s\n%d", info, theight))
							if app != nil {
								fmt.Fprintf(structure, langFmt.Sprintf("%s", info))
							} else {
								fmt.Printf(langFmt.Sprintf("%s", info))
							}
						}
						if app != nil {
							app.Draw()
						}
					}

					//fmt.Printf("%05d files\r", counter)
					var SeriesDescription string
					SeriesDescriptionVal, err := dataset.FindElementByTag(tag.SeriesDescription)
					if err == nil {
						SeriesDescription = dicom.MustGetStrings(SeriesDescriptionVal.Value)[0]
						if SeriesDescription != "" {
							description.SeriesDescription = SeriesDescription
						}
					}
					var PatientID string
					PatientIDVal, err := dataset.FindElementByTag(tag.PatientID)
					if err == nil {
						PatientID = dicom.MustGetStrings(PatientIDVal.Value)[0]
						if PatientID != "" {
							description.PatientID = PatientID
						}
					}
					var PatientName string
					PatientNameVal, err := dataset.FindElementByTag(tag.PatientName)
					if err == nil {
						PatientName = dicom.MustGetStrings(PatientNameVal.Value)[0]
						if PatientName != "" {
							description.PatientName = PatientName
						}
					}
					var SequenceName string
					SequenceNameVal, err := dataset.FindElementByTag(tag.SequenceName)
					if err == nil {
						SequenceName = dicom.MustGetStrings(SequenceNameVal.Value)[0]
						if SequenceName != "" {
							description.SequenceName = SequenceName
						}
					}
					var StudyDate string
					StudyDateVal, err := dataset.FindElementByTag(tag.StudyDate)
					if err == nil {
						StudyDate = dicom.MustGetStrings(StudyDateVal.Value)[0]
						if StudyDate != "" {
							description.StudyDate = StudyDate
						}
					}
					var StudyTime string
					StudyTimeVal, err := dataset.FindElementByTag(tag.StudyTime)
					if err == nil {
						StudyTime = dicom.MustGetStrings(StudyTimeVal.Value)[0]
						if StudyTime != "" {
							description.StudyTime = StudyTime
						}
					}
					var SeriesTime string
					SeriesTimeVal, err := dataset.FindElementByTag(tag.SeriesTime)
					if err == nil {
						SeriesTime = dicom.MustGetStrings(SeriesTimeVal.Value)[0]
						if SeriesTime != "" {
							description.SeriesTime = SeriesTime
						}
					}
					var SeriesNumber string
					SeriesNumberVal, err := dataset.FindElementByTag(tag.SeriesNumber)
					if err == nil {
						SeriesNumber = dicom.MustGetStrings(SeriesNumberVal.Value)[0]
						if SeriesNumber != "" {
							description.SeriesNumber = SeriesNumber
						}
					}
					var StudyInstanceUID string
					StudyInstanceUIDVal, err := dataset.FindElementByTag(tag.StudyInstanceUID)
					if err == nil {
						StudyInstanceUID = dicom.MustGetStrings(StudyInstanceUIDVal.Value)[0]
						if StudyInstanceUID != "" {
							description.StudyInstanceUID = StudyInstanceUID
						}
					}
					var Modality string
					ModalityVal, err := dataset.FindElementByTag(tag.Modality)
					if err == nil {
						Modality = dicom.MustGetStrings(ModalityVal.Value)[0]
						if Modality != "" {
							description.Modality = Modality
						}
					}

					outputPath := destination_path
					inputFile, _ := os.Open(path)
					data, _ := ioutil.ReadAll(inputFile)
					// what is the next unused filename? We can have this case if other series are exported as well
					outputPathFileName := fmt.Sprintf("%s/%06d.dcm", outputPath, counter)
					_, err = os.Stat(outputPathFileName)
					for !os.IsNotExist(err) {
						counter = counter + 1
						outputPathFileName := fmt.Sprintf("%s/%06d.dcm", outputPath, counter)
						_, err = os.Stat(outputPathFileName)
					}
					ioutil.WriteFile(outputPathFileName, data, 0644)

					// We can do a better destination path here. The friendly way of doing this is
					// to provide separate folders aka the BIDS way.
					// We can create a shadow structure that uses symlinks and sorts everything into
					// sub-folders. Lets create a data view and place the info in that directory.
					symOrder := sort_dicom
					if symOrder {
						symOrderPath := filepath.Join(dest_path, "input_view_dicom_series")
						if _, err := os.Stat(symOrderPath); os.IsNotExist(err) {
							err := os.Mkdir(symOrderPath, 0755)
							if err != nil {
								exitGracefully(errors.New("could not create symlink data directory"))
							}
						}
						symOrderPatientPath := filepath.Join(symOrderPath, PatientID+"_"+PatientName)
						if _, err := os.Stat(symOrderPatientPath); os.IsNotExist(err) {
							err := os.Mkdir(symOrderPatientPath, 0755)
							if err != nil {
								exitGracefully(errors.New("could not create symlink data directory"))
							}
						}
						symOrderPatientDatePath := filepath.Join(symOrderPatientPath, StudyDate+"_"+StudyTime)
						if _, err := os.Stat(symOrderPatientDatePath); os.IsNotExist(err) {
							err := os.Mkdir(symOrderPatientDatePath, 0755)
							if err != nil {
								exitGracefully(errors.New("could not create symlink data directory"))
							}
						}
						symOrderPatientDateSeriesNumber := filepath.Join(symOrderPatientDatePath, SeriesNumber+"_"+SeriesDescription)
						if _, err := os.Stat(symOrderPatientDateSeriesNumber); os.IsNotExist(err) {
							err := os.Mkdir(symOrderPatientDateSeriesNumber, 0755)
							if err != nil {
								exitGracefully(errors.New("could not create symlink data directory"))
							}
						}
						// now create symbolic link here to our outputPath + counter .dcm == outputPathFileName
						// this prevents any duplication of space taken up by the images
						symlink := filepath.Join(symOrderPatientDateSeriesNumber, fmt.Sprintf("%06d.dcm", counter))
						relativeDataPath := fmt.Sprintf("../../../../input/%06d.dcm", counter)
						os.Symlink(relativeDataPath, symlink)
					}

					//fmt.Println("path: ", fmt.Sprintf("%s/%06d.dcm", outputPath, counter))
					counter = counter + 1
				}
			}
			return nil
		})
		if err != nil {
			fmt.Println("Warning: could not walk this path")
		}
	}
	description.NumFiles = counter
	return counter, description
}

// dataSets parses the config.Data path for DICOM files.
// It returns the detected studies and series as collections of paths.
func dataSets(config Config) (map[string]map[string]SeriesInfo, error) {
	var datasets = make(map[string]map[string]SeriesInfo)
	if config.Data.Path == "" {
		return datasets, fmt.Errorf("no data path for example data has been specified. Use\n\tror config --data \"path-to-data\" to set such a directory of DICOM data")
	}
	var input_path_list []string
	if _, err := os.Stat(config.Data.Path); err != nil && os.IsNotExist(err) {
		// could be list of paths if we have a glob string
		input_path_list, err = filepath.Glob(config.Data.Path)
		if err != nil || len(input_path_list) < 1 {
			exitGracefully(errors.New("data path does not exist or is empty"))
		}
	} else {
		input_path_list = append(input_path_list, config.Data.Path)
	}
	//fmt.Println("Found data directory, start parsing DICOM files...")
	counter := 0
	nonDICOM := 0
	langFmt := message.NewPrinter(language.English)
	for p := range input_path_list {
		err := filepath.Walk(input_path_list[p], func(path string, info os.FileInfo, err error) error {
			//fmt.Println(path)
			if info.IsDir() && (info.Name() == ".DS_Store" || info.Name() == "$RECYCLE.BIN") {
				return filepath.SkipDir
			}
			if info.IsDir() {
				return nil
			}
			if err != nil {
				return err
			}
			// every once in a while we should save the datasets - so we can break reading without lossing work
			if counter > 0 && counter%1000 == 0 {
				dir_path := input_dir + "/.ror/config"
				config2, err := readConfig(dir_path)
				if err != nil {
					exitGracefully(errors.New("could not read config file"))
				}
				config2.Data.DataInfo = datasets
				// do we need to copy this - do we need to copy more???
				config2.Data.Path = config.Data.Path

				// write out config again
				config2.writeConfig()
				app.Sync()
				//file, _ := json.MarshalIndent(config2, "", " ")
				//_ = ioutil.WriteFile(dir_path, file, 0600)
			}

			// we could be faster here if we ignore zip files, those are large and we don't want them (?)
			// there is a way to detect a zip file, but we should just check the size first as that might be
			// an indicator as well - might be faster just reading otherwise?
			// https://www.socketloop.com/tutorials/golang-how-to-tell-if-a-file-is-compressed-either-gzip-or-zip
			if filepath.Ext(info.Name()) == ".zip" {
				// ignore compressed files
				nonDICOM = nonDICOM + 1
				return nil
			}

			//fmt.Println("look at file: ", path)
			dataset, err := dicom.ParseFile(path, nil)                    // See also: dicom.Parse which has a generic io.Reader API.
			if err != nil && fmt.Sprintf("%s", err) == "unexpected EOF" { // we should check here if dataset is any good...
				// maybe the dataset is ok and the error isjust an "unexpected EOF" ?
				//fmt.Println("unexpected EOF, still try to read now")

				// this seems to happen if the DICOM file has some tags usch as 0009,10c1 with an undeclared value representation
				// the library still reads it but does not continue aftwards. So the dataset structure stops and there is no
				// StudyInstanceUID. An example for this is:
				// 	 ../SmallAnimalImaging/b/left/00004689.dcm

				err = nil
			}

			if err == nil {
				StudyInstanceUIDVal, err := dataset.FindElementByTag(tag.StudyInstanceUID)
				if err == nil {
					var StudyInstanceUID string = ""
					var SeriesInstanceUID string = ""
					var SeriesDescription string
					var SeriesNumber int
					var SequenceName string
					var StudyDescription string
					var Modality string
					var Manufacturer string
					var ManufacturerModelName string
					var PatientID string
					var PatientName string

					removeElement := func(s []*dicom.Element, i int) []*dicom.Element {
						s[i] = s[len(s)-1]
						return s[:len(s)-1]
					}

					var all_dicom []*dicom.Element = dataset.Elements
					// we should clean out the larger elements based on VR
					for i := 0; i < len(all_dicom); i++ {
						if all_dicom[i].ValueRepresentation == tag.VRUInt16List ||
							all_dicom[i].ValueRepresentation == tag.VRUInt32List ||
							all_dicom[i].ValueRepresentation == tag.VRBytes ||
							all_dicom[i].ValueRepresentation == tag.VRPixelData {
							all_dicom = removeElement(all_dicom, i) // append(all[:i], all[i+1:]...)
							i--
						}
					}
					// now convert for the All secion
					var all []TagAndValue = make([]TagAndValue, len(all_dicom))
					for i := 0; i < len(all_dicom); i++ {
						all[i].Tag.Element = all_dicom[i].Tag.Element
						all[i].Tag.Group = all_dicom[i].Tag.Group

						switch all_dicom[i].Value.ValueType() {
						case dicom.Strings:
							all[i].Value = all_dicom[i].Value.GetValue().([]string)
						case dicom.Ints:
							all[i].Value = []string{}
							for _, v := range all_dicom[i].Value.GetValue().([]int) {
								all[i].Value = append(all[i].Value, fmt.Sprintf("%d", v))
							}
						case dicom.Floats:
							all[i].Value = []string{}
							for _, v := range all_dicom[i].Value.GetValue().([]float64) {
								all[i].Value = append(all[i].Value, fmt.Sprintf("%f", v))
							}
						default:
							// todo: handle sequences here
							//fmt.Printf("Warning: we don't know that type yet %v\n", all_dicom[i].Value.ValueType())
							// ...
						}
					}

					showImages := true
					if showImages {
						// create a human readable summary line for the whole dataset
						numStudies := len(datasets)
						numSeries := 0
						numImages := 0
						var participants map[string]bool = make(map[string]bool)
						var modalities map[string]bool = make(map[string]bool)
						for _, v := range datasets {
							numSeries += len(v)
							for _, vv := range v {
								numImages += vv.NumImages
								modalities[vv.Modality] = true
								participants[fmt.Sprintf("%s%s", vv.PatientID, vv.PatientName)] = true
							}
						}
						numModalities := len(modalities)
						numParticipants := len(participants)
						// this is what we have in here from before, it does not contain the current image...
						s1 := "y"
						if numStudies != 1 {
							s1 = "ies"
						}
						s2 := ""
						if numImages != 1 {
							s2 = "s"
						}
						s3 := ""
						if nonDICOM != 1 {
							s3 = "s"
						}
						s4 := "y"
						if numModalities != 1 {
							s4 = "ies"
						}
						s5 := ""
						if numParticipants != 1 {
							s5 = "s"
						}
						var dataset_info string = langFmt.Sprintf("%d Participant%s\n%d Stud%s\n%d Series\n%d Image%s\n%d Modalit%s, and\n%d Non-DICOM file%s",
							numParticipants, s5, numStudies, s1, numSeries, numImages, s2, numModalities, s4, nonDICOM, s3)
						if app != nil {
							footer.Clear()
							structure.Clear()
							viewer.Clear()
							orig_width, orig_height := showDataset(dataset, counter, path, dataset_info, viewer)
							fmt.Fprintf(structure, langFmt.Sprintf("%s", dataset_info))
							fmt.Fprintf(footer, langFmt.Sprintf("[%d] %s (%dx%d)\n", counter+1, path, orig_width, orig_height))
							app.Draw()
						} else {
							orig_width, orig_height := showDataset(dataset, counter, path, dataset_info, nil)
							fmt.Printf(langFmt.Sprintf("[%d] %s (%dx%d)\n", counter+1, path, orig_width, orig_height))
						}
					} else {
						fmt.Printf("%05d files\r", counter)
					}

					counter = counter + 1
					StudyInstanceUID = dicom.MustGetStrings(StudyInstanceUIDVal.Value)[0]
					if StudyInstanceUID == "" {
						// no study instance uid found, skip this series because we cannot reference it later
						fmt.Printf("We could not find a StudyInstanceUID here: %v\n", StudyInstanceUIDVal)
						return nil
					}

					PatientIDVal, err := dataset.FindElementByTag(tag.PatientID)
					if err == nil {
						PatientID = dicom.MustGetStrings(PatientIDVal.Value)[0]
					}
					PatientNameVal, err := dataset.FindElementByTag(tag.PatientName)
					if err == nil {
						PatientName = dicom.MustGetStrings(PatientNameVal.Value)[0]
					}

					SeriesInstanceUIDVal, err := dataset.FindElementByTag(tag.SeriesInstanceUID)
					if err == nil {
						SeriesInstanceUID = dicom.MustGetStrings(SeriesInstanceUIDVal.Value)[0]
					}
					if SeriesInstanceUID == "" {
						// no series instance uid skip this file
						fmt.Printf("We could not find a SeriesInstanceUID here: %v\n", SeriesInstanceUIDVal)
						return nil
					}
					SeriesDescriptionVal, err := dataset.FindElementByTag(tag.SeriesDescription)
					if err == nil {
						SeriesDescription = dicom.MustGetStrings(SeriesDescriptionVal.Value)[0]
					}
					SeriesNumberVal, err := dataset.FindElementByTag(tag.SeriesNumber)
					if err == nil {
						SeriesNumber, err = strconv.Atoi(dicom.MustGetStrings(SeriesNumberVal.Value)[0])
						if err != nil {
							SeriesNumber = 0
						}
					}
					SequenceNameVal, err := dataset.FindElementByTag(tag.SequenceName)
					if err == nil {
						SequenceName = dicom.MustGetStrings(SequenceNameVal.Value)[0]
					}
					StudyDescriptionVal, err := dataset.FindElementByTag(tag.StudyDescription)
					if err == nil {
						StudyDescription = dicom.MustGetStrings(StudyDescriptionVal.Value)[0]
					}
					ModalityVal, err := dataset.FindElementByTag(tag.Modality)
					if err == nil {
						Modality = dicom.MustGetStrings(ModalityVal.Value)[0]
					}
					ManufacturerVal, err := dataset.FindElementByTag(tag.Manufacturer)
					if err == nil {
						Manufacturer = dicom.MustGetStrings(ManufacturerVal.Value)[0]
					}
					ManufacturerModelNameVal, err := dataset.FindElementByTag(tag.ManufacturerModelName)
					if err == nil {
						ManufacturerModelName = dicom.MustGetStrings(ManufacturerModelNameVal.Value)[0]
					}
					abs_path, err := filepath.Abs(path)
					if err != nil {
						abs_path = path
					}
					var path_pieces string = filepath.Dir(abs_path)

					if _, ok := datasets[StudyInstanceUID]; ok {
						if val, ok := datasets[StudyInstanceUID][SeriesInstanceUID]; ok {
							// largest common path
							var lcp string = "-1"
							var l1 = strings.Split(val.Path, string(os.PathSeparator))
							var l2 = strings.Split(path_pieces, string(os.PathSeparator))
							//fmt.Println(l1, l2)
							for i, j := 0, 0; i < len(l1) && j < len(l2); i, j = i+1, j+1 {
								if l1[i] == l2[j] {
									if lcp == "-1" {
										lcp = l1[i]
									} else {
										lcp = fmt.Sprintf("%s%s%s", lcp, string(os.PathSeparator), l1[i])
									}
								} else {
									//fmt.Printf("Break at \"%s\", for \"%s\", \"%s\"\n", lcp, l1, l2)
									break
								}
							}
							tmp_with_double := append(val.ClassifyTypes, ClassifyDICOM(dataset)...)
							// compute a unique list of entries in val.Classify
							var unique_map map[string]string = make(map[string]string)
							for _, v := range tmp_with_double {
								unique_map[v] = ""
							}
							val.ClassifyTypes = make([]string, 0)
							for k := range unique_map {
								val.ClassifyTypes = append(val.ClassifyTypes, k)
							}
							datasets[StudyInstanceUID][SeriesInstanceUID] = SeriesInfo{NumImages: val.NumImages + 1,
								SeriesDescription:     SeriesDescription,
								SeriesNumber:          SeriesNumber,
								SequenceName:          SequenceName,
								Modality:              Modality,
								Manufacturer:          Manufacturer,
								ManufacturerModelName: ManufacturerModelName,
								StudyDescription:      StudyDescription,
								Path:                  lcp,
								PatientID:             PatientID,
								PatientName:           PatientName,
								All:                   val.All,
								ClassifyTypes:         val.ClassifyTypes, // only parse the first image? No, we need to parse all because we have to collect all possible classes for Localizer (aixal + coronal + sagittal)
							}
						} else {
							// if there is no SeriesInstanceUID but there is a StudyInstanceUID we could have
							// other series already in the list

							//fmt.Printf("WE have this study, add another SERIES NOW %d\n", len(datasets[StudyInstanceUID]))
							datasets[StudyInstanceUID][SeriesInstanceUID] = SeriesInfo{NumImages: 1,
								SeriesDescription:     SeriesDescription,
								SeriesNumber:          SeriesNumber,
								SequenceName:          SequenceName,
								Modality:              Modality,
								Manufacturer:          Manufacturer,
								ManufacturerModelName: ManufacturerModelName,
								StudyDescription:      StudyDescription,
								PatientID:             PatientID,
								PatientName:           PatientName,
								Path:                  path_pieces,
								All:                   all,
								ClassifyTypes:         ClassifyDICOM(dataset),
							}
						}
					} else {
						datasets[StudyInstanceUID] = make(map[string]SeriesInfo)
						datasets[StudyInstanceUID][SeriesInstanceUID] = SeriesInfo{NumImages: 1,
							SeriesDescription:     SeriesDescription,
							SeriesNumber:          SeriesNumber,
							SequenceName:          SequenceName,
							Modality:              Modality,
							Manufacturer:          Manufacturer,
							ManufacturerModelName: ManufacturerModelName,
							StudyDescription:      StudyDescription,
							PatientID:             PatientID,
							PatientName:           PatientName,
							Path:                  path_pieces,
							All:                   all,
							ClassifyTypes:         ClassifyDICOM(dataset),
						}
					}
				} else {
					//fmt.Println("NO StudyInstanceUID found", err, dataset)
					return nil
				}
			} else {
				nonDICOM = nonDICOM + 1
				//fmt.Println("NONDICOM FILE: ", path, err, dataset)
				if app != nil {
					footer.Clear()
					fmt.Fprintf(footer, langFmt.Sprintf("[%d] non-DICOM file %s\n", nonDICOM, path))
					app.Draw()
				}
			}
			return nil
		})
		if err != nil {
			fmt.Println("Warning: could not walk this path")
		}
	}

	return datasets, nil
}

// createStub will check if the folder exists and create a text file
// @param p: the path to the file
// @param str: the content of the file
func createStub(p string, str string) {
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		fmt.Println("This directory already contains an " + filepath.Base(p) + ", don't overwrite. Skip writing...")
	} else {
		err := os.MkdirAll(filepath.Dir(p), 0777)
		if err != nil {
			fmt.Println("Error creating the required directories for ", filepath.Dir(p))
		}
		f, err := os.Create(p)
		check(err)
		_, err = f.WriteString(str)
		check(err)
		f.Sync()
	}
}

// Could we create an ast at random that is useful?
// We would need to check how good an ast is given the
// data. A likelihood function would incorporate
// - ratio of the detected datasets given the number of studies/patients (max entropy?)
// - one over the complexity of the ast to prefer simple ast's (one over total number of rules)
// How about longitudinal data? How many series per study is best?
//   We could use the mean over the average number of image series per study?
// How would we generate new rules for monte-carlo testing?
// - We can add a new rule to a ruleset by selecting a new variable
// - We can change an existing rule by changing theh numeric value for '<' and '>'
// - We can add a new ruleset with a random rule
func (ast AST) improveAST(datasets map[string]map[string]SeriesInfo) (AST, float64) {
	// collect all the values in all the SeriesInfo fields
	tmpTargetValues := make(map[string]map[string]bool)
	tmpTargetValues["StudyDescription"] = make(map[string]bool)
	tmpTargetValues["SeriesDescription"] = make(map[string]bool)
	tmpTargetValues["Modality"] = make(map[string]bool)
	tmpTargetValues["SequenceName"] = make(map[string]bool)
	tmpTargetValues["Manufacturer"] = make(map[string]bool)
	tmpTargetValues["NumImages"] = make(map[string]bool)
	tmpTargetValues["SeriesNumber"] = make(map[string]bool)
	tmpTargetValues["ManufacturerModelName"] = make(map[string]bool)
	for _, v := range datasets {
		for _, v2 := range v {
			tmpTargetValues["SeriesDescription"][v2.SeriesDescription] = true
			tmpTargetValues["StudyDescription"][v2.StudyDescription] = true
			tmpTargetValues["Modality"][v2.Modality] = true
			tmpTargetValues["SequenceName"][v2.SequenceName] = true
			tmpTargetValues["Manufacturer"][v2.Manufacturer] = true
			tmpTargetValues["NumImages"][fmt.Sprintf("%d", v2.NumImages)] = true
			tmpTargetValues["SeriesNumber"][fmt.Sprintf("%d", v2.SeriesNumber)] = true
			tmpTargetValues["ManufacturerModelName"][v2.ManufacturerModelName] = true
		}
	}
	targetValues := make(map[string][]string)
	targetValues["StudyDescription"] = []string{}
	targetValues["SeriesDescription"] = []string{}
	targetValues["Modality"] = []string{}
	targetValues["SequenceName"] = []string{}
	targetValues["Manufacturer"] = []string{}
	targetValues["NumImages"] = []string{}
	targetValues["SeriesNumber"] = []string{}
	targetValues["ManufacturerModelName"] = []string{}
	targetType := func(s string) string {
		if s == "NumImages" || s == "SeriesNumber" {
			return "numeric"
		}
		return "text"
	}
	for k, v := range tmpTargetValues {
		for k2 := range v {
			if k2 == "" {
				continue
			}
			if k == "StudyDescription" {
				targetValues["StudyDescription"] = append(targetValues["StudyDescription"], k2)
			}
			if k == "SeriesDescription" {
				targetValues["SeriesDescription"] = append(targetValues["SeriesDescription"], k2)
			}
			if k == "Modality" {
				targetValues["Modality"] = append(targetValues["Modality"], k2)
			}
			if k == "SequenceName" {
				targetValues["SequenceName"] = append(targetValues["SequenceName"], k2)
			}
			if k == "Manufacturer" {
				targetValues["Manufacturer"] = append(targetValues["Manufacturer"], k2)
			}
			if k == "NumImages" {
				targetValues["NumImages"] = append(targetValues["NumImages"], k2)
			}
			if k == "SeriesNumber" {
				targetValues["SeriesNumber"] = append(targetValues["SeriesNumber"], k2)
			}
			if k == "ManufacturerModelName" {
				targetValues["ManufacturerModelName"] = append(targetValues["ManufacturerModelName"], k2)
			}
		}
	}
	// we might have some entries that only have empty values, we don't like those as they
	// always match to anything
	for k, v := range targetValues {
		if len(v) == 0 {
			delete(targetValues, k)
		}
	}

	// likelihood: we want to minimize this function
	likelihood := func(ast AST) float64 {
		allSeriesNum := 0.0
		for _, v := range datasets {
			allSeriesNum = allSeriesNum + float64(len(v))
		}

		// compute the match with the data
		a, _ := findMatchingSets(ast, datasets)
		//  we like to have all a's equally big (studyinstanceuid with same #seriesinstanceuid)
		var sumX float64
		for _, v := range a {
			if len(v) == 0 {
				continue
			}
			//numSeries := len(a[k])
			// number of series for this study
			SeriesInstanceUID := v[0]
			var numSeriesByStudy float64 = 0.0
		L:
			for _, vv := range datasets {
				for siuid := range vv {
					if SeriesInstanceUID == siuid {
						numSeriesByStudy = float64(len(vv))
						break L
					}
				}
			}

			numSelected := float64(len(v)) / numSeriesByStudy
			sumX += float64(numSelected)
		}
		if len(a) > 0 {
			sumX = sumX / allSeriesNum
		} else {
			sumX = 0
		}
		// mean should be close to 0.5

		// compute penalty for the complexity of the rules, more rules is worse
		var total float64
		for _, rulelist := range ast.Rules {
			total = total + float64(len(rulelist))
		}
		b := sumX + 1.0/math.Log2(total+1.0)
		return b
	}
	// addRule: add a single rule
	addRule := func(rules *[]Rule, targetValues map[string][]string) bool {
		// do we have access to targetValues here?
		// what are the possible fields we can match with?
		// fields := []string{"SeriesDescription", "StudyDescription", "NumImages", "Modality"}
		operators := []string{"==", "contains"}
		operatorsNumeric := []string{"<", ">"}
		fieldIdx := rand.Intn((len(targetValues) - 0) + 0)

		counter := 0
		t := ""
		val := ""
		// this does not work as the sorting order is not guaranteed
		for k := range targetValues {
			if counter == fieldIdx {
				t = k
				val = targetValues[k][rand.Intn((len(targetValues[k])-0)+0)]
			}
			counter = counter + 1
		}

		var op string = ""
		if targetType(t) == "numeric" {
			operatorIdx := rand.Intn((len(operatorsNumeric) - 0) + 0)
			op = operatorsNumeric[operatorIdx]
		} else {
			operatorIdx := rand.Intn((len(operators) - 0) + 0)
			op = operators[operatorIdx]
		}

		r := Rule{
			Tag:      []string{t},
			Value:    val,
			Operator: op,
		}
		if op == ">" || op == "<" {
			_, err := strconv.ParseFloat(val, 32)
			if err != nil {
				r.Value = rand.Intn(100)
			}
		}

		// what are possible fields values? We would need to know what we can equate our value to
		// we need to parse all the dataset for this to know what is available...
		*rules = append(*rules, r)

		return true
	}

	// change an existing rule (here just create a new one)
	changeRule := func(rule *Rule, targetValues map[string][]string) bool {
		// we can change a rule based on the operator (like < we can change the value)
		rr := make([]Rule, 0)
		ok := addRule(&rr, targetValues)
		if ok {
			*rule = rr[0]
		} else {
			return false
		}

		return true
	}

	// addRules: add a new complete rule set
	addRules := func(rules *[][]Rule, targetValues map[string][]string) bool {
		// get a new rule
		rr := make([]Rule, 0)
		ok := addRule(&rr, targetValues)
		if ok {
			*rules = append(*rules, rr)
		} else {
			return false
		}
		return true
	}

	// changeRules adjust the rules once using Metropolis-Hastings
	changeRules := func(ast AST, targetValues map[string][]string) bool {
		// pick a random rule
		var rulesetIdx int = -1
		if len(ast.Rules) > 0 {
			rulesetIdx = rand.Intn((len(ast.Rules) - 0) + 0)
		}
		if rulesetIdx == -1 {
			return false
		}
		var ruleIdx int = -1
		if len(ast.Rules[rulesetIdx]) > 0 {
			ruleIdx = rand.Intn((len(ast.Rules[rulesetIdx]) - 0) + 0)
		}
		if ruleIdx > -1 {
			ok := changeRule(&ast.Rules[rulesetIdx][ruleIdx], targetValues)
			// or add a new rule
			if !ok {
				ok = addRule(&ast.Rules[rulesetIdx], targetValues)
				if !ok {
					ok = addRules(&ast.Rules, targetValues)
					if !ok {
						fmt.Println("We failed with changing anything.")
						return false
					}
				}
			}
		} else {
			return false
		}
		return true
	}

	// Metropolis
	l := likelihood(ast)
	var bestRulesetEver AST
	foundBestRuleset := false
	bestL2 := math.Inf(1)
	for i := 0; i < 100; i++ {
		fmt.Printf("\033[A\033[2K%d/100 %.3f\n", i+1, bestL2)
		// make a copy of the rule
		jast, _ := json.Marshal(ast)
		var copyRule AST
		json.Unmarshal(jast, &copyRule)
		if !foundBestRuleset {
			json.Unmarshal(jast, &bestRulesetEver)
			foundBestRuleset = true
			bestL2 = likelihood(bestRulesetEver)
		}

		ok := changeRules(copyRule, targetValues)
		if !ok {
			fmt.Println("End here, no change to the rules could be implemented")
			return copyRule, likelihood(copyRule)
		}
		l2 := likelihood(copyRule)
		if l2 > l {
			ast = copyRule
			l = l2
			if bestL2 < l2 {
				jast, _ := json.Marshal(ast)
				json.Unmarshal(jast, &bestRulesetEver)
				bestL2 = l2
			}
			//fmt.Printf("improve rule now: %f\n", l2)
		} else {
			var prob float64 = rand.Float64()
			if prob > 0.99 {
				ast = copyRule
				l = l2
				if bestL2 < l2 {
					jast, _ := json.Marshal(ast)
					json.Unmarshal(jast, &bestRulesetEver)
					bestL2 = l2
				}
				//fmt.Printf("not improvement but Metropolis update: %f\n", l2)
			}
		}
	}
	return bestRulesetEver, likelihood(bestRulesetEver)
}

// findMatchingSets returns all matching sets for this rule and the provided data
// It also returns a list of the names given to each rule in select.
func findMatchingSets(ast AST, dataInfo map[string]map[string]SeriesInfo) ([][]string, [][]string) {

	var selectFromB [][]string
	var names [][]string = make([][]string, 0)
	// can only access the information in config.Data for these matches
	seriesByStudy := make(map[string]map[string][]int)
	seriesByPatient := make(map[string]map[string][]int)
	for StudyInstanceUID, value := range dataInfo {
		// we can check on the study or the series level or the patient level
		for SeriesInstanceUID, value2 := range value {
			// we assume here that we are in the series level...
			var matches bool = false
			var matchesIdx int = -1
			for idx, ruleset := range ast.Rules { // todo: check if this works if a ruleset matches the 2 series
				if value2.evalRules(ruleset) { // check if this ruleset fits with this series
					matches = true
					matchesIdx = idx
					break
				}
			}
			if matches {
				if _, ok := seriesByStudy[StudyInstanceUID]; !ok {
					seriesByStudy[StudyInstanceUID] = make(map[string][]int)
				}
				if _, ok := seriesByStudy[StudyInstanceUID][SeriesInstanceUID]; !ok {
					seriesByStudy[StudyInstanceUID][SeriesInstanceUID] = []int{matchesIdx}
				} else {
					seriesByStudy[StudyInstanceUID][SeriesInstanceUID] = append(seriesByStudy[StudyInstanceUID][SeriesInstanceUID], matchesIdx)
				}
				PatientName := value2.PatientID + value2.PatientName
				if _, ok := seriesByPatient[PatientName]; !ok {
					seriesByPatient[PatientName] = make(map[string][]int)
				}
				if _, ok := seriesByPatient[PatientName][SeriesInstanceUID]; !ok {
					seriesByPatient[PatientName][SeriesInstanceUID] = []int{matchesIdx}
				} else {
					seriesByPatient[PatientName][SeriesInstanceUID] = append(seriesByPatient[PatientName][SeriesInstanceUID], matchesIdx)
				}
				// single level append here
				selectFromB = append(selectFromB, []string{SeriesInstanceUID})
				names = append(names, []string{ast.Rule_list_names[matchesIdx]})
			}
		}
	}
	if ast.Output_level == "study" {
		// If we want to export by study we need to export all studies where all the individual rules
		// resulted in a match at the series level. But we will export matched series for these studies only.
		selectFromB = make([][]string, 0)
		names = make([][]string, 0)
		for _, value := range seriesByStudy {
			// which rules need to match?
			// all rules from 0..len(ast.Rules)
			allThere := true
			currentNamesByRule := make([]string, 0)
			for r := 0; r < len(ast.Rules); r++ {
				thisThere := false
				for _, value2 := range value {
					for _, value3 := range value2 {
						// each one is an integer, we look for r here
						if value3 == r {
							currentNamesByRule = append(currentNamesByRule, ast.Rule_list_names[r])
							thisThere = true
						}
					}
				}
				if !thisThere {
					allThere = false
					break
				}
			}
			if allThere {
				// only append our series for this study
				// append all SeriesInstanceUIDs now
				var ss []string
				for k := range value {
					ss = append(ss, k)
				}
				selectFromB = append(selectFromB, ss)
				names = append(names, currentNamesByRule)
			}
		}
	} else if ast.Output_level == "patient" {
		// If we want to export by study we need to export all studies where all the individual rules
		// resulted in a match at the series level. But we will export matched series for these studies only.
		selectFromB = make([][]string, 0)
		names = make([][]string, 0)
		for _, value := range seriesByPatient {
			// which rules need to match?
			// all rules from 0..len(ast.Rules)
			allThere := true
			currentNamesByRule := make([]string, 0)
			for r := 0; r < len(ast.Rules); r++ {
				thisThere := false
				for _, value2 := range value {
					for _, value3 := range value2 {
						// each one is an integer, we look for r here
						if value3 == r {
							currentNamesByRule = append(currentNamesByRule, ast.Rule_list_names[r])
							thisThere = true
						}
					}
				}
				if !thisThere {
					allThere = false
					break
				}
			}
			if allThere {
				// only append our series for this study
				// append all SeriesInstanceUIDs now
				var ss []string
				for k := range value {
					ss = append(ss, k)
				}
				selectFromB = append(selectFromB, ss)
				names = append(names, currentNamesByRule)
			}
		}
	} else if ast.Output_level == "project" {
		// If we want to export all matching patient/studies/series where all individual rules
		// resulted in a match at the series level. But we will export matched series only.
		// there will be a single output level with all data in it
		selectFromB = make([][]string, 0)
		names = make([][]string, 0)
		var ss []string
		currentNamesByRule := make([]string, 0)
		for _, value := range seriesByPatient {
			// which rules need to match?
			// all rules from 0..len(ast.Rules)
			allThere := true
			for r := 0; r < len(ast.Rules); r++ {
				thisThere := false
				for _, value2 := range value {
					for _, value3 := range value2 {
						// each one is an integer, we look for r here
						if value3 == r {
							currentNamesByRule = append(currentNamesByRule, ast.Rule_list_names[r])
							thisThere = true
						}
					}
				}
				if !thisThere {
					allThere = false
					break
				}
			}
			if allThere {
				// only append our series for this study
				// append all SeriesInstanceUIDs now
				for k := range value {
					ss = append(ss, k)
				}
			}
		}
		selectFromB = append(selectFromB, ss)
		names = append(names, currentNamesByRule)
	}

	// we need to check the CheckRules as well - if we have those we might loose some more entries here
	if ast.CheckRules != nil {
		checkCheckRules := func(entry []string, names []string, ast AST, dataInfo map[string]map[string]SeriesInfo) []string {
			// entry is now a list of SeriesInstanceUIDs
			okSeriesIDS := make([]string, 0)
			for _, ruleset := range ast.CheckRules {
				// does this ruleset work for all our selected series?
				for _, rule := range ruleset {
					// each one is an integer, we look for r here
					tag1 := rule.Tag
					tag2 := rule.Tag2
					if len(tag1) == 3 && len(tag2) == 3 {
						// find the correct series
						series_name1 := tag1[0]
						series_name2 := tag2[0]
						series_idx1 := -1
						series_idx2 := -1
						// find the correspondingly named series
						// but for a rule we need to compare two series
						// one name can happen several times so we need to
						// collect all possible sets
						pairs := make([][]int, 0)
						for i, name := range names {
							if name == series_name1 {
								series_idx1 = i
							}
							if name == series_name2 {
								series_idx2 = i
							}
							if series_idx1 != -1 && series_idx2 != -1 {
								// one pair
								p := []int{series_idx1, series_idx2}
								// try to find if that pair exists already
								found := false
								for _, ps := range pairs {
									if ps[0] == p[0] && ps[1] == p[1] {
										found = true
										break
									}
								}
								if !found {
									pairs = append(pairs, []int{series_idx1, series_idx2})
								}
							}
						}
						if len(pairs) == 0 {
							// we will now be able to check if the rule applies
							//fmt.Println("IDS could not be found with", series_idx1, series_idx2)
							continue
						}
						// what are the values we have for the two series?
						for _, p := range pairs {
							SeriesInstanceUID1 := entry[p[0]]
							SeriesInstanceUID2 := entry[p[1]]
							ok := evalCheckRule(rule, SeriesInstanceUID1, SeriesInstanceUID2, dataInfo)
							if ok {
								// that means we will keep these series ids
								//fmt.Println("This rule works!")
								okSeriesIDS = append(okSeriesIDS, SeriesInstanceUID1)
								okSeriesIDS = append(okSeriesIDS, SeriesInstanceUID2)
							}
						}
					}
					//fmt.Println("%v\n", rule)
				}
			}
			return okSeriesIDS
		}
		for idx, set := range selectFromB {
			set_names := names[idx]
			okSeriesIDS := checkCheckRules(set, set_names, ast, dataInfo)
			//if len(okSeriesIDS) > 0 { // we actually have some series we want to keep
			//	fmt.Println("We have some series that work with this rule, some we might have to remove")
			//} else {
			//	fmt.Println("We need to remove all entries")
			//}
			contains := func(s []string, e string) bool {
				for _, a := range s {
					if a == e {
						return true
					}
				}
				return false
			}
			remove := func(s []string, i int) []string {
				s[i] = s[len(s)-1]
				return s[:len(s)-1]
			}
			for idx2 := 0; idx2 < len(selectFromB[idx]); idx2++ {
				v := selectFromB[idx][idx2]
				if !contains(okSeriesIDS, v) {
					// remove this series
					//fmt.Println("Removing", v)
					selectFromB[idx] = remove(selectFromB[idx], idx2)
					names[idx] = remove(names[idx], idx2)
					idx2--
				}
			}

		}
	}

	// we should also allow the sorting of the results
	// maybe easiest for now if we sort by number of image series?
	// for each patient we would like to sort by date as well
	order := argsort.SortSlice(selectFromB, func(i, j int) bool {
		l1 := 0
		for _, a := range selectFromB[i] {
			//  ok, a is a series instance uid, I need to get the info from that series
		L1:
			for _, b := range dataInfo {
				for SeriesInstanceUID, c := range b {
					if SeriesInstanceUID == a {
						l1 += c.NumImages
						break L1
					}
				}
			}
		}
		l2 := 0
		for _, a := range selectFromB[j] {
		L2:
			for _, b := range dataInfo {
				for SeriesInstanceUID, c := range b {
					if SeriesInstanceUID == a {
						l2 += c.NumImages
						break L2
					}
				}
			}
		}
		return l1 > l2
	})
	// fmt.Println("%v", order)
	// we should apply the order now
	var outSelect [][]string
	var outNames [][]string
	for i := 0; i < len(order); i++ {
		outSelect = append(outSelect, selectFromB[order[i]])
		outNames = append(outNames, names[order[i]])
	}

	//return selectFromB, names
	return outSelect, outNames
}

func humanizeFilter(ast AST) string {
	// create a human readeable string from the AST
	var ss string

	switch ast.Output_level {
	case "series":
		ss = fmt.Sprintf("%s\nWe will run processing on any single image series that matches.", ss)
	case "study":
		ss = fmt.Sprintf("%s\nWe will run processing on data containing a single study and its matching image series.", ss)
	case "patient":
		ss = fmt.Sprintf("%s\nWe will run processing on data containing all studies of a patient for which those studies have the correct number of matching image series.", ss)
	case "project":
		ss = fmt.Sprintf("%s\nWe will run processing on all data with matching image series.", ss)
	}

	if len(ast.Rules) == 1 {
		ss = fmt.Sprintf("%s\nWe will select cases with a single matching image series.\n", ss)
	} else {
		ss = fmt.Sprintf("%s\nWe will select cases with %d image series.\n", ss, len(ast.Rules))
	}
	fmt.Printf("To use this select statement call:\n%s config --select '\n%s\n'\n", own_name, ast2Select(ast))

	return ss
}

// ast2Select create a select statement from the AST
func ast2Select(ast AST) string {
	//sep1 := "  "
	//sep2 := "\n"
	stm := fmt.Sprintf("SELECT %s\n  FROM study", ast.Output_level)
	for idx2, rules := range ast.Rules {
		s := ""
		for idx, rule := range rules {
			a := ""
			if idx > 0 {
				a = "\n    AND\n"
			}
			opstr := "containing"
			if rule.Operator == "contains" {
				opstr = "containing"
			} else if rule.Operator == "<" {
				opstr = "<"
			} else if rule.Operator == ">" {
				opstr = ">"
			} else if rule.Operator == "=" {
				opstr = "="
			} else if rule.Operator == "regexp" {
				opstr = "regexp"
			}
			if len(rule.Tag) == 2 {
				s = fmt.Sprintf("%s%s (%s,%s) %s %v", s, a, rule.Tag[0], rule.Tag[1], opstr, rule.Value)
			} else {
				s = fmt.Sprintf("%s%s %s %s %v", s, a, rule.Tag[0], opstr, rule.Value)
			}
		}
		if idx2 > 0 {
			stm = fmt.Sprintf("%s\n  ALSO\n    WHERE series NAMED \"%s\" HAS\n      %s", stm, ast.Rule_list_names[idx2], s)
		} else {
			stm = fmt.Sprintf("%s\n    WHERE series NAMED \"%s\" HAS\n      %s", stm, ast.Rule_list_names[idx2], s)
		}
	}
	// Now add a section for the CheckRules if there are any
	if ast.CheckRules != nil {
		stm = fmt.Sprintf("%s\nCheck\n  ", stm)
		for _, rulesets := range ast.CheckRules {
			for idx, rule := range rulesets {
				a := ""
				if idx > 0 {
					a = "\n    AND\n"
				}
				s := ""
				opstr := "containing"
				if rule.Operator == "contains" {
					opstr = "containing"
				} else if rule.Operator == "==" {
					opstr = "="
				}
				if len(rule.Tag) == 3 {
					stm = fmt.Sprintf("%s%s%s %s@(%s,%s) %s %s@(%s,%s)", stm, s, a, rule.Tag[0], rule.Tag[1], rule.Tag[2], opstr, rule.Tag2[0], rule.Tag2[1], rule.Tag2[2])
				}
			}
		}
	}

	return stm
}

func callProgram(config Config, triggerWaitTime string, trigger_container string, dir string) {
	if config.CallString == "" {
		exitGracefully(fmt.Errorf("could not run trigger command, no CallString defined\n\n\t%s config --call \"python ./stub.py\"", own_name))
	}

	// wait for some seconds, why do we support this?
	if triggerWaitTime != "" {
		sec, _ := time.ParseDuration(triggerWaitTime)
		time.Sleep(sec)
	}

	cmd_str := config.CallString
	r := regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)`)
	arr := r.FindAllString(cmd_str, -1)
	arr = append(arr, string(dir))
	// cmd := exec.Command("python", "stub.py", dir)
	var cmd *exec.Cmd
	var cmd_string []string
	if trigger_container != "" {
		cmd_string = []string{"docker", "run", "--rm", "-v",
			fmt.Sprintf("%s:/data", strings.Replace(dir, " ", "\\ ", -1)), trigger_container, "/bin/bash", "-c",
			fmt.Sprintf("cd /app; %s /data/", cmd_str)}
		fmt.Println(strings.Join(cmd_string, " "))
		cmd = exec.Command(cmd_string[0], cmd_string[1:]...)
	} else {
		fmt.Println(arr)
		cmd_string = arr
		cmd = exec.Command(arr[0], arr[1:]...)
	}
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	exitCode := cmd.Run()
	if exitCode != nil {
		exitGracefully(fmt.Errorf("could not run trigger command\n\t%s\nError code: %s\n\t%s", strings.Join(arr[:], " "), exitCode.Error(), errb.String()))
	}
	// store stdout and stderr as log files
	if _, err := os.Stat(dir + "/log"); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(dir+"/log", 0755); os.IsExist(err) {
			exitGracefully(errors.New("directory exist already"))
		}
	}
	// write the log files
	var stdout_log string = fmt.Sprintf("%s/log/stdout.log", dir)
	f_log_stdout, err := os.OpenFile(stdout_log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		exitGracefully(errors.New("could not open file " + stdout_log))
	}
	defer f_log_stdout.Close()
	if _, err := f_log_stdout.WriteString(strings.Join(cmd_string, " ") + "\n" + outb.String()); err != nil {
		exitGracefully(errors.New("could not write to log/stdout.log"))
		// log.Println(err)
	}

	var stderr_log string = fmt.Sprintf("%s/log/stderr.log", dir)
	f_log_stderr, err := os.OpenFile(stderr_log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		exitGracefully(errors.New("could not open " + stderr_log))
	}
	defer f_log_stderr.Close()
	if _, err := f_log_stderr.WriteString(errb.String()); err != nil {
		exitGracefully(errors.New("could not add to " + stderr_log))
		// log.Println(err)
	}
}

var app *tview.Application = nil

func main() {

	rand.Seed(time.Now().UnixNano())
	// disable logging
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)

	const (
		defaultInputDir    = "Specify where you want to setup shop"
		defaultTriggerTime = "A wait time in seconds or minutes before the computation is triggered"
		errorConfigFile    = "the current directory is not an ror directory. Change to the correct directory first or create a new folder by running\n\n\tror init project01\n "
	)

	initCommand := flag.NewFlagSet("init", flag.ContinueOnError)
	configCommand := flag.NewFlagSet("config", flag.ContinueOnError)
	triggerCommand := flag.NewFlagSet("trigger", flag.ContinueOnError)
	statusCommand := flag.NewFlagSet("status", flag.ContinueOnError)
	buildCommand := flag.NewFlagSet("build", flag.ContinueOnError)
	annotateCommand := flag.NewFlagSet("annotate", flag.ContinueOnError)

	initCommand.StringVar(&input_dir, "input_dir", ".", defaultInputDir)
	var init_help bool
	initCommand.BoolVar(&init_help, "help", false, "Show help for init.")

	//initCommand.StringVar(&input_dir, "i", ".", defaultInputDir)
	var author_name string
	configCommand.StringVar(&author_name, "author_name", "", "Author name used to publish your workflow.")
	initCommand.StringVar(&author_name, "author_name", "", "Author name used to publish your workflow.")
	var author_email string
	configCommand.StringVar(&author_email, "author_email", "", "Author email used to publish your workflow.")
	initCommand.StringVar(&author_email, "author_email", "", "Author email used to publish your workflow.")
	var init_type string
	initCommand.StringVar(&init_type, "type", "", "Type of project. The supported types are \"python\", \"notebook\", \"bash\", \"webapp\". Based on\nthis choice you will get a different initial directory structure.")

	var data_path string
	configCommand.StringVar(&data_path, "data", "", "Path to a folder with DICOM files. If you want to specify a subset of folders\nuse double quotes for the path and the glob syntax. For example all folders that\nstart with numbers 008 and 009 would be read with --data \"path/to/data/0[8-9]*\"")
	var call_string string
	configCommand.StringVar(&call_string, "call", "", "The command line to call the workflow. A path-name with the data will be appended\nto this string.")
	var project_name_string string
	configCommand.StringVar(&project_name_string, "project_name", "", "The name of the project. This string will be used in the container name.")
	var no_sort_dicom bool
	configCommand.BoolVar(&no_sort_dicom, "no_sort_dicom", false, "Do not create an additional input_view_dicom_series/ folder that contains sorted DICOM files by\nstudy and series. If set (--no_sort_dicom=1) DICOM files are written into input/,\nno sub-folder is created. If not set (--no_sort_dicom=0) DICOM files are written\ninto input/ and an additional input_view_dicom_series/ folder will contain a directory structure\nby participant, study, and series with symbolic links to the input/ files.")
	var config_help bool
	configCommand.BoolVar(&config_help, "help", false, "Print help for config.")
	var project_token string
	configCommand.StringVar(&project_token, "token", "", "The token generated by the research information system for your workflow.")
	var config_suggest bool
	configCommand.BoolVar(&config_suggest, "suggest", false, "Suggest a selection rule.")

	var triggerWaitTime string
	triggerCommand.StringVar(&triggerWaitTime, "delay", "0s", defaultTriggerTime)
	var trigger_test bool
	triggerCommand.BoolVar(&trigger_test, "test", false, "Don't actually run anything, just show what you would do.")
	var trigger_keep bool
	triggerCommand.BoolVar(&trigger_keep, "keep", false, "Keep the data directory around for testing.")
	var trigger_each bool
	triggerCommand.BoolVar(&trigger_each, "each", false, "Trigger for each found series, not just for a single random one.")
	var trigger_container string
	triggerCommand.StringVar(&trigger_container, "cont", "", "Trigger using a container instead of a local workflow.")
	var trigger_help bool
	triggerCommand.BoolVar(&trigger_help, "help", false, "Show help for trigger")
	var trigger_last bool
	triggerCommand.BoolVar(&trigger_last, "last", false, "Trigger the last created workflow.")

	var status_detailed bool
	statusCommand.BoolVar(&status_detailed, "all", false, "Display all information.")
	var status_help bool
	statusCommand.BoolVar(&status_help, "help", false, "Show help for status.")
	var status_tui bool
	statusCommand.BoolVar(&status_tui, "tui", false, "Show the datasets.")

	var build_help bool
	buildCommand.BoolVar(&build_help, "help", false, "Show help for build.")

	var config_series_filter string
	configCommand.StringVar(&config_series_filter, "select", "",
		"Filter applied to series before trigger. This regular expression should\n"+
			"match anything in the string build by StudyInstanceUID: %s, \n"+
			"SeriesInstanceUID: %s, SeriesDescription: %s, ... As an example you might search\n"+
			"for a any series with a SeriesDescription starting with \"T1\" and ending in \"_2mm\"\n"+
			"with --select \"SeriesDescription: T1.*_2mm\". The default value matches any\nseries.\n"+
			"Also, it is now possible to specify more complex selections using a variant of the\n"+
			"standard query language. Here an example:\n"+
			"\t\"select study from study where series has ClassifyTypes containing T1\n"+
			"\tand SeriesDescription regexp \"^B\" also where series has ClassifyType\n"+
			"\tcontaining DIFFUSION also where series has ClassifyTypes containing RESTING\"\n"+
			"This filter should export all studies of a patient that have matching\n"+
			"series classified as T1, as Diffusion or as resting state scans. A slightly shorter\n"+
			"and valid version of the above filter would be:\n\t"+
			"Select study where ClassifyTypes containing T1 and SeriesDescription regexp \"^B\"\n"+
			"\talso where ClassifyType containing DIFFUSION also where ClassifyTypes containing RESTING")

	var config_temp_directory string
	configCommand.StringVar(&config_temp_directory, "temp_directory", "", "Specify a directory for the temporary folders used in the trigger")

	var show_version bool
	flag.BoolVar(&show_version, "version", false, "Show the version number.")

	var annotate_help bool
	annotateCommand.BoolVar(&annotate_help, "help", false, "Show help for annotate.")

	var annotate_ontology string
	annotateCommand.StringVar(&annotate_ontology, "ontology", "", "Ontology to use for annotation.")

	var user_name string
	user, err := user.Current()
	if err != nil {
		user_name = user.Username
		fmt.Println("got a user name ", user_name)
	}

	own_name = os.Args[0]
	// Showing useful information when the user enters the --help option
	flag.Usage = func() {
		fmt.Printf("ror - Research Workflow Processing\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Println(" A tool to simulate research information system workflows. The program")
		fmt.Println(" can create workflow projects and trigger a processing step similar to")
		fmt.Printf(" automated processing steps run in the research information system.\n\n")
		fmt.Printf("Usage: %s [init|config|status|trigger|build] [options]\n\tStart with init to create a new project folder:\n\n\t%s init <project>\n\n", os.Args[0], os.Args[0])
		fmt.Printf("Option init:\n  Create a new workflow project.\n\n")
		initCommand.PrintDefaults()
		fmt.Printf("\nOption config:\n  Change the current settings of your project and parse example data folders for trigger.\n\n")
		configCommand.PrintDefaults()
		fmt.Printf("\nOption status:\n  List the current settings of your project.\n\n")
		statusCommand.PrintDefaults()
		fmt.Printf("\nOption trigger:\n  Trigger a processing workflow locally.\n\n")
		triggerCommand.PrintDefaults()
		fmt.Printf("\nOption build:\n  Create a containerized version of your workflow.\n\n")
		buildCommand.PrintDefaults()
		fmt.Printf("\nOption annotate:\n  Add annotations to datasets.\n\n")
		annotateCommand.PrintDefaults()
		fmt.Println("")
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(-1)
	}

	if false {
		//
		// test the expression parser for select
		//
		InitParser()
		line := []byte("select patient from study")
		fmt.Printf("TEST EXPRESSION PARSER: %s\n", string(line))
		yyParse(&exprLex{line: line})
		s, _ := json.MarshalIndent(ast, "", "  ")
		fmt.Printf("ast is: %s\n", string(s))

		InitParser()
		line = []byte("select patient from study where series has ClassifyType containing T1 and SeriesDescription containing axial")
		fmt.Printf("TEST EXPRESSION PARSER: %s\n", string(line))
		yyParse(&exprLex{line: line})
		s, _ = json.MarshalIndent(ast, "", "  ")
		fmt.Printf("ast is: %s\n", string(s))

		InitParser()
		line = []byte("select patient from study where series has ClassifyType containing T1 and SeriesDescription containing axial also where series has ClassifyType containing DIFFUSION also where series has ClassifyType containing RESTING")
		fmt.Printf("TEST EXPRESSION PARSER: %s\n", string(line))
		yyParse(&exprLex{line: line})
		s, _ = json.MarshalIndent(ast, "", "  ")
		fmt.Printf("ast is: %s\n", string(s))
	}

	switch os.Args[1] {
	case "init", "create":
		if len(os.Args[2:]) == 0 {
			initCommand.PrintDefaults()
			return
		}
		if err := initCommand.Parse(os.Args[2:]); err == nil {
			if init_help {
				initCommand.PrintDefaults()
				return
			}
			// we expect a path first
			values := initCommand.Args()
			if len(values) != 1 {
				exitGracefully(errors.New("we need a single path entry specified"))
			} else {
				input_dir = initCommand.Arg(0)
			}

			dir_path := input_dir + "/.ror"
			if _, err := os.Stat(dir_path); !os.IsNotExist(err) {
				exitGracefully(errors.New("this directories has already been initialized. Delete the .ror directory to do this again"))
			}
			// do we know the author information?, do we need to know?
			// Instead we should ask for the user token and secret so we can
			// accept uploads from users to the research information system
			// With the token we can identify the project and with the secret
			// we can check if their information is not tampered with.
			// We could use the REDCap token for a user. That way we have control
			// over the metadata as well - but we would expose REDCap.
			if author_name == "" {
				author_name = "Anonymous"
			}
			if author_email == "" {
				author_email = "no-reply@unkown.org"
			}
			// do not ask for author information... we can do this at any time in the future
			if false && (author_name == "" || author_email == "") {

				reader := bufio.NewReader(os.Stdin)
				// we can ask interactively about the author information
				if author_name == "" {
					fmt.Printf("Author name: ")
					author_name, err = reader.ReadString('\n')
					if err != nil {
						msg := "we need your name. Add with\n\t--author_name \"<name>\""
						exitGracefully(errors.New(msg))
					}
					author_name = strings.TrimSuffix(author_name, "\n")
					if len(author_name) < 2 {
						fmt.Println("Does not look like a name, but you know best.")
					}
				}
				if author_email == "" {
					fmt.Printf("Author email: ")
					author_email, err = reader.ReadString('\n')
					if err != nil {
						msg := "we need your your email. Add with\n\t--author_email \"email@home\""
						exitGracefully(errors.New(msg))
					}
					author_email = strings.TrimSuffix(author_email, "\n")
					var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
					//	"^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
					var isEmail = true
					//e, err := mail.ParseAddress(author_email)
					//isEmail = (err == nil)
					//if len(author_email) < 3 && len(author_email) > 254 {
					//	isEmail = false
					//}
					isEmail = emailRegex.MatchString(author_email)
					if !isEmail {
						fmt.Println("Does not look like an email - but you know best.")
					}
				}
			}
			if init_type == "" {
				reader := bufio.NewReader(os.Stdin)
				fmt.Printf("Project type (python, notebook, bash, webapp): ")
				init_type, err = reader.ReadString('\n')
				if err != nil {
					init_type = "notebook"
				}
				init_type = strings.TrimSuffix(init_type, "\n")
				if init_type != "python" && init_type != "notebook" && init_type != "bash" && init_type != "webapp" {
					init_type = "notebook"
					fmt.Println("Warning: That is not a type we know. We will give you a python notebook project to get started.")
				}
			}
			// now we can create the folder - not earlier
			if _, err := os.Stat(input_dir); os.IsNotExist(err) {
				// take care of creating a safe directory here... how is this on Windows?
				if err := os.Mkdir(input_dir, 0755); os.IsExist(err) {
					exitGracefully(errors.New("directory exist already"))
				}
			}

			if err := os.Mkdir(dir_path, 0700); os.IsExist(err) {
				exitGracefully(errors.New("directory already exists"))
			}
			var annotate Annotate
			annotate.Ontology = nil // by default we don't have an ontology available
			data := Config{
				Date: time.Now().String(),
				Author: AuthorInfo{
					Name:  author_name,
					Email: author_email,
				},
				CallString:       "python ./stub.py",
				SeriesFilter:     ".*",
				SeriesFilterType: "glob",
				ProjectType:      init_type,
				SortDICOM:        true,
				ProjectName:      path.Base(input_dir),
				ProjectToken:     project_token,
				LastDataFolder:   "",
				Annotate:         annotate,
			}
			if init_type == "bash" {
				data.CallString = "./stub.sh"
			} else if init_type == "webapp" {
				data.CallString = "open http://127.0.0.1:8000"
			}
			if !data.writeConfig() {
				exitGracefully(errors.New("could not write config file"))
			}
			//file, _ := json.MarshalIndent(data, "", " ")
			//_ = ioutil.WriteFile(dir_path+"/config", file, 0600)

			readme_path := filepath.Join(input_dir, "README.md")
			createStub(readme_path, readme)

			if data.ProjectType == "python" || data.ProjectType == "notebook" { // plain python
				stub_path := filepath.Join(input_dir, "stub.py")
				createStub(stub_path, stub_py)
			}
			if data.ProjectType == "notebook" {
				stubipynb_path := filepath.Join(input_dir, "stub.ipynb")
				createStub(stubipynb_path, stub_ipynb)
			}
			if data.ProjectType == "webapp" {
				webapp_index_path := filepath.Join(input_dir, "index.html")
				createStub(webapp_index_path, webapp_index)

				webapp_all_path := filepath.Join(input_dir, "js", "all.js")
				createStub(webapp_all_path, webapp_js_all)

				webapp_js_bootstrap_path := filepath.Join(input_dir, "js", "bootstrap.min.js")
				createStub(webapp_js_bootstrap_path, webapp_js_boostrap)

				webapp_js_colorbrewer_path := filepath.Join(input_dir, "js", "colorbrewer.js")
				createStub(webapp_js_colorbrewer_path, webapp_js_colorbrewer)

				webapp_js_jquery_path := filepath.Join(input_dir, "js", "jquery-3.2.1.min.js")
				createStub(webapp_js_jquery_path, webapp_js_jquery)

				webapp_js_popper_path := filepath.Join(input_dir, "js", "popper.min.js")
				createStub(webapp_js_popper_path, webapp_js_popper)

				webapp_css_style_path := filepath.Join(input_dir, "css", "style.css")
				createStub(webapp_css_style_path, webapp_css_style)

				webapp_css_bootstrap_path := filepath.Join(input_dir, "css", "bootstrap.min.css")
				createStub(webapp_css_bootstrap_path, webapp_css_bootstrap)
			}
			if data.ProjectType == "bash" {
				stub_path2 := input_dir + "/stub.sh"
				if _, err := os.Stat(stub_path2); !os.IsNotExist(err) {
					fmt.Println("This directory already contains a stub.sh, don't overwrite. Skip writing...")
				} else {
					f, err := os.Create(stub_path2)
					check(err)
					_, err = f.WriteString(stub_sh)
					check(err)
					f.Sync()
					// make the file executable
					err = os.Chmod(stub_path2, 0755)
					if err != nil {
						fmt.Println("Warning: could not make the stub.sh executable, try your luck on your own.")
					}
				}
			}
			// virtualization environment
			virt_path := input_dir + "/.ror/virt"
			if err := os.Mkdir(virt_path, 0755); os.IsExist(err) {
				exitGracefully(errors.New("directory exist already"))
			}
			// classification rules so we can overwrite what ror does on its own
			classify_dicom_path2 := input_dir + "/.ror/classifyDICOM.json"
			createStub(classify_dicom_path2, classifyRules)

			if data.ProjectType == "python" || data.ProjectType == "notebook" {
				requirements_path2 := filepath.Join(virt_path, "requirements.txt")
				createStub(requirements_path2, requirements)

				requirements_path3 := filepath.Join(virt_path, "requirements.yml")
				createStub(requirements_path3, requirements_yml)
			}
			dockerignore_path2 := filepath.Join(virt_path, ".dockerignore")
			createStub(dockerignore_path2, dockerignore)

			dockerfile_path2 := filepath.Join(virt_path, "Dockerfile")
			if data.ProjectType == "bash" {
				createStub(dockerfile_path2, dockerfile_bash)
			} else if data.ProjectType == "python" || data.ProjectType == "notebook" {
				createStub(dockerfile_path2, dockerfile)
			} else if data.ProjectType == "webapp" {
				createStub(dockerfile_path2, webapp_dockerfile)
			}

			fmt.Printf("\nInit new project folder \"%s\" done.\n", input_dir)
			fmt.Printf("You might want to add a data folder with DICOM files to get started\n\n\tcd \"%s\"\n\t%s config --data <data folder>\n\n", input_dir, own_name)
			fmt.Println("Careful with using a data folder with too many files. Each time you trigger a\n" +
				"computation ror needs to look at each of the files. This might take\n" +
				"a long time. Test with a few hundred DICOM files first.\n\n" +
				"If you don't have any readily available DICOM data you might want to download some by\n" +
				" mkdir data; cd data;\n" +
				" git clone https://github.com/ImagingInformatics/hackathon-dataset.git\n" +
				" cd hackathon-dataset\n" +
				" git submodule update --init --recursive")
		}
	case "config":
		if len(os.Args[2:]) == 0 {
			configCommand.PrintDefaults()
			return
		}
		if err := configCommand.Parse(os.Args[2:]); err == nil {
			if config_help {
				configCommand.PrintDefaults()
				return
			}

			//fmt.Println("Config")
			// are we init already?
			dir_path := input_dir + "/.ror/config"
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New(errorConfigFile))
			}

			var studies map[string]map[string]SeriesInfo
			if data_path != "" {
				if _, err := os.Stat(data_path); os.IsNotExist(err) {
					// the data path could also be a glob string (has to be enclosed on double quotes)
					files, err := filepath.Glob(data_path)
					if err != nil || len(files) < 1 {
						exitGracefully(errors.New("this data path does not exist or contains no data"))
					}
				}
				// try to play with a tui interface here
				if true {
					newPrimitive := func(text string) *tview.TextView {
						return tview.NewTextView().
							SetTextAlign(tview.AlignLeft).
							SetText(text)
					}
					structure = newPrimitive("")
					structure.SetBorder(true).SetTitle("Database")
					viewer = newPrimitive("")
					footer = newPrimitive(fmt.Sprintf("Start looking for files in %s", data_path))
					footer.SetBorder(true)
					footer.SetTitle("File")
					viewer.SetBorder(true).SetTitle("DICOM")

					flex := tview.NewFlex().SetDirection(tview.FlexRow).
						AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
							AddItem(structure, 30, 1, false).
							AddItem(viewer, 0, 1, true), 0, 1, false).
						AddItem(footer, 3, 1, false)

					tviewrun := func() {
						app = tview.NewApplication()
						if err := app.SetRoot(flex, true).EnableMouse(false).Run(); err != nil {
							fmt.Println("No text user interface provided, fall back to text only")
							// panic(err)
							app = nil
							return
						}
						app.Stop()
					}
					go tviewrun()
					//defer app.Stop()
				}
				//defer app.Stop()
				fmt.Println("")

				config.Data.Path = data_path
				studies, err = dataSets(config)
				check(err)
				app.Stop()
				if len(studies) == 0 {
					fmt.Println("We did not find any DICOM files in the folder you provided. Please check if the files are available, un-compress any zip files to make the accessible to this tool.")
				} else {
					postfix := "ies"
					if len(studies) == 1 {
						postfix = "y"
					}
					fmt.Printf("Found %d DICOM stud%s.\n", len(studies), postfix)
				}

				// update the config file now - the above dataSets can take a long time!
				config, err = readConfig(dir_path)
				if err != nil {
					exitGracefully(errors.New(errorConfigFile))
				}
				config.Data.DataInfo = studies
				config.Data.Path = data_path
				if config_temp_directory == "" {
					fmt.Printf("For testing a workflow you might next want to set the temp directory\n\n\t"+
						"%s config --temp_directory \"<folder>\"\n\nExample trigger data folders will appear there.\n",
						own_name)
				}
			}
			if author_name != "" {
				config.Author.Name = author_name
			}
			if author_email != "" {
				config.Author.Email = author_email
			}
			if project_token != "" {
				config.ProjectToken = project_token
			}
			if config_series_filter != "" {
				// if we want comments we should use /* */, would be good if we can keep them in the code
				// we can remove them before we parse...
				// we might have newlines in the filter string, remove those first before we safe
				//	config_series_filter = strings.Replace(config_series_filter, "\n", "", -1)
				// we might also have too many spaces in the filter string, remove those as well
				//	space := regexp.MustCompile(`\s+`)
				//	config_series_filter := space.ReplaceAllString(config_series_filter, " ")
				comments := regexp.MustCompile("/[*]([^*]|[\r\n]|([*]+([^*/]|[\r\n])))*[*]+/")
				series_filter_no_comments := comments.ReplaceAllString(config_series_filter, " ")

				// now parse the input string
				InitParser()
				//yyErrorVerbose = true
				yyDebug = 1

				line := []byte(series_filter_no_comments)
				yyParse(&exprLex{line: line})
				if !errorOnParse {
					s, _ := json.MarshalIndent(ast, "", "  ")
					ss := humanizeFilter(ast)
					fmt.Printf("Parsing series filter successful\n%s\n%s\n", string(s), ss)
					config.SeriesFilterType = "select"
					// check if we have any matches - cheap for us here
					matches, _ := findMatchingSets(ast, config.Data.DataInfo)
					postfix := "s"
					if len(matches) == 1 {
						postfix = ""
					}
					fmt.Printf("Given our current test data we can identify %d matching dataset%s.\n", len(matches), postfix)

				} else {
					// maybe its a simple glob expression? We should add in any case
					fmt.Println("We tried to parse the series filter but failed. Maybe you just want to grep?")
					config.SeriesFilterType = "glob"
				}
				config.SeriesFilter = config_series_filter
			}
			if call_string != "" {
				//fmt.Println("Set the call string to :", call_string)
				config.CallString = call_string
			}
			if no_sort_dicom {
				config.SortDICOM = false
			} else {
				config.SortDICOM = true
			}
			if project_name_string != "" {
				project_name_string = strings.Replace(project_name_string, " ", "_", -1)
				project_name_string = strings.ToLower(project_name_string)
				config.ProjectName = project_name_string
			}
			if config_temp_directory != "" {
				if _, err := os.Stat(config_temp_directory); os.IsNotExist(err) {
					exitGracefully(errors.New("this temp_directory path does not exist"))
				}
				config.TempDirectory = config_temp_directory
				fmt.Printf("You can trigger a workflow now. Use\n\n\t%s trigger --keep\n\nto leave the data folder in the temp directory for inspection.\n", own_name)
			}
			if config_suggest {
				if config.Data.DataInfo == nil {
					exitGracefully(errors.New("to suggest a selection we need some data first. Use config --data <path to DICOMs>"))
				}

				// get dataset and ast from config
				// create an ast
				// fmt.Println("Suggested abstract syntax tree for your data:")
				InitParser()
				line := []byte("Select series from series where series has ClassifyType containing CT")
				yyParse(&exprLex{line: line})

				ast, _ := ast.improveAST(config.Data.DataInfo)

				//s, l := json.MarshalIndent(ast, "", "  ")
				//fmt.Printf("Suggested abstract syntax tree for your data [%f]\n%s\n", l, string(s))
				fmt.Println(humanizeFilter(ast))

				matches, _ := findMatchingSets(ast, config.Data.DataInfo)
				postfix := "s"
				if len(matches) == 1 {
					postfix = ""
				}
				fmt.Printf("Given our current test data we can identify %d matching dataset%s.\n", len(matches), postfix)
			}
			// write out config again
			if !config.writeConfig() {
				exitGracefully(errors.New("failed to write config file"))
			}
			//file, _ := json.MarshalIndent(config, "", " ")
			//_ = ioutil.WriteFile(dir_path, file, 0600)
		}
	case "status":
		if err := statusCommand.Parse(os.Args[2:]); err == nil {
			if status_help {
				statusCommand.PrintDefaults()
				return
			}

			// we might have a folder name after all the arguments to look into
			values := statusCommand.Args()
			if len(values) == 1 {
				input_dir = statusCommand.Arg(0)
			}

			dir_path := input_dir + "/.ror/config"
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New(errorConfigFile))
			}

			if status_tui {
				// We want to setup a screen where we can see the list of raw data and the list of
				// matching datasets. We want to be able to see the images in the dataset and we want
				// to be able to trigger a workflow.
				var statusTui StatusTUI
				statusTui.dataSets = config.Data.DataInfo
				if config.SeriesFilterType != "select" {
					exitGracefully(errors.New("we can only work with Select filters"))
				}
				InitParser()
				line := []byte(config.SeriesFilter)
				yyParse(&exprLex{line: line})
				statusTui.ast = ast
				statusTui.Init()
			}

			if !status_detailed {
				// remove some info that takes up a lot of space
				// we would like to hide the big field:
				// 	config.Data.DataInfo = nil
				// is not an option as we need the field again later
				// try to make a copy of the config using Marshal and Unmarshal
				tt, err := json.Marshal(config)
				if err == nil {
					var newConfig Config
					json.Unmarshal(tt, &newConfig)
					newConfig.Data.DataInfo = nil     // hide the data
					newConfig.ProjectToken = "hidden" // hide the project token
					file, _ := json.MarshalIndent(newConfig, "", " ")
					fmt.Println(string(file))
				} else {
					fmt.Printf("Error: could not marshal the config again %s", string(tt))
				}
			} else {
				file, _ := json.MarshalIndent(config, "", " ")
				fmt.Println(string(file))
			}
			if status_detailed {
				counterStudy := 0
				// find all patients, sort by them and print out the studies
				var participantsMap map[string]bool = make(map[string]bool)
				for _, element := range config.Data.DataInfo {
					for _, element2 := range element {
						name := element2.PatientID
						if element2.PatientName != "" && element2.PatientName != name {
							name = name + "-" + element2.PatientName
						}
						participantsMap[name] = true
					}
				}
				var participants []string = make([]string, 0, len(participantsMap))
				for k := range participantsMap {
					participants = append(participants, k)
				}
				sort.Strings(participants)

				fmt.Printf("\nData summary\n\n")

				for pidx, p := range participants {
					counter3 := 0
					for key, element := range config.Data.DataInfo {
						counter2 := 0
						// TODO: we should sort element by SeriesNumber
						for key2, element2 := range element {
							name := element2.PatientID
							if element2.PatientName != "" && element2.PatientName != name {
								name = name + "-" + element2.PatientName
							}
							studyDate := ""
							for _, a := range element2.All {
								t := tag.StudyDate
								if a.Tag == t {
									studyDate = strings.Join(a.Value, ",")
									layout := "20060102"
									t, err := time.Parse(layout, studyDate)
									if err == nil {
										studyDate = t.Format("2006/01/02")
									}
									break
								}
							}
							// TODO: This is not correct, it might happen that the PatientName for
							// some of the images is empty. Those would not be printed even
							// if they are in the same study.
							if name != p {
								continue
							}
							counter2 = counter2 + 1
							counter3 = counter3 + 1
							if counter3 == 1 { // change in patient
								fmt.Printf("Patient [%d/%d]: %s\n", pidx+1, len(participants), name)
							}
							if counter2 == 1 { // change in study
								counterStudy = counterStudy + 1
								fmt.Printf("  Study: %s %s (%d/%d)\n",
									studyDate, key, counterStudy,
									len(config.Data.DataInfo))
							}
							var de string = element2.SeriesDescription
							if de == "" {
								de = "no series description"
							} else {
								de = fmt.Sprintf("description \"%s\"", de)
							}
							postfix := "s"
							if element2.NumImages == 1 {
								postfix = ""
							}
							fmt.Printf("    %s (%d/%d) %d %s image%s, series: %d, %s\n",
								key2,
								counter2,
								len(element),
								element2.NumImages,
								element2.Modality,
								postfix,
								element2.SeriesNumber,
								de)
						}
					}
					fmt.Println("")
				}
			} else {
				fmt.Println("This short status does not contain data information. Use the --all option to obtain all info.")
			}
			if status_detailed && config.SeriesFilterType == "select" {
				comments := regexp.MustCompile("/[*]([^*]|[\r\n]|([*]+([^*/]|[\r\n])))*[*]+/")
				series_filter_no_comments := comments.ReplaceAllString(config.SeriesFilter, " ")

				// now parse the input string
				InitParser()
				line := []byte(series_filter_no_comments)
				yyParse(&exprLex{line: line})
				if !errorOnParse {
					s, _ := json.MarshalIndent(ast, "", "  ")
					ss := humanizeFilter(ast)
					fmt.Printf("Parsing series filter successful\n%s\n%s\n", string(s), ss)
					config.SeriesFilterType = "select"
					// check if we have any matches - cheap for us here
					matches, _ := findMatchingSets(ast, config.Data.DataInfo)
					postfix := "s"
					if len(matches) == 1 {
						postfix = ""
					}
					fmt.Printf("Given our current test data we can identify %d matching dataset%s.\n", len(matches), postfix)
				}
			}
			// we can suggest a select filter for this dataset
			if false {
				// get dataset and ast from config
				// create an ast
				// fmt.Println("Suggested abstract syntax tree for your data:")
				InitParser()
				line := []byte("Select series from series where series has ClassifyType containing CT")
				yyParse(&exprLex{line: line})

				ast, l := ast.improveAST(config.Data.DataInfo)

				s, _ := json.MarshalIndent(ast, "", "  ")
				fmt.Printf("Suggested abstract syntax tree for your data [%f]\n%s\n", l, string(s))
				fmt.Println(humanizeFilter(ast))

				matches, _ := findMatchingSets(ast, config.Data.DataInfo)
				postfix := "s"
				if len(matches) == 1 {
					postfix = ""
				}
				fmt.Printf("Given our current test data we can identify %d matching dataset%s.\n", len(matches), postfix)
			}
		}
	case "trigger":
		if err := triggerCommand.Parse(os.Args[2:]); err == nil {
			if trigger_help {
				triggerCommand.PrintDefaults()
				return
			}

			dir_path := input_dir + "/.ror/config"
			// we have a couple of example datasets that we can select
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New(errorConfigFile))
			}

			if trigger_last {
				// we would like to run a specific folder with the call string
				folder := config.LastDataFolder
				if folder == "" {
					exitGracefully(fmt.Errorf("there is no folder with data. Create one with 'ror trigger --keep'"))
				}
				if _, err := os.Stat(folder); os.IsNotExist(err) {
					exitGracefully(fmt.Errorf("%s could not be found. Create one with 'ror trigger --keep'", folder))
				}
				callProgram(config, triggerWaitTime, trigger_container, folder)
			}

			// make sure we have updated classifyRules.json loaded here ... just in case if the user
			// puts his/her own rules into .ror/classifyRules.json
			classifyDICOM_path := input_dir + "/.ror/classifyDICOM.json"
			if _, err := os.Stat(classifyDICOM_path); !os.IsNotExist(err) {
				// read the classifyDICOM
				classifyDICOMFile, err := os.Open(classifyDICOM_path)
				if err != nil {
					classifyRules_new_set, err := ioutil.ReadAll(classifyDICOMFile)
					if err != nil {
						classifyRules = string(classifyRules_new_set)
					}
				}
			}

			selectFromA := make(map[string]string)
			// we can have sets of values to export, so instead of a single series we should have here
			// a list of series instance uids. In case we export by series we have a single entry, if
			// we export on the study or patient level we have more series. Picking one entry means
			// exporting all the series in the entry.
			var selectFromB [][]string = nil
			var selectFromBNames [][]string = nil
			for StudyInstanceUID, value := range config.Data.DataInfo {
				for SeriesInstanceUID, value2 := range value {
					selectFromA[SeriesInstanceUID] = fmt.Sprintf("StudyInstanceUID: %s, SeriesInstanceUID: %s, SeriesDescription: %s, "+
						"NumImages: %d, SeriesNumber: %d, SequenceName: %s, Modality: %s, Manufacturer: %s, ManufacturerModelName: %s, "+
						"StudyDescription: %s, PatientID: %s, PatientName: %s, ClassifyType: %s",
						StudyInstanceUID, SeriesInstanceUID, value2.SeriesDescription, value2.NumImages, value2.SeriesNumber, value2.SequenceName, value2.Modality,
						value2.Manufacturer, value2.ManufacturerModelName, value2.StudyDescription, value2.PatientID, value2.PatientName,
						strings.Join(value2.ClassifyTypes, " "),
					)
				}
			}
			if len(selectFromA) == 0 {
				exitGracefully(fmt.Errorf("there is no data. Did you forget to specify a data folder?\n\n\t%s config --data <folder>", own_name))
			}

			// check if we have a trivial filter (glob) or a proper rule filter
			if config.SeriesFilterType == "glob" {
				mm := regexp.MustCompile(config.SeriesFilter)
				for key, value := range selectFromA {
					if mm.MatchString(value) {
						selectFromB = append(selectFromB, []string{key})
						selectFromBNames = append(selectFromBNames, []string{"no-name"})
					}
				}
			} else if config.SeriesFilterType == "select" {
				// We need to do things differently if we select Output_level that is not
				// "series"
				comments := regexp.MustCompile("/[*]([^*]|[\r\n]|([*]+([^*/]|[\r\n])))*[*]+/")
				series_filter_no_comments := comments.ReplaceAllString(config.SeriesFilter, " ")

				// its a rule so behave accordingly, check for each rule set if the current series matches
				InitParser()
				line := []byte(series_filter_no_comments)
				yyParse(&exprLex{line: line})
				if !errorOnParse {
					//if ast.Output_level != "series" && ast.Output_level != "study" {
					//	exitGracefully(fmt.Errorf("we only support \"Select <series>\" and \"Select <study>\" for now as the output level"))
					//}
					selectFromB, selectFromBNames = findMatchingSets(ast, config.Data.DataInfo)
					fmt.Printf("NAMES ARE: %v\n", selectFromBNames)
				}
				//s, _ = json.MarshalIndent(ast, "", "  ")
				//fmt.Printf("ast is: %s\n", string(s))

			} else {
				exitGracefully(fmt.Errorf("Error: unknown SeriesFilterType"))
			}
			if len(selectFromB) == 0 {
				exitGracefully(fmt.Errorf("found %d series, but there is no matching data after applying your series_filter. Did you specify a filter that does not work or is too restrictive?\n\n\t%s\n\n ", len(selectFromA), config.SeriesFilter))
			}
			// if trigger_each we want to run this for all of them, not just a single one
			var runIdx []int
			// if we are on the series level we export a single series here, but we can also be on the study or patient level and export more
			if !trigger_each {
				runIdx = []int{rand.Intn((len(selectFromB) - 0) + 0)}
			} else {
				runIdx = []int{0}
				for i := 1; i < len(selectFromB); i++ {
					runIdx = append(runIdx, i)
				}
			}
			output_json_array := []string{}
			for _, idx := range runIdx {
				fmt.Printf("found %d matching series sets. Picked index %d, trigger series: %s\n", len(selectFromB), idx, strings.Join(selectFromB[idx], ", "))

				dir, err := ioutil.TempDir(config.TempDirectory, fmt.Sprintf("ror_trigger_run_%s_*", time.Now().Weekday()))
				if err != nil {
					fmt.Printf("%s", err)
					exitGracefully(errors.New("could not create the temporary directory for the trigger"))
				}
				if !trigger_keep {
					defer os.RemoveAll(dir)
				} else {
					fmt.Printf("trigger data directory is \"%s\"\n", dir)
				}
				if trigger_keep {
					// change the LastDataFolder in config
					dir_path := input_dir + "/.ror/config"
					// we have a couple of example datasets that we can select
					config, err := readConfig(dir_path)
					if err != nil {
						exitGracefully(errors.New(errorConfigFile))
					}
					config.LastDataFolder = dir
					// and save again
					if !config.writeConfig() {
						exitGracefully(errors.New(errorConfigFile))
					}
					//file, _ := json.MarshalIndent(config, "", " ")
					//_ = ioutil.WriteFile(dir_path, file, 0600)
				}

				// we should copy all files into this directory that we need for processing
				// the study we want is this one selectFromB[idx]
				// look for the path stored in that study

				// export each series from the current set in selectFromB
				var description []Description
				for idx2, thisSeriesInstanceUID := range selectFromB[idx] {
					var closestPath string = ""
					var classifyTypes []string
					for _, value := range config.Data.DataInfo {
						for SeriesInstanceUID, value2 := range value {
							if SeriesInstanceUID == thisSeriesInstanceUID {
								closestPath = value2.Path
								classifyTypes = value2.ClassifyTypes
							}
						}
					}
					if closestPath == "" {
						fmt.Println("ERROR: Could not detect the closest PATH")
						closestPath = config.Data.Path
					}

					numFiles, descr := copyFiles(thisSeriesInstanceUID, closestPath, dir, config.SortDICOM, classifyTypes)
					descr.NameFromSelect = selectFromBNames[idx][idx2]
					// we should merge the different descr together to get description
					description = append(description, descr)
					fmt.Println("Found", numFiles, "files.")
				}
				// write out a description
				file, _ := json.MarshalIndent(description, "", " ")
				_ = ioutil.WriteFile(dir+"/descr.json", file, 0644)
				if !trigger_test {
					// check if the call string is empty
					callProgram(config, triggerWaitTime, trigger_container, dir)

					// we can check if we have an output folder now
					path_string := dir + "/output/output.json"
					if _, err := os.Stat(path_string); err != nil && !os.IsNotExist(err) {
						exitGracefully(fmt.Errorf("run finished but no output/output.json file found. Consider creating such a file in your program"))
					}

					// plot the output.json as a result object to screen
					jsonFile, err := os.Open(path_string)
					// if we os.Open returns an error then handle it
					if err != nil {
						fmt.Println(err)
					}
					//fmt.Println("Successfully Opened users.json")
					// defer the closing of our jsonFile so that we can parse it later on
					defer jsonFile.Close()

					byteValue, _ := ioutil.ReadAll(jsonFile)
					output_json_array = append(output_json_array, string(byteValue))
					//fmt.Println(string(byteValue))

					//fmt.Println("Done.")
				} else {
					fmt.Printf("Test only. Make sure you also use '--keep' and call something like this:\n\t%s %s\n", config.CallString, dir)
				}
			}
			if !trigger_test {
				fmt.Println("[", strings.Join(output_json_array[:], ", "), "]")
			}
		}
	case "build":
		if err := buildCommand.Parse(os.Args[2:]); err == nil {
			if build_help {
				buildCommand.PrintDefaults()
				return
			}
			// we should just gather the requirements for now
			dir_path := input_dir + "/.ror/config"
			// we have a couple of example datasets that we can select
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New(errorConfigFile))
			}
			projectName := config.ProjectName
			// remove any spaces in the project name, make it lower-case
			projectName = strings.Replace(projectName, " ", "_", -1)
			projectName = strings.ToLower(projectName)
			fmt.Println("\nWe will assume a python/pip based workflow and fall back to using conda.")
			fmt.Println("There is no automated build yet, please follow these instructions.")
			fmt.Println("\nThere are only two steps that need to be done, create a list of")
			fmt.Println("requirements and build a container. Run pip freeze to update the")
			fmt.Println("list of python packages (requires pip):")
			fmt.Println("\n\tpip list --format=freeze >", path.Join(input_dir, ".ror", "virt", "requirements.txt"))
			fmt.Println("\nCreate a container of your workflow:")
			fmt.Println("\n\tdocker build --no-cache -t", fmt.Sprintf("workflow_%s", projectName), "-f", path.Join(input_dir, ".ror", "virt", "Dockerfile"), ".")
			fmt.Println("\nNote: This build might fail if pip is not able to resolve all the requirements")
			//fmt.Println("In this case it might help to update all packages first with something like:")
			//fmt.Println("\n\tpip list --outdated --format=freeze | grep -v '^\\-e' | cut -d = -f 1 | xargs -n1 pip install -U ")
			fmt.Println("inside the container. If that is the case it is best to use a virtual environment.")
			fmt.Println("The list of dependencies inside a new virtual environment easier to handle as only")
			fmt.Println("the essential packages for your workflow will be part of the container.")
			fmt.Println("\nCreate a new conda environment with")
			fmt.Printf("\n\tconda create --name workflow_%s python=3.8\n", projectName)
			fmt.Printf("\tconda activate workflow_%s\n", projectName)
			fmt.Printf("\tconda install -c conda-forge pydicom numpy matplotlib\n")
			fmt.Printf("\nAdjust the list of packages based on your workflow. The above list should be\n")
			fmt.Printf("sufficient for the default workflow. Now repeat the above steps.\n")
			fmt.Println("\nA corresponding yml file can be created with:")
			fmt.Println("\n\tconda env export --name \"name\" >", path.Join(input_dir, ".ror", "virt", "requirements.yml"))
			fmt.Println("\nwhere \"name\" is the name of your docker environment. Create the container now with:")
			fmt.Println("\n\tdocker build --no-cache --build-arg conda-env=\"name\" -t", fmt.Sprintf("workflow_%s", projectName), "-f", path.Join(input_dir, ".ror", "virt", "Dockerfile"), ".")

			fmt.Println("\n\nSimulate a docker based processing workflow using one of the trigger generated folders:")
			abs_temp_path, err := filepath.Abs(config.TempDirectory)
			if err != nil {
				fmt.Println("error computing the absolution path of the temp_directory")
			}
			// is there a ror_trigger folder?
			folders, err := filepath.Glob(fmt.Sprintf("%s/ror_trigger_run_*", abs_temp_path))
			var folder string
			if err != nil || len(folders) < 1 {
				fmt.Printf("Error: Could not find an example data folder in the temp directory %s.\n\tCreate one with\n\n\t%s trigger --keep\n\n",
					abs_temp_path, own_name)
				folder = "<ror_trigger_run_folder>"
			} else {
				// first folder found in temp_directory
				folder = folders[0]
			}

			fmt.Println("\n\tdocker run --rm -it \\\n\t",
				"-v", fmt.Sprintf("\"%s/%s\":/data", abs_temp_path, filepath.Base(folder)), "\\\n\t",
				fmt.Sprintf("workflow_%s", projectName),
				"/bin/bash -c", fmt.Sprintf("\"cd /app; %s /data/\"", config.CallString),
			)
			fmt.Println("")
			fmt.Println("If the above call was sufficient to run your workflow, we can now submit.")
		}
	case "annotate":
		if err := annotateCommand.Parse(os.Args[2:]); err == nil {
			if annotate_help {
				annotateCommand.PrintDefaults()
				return
			}
			dir_path := input_dir + "/.ror/config"
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New(errorConfigFile))
			}
			// add the ontology
			if annotate_ontology != "" {
				if _, err := os.Stat(annotate_ontology); err != nil && os.IsNotExist(err) {
					fmt.Printf("file %s does not exist", annotate_ontology)
				} else {
					fi, err := os.Open(annotate_ontology)
					if err == nil {
						byteValue, err := io.ReadAll(fi)
						if err != nil {
							log.Fatal(err)
						}
						json.Unmarshal(byteValue, &config.Annotate.Ontology)

						if !config.writeConfig() {
							exitGracefully(errors.New("failed to write config file"))
						}
					}
					defer fi.Close()
				}
			}

			var annotate_tui bool = true
			if annotate_tui {
				// We want to setup a screen where we can see the list of raw data and the list of
				// matching datasets. We want to be able to see the images in the dataset and we want
				// to be able to trigger a workflow.
				var annotateTui AnnotateTUI
				annotateTui.dataSets = config.Data.DataInfo
				annotateTui.ontology = config.Annotate.Ontology
				if config.SeriesFilterType != "select" {
					exitGracefully(errors.New("Need a Select filter, use 'ror config --suggest' to create one."))
				}
				InitParser()
				line := []byte(config.SeriesFilter)
				yyParse(&exprLex{line: line})
				annotateTui.ast = ast
				annotateTui.Init()
			}

		}
	default:
		// fall back to parsing without a command
		flag.Parse()
		if show_version {
			timeThen := time.Now()
			setTime := false
			if compileDate != "" {
				layout := ".20060102.150405"
				t, err := time.Parse(layout, compileDate)
				if err == nil {
					timeThen = t
					setTime = true
				}
			}

			fmt.Printf("ror version %s%s", version, compileDate)
			if setTime {
				fmt.Printf(" build %.0f days ago\n", math.Round(time.Since(timeThen).Hours()/24))
			} else {
				fmt.Println()
			}
			os.Exit(0)
		} else {
			flag.Usage()
			os.Exit(-1)
		}
	}
}
