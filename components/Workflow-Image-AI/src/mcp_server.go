package main

/*
   The following discussion worked:

   Notify mcp/ror that its root should be /Users/..../bla


*/

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suyashkumar/dicom/pkg/tag"
)

func startMCP(useHttp string, rootFolder string) {
	// if the useHttp string is empty use stdin/stdout
	if useHttp == "" {
		log.Println("Starting MCP server using stdin/stdout")
	}

	opts := &mcp.ServerOptions{
		Instructions:      "Use this server with the MCP protocol in vcode or other clients.",
		CompletionHandler: complete, // support completions by setting this handler
		RootsListChangedHandler: func(ctx context.Context, req *mcp.RootsListChangedRequest) {
			// notificationChans["roots"] <- 0
			// fmt.Printf("got a root change request %v", req)
			// should we reject a change of the root if its not in the initial root folder?
		},
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "ror", Version: version}, opts)

	// configure the root directory that the server can access
	if rootFolder != "" {
		input_dir = rootFolder
		// make sure the rootFolder is an absolute path
		/*	absRoot, err := makeAbsolutePath(rootFolder)
			if err != nil {
				log.Fatalf("Could not make root folder absolute: %v", err)
			}
			rootFolder = absRoot
			server.Roots = append(server.Roots, mcp.Root{URI: "file://" + rootFolder, Name: "RootFolder"})
			input_dir = rootFolder
			log.Printf("Setting the MCP root folder to %s", rootFolder) */
	} else {
		log.Printf("No root folder specified, please set one up using the MCP Inspector or other client (see --working_directory).")
	}

	// Add tools that exercise different features of the protocol.
	//mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, contentTool)
	//mcp.AddTool(server, &mcp.Tool{Name: "greet (structured)"}, structuredTool) // returns structured output
	mcp.AddTool(server, &mcp.Tool{Name: "ror/info", Description: "ROR (helm) is a set of workflow tools for research PACS. There are tools for clearing out current data and adding new DICOM data."}, rorTool) // returns structured output
	//mcp.AddTool(server, &mcp.Tool{Name: "ping"}, pingingTool)                                                                                                                                                   // performs a ping
	//mcp.AddTool(server, &mcp.Tool{Name: "log"}, loggingTool)                                                                                                                                                    // performs a log
	//mcp.AddTool(server, &mcp.Tool{Name: "sample"}, samplingTool)                                                                                                                                                // performs sampling
	//mcp.AddTool(server, &mcp.Tool{Name: "elicit"}, elicitingTool)                                                                                                                                               // performs elicitation
	mcp.AddTool(server, &mcp.Tool{
		Name:        "roots",
		Description: "Manage the ror roots. Use roots/list to see the currently configured roots.",
	}, rootsTool) // does everything with the ror folder?                                                                                                                                                                // lists roots
	mcp.AddTool(server, &mcp.Tool{
		Name:        "roots/list",
		Description: "List the currently configured roots.",
	}, rootsListTool) // lists roots

	mcp.AddTool(server, &mcp.Tool{
		Name:        "project/init",
		Description: "Create a ror project folder. In a later step add data and trigger workflows for matching DICOM series or studies.",
	}, projectTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "data/clear",
		Description: "Delete all imported data references from the current ror project.",
	}, clearOutDataCacheTool) // returns structured output

	mcp.AddTool(server, &mcp.Tool{
		Name: "data/add",
		Description: "Add a new data folder. Adding data will require ror to parse the whole directory which takes some time. " +
			"Wait for this operation to finish before querying the resources again.",
	}, addDataCacheTool) // returns structured output

	mcp.AddTool(server, &mcp.Tool{
		Name:        "data/list",
		Description: "Get detailed information on the currently loaded data.",
	}, dataInfoTool)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_patients_info",
		Description: "Get detailed information about the list of patients or, in the context of a research study the list of participants.\n" +
			"\nReturns an object with a 'patients' property containing an array of patient identifiers.",
		OutputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"message": {Type: "string"},
				"patients": {
					Type:  "array",
					Items: &jsonschema.Schema{Type: "string"},
				},
			},
		},
	}, dataListPatients)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_study_info",
		Description: "Get detailed information about the list of studies for a given patient or participant.\n" +
			"\nParameters:\n" +
			"- Name: The patient or participant name to query. If the value is an empty string studies for all are returned.\n" +
			"\nReturns an object with 'studies' property containing a map of study IDs to study information.",
		OutputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"message": {Type: "string"},
				"studies": {
					Type: "object",
					AdditionalProperties: &jsonschema.Schema{
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"patient_id":        {Type: "string"},
							"patient_name":      {Type: "string"},
							"study_date":        {Type: "string"},
							"study_description": {Type: "string"},
						},
					},
				},
			},
		},
	}, dataListStudies)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_series_info",
		Description: "Get detailed information about the list of image series.\n" +
			"\nParameters:\n" +
			"- StudyInstanceUID: The StudyInstanceUID information for the study to query.\n" +
			"\nReturns an array of series with properties such as PatientID, PatientName, StudyDate, SeriesDescription, Modality and number of images.",
		OutputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"message": {Type: "string"},
				"series": {
					Type: "array",
					Items: &jsonschema.Schema{
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"patient_id":          {Type: "string"},
							"patient_name":        {Type: "string"},
							"study_date":          {Type: "string"},
							"series_description":  {Type: "string"},
							"modality":            {Type: "string"},
							"number_of_images":    {Type: "integer"},
							"series_instance_uid": {Type: "string"},
							"study_instance_uid":  {Type: "string"},
						},
					},
				},
			},
		},
	}, dataListSeries)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_series_tags",
		Description: "Get a list of tags for a DICOM series.",
		OutputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"message": {Type: "string"},
				"tags": {
					Type: "array",
					Items: &jsonschema.Schema{
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"group":   {Type: "integer"},
							"element": {Type: "integer"},
							"value":   {Type: "string"},
							"vr":      {Type: "string"},
						},
					},
				},
			},
		},
	}, dataListTags)

	//mcp.AddTool(server, &mcp.Tool{Name: "change/root", Description: "Change to a new ror folder."}, changeRootTool)                                                                                                                                                   // returns structured output

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_current_select_statement",
		Description: "Get the current select statement used to filter the DICOM studies and series.",
		OutputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"message": {Type: "string"},
				"select_statement": {
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"Output_level": {Type: "string"},
						"Select_level": {Type: "string"},
						"RulesTree": {
							Type: "array",
							Items: &jsonschema.Schema{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"Name": {Type: "string"},
									"Rs": {
										Type: "object",
										Properties: map[string]*jsonschema.Schema{
											"Leaf1":    {Type: "string"},
											"Operator": {Type: "string"},
											"Leaf2":    {Type: "string"},
										},
									},
								},
							},
						},
					},
				},
				"match_count": {Type: "integer"},
			},
		},
	}, showSelectTool) // support completions
	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_new_select_statement",
		Description: "Set a new select statement.",
		OutputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"message": {Type: "string"},
			},
		},
	}, setSelectTool) // support completions

	// Add a basic prompt.
	server.AddPrompt(&mcp.Prompt{Name: "greet"}, prompt)

	server.AddPrompt(&mcp.Prompt{Name: "ror/process_data",
		Description: "Workflow to create a new ROR database from a folder with DICOM images.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "directory",
				Title:       "Data Directory",
				Description: "Directory with DICOM images to process.",
				Required:    true,
			},
			{
				Name:        "work_folder",
				Title:       "Work Folder",
				Description: "Work folder to store the database in.",
				Required:    true,
			},
		},
	}, createNewRORDatabase)

	server.AddPrompt(&mcp.Prompt{Name: "ror/analyze_data",
		Description: "Workflow to summarize information about DICOM data in a folder.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "directory",
				Title:       "Data Directory",
				Description: "Directory with DICOM images to process.",
				Required:    true,
			},
		},
	}, analyzeDICOMData)

	server.AddPrompt(&mcp.Prompt{Name: "ror/get_test_dicom_data",
		Description: "Workflow to download some test DICOM data using git.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "directory",
				Title:       "Data Directory",
				Description: "Directory to store the DICOM images.",
				Required:    true,
			},
		},
	}, downloadDICOMData)

	// Add an embedded resource.
	server.AddResource(&mcp.Resource{
		Name:     "info",
		MIMEType: "text/plain",
		URI:      "embedded:info",
	}, embeddedResource)
	server.AddResource(&mcp.Resource{
		Name: "data",
		//MIMEType: "text/plain",
		MIMEType: "application/json",
		URI:      "embedded:data",
	}, embeddedResource)
	server.AddResource(&mcp.Resource{
		Name:     "numstudies",
		MIMEType: "text/plain",
		URI:      "embedded:numstudies",
	}, embeddedResource)
	server.AddResource(&mcp.Resource{
		Name:     "numseries",
		MIMEType: "text/plain",
		URI:      "embedded:numseries",
	}, embeddedResource)
	server.AddResource(&mcp.Resource{
		Name:     "numimages",
		MIMEType: "text/plain",
		URI:      "embedded:numimages",
	}, embeddedResource)
	server.AddResource(&mcp.Resource{
		Name:     "numparticipants",
		MIMEType: "text/plain",
		URI:      "embedded:numparticipants",
	}, embeddedResource)
	server.AddResource(&mcp.Resource{
		Name:        "tag",
		Description: "Get DICOM tag name from tag group/element",
		MIMEType:    "text/plain",
		URI:         "embedded:tag",
	}, embeddedResource)
	server.AddResource(&mcp.Resource{
		Name:        "tagname",
		Description: "Get DICOM tag group/element from tag name",
		MIMEType:    "text/plain",
		URI:         "embedded:tagname",
	}, embeddedResource)

	// Serve over stdio, or streamable HTTP if -http is set.
	if useHttp != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("MCP handler listening at %s", useHttp)
		http.ListenAndServe(useHttp, handler)
	} else {
		t := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
		if err := server.Run(context.Background(), t); err != nil {
			log.Printf("Server failed: %v", err)
		}
	}

}

