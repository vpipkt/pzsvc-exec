// Copyright 2016, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

type configType struct {
	CliCmd      string
	VersionCmd	string
	VersionStr	string
	PzAddr      string
	AuthEnVar	string
	SvcName     string
	URL         string
	Port        int
	Description	string
	Attributes	map[string]string
}

type outStruct struct {
	InFiles		map[string]string
	OutFiles	map[string]string
	ProgReturn	string
	Errors		[]string
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("error: Insufficient parameters.  You must specify a config file.")
		return
	}
	
	// First argument after the base call should be the path to the config file.
	// ReadFile returns the contents of the file as a byte buffer.
	configBuf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("error:", err)
	}
	var configObj configType
	err = json.Unmarshal(configBuf, &configObj)
	if err != nil {
		fmt.Println("error:", err.Error())
	}
	canReg, canFile, hasAuth := checkConfig(&configObj)

	var authKey string
	if hasAuth {
		authKey = os.Getenv(configObj.AuthEnVar)
		if authKey == "" {
			fmt.Println("Error: no auth key at AuthEnVar.  Registration disabled, and client will have to provide authKey.")
			hasAuth = false
			canReg = false
		}
	}

	if configObj.Port <= 0 {
		configObj.Port = 8080
	}
	portStr := ":" + strconv.Itoa(configObj.Port)
	
	version := getVersion(configObj)

	if canReg {
		fmt.Println("About to manage registration.")
		err = pzsvc.ManageRegistration(	configObj.SvcName,
										configObj.Description,
										configObj.URL,
										configObj.PzAddr,
										version,
										authKey,
										configObj.Attributes )
		if err != nil {
			fmt.Println("error:", err.Error())
		}
		fmt.Println("Registration managed.")
	}
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.URL.Path {
		case "/":
			{
				fmt.Fprintf(w, "Hello.  This is pzsvc-exec")
				if configObj.SvcName != "" {
					fmt.Fprintf(w, ", serving %s", configObj.SvcName)
				}
				fmt.Fprintf(w, ".\nWere you possibly looking for the /help or /execute endpoints?")
			}
		case "/execute":
			{
				// the other options are shallow and informational.  This is the
				// place where the work gets done.
				output := execute (w, r, configObj, authKey, version, canFile)
				printJSON(w, output)
			}
		case "/description":
			if configObj.Description == "" {
				fmt.Fprintf(w, "No description defined")
			} else {
				fmt.Fprintf(w, configObj.Description)
			}
		case "/attributes":
			if configObj.Attributes == nil {
				fmt.Fprintf(w, "{ }")
			} else {
				printJSON(w, configObj.Attributes)
			}
		case "/help":
			printHelp(w)
		case "/version":
			fmt.Fprintf(w, version)
		default:
			fmt.Fprintf(w, "Endpoint undefined.  Try /help?\n")
		}
	})

	log.Fatal(http.ListenAndServe(portStr, nil))
}

