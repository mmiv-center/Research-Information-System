package main

// Importing all the required packages for our tests to work
import (
	"flag"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func Test_processCsvFile(t *testing.T) {
	// Defining the maps we're expenting to get from our function
	wantMapSlice := []map[string]string{
		{"COL1": "1", "COL2": "2", "COL3": "3"},
		{"COL1": "4", "COL2": "5", "COL3": "6"},
	}
	// Defining our test cases
	tests := []struct {
		name      string // The name of the test
		csvString string // The content of our tested CSV file
		separator string // The separator used for each test case
	}{
		{"Comma separator", "COL1,COL2,COL3\n1,2,3\n4,5,6\n", "comma"},
		{"Semicolon separator", "COL1;COL2;COL3\n1;2;3\n4;5;6\n", "semicolon"},
	}
	// Iterating our test cases as usual
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Creating a CSV temp file for testing
			tmpfile, err := ioutil.TempFile("", "test*.csv")
			check(err)

			defer os.Remove(tmpfile.Name())            // Removing the CSV test file before living
			_, err = tmpfile.WriteString(tt.csvString) // Writing the content of the CSV test file
			check(err)
			tmpfile.Sync() // Persisting data on disk
			// Defining the inputFile struct that we're going to use as one parameter of our function
			testFileData := inputFile{
				filepath:  tmpfile.Name(),
				pretty:    false,
				separator: tt.separator,
			}
			// Defining the writerChanel
			writerChannel := make(chan map[string]string)
			// Calling the targeted function as a go routine
			go processCsvFile(testFileData, writerChannel)
			// Iterating over the slice containing the expected map values
			for _, wantMap := range wantMapSlice {
				record := <-writerChannel                // Waiting for the record that we want to compare
				if !reflect.DeepEqual(record, wantMap) { // Making the corresponding test assertion
					t.Errorf("processCsvFile() = %v, want %v", record, wantMap)
				}
			}
		})
	}
}

func Test_checkIfValidFile(t *testing.T) {
	// Creating a temporal and empty CSV file
	tmpfile, err := ioutil.TempFile("", "test*.csv")
	if err != nil {
		panic(err) // This should never happen
	}
	// Once all the tests are done. We delete the temporal file
	defer os.Remove(tmpfile.Name())
	// Defining the struct we're going to use
	tests := []struct {
		name     string
		filename string
		want     bool
		wantErr  bool
	}{ // Defining our test cases
		{"File does exist", tmpfile.Name(), true, false},
		{"File does not exist", "nowhere/test.csv", false, true},
		{"File is not csv", "test.txt", false, true},
	}
	// Iterating over our test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkIfValidFile(tt.filename)
			// Checking the error
			if (err != nil) != tt.wantErr {
				t.Errorf("checkIfValidFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Checking the returning value
			if got != tt.want {
				t.Errorf("checkIfValidFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getFileData(t *testing.T) {
	// Defining our test slice. Each unit test should have the following properties:
	tests := []struct {
		name    string    // The name of the test
		want    inputFile // What inputFile instance we want our function to return.
		wantErr bool      // whether or not we want an error.
		osArgs  []string  // The command arguments used for this test
	}{
		// Here we're declaring each unit test input and output data as defined before
		{"Default parameters", inputFile{"test.csv", "comma", false}, false, []string{"cmd", "test.csv"}},
		{"No parameters", inputFile{}, true, []string{"cmd"}},
		{"Semicolon enabled", inputFile{"test.csv", "semicolon", false}, false, []string{"cmd", "--separator=semicolon", "test.csv"}},
		{"Pretty enabled", inputFile{"test.csv", "comma", true}, false, []string{"cmd", "--pretty", "test.csv"}},
		{"Pretty and semicolon enabled", inputFile{"test.csv", "semicolon", true}, false, []string{"cmd", "--pretty", "--separator=semicolon", "test.csv"}},
		{"Separator not identified", inputFile{}, true, []string{"cmd", "--separator=pipe", "test.csv"}},
	}
	// Iterating over the previous test slice
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Saving the original os.Args reference
			actualOsArgs := os.Args
			// This defer function will run after the test is done
			defer func() {
				os.Args = actualOsArgs                                           // Restoring the original os.Args reference
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError) // Reseting the Flag command line. So that we can parse flags again
			}()

			os.Args = tt.osArgs             // Setting the specific command args for this test
			got, err := getFileData()       // Runing the function we want to test
			if (err != nil) != tt.wantErr { // Asserting whether or not we get the corret error value
				t.Errorf("getFileData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) { // Asserting whether or not we get the corret wanted value
				t.Errorf("getFileData() = %v, want %v", got, tt.want)
			}
		})
	}
}
