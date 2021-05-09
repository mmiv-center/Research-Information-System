// Code written 2021 by Hauke Bartsch.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

//go:embed templates/README.md
var readme string

type inputFile struct {
	filepath  string
	separator string
	pretty    bool
}

func exitGracefully(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func check(e error) {
	if e != nil {
		exitGracefully(e)
	}
}

func getJSONFunc(pretty bool) (func(map[string]string) string, string) {
	// Declaring the variables we're going to return at the end
	var jsonFunc func(map[string]string) string
	var breakLine string
	if pretty { //Pretty is enabled, so we should return a well-formatted JSON file (multi-line)
		breakLine = "\n"
		jsonFunc = func(record map[string]string) string {
			jsonData, _ := json.MarshalIndent(record, "   ", "   ") // By doing this we're ensuring the JSON generated is indented and multi-line
			return "   " + string(jsonData)                         // Transforming from binary data to string and adding the indent characets to the front
		}
	} else { // Now pretty is disabled so we should return a compact JSON file (one single line)
		breakLine = "" // It's an empty string because we never break lines when adding a new JSON object
		jsonFunc = func(record map[string]string) string {
			jsonData, _ := json.Marshal(record) // Now we're using the standard Marshal function, which generates JSON without formating
			return string(jsonData)             // Transforming from binary data to string
		}
	}

	return jsonFunc, breakLine // Returning everythinbg
}

func createStringWriter(csvPath string) func(string, bool) {
	jsonDir := filepath.Dir(csvPath)                                                       // Getting the directory where the CSV file is
	jsonName := fmt.Sprintf("%s.json", strings.TrimSuffix(filepath.Base(csvPath), ".csv")) // Declaring the JSON filename, using the CSV file name as base
	finalLocation := filepath.Join(jsonDir, jsonName)                                      // Declaring the JSON file location, using the previous variables as base
	// Opening the JSON file that we want to start writing
	f, err := os.Create(finalLocation)
	check(err)
	// This is the function we want to return, we're going to use it to write the JSON file
	return func(data string, close bool) { // 2 arguments: The piece of text we want to write, and whether or not we should close the file
		_, err := f.WriteString(data) // Writing the data string into the file
		check(err)
		// If close is "true", it means there are no more data left to be written, so we close the file
		if close {
			f.Close()
		}
	}
}

func writeJSONFile(csvPath string, writerChannel <-chan map[string]string, done chan<- bool, pretty bool) {
	writeString := createStringWriter(csvPath) // Instanciating a JSON writer function
	jsonFunc, breakLine := getJSONFunc(pretty) // Instanciating the JSON parse function and the breakline character
	// Log for informing
	fmt.Println("Writing JSON file...")
	// Writing the first character of our JSON file. We always start with a "[" since we always generate array of record
	writeString("["+breakLine, false)
	first := true
	for {
		// Waiting for pushed records into our writerChannel
		record, more := <-writerChannel
		if more {
			if !first { // If it's not the first record, we break the line
				writeString(","+breakLine, false)
			} else {
				first = false // If it's the first one, we don't break the line
			}

			jsonData := jsonFunc(record) // Parsing the record into JSON
			writeString(jsonData, false) // Writing the JSON string with our writer function
		} else { // If we get here, it means there aren't more record to parse. So we need to close the file
			writeString(breakLine+"]", true) // Writing the final character and closing the file
			fmt.Println("Completed!")        // Logging that we're done
			done <- true                     // Sending the signal to the main function so it can correctly exit out.
			break                            // Stoping the for-loop
		}
	}
}

func processLine(headers []string, dataList []string) (map[string]string, error) {
	// Validating if we're getting the same number of headers and columns. Otherwise, we return an error
	if len(dataList) != len(headers) {
		return nil, errors.New("Line doesn't match headers format. Skipping")
	}
	// Creating the map we're going to populate
	recordMap := make(map[string]string)
	// For each header we're going to set a new map key with the corresponding column value
	for i, name := range headers {
		recordMap[name] = dataList[i]
	}
	// Returning our generated map
	return recordMap, nil
}

func processCsvFile(fileData inputFile, writerChannel chan<- map[string]string) {
	// Opening our file for reading
	file, err := os.Open(fileData.filepath)
	// Checking for errors, we shouldn't get any
	check(err)
	// Don't forget to close the file once everything is done
	defer file.Close()

	// Defining a "headers" and "line" slice
	var headers, line []string
	// Initializing our CSV reader
	reader := csv.NewReader(file)
	// The default character separator is comma (,) so we need to change to semicolon if we get that option from the terminal
	if fileData.separator == "semicolon" {
		reader.Comma = ';'
	}
	// Reading the first line, where we will find our headers
	headers, err = reader.Read()
	check(err) // Again, error checking
	// Now we're going to iterate over each line from the CSV file
	for {
		// We read one row (line) from the CSV.
		// This line is a string slice, with each element representing a column
		line, err = reader.Read()
		// If we get to End of the File, we close the channel and break the for-loop
		if err == io.EOF {
			close(writerChannel)
			break
		} else if err != nil {
			exitGracefully(err) // If this happens, we got an unexpected error
		}
		// Processiong a CSV line
		record, err := processLine(headers, line)

		if err != nil { // If we get an error here, it means we got a wrong number of columns, so we skip this line
			fmt.Printf("Line: %sError: %s\n", line, err)
			continue
		}
		// Otherwise, we send the processed record to the writer channel
		writerChannel <- record
	}
}

func checkIfValidFile(filename string) (bool, error) {
	// Checking if entered file is CSV by using the filepath package from the standard library
	if fileExtension := filepath.Ext(filename); fileExtension != ".csv" {
		return false, fmt.Errorf("File %s is not CSV", filename)
	}

	// Checking if filepath entered belongs to an existing file. We use the Stat method from the os package (standard library)
	if _, err := os.Stat(filename); err != nil && os.IsNotExist(err) {
		return false, fmt.Errorf("File %s does not exist", filename)
	}
	// If we get to this point, it means this is a valid file
	return true, nil
}

func getFileData() (inputFile, error) {
	// We need to validate that we're getting the correct number of arguments
	if len(os.Args) < 2 {
		return inputFile{}, errors.New("A filepath argument is required")
	}

	// Defining option flags. For this, we're using the Flag package from the standard library
	// We need to define three arguments: the flag's name, the default value, and a short description (displayed whith the option --help)
	separator := flag.String("separator", "comma", "Column separator")
	pretty := flag.Bool("pretty", false, "Generate pretty JSON")

	flag.Parse() // This will parse all the arguments from the terminal

	fileLocation := flag.Arg(0) // The only argument (that is not a flag option) is the file location (CSV file)

	// Validating whether or not we received "comma" or "semicolon" from the parsed arguments.
	// If we dind't receive any of those. We should return an error
	if !(*separator == "comma" || *separator == "semicolon") {
		return inputFile{}, errors.New("Only comma or semicolon separators are allowed")
	}

	// If we get to this endpoint, our programm arguments are validated
	// We return the corresponding struct instance with all the required data
	return inputFile{fileLocation, *separator, *pretty}, nil
}

type AuthorInfo struct {
	Name, Email string
}

type Config struct {
	Date   string
	Data   string
	Author AuthorInfo
}

// readConfig parses a provided config file as JSON.
// It returns the parsed code as a marshaled structure.
func readConfig(path_string string) (Config, error) {
	// todo: check directories up as well
	if _, err := os.Stat(path_string); err != nil && os.IsNotExist(err) {
		return Config{}, fmt.Errorf("File %s does not exist", path_string)
	}
	jsonFile, err := os.Open(path_string)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
		return Config{}, fmt.Errorf("Could not open the file ", path_string)
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

// dataSets parses the config.Data path for DICOM files.
// It returns the detected studies and series as collections of paths.
func dataSets(config Config) map[string]int {
	var datasets = make(map[string]int)

	err := filepath.Walk(config.Data, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return err
		}
		dataset, err := dicom.ParseFile(path, nil) // See also: dicom.Parse which has a generic io.Reader API.
		if err == nil {
			StudyInstanceUIDVal, err := dataset.FindElementByTag(tag.StudyInstanceUID)
			if err == nil {
				StudyInstanceUID := dicom.MustGetStrings(StudyInstanceUIDVal.Value)[0]
				if val, ok := datasets[StudyInstanceUID]; ok {
					datasets[StudyInstanceUID] = val + 1
				} else {
					datasets[StudyInstanceUID] = 1
				}
			}
		}

		return err
	})
	if err != nil {
		fmt.Println("Warning: could not walk this path")
	}

	return datasets
}