// needs folder with data and a folder location to work in
func createNewRORDatabase(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Workflow to create a new ROR working directory and add DICOM image data",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{Text: "Create a new ROR database from DICOM images in directory '" +
					req.Params.Arguments["directory"] + "' and store it in work folder '" + req.Params.Arguments["work_folder"] +
					"'.\n\nStep by step instructions are: \n\n" +
					"  1. Initialize a new ROR project in the work folder with\n" +
					"     'ror init " + req.Params.Arguments["work_folder"] + "'.\n\n" +
					"  2. Add the DICOM data from the directory with\n" +
					"     'ror config --data " + req.Params.Arguments["directory"] + " --temp_directory " + req.Params.Arguments["directory"] + "'."},
			},
		},
	}, nil
}

func analyzeDICOMData(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Workflow to summarize information about a DICOM directory",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{Text: "Summarize information about a folder with DICOM images in '" +
					req.Params.Arguments["directory"] + "'.\n\nStep by step instructions are: \n\n" +
					"  1. Create a new temporary ROR working directory and add the data.\n\n" +
					"  2. Show the number of participants, number of DICOM studies and number of DICOM series.\n\n" +
					"  3. Show a list of image modalities and the number of series for each.\n\n" +
					"  4. Show a list of sequence names and the number of studies for each.\n\n" +
					"  5. Show the date range of the studies in the data folder.\n\n" +
					"  6. Show a summary of the StudyDescription DICOM tags.\n\n" +
					"  7. Create up to three examples of image types (like T1, T2, FLAIR) for which the majority of studies have matching image series.",
				},
			},
		},
	}, nil
}