// execute does the primary work for pzsvc-exec.  Given a request and various
// blocks of config data, it creates a temporary folder to work in, downloads
// any files indicated in the request (if the configs support it), executes
// the command indicated by the combination of request and configs, uploads
// any files indicated by the request (if the configs support it) and cleans
// up after itself
func execute(w http.ResponseWriter, r *http.Request, configObj configType, authKey, version string, canFile bool) outStruct {

	var output outStruct
	output.InFiles = make(map[string]string)
	output.OutFiles = make(map[string]string)

	if r.Method != "POST" {
		output.Errors = append(output.Errors, "This endpoint does not support that method.  Please try again with POST.")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return output
	}

	cmdParam := r.FormValue("cmd")
	cmdParamSlice := splitOrNil(cmdParam, " ")
	cmdConfigSlice := splitOrNil(configObj.CliCmd, " ")
	cmdSlice := append(cmdConfigSlice, cmdParamSlice...)

	inFileSlice := splitOrNil(r.FormValue("inFiles"), ",")
	outTiffSlice := splitOrNil(r.FormValue("outTiffs"), ",")
	outTxtSlice := splitOrNil(r.FormValue("outTxts"), ",")
	outGeoJSlice := splitOrNil(r.FormValue("outGeoJson"), ",")
	
	if 	r.FormValue("authKey") != "" {
		authKey = r.FormValue("authKey")
	}

	if !canFile && (len(inFileSlice) + len(outTiffSlice) + len(outTxtSlice) + len(outGeoJSlice) != 0) {
		output.Errors = append(output.Errors, "Cannot complete.  File up/download not enabled in config file.")
		w.WriteHeader(http.StatusForbidden)
		return output
	}

	if authKey == "" && (len(inFileSlice) + len(outTiffSlice) + len(outTxtSlice) + len(outGeoJSlice) != 0) {
		output.Errors = append(output.Errors, "Cannot complete.  Auth Key not available.")
		w.WriteHeader(http.StatusForbidden)
		return output
	}

	runID, err := psuUUID()
	handleError(&output, err, w, http.StatusInternalServerError)

	err = os.Mkdir("./"+runID, 0777)
	handleError(&output, err, w, http.StatusInternalServerError)
	defer os.RemoveAll("./" + runID)

	err = os.Chmod("./"+runID, 0777)
	handleError(&output, err, w, http.StatusInternalServerError)

	// this is done to enable use of handleFList, which lets us
	// reduce a fair bit of code duplication in plowing through
	// our upload/download lists.  handleFList gets used a fair
	// bit more after the execute call.
	downlFunc := func(dataID, fType string) (string, error) {
		return pzsvc.Download(dataID, runID, configObj.PzAddr, authKey)
	}
	handleFList(inFileSlice, downlFunc, "", &output, output.InFiles, w)

	if len(cmdSlice) == 0 {
		output.Errors = append(output.Errors, `No cmd or CliCmd.  Please provide "cmd" param.`)
		w.WriteHeader(http.StatusBadRequest)
		return output
	}

	fmt.Printf("Executing \"%s\".\n", configObj.CliCmd+" "+cmdParam)

	// we're calling this from inside a temporary subfolder.  If the
	// program called exists inside the initial pzsvc-exec folder, that's
	// probably where it's called from, and we need to acccess it directly.
	_, err = os.Stat(fmt.Sprintf("./%s", cmdSlice[0]))
	if err == nil || !(os.IsNotExist(err)){
		// ie, if there's a file in the start folder named the same thing
		// as the base command
		cmdSlice[0] = ("../" + cmdSlice[0])
	}

	clc := exec.Command(cmdSlice[0], cmdSlice[1:]...)
	clc.Dir = runID

	var b bytes.Buffer
	clc.Stdout = &b
	clc.Stderr = os.Stderr

	err = clc.Run()
	handleError(&output, err, w, http.StatusBadRequest)
	
	output.ProgReturn = b.String()
				
	fmt.Printf("Program output: %s\n", output.ProgReturn)

	attMap := make(map[string]string)
	attMap["algoName"] = configObj.SvcName
	attMap["algoVersion"] = version
	attMap["algoCmd"] = configObj.CliCmd + " " + cmdParam
	attMap["algoProcTime"] = time.Now().UTC().Format("20060102.150405.99999")
	
	// this is the other spot that handleFlist gets used, and works on the
	// same principles.

	ingFunc := func(fName, fType string) (string, error) {
		return pzsvc.IngestFile(fName, runID, fType, configObj.PzAddr, configObj.SvcName, version, authKey, attMap)
	}

	handleFList(outTiffSlice, ingFunc, "raster", &output, output.OutFiles, w)
	handleFList(outTxtSlice, ingFunc, "text", &output, output.OutFiles, w)
	handleFList(outGeoJSlice, ingFunc, "geojson", &output, output.OutFiles, w)
	
	return output
}

type rangeFunc func(string, string) (string, error)

func handleFList(fList []string, lFunc rangeFunc, fType string, output *outStruct, fileRec map[string]string, w http.ResponseWriter) {
	for _, f := range fList {
		outStr, err := lFunc(f, fType)
		if err != nil {
			output.Errors = append(output.Errors, err.Error())
			w.WriteHeader(http.StatusBadRequest)
		} else {
			fileRec[f] = outStr
		}
	}
}

func handleError(output *outStruct, err error, w http.ResponseWriter, httpStat int) {
	if (err != nil) {
		output.Errors = append(output.Errors, err.Error())
		w.WriteHeader(httpStat)
	}
	return
}

func splitOrNil(inString, knife string) []string {
	if inString == "" {
		return nil
	}
	return strings.Split(inString, knife)
}