func main() {

	const (
		defaultInputDir    = "Specify where you want to setup shop"
		defaultTriggerTime = "now"
	)

	initCommand := flag.NewFlagSet("init", flag.ContinueOnError)
	configCommand := flag.NewFlagSet("config", flag.ContinueOnError)
	triggerCommand := flag.NewFlagSet("trigger", flag.ContinueOnError)
	statusCommand := flag.NewFlagSet("status", flag.ContinueOnError)

	var input_dir string
	initCommand.StringVar(&input_dir, "input_dir", ".", defaultInputDir)
	initCommand.StringVar(&input_dir, "i", ".", defaultInputDir)
	var author_name string
	configCommand.StringVar(&author_name, "author_name", "", "Your name.")
	initCommand.StringVar(&author_name, "author_name", "", "Your name.")
	var author_email string
	configCommand.StringVar(&author_email, "author_email", "", "Your email.")
	initCommand.StringVar(&author_email, "author_email", "", "Your email.")
	var data_path string
	configCommand.StringVar(&data_path, "data", "data", "Path to a folder with folders of DICOM files.")

	var trigger string
	triggerCommand.StringVar(&trigger, "trigger", "now", defaultTriggerTime)

	// Showing useful information when the user enters the --help option
	flag.Usage = func() {
		fmt.Printf("Usage: %s [init|trigger|status|config] [options]\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(-1)
	}

	switch os.Args[1] {
	case "init":
		if err := initCommand.Parse(os.Args[2:]); err == nil {
			fmt.Println("Asked to init, in directory:", input_dir)

			if _, err := os.Stat(input_dir); os.IsNotExist(err) {
				if err := os.Mkdir(input_dir, 0755); os.IsExist(err) {
					exitGracefully(errors.New("Directory exist already."))
				}
			}

			dir_path := input_dir + "/.rpp"
			if _, err := os.Stat(dir_path); !os.IsNotExist(err) {
				exitGracefully(errors.New("This directories has already been initialized. Delete the .rpp directory to do this again."))
			} else {
				// do we know the author information?
				if author_name == "" || author_email == "" {
					exitGracefully(errors.New("We need your name and your email with --author_name and --author_email"))
				}

				if err := os.Mkdir(dir_path, 0755); os.IsExist(err) {
					exitGracefully(errors.New("Directory exist already."))
				}
				data := Config{
					Date: time.Now().String(),
					Author: AuthorInfo{
						Name:  author_name,
						Email: author_email,
					},
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
					f.Sync()
				}
				//fmt.Println("Initialized this folder.")
			}
		}
	case "config":
		if err := configCommand.Parse(os.Args[2:]); err == nil {
			//fmt.Println("Config")
			// are we init already?
			dir_path := input_dir + "/.rpp/config"
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New("Could not read the config file"))
			}

			if author_name != "" {
				config.Author.Name = author_name
			}
			if author_email != "" {
				config.Author.Email = author_email
			}
			if data_path != "" {
				if _, err := os.Stat(data_path); os.IsNotExist(err) {
					exitGracefully(errors.New("This data path does not exist."))
				}
				config.Data = data_path
			}
			// write out config again
			file, _ := json.MarshalIndent(config, "", " ")
			_ = ioutil.WriteFile(dir_path, file, 0644)
		}
	case "status":
		if err := statusCommand.Parse(os.Args[2:]); err == nil {
			dir_path := input_dir + "/.rpp/config"
			config, err := readConfig(dir_path)
			if err != nil {
				exitGracefully(errors.New("Could not read the config file"))
			}
			file, _ := json.MarshalIndent(config, "", " ")
			fmt.Println(string(file))

			studies := dataSets(config)
			for key, element := range studies {
				fmt.Println("Study:", key, "num image:", element)
			}
		}
	case "trigger":
		if err := triggerCommand.Parse(os.Args[2:]); err == nil {
			fmt.Println("Asked to trigger")
			fmt.Println("TOBD")
		}
	}

	//if input_dir == "" {
	//	exitGracefully(errors.New("A location to create is required for init"))
	//}

	// Declaring the channels that our go-routines are going to use
	//writerChannel := make(chan map[string]string)
	//done := make(chan bool)
	// Running both of our go-routines, the first one responsible for reading and the second one for writing
	//go processCsvFile(fileData, writerChannel)
	//go writeJSONFile(fileData.filepath, writerChannel, done, fileData.pretty)
	// Waiting for the done channel to receive a value, so that we can terminate the programn execution
	//<-done
}