func downloadDICOMData(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Workflow to download test DICOM data using git",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{Text: "Download test DICOM data using git into the directory: " + req.Params.Arguments["directory"] +
					"\n\nStep by step instructions are: \n\n" +
					"  1. Create a new directory and change into it with \n" +
					"     'mkdir " + req.Params.Arguments["directory"] + "; cd " + req.Params.Arguments["directory"] + "'\n\n" +
					"  2. Clone the hackathon dataset with\n" +
					"     'git clone https://github.com/ImagingInformatics/hackathon-dataset.git'\n\n" +
					"  3. Enter the new directory and update the submodules to download the DICOM images with\n" +
					"     'cd hackathon-dataset; git submodule update --init --recursive'"},
			},
		},
	}, nil
}

func prompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {

	return &mcp.GetPromptResult{
		Description: "Hi prompt",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: "Say hi to " + req.Params.Arguments["name"]},
			},
		},
	}, nil
}

var embeddedResources = map[string]string{
	"info":            "This is the 'ror' tool server. 'ror' is a tool to create workflows for the research picture archive and communication systems (PACS).",
	"data":            "", // config.Data.Path,
	"numstudies":      "",
	"numseries":       "",
	"numimages":       "",
	"numparticipants": "",
}

func getInputDir(ctx context.Context, session *mcp.ServerSession) (string, error) {
	if input_dir != "" {
		return input_dir, nil
	}
	res, err := session.ListRoots(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("listing roots failed: %v", err)
	}
	var allroots []string
	for _, r := range res.Roots {
		uri_temp := strings.TrimPrefix(r.URI, "file://")
		allroots = append(allroots, uri_temp)
	}
	if len(allroots) == 0 {
		return "", fmt.Errorf("no roots defined, setup a root first")
	}
	dir_path := allroots[0] // should be "./.ror/config"
	if len(allroots) > 1 {
		log.Printf("Warning: Multiple roots defined: %v, selecting %s", allroots, dir_path)
	}
	return dir_path, nil
}