func psuUUID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func printJSON(w http.ResponseWriter, output interface{}) {
	outBuf, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(w, `{"Errors":"Json marshalling failure.  Data not reportable."}`)
	}

	fmt.Fprintf(w, "%s", string(outBuf))
}

func getVersion(configObj configType) string {
	vCmdSlice := splitOrNil(configObj.VersionCmd, " ")
	if vCmdSlice != nil {
		vCmd := exec.Command (vCmdSlice[0], vCmdSlice[1:]...)
		verB, err := vCmd.Output()
		if err != nil {
			fmt.Println("error: VersionCmd failed: " + err.Error())
		}
		return string(verB)
	}
	return configObj.VersionStr
}

// checkConfig takes an input config file, checks it over for issues,
// and outputs any issues or concerns to std.out.  It returns whether
// or not the config file permits autoregistration, and whether or not
// it permits file upload/download.
func checkConfig (configObj *configType) (bool, bool, bool) {
	canReg := true
	canFile := true
	hasAuth := true
	if configObj.CliCmd == "" {
		fmt.Println(`Config: Warning: CliCmd is blank.  This is a major security vulnerability.`)
	}
	
	if configObj.PzAddr == "" {
		fmt.Println(`Config: PzAddr not specified.  Autoregistration and file upload/download disabled.`)
		canFile = false
		hasAuth = false
		canReg = false
	} else if configObj.AuthEnVar == "" {
		fmt.Println(`Config: AuthEnVar was not specified.  Client will have to provide authKey.  Autoregistration disabled.`)
		hasAuth = false
		canReg = false
	} else if configObj.SvcName == "" {
		fmt.Println(`Config: SvcName not specified.  Autoregistration disabled.`)
		canReg = false
	} else if configObj.URL == "" {
		fmt.Println(`Config: URL not specified for this service.  Autoregistration disabled.`)
		canReg = false
	}
	
	if !canFile {
		if configObj.PzAddr != "" {
			fmt.Println(`Config: PzAddr was specified, but is meaningless without upload/download/autoregistration.`)
		}
		if configObj.VersionCmd != "" {
			fmt.Println(`Config: VersionCmd was specified, but is meaningless without upload/download/autoregistration.`)
		}
		if configObj.VersionStr != "" {
			fmt.Println(`Config: VersionStr was specified, but is meaningless without upload/download/autoregistration.`)
		}
		if configObj.AuthEnVar != "" {
			fmt.Println(`Config: AuthEnVar was specified, but is meaningless without upload/download/autoregistration.`)
		}	
	} else {
		if configObj.VersionCmd == "" && configObj.VersionStr == "" {
			fmt.Println(`Config: neither VersionCmd nor VersionStr was specified.  Version will be left blank.`)
		}
		if configObj.VersionCmd != "" && configObj.VersionStr != "" {
			fmt.Println(`Config: Both VersionCmd and VersionStr were specified.  Redundant.  Default to VersionCmd.`)
		}
	}
	
	if !canReg {
		if configObj.SvcName != "" {
			fmt.Println(`Config: SvcName was specified, but is meaningless without autoregistration.`)
		}
		if configObj.URL != "" {
			fmt.Println(`Config: URL was specified, but is meaningless without autoregistration.`)
		}
	} else {
		if configObj.Description == "" {
			fmt.Println(`Config: Description not specified.  When autoregistering, descriptions are strongly encouraged.`)
		}
	}

	if configObj.Port <= 0 {
		fmt.Println(`Config: Port not specified, or incorrect format.  Default to 8080.`)
	}
	
	return canReg, canFile, hasAuth
}

func printHelp(w http.ResponseWriter) {
	fmt.Fprintln(w, `pzsvc-exec endpoints as follows:`)
	fmt.Fprintln(w, `- '/': entry point.  Displays base command if any, and suggests other endpoints.`)
	fmt.Fprintln(w, `- '/execute': The meat of the program.  Downloads files, executes on them, and uploads the results.`)
	fmt.Fprintln(w, `See the Service Request Format section of the Readme for interface details.`)
	fmt.Fprintln(w, `(Readme available at https://github.com/venicegeo/pzsvc-exec).`)
	fmt.Fprintln(w, `- '/description': When enabled, provides a description of this particular pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/attributes': When enabled, provides a list of key/value attributes for this pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/version': When enabled, provides version number for the application served by this pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/help': This screen.`)
}
