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
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

const version string = "0.0.1"

//go:embed templates/README.md
var readme string

//go:embed templates/stub.py
var stub_py string

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
}

type SeriesInfo struct {
	SeriesDescription string
	NumImages         int
	SeriesNumber      int
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
	SequenceName      string
}

// copyFiles will copy all DICOM files that fit the string to the dest_path directory.
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
	counter := 0
	err := filepath.Walk(source_path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}
		//fmt.Println("look at file: ", path)
		dataset, err := dicom.ParseFile(path, nil) // See also: dicom.Parse which has a generic io.Reader API.
		if err == nil {
			SeriesInstanceUIDVal, err := dataset.FindElementByTag(tag.SeriesInstanceUID)
			if err == nil {
				var SeriesInstanceUID string
				SeriesInstanceUID = dicom.MustGetStrings(SeriesInstanceUIDVal.Value)[0]
				if SeriesInstanceUID != SelectedSeriesInstanceUID {
					return nil // ignore that file
				}
				fmt.Printf("%05d files\r", counter)
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
				var SequenceName string
				SequenceNameVal, err := dataset.FindElementByTag(tag.SequenceName)
				if err == nil {
					SequenceName = dicom.MustGetStrings(SequenceNameVal.Value)[0]
					if SequenceName != "" {
						description.SequenceName = SequenceName
					}
				}

				outputPath := destination_path
				inputFile, _ := os.Open(path)
				data, _ := ioutil.ReadAll(inputFile)
				ioutil.WriteFile(fmt.Sprintf("%s/%06d.dcm", outputPath, counter), data, 0644)

				counter++
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

				fmt.Printf("%05d files\r", counter)
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
				if _, ok := datasets[StudyInstanceUID]; ok {
					if val, ok := datasets[StudyInstanceUID][SeriesInstanceUID]; ok {
						datasets[StudyInstanceUID][SeriesInstanceUID] = SeriesInfo{NumImages: val.NumImages + 1, SeriesDescription: SeriesDescription, SeriesNumber: SeriesNumber}
					} else {
						datasets[StudyInstanceUID] = make(map[string]SeriesInfo)
						datasets[StudyInstanceUID][SeriesInstanceUID] = SeriesInfo{NumImages: 1, SeriesDescription: SeriesDescription, SeriesNumber: SeriesNumber}
					}
				} else {
					datasets[StudyInstanceUID] = make(map[string]SeriesInfo)
					datasets[StudyInstanceUID][SeriesInstanceUID] = SeriesInfo{NumImages: 1, SeriesDescription: SeriesDescription, SeriesNumber: SeriesNumber}
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

	const (
		defaultInputDir    = "Specify where you want to setup shop"
		defaultTriggerTime = "A wait time in seconds before the computation is triggered (2s, or 7m, etc.)"
		errorConfigFile    = "the current directory is not an rpp directory. Change to the correct directory or run\n\t rpp init project01\nfirst to create a new project01 folder in the current location"
	)

	initCommand := flag.NewFlagSet("init", flag.ContinueOnError)
	configCommand := flag.NewFlagSet("config", flag.ContinueOnError)
	triggerCommand := flag.NewFlagSet("trigger", flag.ContinueOnError)
	statusCommand := flag.NewFlagSet("status", flag.ContinueOnError)

	var input_dir string
	initCommand.StringVar(&input_dir, "input_dir", ".", defaultInputDir)
	//initCommand.StringVar(&input_dir, "i", ".", defaultInputDir)
	var author_name string
	configCommand.StringVar(&author_name, "author_name", "", "Your name \"A User\".")
	initCommand.StringVar(&author_name, "author_name", "", "Your name \"A User\".")
	var author_email string
	configCommand.StringVar(&author_email, "author_email", "", "Your email.")
	initCommand.StringVar(&author_email, "author_email", "", "Your email.")
	var data_path string
	configCommand.StringVar(&data_path, "data", "", "Path to a folder with folders of DICOM files.")

	var trigger string
	triggerCommand.StringVar(&trigger, "trigger", "0s", defaultTriggerTime)
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
	configCommand.StringVar(&config_temp_directory, "temp_directory", "", "Specify a directory for the temporary folders used in the trigger.\n")

	var show_version bool
	flag.BoolVar(&show_version, "version", false, "Show the version number.")

	var user_name string
	user, err := user.Current()
	if err != nil {
		user_name = user.Username
		fmt.Println("got a user name ", user_name)
	}

	// Showing useful information when the user enters the --help option
	flag.Usage = func() {
		fmt.Printf("RPP - Remote Pipeline Processing\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Usage: %s [init|trigger|status|config] [options]\n\tStart with init to create a new project folder.\n\t%s init <project>\n", os.Args[0], os.Args[0])
		fmt.Printf("Option init:\n")
		initCommand.PrintDefaults()
		fmt.Printf("Option config:\n")
		configCommand.PrintDefaults()
		fmt.Printf("Option status:\n")
		statusCommand.PrintDefaults()
		fmt.Printf("Option trigger:\n")
		triggerCommand.PrintDefaults()
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
						fmt.Printf("Your name: ")
						author_name, err = reader.ReadString('\n')
						if err != nil {
							msg := "we need your name. Add with\n\t--author_name \"<name>\""
							exitGracefully(errors.New(msg))
						}
					}
					if author_email == "" {
						fmt.Printf("Your email: ")
						author_email, err = reader.ReadString('\n')
						if err != nil {
							msg := "we need your your email. Add with\n\t--author_email \"email@home\""
							exitGracefully(errors.New(msg))
						}
					}

				}

				if err := os.Mkdir(dir_path, 0755); os.IsExist(err) {
					exitGracefully(errors.New("directory exist already"))
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
			}
			fmt.Printf("Init folder %s done\n", input_dir)
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
			if config_temp_directory != "" {
				config.TempDirectory = config_temp_directory
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
			var selectFromB []string
			for StudyInstanceUID, value := range config.Data.DataInfo {
				for SeriesInstanceUID, value2 := range value {
					selectFromA[SeriesInstanceUID] = fmt.Sprintf("StudyInstanceUID: %s, SeriesInstanceUID: %s, SeriesDescription: %s, NumImages: %d, SeriesNumber: %d", StudyInstanceUID, SeriesInstanceUID, value2.SeriesDescription, value2.NumImages, value2.SeriesNumber)
				}
			}
			mm := regexp.MustCompile(config.SeriesFilter)
			for key, value := range selectFromA {
				if mm.MatchString(value) {
					selectFromB = append(selectFromB, key)
				}
			}
			idx := rand.Intn((len(selectFromB) - 0) + 0)
			fmt.Printf("found %d matching series. Picked index %d, run with series: %s\n", len(selectFromB), idx, selectFromB[idx])

			dir, err := ioutil.TempDir(config.TempDirectory, fmt.Sprintf("rpp_trigger_run_%s_*", time.Now().Weekday()))
			if err != nil {
				exitGracefully(errors.New("could not create the temporary directory for the trigger"))
			}
			if !trigger_keep {
				defer os.RemoveAll(dir)
			} else {
				fmt.Printf("trigger data directory is \"%s\"\n", dir)
			}
			// we should copy all files into this directory that we need for processing
			// the study we want is this one selectFromB[idx]
			numFiles, description := copyFiles(selectFromB[idx], config.Data.Path, dir)
			fmt.Println("Found", numFiles, "files.")
			// write out a description
			file, _ := json.MarshalIndent(description, "", " ")
			_ = ioutil.WriteFile(dir+"/descr.json", file, 0644)
			if !trigger_test {
				// wait for some seconds
				if trigger != "" {
					sec, _ := time.ParseDuration(trigger)
					time.Sleep(sec)
				}

				cmd_str := fmt.Sprintf("python ./stub.py \"%s/\"", dir)
				cmd := exec.Command("python", "stub.py", dir)
				var outb, errb bytes.Buffer
				cmd.Stdout = &outb
				cmd.Stderr = &errb
				err := cmd.Run()
				if err != nil {
					exitGracefully(fmt.Errorf("could not run trigger command\n\t%s", cmd_str))
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

				fmt.Println("Done.")
				// we can check if we have an output folder now
				if _, err := os.Stat(dir + "/output/output.json"); err != nil && !os.IsNotExist(err) {
					exitGracefully(fmt.Errorf("run finished but no output/output.json file found. Consider creating such a file in your program"))
				}
			}
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