// add all fields to the embeddedResources global variable (update them)
func fillInEmbeddedResources(ctx context.Context, session *mcp.ServerSession) (map[string]string, error) {
	var err error
	if input_dir, err = getInputDir(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	dir_path := input_dir + "/.ror/config" // should be "./.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}
	embeddedResources["numstudies"] = fmt.Sprintf("%d", len(config.Data.DataInfo))

	var datasets = make(map[string]map[string]SeriesInfo)
	datasets = config.Data.DataInfo
	numSeries := 0
	for _, v := range datasets {
		numSeries += len(v)
	}
	embeddedResources["numseries"] = fmt.Sprintf("%d", numSeries)

	datasets = config.Data.DataInfo
	numImages := 0
	for _, v := range datasets {
		for _, vv := range v {
			numImages += vv.NumImages
		}
	}
	embeddedResources["numimages"] = fmt.Sprintf("%d", numImages)

	datasets = config.Data.DataInfo
	var participants map[string]bool = make(map[string]bool)
	for _, v := range datasets {
		for _, vv := range v {
			participants[fmt.Sprintf("%s%s", vv.PatientID, vv.PatientName)] = true
		}
	}
	numParticipants := len(participants)
	embeddedResources["numparticipants"] = fmt.Sprintf("%d", numParticipants)
	return embeddedResources, nil
}

func embeddedResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	u, err := url.Parse(req.Params.URI)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "embedded" {
		return nil, fmt.Errorf("wrong scheme: %q", u.Scheme)
	}
	key := u.Opaque
	if strings.HasPrefix(key, "tag/") {
		// Parse group/element from URI like embedded:tag/0008/0020
		parts := strings.Split(key, "/")
		if len(parts) == 3 {
			groupStr := parts[1]
			elementStr := parts[2]
			group, err := strconv.ParseUint(groupStr, 16, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid group: %s", groupStr)
			}
			element, err := strconv.ParseUint(elementStr, 16, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid element: %s", elementStr)
			}
			t := tag.Tag{Group: uint16(group), Element: uint16(element)}
			info, err := tag.Find(t)
			if err != nil {
				return nil, fmt.Errorf("tag not found: %v", err)
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{URI: req.Params.URI, MIMEType: "text/plain", Text: info.Name},
				},
			}, nil
		} else {
			return nil, fmt.Errorf("invalid tag URI format, expected embedded:tag/group/element")
		}
	} else if strings.HasPrefix(key, "tagname/") {
		// Parse name from URI like embedded:tagname/StudyDate
		parts := strings.Split(key, "/")
		if len(parts) == 2 {
			name := parts[1]
			info, err := tag.FindByName(name)
			if err != nil {
				return nil, fmt.Errorf("tag name not found: %v", err)
			}
			text := fmt.Sprintf("%04X,%04X", info.Tag.Group, info.Tag.Element)
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{URI: req.Params.URI, MIMEType: "text/plain", Text: text},
				},
			}, nil
		} else {
			return nil, fmt.Errorf("invalid tagname URI format, expected embedded:tagname/name")
		}
	}
	text, ok := embeddedResources[key]
	if !ok {
		return nil, fmt.Errorf("no embedded resource named %q", key)
	}
	// add the current value for resource "data"
	// our input_dir should be the root folder
	// get the roots - use the first entry
	//res, err := req.Session.ListRoots(ctx, nil)
	//if err != nil {
	//	return nil, fmt.Errorf("listing roots failed: %v", err)
	//}
	//var allroots []string
	//for _, r := range res.Roots {
	//	uri_temp := strings.TrimPrefix(r.URI, "file://")
	//	allroots = append(allroots, uri_temp)
	//}
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	dir_path := input_dir + "/.ror/config" // should be "./.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}
	if key == "data" {
		// instead of a simple string try to return a json object with the path inside
		var obj = map[string]string{"path": config.Data.Path} // this is relative to the ror directory
		var obj_json []byte
		if obj_json, err = json.MarshalIndent(obj, "", " "); err != nil {
			return nil, fmt.Errorf("failed to marshal json: %v", err)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: req.Params.URI, MIMEType: "application/json", Text: string(obj_json)},
			},
		}, nil
	}
	if key == "numstudies" {
		text = fmt.Sprintf("%d", len(config.Data.DataInfo))
	}
	if key == "numseries" {
		var datasets = make(map[string]map[string]SeriesInfo)
		datasets = config.Data.DataInfo
		numSeries := 0
		for _, v := range datasets {
			numSeries += len(v)
		}
		text = fmt.Sprintf("%d", numSeries)
	}
	if key == "numimages" {
		var datasets = make(map[string]map[string]SeriesInfo)
		datasets = config.Data.DataInfo
		numImages := 0
		for _, v := range datasets {
			for _, vv := range v {
				numImages += vv.NumImages
			}
		}
		text = fmt.Sprintf("%d", numImages)
	}
	if key == "numparticipants" {
		var datasets = make(map[string]map[string]SeriesInfo)
		datasets = config.Data.DataInfo
		var participants map[string]bool = make(map[string]bool)
		for _, v := range datasets {
			for _, vv := range v {
				participants[fmt.Sprintf("%s%s", vv.PatientID, vv.PatientName)] = true
			}
		}
		numParticipants := len(participants)
		text = fmt.Sprintf("%d", numParticipants)
	}

	if text == "" {
		text = "empty string"
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "text/plain", Text: text},
		},
	}, nil
}

// for dataListStudies
type args struct {
	Name string `json:"name" jsonschema:"The patient or participant name to query"`
}

type argsPath struct {
	Path string `json:"path" jsonschema:"the data folder with DICOM images to add"`
}

type argsMessage struct {
	Message string `json:"message" jsonschema:"the message to log"`
}

type argsSelect struct {
	Message    string `json:"message" jsonschema:"general message if the select statement was found"`
	Select     AST    `json:"select_stement" jsonschema:"the select statement to filter for specific DICOM series"`
	MatchCount int    `json:"match_count" jsonschema:"the number of matching series or studies for the select statement"`
}

type setSelectMessage struct {
	Select string `json:"select" jsonschema:"the select statement to filter in DICOM series"`
}

type argsData struct {
	Patient string `json:"patient" jsonschema:"If string is not empty list information from this patient"`
	Study   string `json:"study" jsonschema:"If string is not empty list information from this study"`
	Series  string `json:"series" jsonschema:"If string is not empty list information from this series"`
}

// contentTool is a tool that returns unstructured content.
//
// Since its output type is 'any', no output schema is created.
func contentTool(ctx context.Context, req *mcp.CallToolRequest, args args) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Hi " + args.Name},
		},
	}, nil, nil
}

type result struct {
	Message string `json:"message" jsonschema:"the message to convey"`
}

type resultDataInfo struct {
	Message string `json:"message" jsonschema:"the message to convey"`
	Data    string `json:"data" jsonschema:"a map with the individual DICOM series information"`
}

type resultPatients struct {
	Message  string   `json:"message"`
	Patients []string `json:"patients"`
}

type studyInfo struct {
	PatientID        string `json:"patient_id"`
	PatientName      string `json:"patient_name"`
	StudyDate        string `json:"study_date"`
	StudyDescription string `json:"study_description"`
}

type resultStudies struct {
	Message string               `json:"message"`
	Studies map[string]studyInfo `json:"studies"`
}

type resultStudyInfo struct {
	Message string `json:"message" jsonschema:"the message to convey"`
	Data    string `json:"data" jsonschema:"an array with the DICOM study information, each study entry has properties such as PatientID, PatientName, StudyDate"`
}

