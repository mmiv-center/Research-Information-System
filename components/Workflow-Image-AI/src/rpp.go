// Code written 2021 by Hauke Bartsch.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
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
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"golang.org/x/image/draw"

	"image/color"
	_ "image/jpeg"
)

const version string = "0.0.1"

var own_name string = "rpp"

//go:embed templates/README.md
var readme string

//go:embed templates/stub.py
var stub_py string

//go:embed templates/stub.sh
var stub_sh string

//go:embed templates/requirements.txt
var requirements string

//go:embed templates/Dockerfile
var dockerfile string

//go:embed templates/.dockerignore
var dockerignore string

//go:embed templates/docker-compose.yml
var dockercompose string

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

type Config struct {
	Date          string
	Data          DataInfo
	SeriesFilter  string
	Author        AuthorInfo
	TempDirectory string
	CallString    string
	ProjectName   string
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
}

// readConfig parses a provided config file as JSON.
// It returns the parsed code as a marshaled structure.
func readConfig(path_string string) (Config, error) {
	// todo: check directories up as well
	if _, err := os.Stat(path_string); err != nil && os.IsNotExist(err) {
		return Config{}, fmt.Errorf("file %s does not exist", path_string)
	}
	jsonFile, err := os.Open(path_string)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
		return Config{}, fmt.Errorf("could not open the file %s", path_string)
	}
	//fmt.Println("Successfully Opened users.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize our Users array
	var config Config

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &config)
	return config, nil
}

type Description struct {
	SeriesInstanceUID string
	SeriesDescription string
	NumFiles          int
	PatientID         string
	PatientName       string
	SequenceName      string
	StudyDate         string
	StudyTime         string
	SeriesTime        string
	SeriesNumber      string
	ProcessDataPath   string
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

func reverse(s string) string {
	o := make([]rune, utf8.RuneCountInString(s))
	i := len(o)
	for _, c := range s {
		i--
		o[i] = c
	}
	return string(o)
}

func printImage2ASCII(img image.Image, w, h int) []byte {
	//table := []byte(reverse(ASCIISTR))
	table := []byte(reverse(ASCIISTR2))
	//table := []byte(ASCIISTR3)
	buf := new(bytes.Buffer)

	g := color.Gray16Model.Convert(img.At(0, 0))
	maxVal := reflect.ValueOf(g).FieldByName("Y").Uint()
	minVal := maxVal

	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			g := color.Gray16Model.Convert(img.At(j, i))
			//g := img.At(j, i)
			y := reflect.ValueOf(g).FieldByName("Y").Uint()
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
	var histogram [1024]int
	bins := len(histogram)
	for i := 0; i < bins; i++ {
		histogram[i] = 0
	}

	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			g := color.Gray16Model.Convert(img.At(j, i))
			//g := img.At(j, i)
			y := reflect.ValueOf(g).FieldByName("Y").Uint()
			//if math.IsInf(float64(y), 0) || math.IsNaN(float64(y)) {
			//	continue
			//}
			idx := int(math.Round((float64(y) - float64(minVal)) / float64(maxVal-minVal) * float64(bins-1)))
			idx = int(math.Min(float64(bins)-1, math.Max(0, float64(idx))))
			histogram[idx] += 1
		}
	}
	//fmt.Println(histogram)
	// compute the 2%, 99% borders in the cumulative density
	sum := histogram[0]
	for i := 1; i < bins; i++ {
		sum += histogram[i]
	}
	min2 := 0
	s := histogram[0]
	for i := 1; i < bins; i++ {
		if float32(s) >= (float32(sum) * 2.0 / 100.0) { // sum / 100 = ? / 2
			min2 = int(minVal) + int(float32(i)/float32(bins)*float32(maxVal-minVal))
			break
		}
		s += histogram[i]
	}
	max99 := 0
	s = histogram[0]
	for i := 1; i < bins; i++ {
		if float32(s) >= (float32(sum) * 98.0 / 100.0) { // sum / 100 = ? / 2
			max99 = int(minVal) + int(float32(i)/float32(bins)*float32(maxVal-minVal))
			break
		}
		s += histogram[i]
	}
	//fmt.Println("min2:", min2, "max99:", max99, "true min:", minVal, "true max:", maxVal)

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
			y := reflect.ValueOf(g).FieldByName("Y").Uint()
			//fmt.Println("got a number: ", img.At(j, i))
			pos := int((float32(y) - float32(min2)) * float32(len(table)-1) / float32(denom))
			pos = int(math.Min(float64(len(table)-1), math.Max(0, float64(pos))))
			_ = buf.WriteByte(table[pos])
		}
		_ = buf.WriteByte('\n')
	}
	return buf.Bytes()
}

