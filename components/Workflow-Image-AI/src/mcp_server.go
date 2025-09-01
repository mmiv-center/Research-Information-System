package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func startMCP(useHttp string) {
	// if the useHttp string is empty use stdin/stdout
	if useHttp == "" {
		log.Println("Starting MCP server using stdin/stdout")
	}

	opts := &mcp.ServerOptions{
		Instructions:      "Use this server with the MCP protocol in vcode or other clients.",
		CompletionHandler: complete, // support completions by setting this handler
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "ror", Version: version}, opts)

	// Add tools that exercise different features of the protocol.
	//mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, contentTool)
	//mcp.AddTool(server, &mcp.Tool{Name: "greet (structured)"}, structuredTool) // returns structured output
	mcp.AddTool(server, &mcp.Tool{Name: "ror", Description: "ROR (helm) is a set of tools to create workflows for the research PACS. The list of tool includes clearing out current data and adding new data."}, rorTool) // returns structured output
	mcp.AddTool(server, &mcp.Tool{Name: "ping"}, pingingTool)                                                                                                                                                             // performs a ping
	mcp.AddTool(server, &mcp.Tool{Name: "log"}, loggingTool)                                                                                                                                                              // performs a log
	mcp.AddTool(server, &mcp.Tool{Name: "sample"}, samplingTool)                                                                                                                                                          // performs sampling
	mcp.AddTool(server, &mcp.Tool{Name: "elicit"}, elicitingTool)                                                                                                                                                         // performs elicitation
	mcp.AddTool(server, &mcp.Tool{Name: "roots"}, rootsTool)                                                                                                                                                              // lists roots

	mcp.AddTool(server, &mcp.Tool{Name: "clear", Description: "ROR tool to clear out all data folders."}, clearOutDataCacheTool) // returns structured output
	mcp.AddTool(server, &mcp.Tool{Name: "add", Description: "ROR tool to add a new data folder."}, addDataCacheTool)             // returns structured output

	// Add a basic prompt.
	server.AddPrompt(&mcp.Prompt{Name: "greet"}, prompt)

	// Add an embedded resource.
	server.AddResource(&mcp.Resource{
		Name:     "info",
		MIMEType: "text/plain",
		URI:      "embedded:info",
	}, embeddedResource)
	server.AddResource(&mcp.Resource{
		Name:     "data",
		MIMEType: "text/plain",
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
	res, err := session.ListRoots(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("listing roots failed: %v", err)
	}
	var allroots []string
	for _, r := range res.Roots {
		uri_temp := strings.TrimPrefix(r.URI, "file://")
		allroots = append(allroots, uri_temp)
	}
	dir_path := allroots[0] // should be "./.ror/config"
	return dir_path, nil
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
		text = config.Data.Path // this is relative to the ror directory
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

type args struct {
	Name string `json:"name" jsonschema:"the name to say hi to"`
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

// if we clear out the data cache we need a result that reports the total numbers
type resultDataCache struct {
	Message    string `json:"message" jsonschema:"the message to convey"`
	NumStudies int    `json:"numstudies" jsonschema:"the number of DICOM studies"`
	NumSeries  int    `json:"numseries" jsonschema:"the number of DICOM image series"`
	NumImages  int    `json:"numimages" jsonschema:"the number of DICOM images"`
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

func addDataCacheTool(ctx context.Context, req *mcp.CallToolRequest, args *args) (*mcp.CallToolResult, *resultDataCache, error) {
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

	res, err := req.Session.Elicit(ctx, &mcp.ElicitParams{
		Message: "Where is the data that should be added",
		RequestedSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"newdatapath": {Type: "string", Description: "The directory path on the local machine that contains DICOM data to import.", Examples: []any{"file://somewhere/here/"}},
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("eliciting failed: %v", err)
	}
	// use res to add the data
	fmt.Printf("%v", res)

	// this will use input_dir to write
	if !config.writeConfig() {
		return nil, &resultDataCache{Message: "Error could not write config file into ror directory."}, err
	}

	// return that we cleared out the data cache, return the current number of dataset as well
	return nil, &resultDataCache{Message: "Added the data path", NumStudies: 0, NumSeries: 0, NumImages: 0}, nil
}

// structuredTool returns a structured result.
func structuredTool(ctx context.Context, req *mcp.CallToolRequest, args *args) (*mcp.CallToolResult, *result, error) {
	return nil, &result{Message: "Hi " + args.Name}, nil
}

// rorTool returns a structured result.
func rorTool(ctx context.Context, req *mcp.CallToolRequest, args *args) (*mcp.CallToolResult, *result, error) {
	return nil, &result{Message: "ROR is a tools to create workflows for the research PACS"}, nil
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

func rootsTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	res, err := req.Session.ListRoots(ctx, nil)
	if err != nil {
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
		suggestions = []string{"ror init", "ror trigger", "ror config"}
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