type seriesOutput struct {
	PatientID         string `json:"patient_id" jsonschema:"the patient ID"`
	PatientName       string `json:"patient_name" jsonschema:"the patient name"`
	StudyDate         string `json:"study_date" jsonschema:"the study date"`
	SeriesDescription string `json:"series_description" jsonschema:"the series description"`
	Modality          string `json:"modality" jsonschema:"the modality"`
	NumberOfImages    int    `json:"number_of_images" jsonschema:"the number of images"`
	SeriesInstanceUID string `json:"series_instance_uid" jsonschema:"the series instance UID"`
	StudyInstanceUID  string `json:"study_instance_uid" jsonschema:"the study instance UID"`
}

type resultSeriesInfo struct {
	Message string         `json:"message" jsonschema:"the message to convey"`
	Series  []seriesOutput `json:"series" jsonschema:"an array of DICOM series information"`
}

type TagInfo struct {
	Group   uint16 `json:"group" jsonschema:"the DICOM tag group"`
	Element uint16 `json:"element" jsonschema:"the DICOM tag element"`
	Value   string `json:"value" jsonschema:"the tag value"`
	VR      string `json:"vr" jsonschema:"the value representation"`
}

type resultTags struct {
	Message string    `json:"message" jsonschema:"the message to convey"`
	Tags    []TagInfo `json:"tags" jsonschema:"an array of DICOM tag information"`
}

// if we clear out the data cache we need a result that reports the total numbers
type resultDataCache struct {
	Message         string `json:"message" jsonschema:"the message to convey"`
	NumStudies      int    `json:"numstudies" jsonschema:"the number of DICOM studies"`
	NumSeries       int    `json:"numseries" jsonschema:"the number of DICOM image series"`
	NumImages       int    `json:"numimages" jsonschema:"the number of DICOM images"`
	NumParticipants int    `json:"numparticipants" jsonschema:"the number of unique PatientID DICOM tags"`
}

// TOOL
func projectTool(ctx context.Context, req *mcp.CallToolRequest, args *argsPath) (*mcp.CallToolResult, *result, error) {
	// only init if a directory already exists and its not yet a ror folder

	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err == nil {
		if input_dir == args.Path {
			return nil, &result{Message: "Everything is ok, this folder is already the current working folder."}, err
		} else {
			return nil, &result{Message: "Error, a ror directory was already specified. Restart the mcp server with the new project location folder working_directory."}, err
		}
	}
	if input_dir != args.Path {
		return nil, &result{Message: "Error, the requested directory is different from the working_directory specified when the mcp_server was started."}, err
	}

	// read the config
	dir_path := args.Path + "/.ror/config"
	_, err = readConfig(dir_path)
	if err == nil {
		return nil, &result{Message: "Error, a config file exists already at that location. Remove .ror/ or use a different directory."}, err
	}

	return nil, &result{Message: "Error, not yet implemented."}, err
}

// TOOL
func clearOutDataCacheTool(ctx context.Context, req *mcp.CallToolRequest, args *args) (*mcp.CallToolResult, *resultDataCache, error) {
	// find out if there is data, if there is no ror folder produce an error
	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, &resultDataCache{Message: "Error could not get ror directory."}, err
	}
	// make the config
	dir_path := input_dir + "/.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, &resultDataCache{Message: "Error could not read config file from ror directory."}, err
	}

	config.Data.DataInfo = make(map[string]map[string]SeriesInfo)
	config.Data.Path = ""

	// this will use input_dir to write
	if !config.writeConfig() {
		return nil, &resultDataCache{Message: "Error could not write config file into ror directory."}, err
	}

	// return that we cleared out the data cache, return the current number of dataset as well
	return nil, &resultDataCache{Message: "Removed all data", NumStudies: 0, NumSeries: 0, NumImages: 0}, nil
}

func changeRootTool(ctx context.Context, req *mcp.CallToolRequest, args *args) (*mcp.CallToolResult, *resultDataCache, error) {
	//req.Session.Roots.append({uri: "file://" + args[0], name: "RootFolder"})
	// This is not enough, the getInputDir will lookup the value from the roots again, we need to add the input_dir there.
	// Right now the only place we can add it is from the client (MCP Inspector).
	input_dir = args.Name
	return nil, &resultDataCache{Message: "Changed to the new root path", NumStudies: 0, NumSeries: 0, NumImages: 0}, nil
}

func setSelectTool(ctx context.Context, req *mcp.CallToolRequest, args *setSelectMessage) (*mcp.CallToolResult, *argsMessage, error) {
	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, &argsMessage{Message: "Error could not get ror directory."}, err
	}
	// make the config
	dir_path := input_dir + "/.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, &argsMessage{Message: "Error could not read config file from ror directory."}, err
	}
	config_series_filter := string(args.Select)

	comments := regexp.MustCompile("/[*]([^*]|[\r\n]|([*]+([^*/]|[\r\n])))*[*]+/")
	series_filter_no_comments := comments.ReplaceAllString(config_series_filter, " ")

	// now parse the input string
	InitParser()
	//yyErrorVerbose = true
	yyDebug = 1

	line := []byte(series_filter_no_comments)
	yyParse(&exprLex{line: line})
	msg := ""
	if !errorOnParse {
		//s, _ := json.MarshalIndent(ast, "", "  ")
		ss := humanizeFilter(ast)
		type Msg struct {
			Messages  []string `json:"messages"`
			Ast       AST      `json:"ast"`
			Matches   int      `json:"matches"`
			Complains []string `json:"complains"`
		}
		//fmt.Printf("Parsing series filter successful\n%s\n%s\n", string(s), strings.Join(ss[:], "\n"))
		config.SeriesFilterType = "select"
		// check if we have any matches - cheap for us here
		matches, complains := findMatchingSets(ast, config.Data.DataInfo)
		//fmt.Printf("Given our current test data we can identify %d matching dataset%s.\n", len(matches), postfix)
		out := Msg{Messages: ss, Ast: ast, Matches: len(matches), Complains: complains}
		human_enc, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Println(err)
		}
		msg = fmt.Sprintln(string(human_enc))
	} else {
		// maybe its a simple glob expression? We should add in any case
		//fmt.Println("We tried to parse the series filter but failed. Maybe you just want to grep?")
		// exitGracefully(errors.New("we tried to parse the series filter but failed"))
		config.SeriesFilterType = "glob"
	}

	if config.SeriesFilterType != "select" {
		return nil, &argsMessage{
			Message: "Done. This is now the new select statement: " + config.SeriesFilter + "\nNote:" + msg,
		}, nil
	}
	return nil, &argsMessage{
		Message: "Set a glob type filter because the select statement could not be parsed: " + config.SeriesFilter,
	}, nil
}

func showSelectTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, *argsSelect, error) {
	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, &argsSelect{Message: "Error could not get ror directory."}, err
	}
	// make the config
	dir_path := input_dir + "/.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, &argsSelect{Message: "Error could not read config file from ror directory."}, err
	}

	if len(config.Data.DataInfo) == 0 {
		return nil, &argsSelect{Message: "No data loaded, please add data first using the add/data tool."}, nil
	}

	comments := regexp.MustCompile("/[*]([^*]|[\r\n]|([*]+([^*/]|[\r\n])))*[*]+/")
	series_filter_no_comments := comments.ReplaceAllString(config.SeriesFilter, " ")
	select_str := ""
	// now parse the input string
	InitParser()
	line := []byte(series_filter_no_comments)
	yyParse(&exprLex{line: line})
	matches := make([][]SeriesInstanceUIDWithName, 0)
	if !errorOnParse {
		s, _ := json.MarshalIndent(ast, "", "  ")
		// ss := humanizeFilter(ast)
		select_str += string(s) // strings.Join(ss, " ") // fmt.Sprintf("Parsing series filter\n%s\n%s\n", string(s), ss)
		config.SeriesFilterType = "select"
		// check if we have any matches - cheap for us here
		matches, _ = findMatchingSets(ast, config.Data.DataInfo)
		//postfix := "s"
		//if len(matches) == 1 {
		//	postfix = ""
		//}
		//select_str += fmt.Sprintf("Given our current test data we can identify %d matching dataset%s.\n", len(matches), postfix)
	}

	// return that we cleared out the data cache, return the current number of dataset as well
	return nil, &argsSelect{
		Message:    "Success",
		Select:     ast, // shouldn't this be structured information instead?
		MatchCount: len(matches),
	}, nil
}

func dataListPatients(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, *resultPatients, error) {
	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, &resultPatients{Message: "Error could not get ror directory."}, err
	}
	// make the config
	dir_path := input_dir + "/.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, &resultPatients{Message: "Error could not read config file from ror directory."}, err
	}

	if len(config.Data.DataInfo) == 0 {
		return nil, &resultPatients{Message: "No data loaded, please add data first using the add/data tool."}, nil
	}

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
	return nil, &resultPatients{
		Message:  "List of accessible patient ids from " + config.Data.Path + ". Each patient will have associated studies that in turn have series with tag information.",
		Patients: participants,
	}, nil
}

// name could be a part of a patient name.
func dataListStudies(ctx context.Context, req *mcp.CallToolRequest, args *args) (*mcp.CallToolResult, *resultStudies, error) {
	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, &resultStudies{Message: "Error could not get ror directory."}, err
	}
	// make the config
	dir_path := input_dir + "/.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, &resultStudies{Message: "Error could not read config file from ror directory."}, err
	}

	if len(config.Data.DataInfo) == 0 {
		return nil, &resultStudies{Message: "No data loaded, please add data first using the add/data tool."}, nil
	}
	var studyAndDate map[string]studyInfo = make(map[string]studyInfo, 0)
	for key, element := range config.Data.DataInfo { // study
		for _, element2 := range element { // series
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
			if args.Name != "" {
				if !strings.Contains(name, args.Name) {
					continue
				}
			}
			studyAndDate[key] = studyInfo{
				PatientID:        element2.PatientID,
				PatientName:      element2.PatientName,
				StudyDate:        studyDate,
				StudyDescription: element2.StudyDescription,
			}
			break
			// data += fmt.Sprintf("Patient: %s, Study %s (Date: %s) Series %s: %d images\n", name, key, studyDate, key2, element2.NumImages)
		}
	}
	return nil, &resultStudies{
		Message: "Mapping of StudyInstanceUIDs and their associated PatientID, PatientName, StudyDate and StudyDescription from data path " + config.Data.Path,
		Studies: studyAndDate,
	}, nil
}

type argsSeries struct {
	StudyInstanceUID string `json:"study_instance_uid" jsonschema:"The StudyInstanceUID for the study to query."`
}