type Converted struct {
	Img image.Image
	Mod color.Model
}

// We return the new color model...
func (c *Converted) ColorModel() color.Model {
	return c.Mod
}

// ... but the original bounds
func (c *Converted) Bounds() image.Rectangle {
	return c.Img.Bounds()
}

// At forwards the call to the original image and
// then asks the color model to convert it.
func (c *Converted) At(x, y int) color.Color {
	return c.Mod.Convert(c.Img.At(x, y))
}

func Scale(src image.Image, rect image.Rectangle, scale draw.Scaler) image.Image {
	dst := image.NewRGBA(rect)
	scale.Scale(dst, rect, src, src.Bounds(), draw.Over, nil)
	return dst
}

func showDataset(dataset dicom.Dataset, counter int, path string) {
	pixelDataElement, err := dataset.FindElementByTag(tag.PixelData)
	if err != nil {
		return
	}
	pixelDataInfo := dicom.MustGetPixelDataInfo(pixelDataElement.Value)
	for _, fr := range pixelDataInfo.Frames {
		fmt.Printf("\033[0;0f") // go to top of the screen
		img, _ := fr.GetImage() // The Go image.Image for this frame

		/*
			// If we would use the native frame we could get access to more levels of detail.
			// That would allow us to use more than 8 bit color depth using ASCII...
			native_img, _ := fr.GetNativeFrame()
			fmt.Println("Bits per sample is ", native_img.BitsPerSample)
			var mi int = native_img.Data[0][0]
			var ma int = mi
			for i:=0; i < native_img.Rows; i++ {
				for j:=0; j < native_img.Cols; j++ {
					currValue := native_img.Data[i*native_img.Cols+j][0]
					if currValue < mi {
						mi = currValue
					}
					if currValue > ma {
						ma = currValue
					}
				}
			}
			fmt.Println("mi ma : ", mi, ma)
		*/

		// gr := &Converted{img, color.RGBAModel /*color.GrayModel*/}

		origbounds := img.Bounds()
		orig_width, orig_height := origbounds.Max.X, origbounds.Max.Y
		// newImage := resize.Resize(196/2 , 196/2 / (80/24), gr.Img, resize.Lanczos3) // should use 80x20 aspect ratio for screen
		// golang.org/x/image/draw
		//newImage := Scale(img, image.Rect(0, 0, 196/2 , int(math.Round(196.0 / 2.0 / (80.0/30.0) ) )), draw.ApproxBiLinear)
		newImage := image.NewGray16(image.Rect(0, 0, 196/2, int(math.Round(196.0/2.0/(80.0/30.0)))))
		draw.ApproxBiLinear.Scale(newImage, image.Rect(0, 0, 196/2, int(math.Round(196.0/2.0/(80.0/30.0)))), img, origbounds, draw.Over, nil)

		bounds := newImage.Bounds()
		width, height := bounds.Max.X, bounds.Max.Y
		p := printImage2ASCII(newImage, width, height)
		fmt.Println(string(p))
		fmt.Printf("[%d] %s (%dx%d)\n", counter+1, path, orig_width, orig_height)
	}
}