func dataListSeries(ctx context.Context, req *mcp.CallToolRequest, args *argsSeries) (*mcp.CallToolResult, *resultSeriesInfo, error) {
	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, &resultSeriesInfo{Message: "Error could not get ror directory."}, err
	}
	// make the config
	dir_path := input_dir + "/.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, &resultSeriesInfo{Message: "Error could not read config file from ror directory."}, err
	}

	if len(config.Data.DataInfo) == 0 {
		return nil, &resultSeriesInfo{Message: "No data loaded, please add data first using the add/data tool."}, nil
	}

	var series []seriesOutput = make([]seriesOutput, 0)

	for key, element := range config.Data.DataInfo { // study
		for key2, element2 := range element { // series
			if args.StudyInstanceUID != "" {
				if !strings.Contains(key, args.StudyInstanceUID) {
					continue
				}
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

			series = append(series, seriesOutput{
				PatientID:         element2.PatientID,
				PatientName:       element2.PatientName,
				StudyDate:         studyDate,
				SeriesDescription: element2.SeriesDescription,
				Modality:          element2.Modality,
				NumberOfImages:    element2.NumImages,
				StudyInstanceUID:  key,
				SeriesInstanceUID: key2,
			})
		}
	}
	return nil, &resultSeriesInfo{
		Message: "Series information from data path " + config.Data.Path,
		Series:  series,
	}, nil
}

type argsTags struct {
	SeriesInstanceUID string `json:"series_instance_uid" jsonschema:"the series instance uid to list tags for"`
}

func dataListTags(ctx context.Context, req *mcp.CallToolRequest, args *argsTags) (*mcp.CallToolResult, *resultTags, error) {
	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, &resultTags{Message: "Error could not get ror directory."}, err
	}
	// make the config
	dir_path := input_dir + "/.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, &resultTags{Message: "Error could not read config file from ror directory."}, err
	}

	if len(config.Data.DataInfo) == 0 {
		return nil, &resultTags{Message: "No data loaded, please add data first using the add/data tool."}, nil
	}

	// the returned data can only be very easy to understand, so here we can generate a list of tag structures
	var data []TagInfo = make([]TagInfo, 0)

	for _, element := range config.Data.DataInfo { // study
		for key2, element2 := range element { // series
			if args.SeriesInstanceUID != "" {
				if !strings.Contains(key2, args.SeriesInstanceUID) {
					continue
				}
			}
			for _, a := range element2.All {
				data = append(data, TagInfo{
					Group:   a.Tag.Group,
					Element: a.Tag.Element,
					Value:   strings.Join(a.Value, ","),
					VR:      a.Type,
				})
			}
		}
	}
	return nil, &resultTags{
		Message: "Tag information from data path " + config.Data.Path +
			". Each tag has a group and element and a value.",
		Tags: data,
	}, nil
}

func dataInfoTool(ctx context.Context, req *mcp.CallToolRequest, args *argsData) (*mcp.CallToolResult, *resultDataInfo, error) {
	// find out if there is data, if there is no ror folder produce an error
	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, &resultDataInfo{Message: "Error could not get ror directory."}, err
	}
	// make the config
	dir_path := input_dir + "/.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, &resultDataInfo{Message: "Error could not read config file from ror directory."}, err
	}

	if len(config.Data.DataInfo) == 0 {
		return nil, &resultDataInfo{Message: "No data loaded, please add data first using the add/data tool."}, nil
	}

	data := ""
	if args.Patient != "" {
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
		var participantFound bool = false
		var participantFoundName string = ""
		var participants []string = make([]string, 0, len(participantsMap))
		for k := range participantsMap {
			participants = append(participants, k)
			if k == args.Patient {
				participantFound = true
				participantFoundName = k
			} else if strings.Contains(strings.ToLower(k), strings.ToLower(args.Patient)) {
				participantFound = true
				participantFoundName = k
			}
		}
		if !participantFound {
			return nil, &resultDataInfo{Message: "Participant not found, here is the list of known participant ids: " + strings.Join(participants, ", ")}, nil
		}
		data += fmt.Sprintf("Found patient %s in the loaded data.\n", args.Patient)
		// create the list of studies for that patient
		for key, element := range config.Data.DataInfo {
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
				if name == participantFoundName {
					data += fmt.Sprintf("Study %s (Date: %s) Series %s: %d images\n", key, studyDate, key2, element2.NumImages)
				}
			}
		}
	} else if args.Study != "" {
		data += fmt.Sprintf("Listing information for study %s\n", args.Study)
		for key, element := range config.Data.DataInfo { // study
			for key2, element2 := range element { // series
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
				if key == args.Study {
					data += fmt.Sprintf("Patient: %s, Study %s (Date: %s) Series %s: %d images\n", name, key, studyDate, key2, element2.NumImages)
				}
			}
		}
	} else if args.Series != "" {
		data += fmt.Sprintf("Listing information for series %s\n", args.Series)
		for _, element := range config.Data.DataInfo { // study
			for key2, element2 := range element { // series
				if key2 != args.Series {
					continue
				}
				for _, a := range element2.All {
					data += fmt.Sprintf("Tag: (0x%04x,0x%04x) Value: %s\n", a.Tag.Group, a.Tag.Element, strings.Join(a.Value, ","))
				}
			}
		}
	} else {
		data = getDetailedStatusInfo(config)
	}

	// return that we cleared out the data cache, return the current number of dataset as well
	return nil, &resultDataInfo{
		Message: "Here the info loaded from the data path " + config.Data.Path,
		Data:    data, // shouldn't this be structured information instead?
	}, nil
}