// copyFiles will copy all DICOM files that fit the string to the dest_path directory.
// we could display those images as well on the command line - just to impress
func copyFiles(SelectedSeriesInstanceUID string, source_path string, dest_path string) (int, Description) {

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
	counter := 0
	fmt.Printf("\033[2J\n") // clear the screen
	err := filepath.Walk(source_path, func(path string, info os.FileInfo, err error) error {
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

				// we can get a version of the image, scale it and print out on command line
				showImage := true
				if showImage {
					showDataset(dataset, counter+1, path)
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

				outputPath := destination_path
				inputFile, _ := os.Open(path)
				data, _ := ioutil.ReadAll(inputFile)
				outputPathFileName := fmt.Sprintf("%s/%06d.dcm", outputPath, counter)
				ioutil.WriteFile(outputPathFileName, data, 0644)

				// We can do a better destination path here. The friendly way of doing this is
				// to provide separate folders aka the BIDS way.
				// We can create a shadow structure that uses symlinks and sorts everything into
				// sub-folders. Lets name a directory "_" and place the info in that directory.
				symOrder := true
				if symOrder {
					symOrderPath := filepath.Join(destination_path, "_")
					if _, err := os.Stat(symOrderPath); os.IsNotExist(err) {
						err := os.Mkdir(symOrderPath, 0755)
						if err != nil {
							exitGracefully(errors.New("could not create symlink data directory"))
						}
					}
					symOrderPatientPath := filepath.Join(symOrderPath, PatientID+PatientName)
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
					symlink := filepath.Join(symOrderPatientDateSeriesNumber, fmt.Sprintf("%06d.dcm", counter))
					relativeDataPath := fmt.Sprintf("../../../../%06d.dcm", counter)
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
	description.NumFiles = counter
	return counter, description
}

// dataSets parses the config.Data path for DICOM files.
// It returns the detected studies and series as collections of paths.
func dataSets(config Config) (map[string]map[string]SeriesInfo, error) {
	var datasets = make(map[string]map[string]SeriesInfo)
	if config.Data.Path == "" {
		return datasets, fmt.Errorf("no data path for example data has been specified. Use\n\trpp config --data \"path-to-data\" to set such a directory of DICOM data")
	}

	if _, err := os.Stat(config.Data.Path); err != nil && os.IsNotExist(err) {
		exitGracefully(errors.New("data path does not exist"))
	}
	fmt.Println("Found data directory, start parsing DICOM files...")
	counter := 0
	err := filepath.Walk(config.Data.Path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}
		//fmt.Println("look at file: ", path)
		dataset, err := dicom.ParseFile(path, nil) // See also: dicom.Parse which has a generic io.Reader API.
		if err == nil {
			StudyInstanceUIDVal, err := dataset.FindElementByTag(tag.StudyInstanceUID)
			if err == nil {
				var StudyInstanceUID string
				var SeriesInstanceUID string
				var SeriesDescription string
				var SeriesNumber int
				var SequenceName string
				var StudyDescription string
				var Modality string
				var Manufacturer string
				var ManufacturerModelName string

				showImages := true
				if showImages {
					showDataset(dataset, counter, path)
				} else {
					fmt.Printf("%05d files\r", counter)
				}

				counter = counter + 1
				StudyInstanceUID = dicom.MustGetStrings(StudyInstanceUIDVal.Value)[0]
				SeriesInstanceUIDVal, err := dataset.FindElementByTag(tag.SeriesInstanceUID)
				if err == nil {
					SeriesInstanceUID = dicom.MustGetStrings(SeriesInstanceUIDVal.Value)[0]
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

						datasets[StudyInstanceUID][SeriesInstanceUID] = SeriesInfo{NumImages: val.NumImages + 1,
							SeriesDescription:     SeriesDescription,
							SeriesNumber:          SeriesNumber,
							SequenceName:          SequenceName,
							Modality:              Modality,
							Manufacturer:          Manufacturer,
							ManufacturerModelName: ManufacturerModelName,
							StudyDescription:      StudyDescription,
							Path:                  lcp,
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
							Path:                  path_pieces,
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
						Path:                  path_pieces,
					}
				}
			} else {
				return nil
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Warning: could not walk this path")
	}

	return datasets, nil
}

func main() {

	rand.Seed(time.Now().UnixNano())
	// disable logging
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)

	const (
		defaultInputDir    = "Specify where you want to setup shop"
		defaultTriggerTime = "A wait time in seconds or minutes before the computation is triggered"
		errorConfigFile    = "the current directory is not an rpp directory. Change to the correct directory first or create a new folder by running\n\n\trpp init project01"
	)

	initCommand := flag.NewFlagSet("init", flag.ContinueOnError)
	configCommand := flag.NewFlagSet("config", flag.ContinueOnError)
	triggerCommand := flag.NewFlagSet("trigger", flag.ContinueOnError)
	statusCommand := flag.NewFlagSet("status", flag.ContinueOnError)
	buildCommand := flag.NewFlagSet("build", flag.ContinueOnError)

	var input_dir string
	initCommand.StringVar(&input_dir, "input_dir", ".", defaultInputDir)
	//initCommand.StringVar(&input_dir, "i", ".", defaultInputDir)
	var author_name string
	configCommand.StringVar(&author_name, "author_name", "", "Author name used to publish your workflow.")
	initCommand.StringVar(&author_name, "author_name", "", "Author name used to publish your workflow.")
	var author_email string
	configCommand.StringVar(&author_email, "author_email", "", "Author email used to publish your workflow.")
	initCommand.StringVar(&author_email, "author_email", "", "Author email used to publish your workflow.")
	var data_path string
	configCommand.StringVar(&data_path, "data", "", "Path to a folder with DICOM files.")
	var call_string string
	configCommand.StringVar(&call_string, "call", "python ./stub.py", "The command line to call the workflow. A path-name with the data will be appended\n\tto this string.")

	var triggerWaitTime string
	triggerCommand.StringVar(&triggerWaitTime, "delay", "0s", defaultTriggerTime)
	var trigger_test bool
	triggerCommand.BoolVar(&trigger_test, "test", false, "Don't actually run anything, just show what you would do.")
	var trigger_keep bool
	triggerCommand.BoolVar(&trigger_keep, "keep", false, "Keep the created directory around for testing.")

	var status_detailed bool
	statusCommand.BoolVar(&status_detailed, "detailed", false, "Parse the data folder and extract number of studies and series for the trigger.")

	var config_series_filter string
	configCommand.StringVar(&config_series_filter, "series_filter", ".*",
		"Filter applied to series before trigger. This regular expression should\nmatch anything in the string build by StudyInstanceUID: %%s, \nSeriesInstanceUID: %%s, SeriesDescription: %%s, NumImages: %%d, SeriesNumber: %%d\n")

	var config_temp_directory string
	configCommand.StringVar(&config_temp_directory, "temp_directory", "", "Specify a directory for the temporary folders used in the trigger")

	var show_version bool
	flag.BoolVar(&show_version, "version", false, "Show the version number.")

	var user_name string
	user, err := user.Current()
	if err != nil {
		user_name = user.Username
		fmt.Println("got a user name ", user_name)
	}

	own_name = os.Args[0]
	// Showing useful information when the user enters the --help option
	flag.Usage = func() {
		fmt.Printf("RPP - Remote Pipeline Processing\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Println(" A tool to simulate research information system workflows. The program")
		fmt.Println(" can create workflow projects and trigger a processing step similar to")
		fmt.Printf(" automated processing steps run in the research information system.\n\n")
		fmt.Printf("Usage: %s [init|trigger|status|config|build] [options]\n\tStart with init to create a new project folder:\n\n\t%s init <project>\n\n", os.Args[0], os.Args[0])
		fmt.Printf("Option init:\n  Create a workflow project.\n\n")
		initCommand.PrintDefaults()
		fmt.Printf("\nOption config:\n  Configure your project.\n\n")
		configCommand.PrintDefaults()
		fmt.Printf("\nOption status:\n  List the settings of your project.\n\n")
		statusCommand.PrintDefaults()
		fmt.Printf("\nOption trigger:\n  Trigger a processing workflow locally.\n\n")
		triggerCommand.PrintDefaults()
		fmt.Printf("\nOption build:\n  Create a containerized version of your workflow.\n\n")
		buildCommand.PrintDefaults()
		fmt.Println("")
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(-1)
	}

	switch os.Args[1] {
	case "init":
		if err := initCommand.Parse(os.Args[2:]); err == nil {
			// we expect a path first
			values := initCommand.Args()
			if len(values) != 1 {
				exitGracefully(errors.New("we need a single path entry specified"))
			} else {
				input_dir = initCommand.Arg(0)
			}

			if _, err := os.Stat(input_dir); os.IsNotExist(err) {
				if err := os.Mkdir(input_dir, 0755); os.IsExist(err) {
					exitGracefully(errors.New("directory exist already"))
				}
			}

			dir_path := input_dir + "/.rpp"
			if _, err := os.Stat(dir_path); !os.IsNotExist(err) {
				exitGracefully(errors.New("this directories has already been initialized. Delete the .rpp directory to do this again"))
			} else {
				// do we know the author information?
				if author_name == "" || author_email == "" {

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

				if err := os.Mkdir(dir_path, 0755); os.IsExist(err) {
					exitGracefully(errors.New("directory already exists"))
				}
				data := Config{
					Date: time.Now().String(),
					Author: AuthorInfo{
						Name:  author_name,
						Email: author_email,
					},
					CallString:  "python ./stub.py",
					ProjectName: path.Base(input_dir),
				}
				file, _ := json.MarshalIndent(data, "", " ")
				_ = ioutil.WriteFile(dir_path+"/config", file, 0644)

				readme_path := input_dir + "/README.md"
				if _, err := os.Stat(readme_path); !os.IsNotExist(err) {
					fmt.Println("This directory already contains a README.md, don't overwrite. Skip writing...")
				} else {
					f, err := os.Create(readme_path)
					check(err)
					_, err = f.WriteString(readme)
					check(err)
					f.Sync()
				}
				stub_path := input_dir + "/stub.py"
				if _, err := os.Stat(stub_path); !os.IsNotExist(err) {
					fmt.Println("This directory already contains a stub.py, don't overwrite. Skip writing...")
				} else {
					f, err := os.Create(stub_path)
					check(err)
					_, err = f.WriteString(stub_py)
					check(err)
					f.Sync()
				}
				stub_path2 := input_dir + "/stub.sh"
				if _, err := os.Stat(stub_path2); !os.IsNotExist(err) {
					fmt.Println("This directory already contains a stub.sh, don't overwrite. Skip writing...")
				} else {
					f, err := os.Create(stub_path2)
					check(err)
					_, err = f.WriteString(stub_sh)
					check(err)
					f.Sync()
				}
				// virtualization environment
				virt_path := input_dir + "/.rpp/virt"
				if err := os.Mkdir(virt_path, 0755); os.IsExist(err) {
					exitGracefully(errors.New("directory exist already"))
				}
				requirements_path2 := virt_path + "/requirements.txt"
				if _, err := os.Stat(requirements_path2); !os.IsNotExist(err) {
					fmt.Println("This directory already contains a requirements.txt, don't overwrite. Skip writing...")
				} else {
					f, err := os.Create(requirements_path2)
					check(err)
					_, err = f.WriteString(requirements)
					check(err)
					f.Sync()
				}
				dockerignore_path2 := virt_path + "/.dockerignore"
				if _, err := os.Stat(dockerignore_path2); !os.IsNotExist(err) {
					fmt.Println("This directory already contains a .dockerignore, don't overwrite. Skip writing...")
				} else {
					f, err := os.Create(dockerignore_path2)
					check(err)
					_, err = f.WriteString(dockerignore)
					check(err)
					f.Sync()
				}
				dockerfile_path2 := virt_path + "/Dockerfile"
				if _, err := os.Stat(dockerfile_path2); !os.IsNotExist(err) {
					fmt.Println("This directory already contains a Dockerfile, don't overwrite. Skip writing...")
				} else {
					f, err := os.Create(dockerfile_path2)
					check(err)
					_, err = f.WriteString(dockerfile)
					check(err)
					f.Sync()
				}
				dockercompose_path2 := virt_path + "/docker-compose.yml"
				if _, err := os.Stat(dockercompose_path2); !os.IsNotExist(err) {
					fmt.Println("This directory already contains a docker-compose.yml, don't overwrite. Skip writing...")
				} else {
					f, err := os.Create(dockercompose_path2)
					check(err)
					_, err = f.WriteString(dockercompose)
					check(err)
					f.Sync()
				}

			}
			fmt.Printf("\nInit new project folder \"%s\" done.\n", input_dir)
			fmt.Printf("You might want to add a data folder with DICOM files to get started\n\n\tcd \"%s\"\n\t%s config --data <data folder>\n\n", input_dir, own_name)
			fmt.Println("Careful with using a data folder with too many files. Each time you trigger a\n" +
				"computation rpp needs to look at each of the files. This might take\n" +
				"a long time. Test with a few hundred DICOM files first.")
		}
	case "config":
		if err := configCommand.Parse(os.Args[2:]); err == nil {
			//fmt.Println("Config")
			// are we init already?
			dir_path := input_dir + "/.rpp/config"
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New(errorConfigFile))
			}

			var studies map[string]map[string]SeriesInfo
			if data_path != "" {
				if _, err := os.Stat(data_path); os.IsNotExist(err) {
					exitGracefully(errors.New("this data path does not exist"))
				}
				config.Data.Path = data_path
				studies, err = dataSets(config)
				check(err)
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
			if config_series_filter != "" {
				config.SeriesFilter = config_series_filter
			}
			if call_string != "" {
				config.CallString = call_string
			}
			if config_temp_directory != "" {
				if _, err := os.Stat(config_temp_directory); os.IsNotExist(err) {
					exitGracefully(errors.New("this temp_directory path does not exist"))
				}
				config.TempDirectory = config_temp_directory
				fmt.Printf("You can trigger a workflow now. Use\n\n\t%s trigger --keep\n\nto leave the data folder in the temp directory for inspection.\n", own_name)
			}
			// write out config again
			file, _ := json.MarshalIndent(config, "", " ")
			_ = ioutil.WriteFile(dir_path, file, 0644)
		}
	case "status":
		if err := statusCommand.Parse(os.Args[2:]); err == nil {
			// we might have a folder name after all the arguments to look into
			values := statusCommand.Args()
			if len(values) == 1 {
				input_dir = statusCommand.Arg(0)
			}

			dir_path := input_dir + "/.rpp/config"
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New(errorConfigFile))
			}
			file, _ := json.MarshalIndent(config, "", " ")
			fmt.Println(string(file))
			if status_detailed {
				studies, err := dataSets(config)
				check(err)
				// update the config file now
				config, err := readConfig(dir_path)
				if err != nil {
					exitGracefully(errors.New(errorConfigFile))
				}
				config.Data.DataInfo = studies
				file, _ := json.MarshalIndent(config, "", " ")
				_ = ioutil.WriteFile(dir_path, file, 0644)

				for key, element := range studies {
					fmt.Println("Study:", key)
					for key2, element2 := range element {
						fmt.Printf("\t%s num images: %d, series number: %d, description: \"%s\"\n", key2, element2.NumImages, element2.SeriesNumber, element2.SeriesDescription)
					}
				}
			}
		}
	case "trigger":
		if err := triggerCommand.Parse(os.Args[2:]); err == nil {
			dir_path := input_dir + "/.rpp/config"
			// we have a couple of example datasets that we can select
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New(errorConfigFile))
			}

			selectFromA := make(map[string]string)
			var selectFromB []string = nil
			for StudyInstanceUID, value := range config.Data.DataInfo {
				for SeriesInstanceUID, value2 := range value {
					selectFromA[SeriesInstanceUID] = fmt.Sprintf("StudyInstanceUID: %s, SeriesInstanceUID: %s, SeriesDescription: %s, NumImages: %d, SeriesNumber: %d, SequenceName: %s, Modality: %s, Manufacturer: %s, ManufacturerModelName: %s, StudyDescription: %s",
						StudyInstanceUID, SeriesInstanceUID, value2.SeriesDescription, value2.NumImages, value2.SeriesNumber, value2.SequenceName, value2.Modality,
						value2.Manufacturer, value2.ManufacturerModelName, value2.StudyDescription,
					)
				}
			}
			if len(selectFromA) == 0 {
				exitGracefully(fmt.Errorf("there is no data. Did you forget to specify a data folder?\n\n\t%s config --data <folder>", own_name))
			}

			mm := regexp.MustCompile(config.SeriesFilter)
			for key, value := range selectFromA {
				if mm.MatchString(value) {
					selectFromB = append(selectFromB, key)
				}
			}
			if selectFromB == nil {
				exitGracefully(fmt.Errorf("there is no matching data. Did you specify a filter that does not work?\n\n\t%s status", own_name))
			}

			idx := rand.Intn((len(selectFromB) - 0) + 0)
			fmt.Printf("found %d matching series. Picked index %d, trigger series: %s\n", len(selectFromB), idx, selectFromB[idx])

			dir, err := ioutil.TempDir(config.TempDirectory, fmt.Sprintf("rpp_trigger_run_%s_*", time.Now().Weekday()))
			if err != nil {
				fmt.Printf("%s", err)
				exitGracefully(errors.New("could not create the temporary directory for the trigger"))
			}
			if !trigger_keep {
				defer os.RemoveAll(dir)
			} else {
				fmt.Printf("trigger data directory is \"%s\"\n", dir)
			}
			// we should copy all files into this directory that we need for processing
			// the study we want is this one selectFromB[idx]
			// look for the path stored in that study
			var closestPath string = ""
			for _, value := range config.Data.DataInfo {
				for SeriesInstanceUID, value2 := range value {
					if SeriesInstanceUID == selectFromB[idx] {
						closestPath = value2.Path
					}
				}
			}
			if closestPath == "" {
				fmt.Println("ERROR: Could not detect the closest PATH")
				closestPath = config.Data.Path
			}

			numFiles, description := copyFiles(selectFromB[idx], closestPath, dir)
			fmt.Println("Found", numFiles, "files.")
			// write out a description
			file, _ := json.MarshalIndent(description, "", " ")
			_ = ioutil.WriteFile(dir+"/descr.json", file, 0644)
			if !trigger_test {
				// chheck if the call string is empty
				if config.CallString == "" {
					exitGracefully(fmt.Errorf("could not run trigger command, no CallString defined\n\n\t%s config --call \"python ./stub.py\"", own_name))
				}

				// wait for some seconds
				if triggerWaitTime != "" {
					sec, _ := time.ParseDuration(triggerWaitTime)
					time.Sleep(sec)
				}

				cmd_str := config.CallString
				r := regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)`)
				arr := r.FindAllString(cmd_str, -1)
				arr = append(arr, string(dir))
				fmt.Println(arr)
				// cmd := exec.Command("python", "stub.py", dir)
				cmd := exec.Command(arr[0], arr[1:]...)
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
				if _, err := f_log_stdout.WriteString(outb.String()); err != nil {
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
				fmt.Println(string(byteValue))

				//fmt.Println("Done.")
			} else {
				fmt.Println("Test only. Make sure you also use '--keep' and call something like this:\n\tpython ./stub.py " + dir)
			}
		}
	case "build":
		if err := buildCommand.Parse(os.Args[2:]); err == nil {
			// we should just gather the requirements for now
			dir_path := input_dir + "/.rpp/config"
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
			fmt.Println("There is no automated build yet, please follow these instructions:")
			fmt.Println("\nRun pip freeze to update the list of python packages (requires pip):")
			fmt.Println("\n\tpip list --format=freeze >", path.Join(input_dir, ".rpp", "virt", "requirements.txt"))
			fmt.Println("\nCreate a container of your workflow:")
			fmt.Println("\n\tdocker build --no-cache -t", fmt.Sprintf("workflow_%s", projectName), "-f", path.Join(input_dir, ".rpp", "virt", "Dockerfile"), ".")
			fmt.Println("\nThis build might fail if pip is not able to resolve all the requirements inside the container.")
			//fmt.Println("In this case it might help to update all packages first with something like:")
			//fmt.Println("\n\tpip list --outdated --format=freeze | grep -v '^\\-e' | cut -d = -f 1 | xargs -n1 pip install -U ")
			fmt.Println("\nIf the above steps do not work it is best to use a virtual environment.")
			fmt.Println("The list of dependencies inside a new virtual environment easier to handle as only")
			fmt.Println("\nthe essential packages for your workflow will be part of the container.")
			fmt.Println("\nCreate a new conda environment with")
			fmt.Printf("\n\tconda create --name workflow_%s python=3.8\n", projectName)
			fmt.Printf("\tconda activate workflow_%s\n", projectName)
			fmt.Printf("\tconda install -c conda-forge pydicom numpy matplotlib\n")
			fmt.Printf("\nAdjust the list of packages based on your workflow. The above list should be\n")
			fmt.Printf("sufficient for the default workflow. Now repeat the above steps.\n")

			fmt.Println("\nSimulate a docker based processing workflow using one of the trigger generated folders:")
			abs_temp_path, err := filepath.Abs(config.TempDirectory)
			if err != nil {
				fmt.Println("error computing the absolution path of the temp_directory")
			}
			// is there a rpp_trigger folder?
			folders, err := filepath.Glob(fmt.Sprintf("%s/rpp_trigger_run_*", abs_temp_path))
			var folder string
			if err != nil || len(folders) < 1 {
				fmt.Printf("Error: Could not find an example data folder in the temp directory %s.\n\tCreate one with\n\n\t%s trigger --keep\n\n",
					abs_temp_path, own_name)
				folder = "<rpp_trigger_run_folder>"
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
	default:
		// fall back to parsing without a command
		flag.Parse()
		if show_version {
			fmt.Printf("rpp version %s\n", version)
			os.Exit(0)
		}
	}
}