func addDataCacheTool(ctx context.Context, req *mcp.CallToolRequest, args *argsPath) (*mcp.CallToolResult, *resultDataCache, error) {
	// ask the user for the directory of the data to add
	// find out if there is data, if there is no ror folder produce an error
	var err error
	if input_dir, err = getInputDir(ctx, req.Session); err != nil {
		return nil, &resultDataCache{Message: "Error could not get ror directory."}, err
	}
	// make the config
	dir_path := input_dir + "/.ror/config"
	config, err := readConfig(dir_path)
	if err != nil {
		return nil, &resultDataCache{Message: "Error could not read config file from ror directory."}, err
	}

	// The following will take a while... should we report back of our progress?
	config.Data.Path = string(args.Path)
	studies, err := dataSets(config, config.Data.DataInfo) // TODO: can we make this create no output on stdout?
	check(err)
	if app != nil {
		app.Stop()
	}
	if len(studies) == 0 {
		return nil, &resultDataCache{Message: "Error we did not find any DICOM files in the folder specified."}, err
		// fmt.Println("We did not find any DICOM files in the folder you provided. Please check if the files are available, un-compress any zip files to make the accessible to this tool.")
	}

	// update the config file now - the above dataSets can take a long time!
	config, err = readConfig(dir_path)
	if err != nil {
		//exitGracefully(errors.New(errorConfigFile))
	}
	config.Data.DataInfo = studies
	config.Data.Path = args.Path

	// this will use input_dir to write
	if !config.writeConfig() {
		return nil, &resultDataCache{Message: "Error could not write config file into ror directory."}, err
	}
	numSeries := 0
	for _, v := range studies {
		numSeries += len(v)
	}
	numImages := 0
	for _, v := range studies {
		for _, vv := range v {
			numImages += vv.NumImages
		}
	}
	var participants map[string]bool = make(map[string]bool)
	for _, v := range studies {
		for _, vv := range v {
			participants[fmt.Sprintf("%s%s", vv.PatientID, vv.PatientName)] = true
		}
	}
	numParticipants := len(participants)

	// return that we cleared out the data cache, return the current number of dataset as well
	return nil, &resultDataCache{
		Message:         "Added the data path " + config.Data.Path,
		NumStudies:      len(studies),
		NumSeries:       numSeries,
		NumImages:       numImages,
		NumParticipants: numParticipants}, nil
}

// structuredTool returns a structured result.
func structuredTool(ctx context.Context, req *mcp.CallToolRequest, args *args) (*mcp.CallToolResult, *result, error) {
	return nil, &result{Message: "Hi " + args.Name}, nil
}

// rorTool returns a structured result.
func rorTool(ctx context.Context, req *mcp.CallToolRequest, args *args) (*mcp.CallToolResult, *result, error) {
	resources, err := fillInEmbeddedResources(ctx, req.Session)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error, could not fill in the resource information, %v", err)},
			},
		}, &result{Message: "ROR is a tool to create workflows for the research PACS"}, nil
	}
	jsonContent, err := json.Marshal(resources)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonContent)},
		},
	}, &result{Message: "ROR is a tool to create workflows for the research PACS"}, nil
}

func pingingTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := req.Session.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("ping failed")
	}
	return nil, nil, nil
}

func loggingTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := req.Session.Log(ctx, &mcp.LoggingMessageParams{
		Data:  "something happened!",
		Level: "error",
	}); err != nil {
		return nil, nil, fmt.Errorf("log failed")
	}
	return nil, nil, nil
}

func rootsListTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	res, err := req.Session.ListRoots(ctx, nil)
	if err != nil {
		if input_dir != "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "file://" + input_dir}, // do we need to add file:// ?
				},
			}, nil, nil
		}
		return nil, nil, fmt.Errorf("listing roots failed: %v", err)
	}
	var allroots []string
	for _, r := range res.Roots {
		allroots = append(allroots, fmt.Sprintf("%s:%s", r.Name, r.URI))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(allroots, ",")},
		},
	}, nil, nil
}

func rootsTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	res, err := req.Session.ListRoots(ctx, nil)
	if err != nil {
		if input_dir != "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: input_dir}, // do we need to add file:// ?
				},
			}, nil, nil
		}
		return nil, nil, fmt.Errorf("listing roots failed: %v", err)
	}
	var allroots []string
	for _, r := range res.Roots {
		allroots = append(allroots, fmt.Sprintf("%s:%s", r.Name, r.URI))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(allroots, ",")},
		},
	}, nil, nil
}

func samplingTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	res, err := req.Session.CreateMessage(ctx, new(mcp.CreateMessageParams))
	if err != nil {
		return nil, nil, fmt.Errorf("sampling failed: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			res.Content,
		},
	}, nil, nil
}

func elicitingTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	res, err := req.Session.Elicit(ctx, &mcp.ElicitParams{
		Message: "provide a random string",
		RequestedSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"random": {Type: "string"},
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("eliciting failed: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: res.Content["random"].(string)},
		},
	}, nil, nil
}

func complete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	var suggestions []string
	switch req.Params.Ref.Type {
	case "ref/prompt":
		suggestions = []string{"ror init", "ror trigger", "ror config", "ror status", "ror mcp"}
	case "ref/resource":
		suggestions = []string{"numstudies", "numseries", "numimages", "numparticipants"}
	default:
		return nil, fmt.Errorf("unrecognized content type %s", req.Params.Ref.Type)
	}

	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Total:  len(suggestions),
			Values: suggestions,
		},
	}, nil
}
