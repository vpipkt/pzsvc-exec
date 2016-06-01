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
	VersionCmd	string // should have either this or VersionStr - not both.  Meaningless without PzAddr
	VersionStr	string // should have either this or VersionCmd - not both.  Meaningless without PzAddr
	PzAddr      string // useless without AuthEnVar
	AuthEnVar	string // meaningless without PzAddr
	SvcName     string // meaningless without PzAddr.  Nearly meaningless without URL
	URL         string // meaningless without PzAddr, SvcName
	Port        int // defaults to 8080
	Description	string
	Attributes	map[string]string // meaningless without PzAddr, URL, SvcName
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

	authKey := os.Getenv(configObj.AuthEnVar)
	cmdConfigSlice := splitOrNil(configObj.CliCmd, " ")
	if configObj.Port <= 0 {
		configObj.Port = 8080
	}
	portStr := ":" + strconv.Itoa(configObj.Port)
	
	version := configObj.VersionStr
	vCmdSlice := splitOrNil(configObj.VersionCmd, " ")
	if vCmdSlice != nil {
		vCmd := exec.Command (vCmdSlice[0], vCmdSlice[1:]...)
		verB, err := vCmd.Output()
		if err != nil {
			fmt.Println("error: VersionCmd failed: " + err.Error())
		}
		version = string(verB)
	}

	if configObj.SvcName != "" && configObj.PzAddr != "" && configObj.URL != "" {
		fmt.Println("About to manage registration.")
		err = pzsvc.ManageRegistration(	configObj.SvcName,
										configObj.Description,
										configObj.URL + "/execute",
										configObj.PzAddr,
										version,
										authKey,
										configObj.Attributes )
		if err != nil {
			fmt.Println("error:", err.Error())
		}
		fmt.Println("Registration managed.")
	}
	
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		switch req.URL.Path {
		case "/":
			fmt.Fprintf(w, "hello.")
		case "/execute":
			{
				// the other options are shallow and informational.  This is the
				// place where the work gets done.
				output := execute (req, cmdConfigSlice, configObj, authKey, version)
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
	// might be time to start looking into that "help" thing.
			fmt.Fprintf(w, "We're sorry, help is not yet implemented.\n")
		case "/version":
			fmt.Fprintf(w, version)
			
		default:
			fmt.Fprintf(w, "Command undefined.  Try help?\n")
		}
	})

	log.Fatal(http.ListenAndServe(portStr, nil))
}

// execute does the primary work for pzsvc-exec.  Given a request and various blocks fo config data
func execute(r *http.Request, cmdConfigSlice []string, configObj configType, authKey, version string) outStruct {

	var output outStruct
	output.InFiles = make(map[string]string)
	output.OutFiles = make(map[string]string)

	if r.Method == "GET" {
		output.Errors = append(output.Errors, "This endpoint no longer supports GET.  Please try again with POST")
		return output
	}

	cmdParam := r.FormValue("cmd")
	cmdParamSlice := splitOrNil(cmdParam, " ")
	cmdSlice := append(cmdConfigSlice, cmdParamSlice...)

	inFileSlice := splitOrNil(r.FormValue("inFiles"), ",")
	outTiffSlice := splitOrNil(r.FormValue("outTiffs"), ",")
	outTxtSlice := splitOrNil(r.FormValue("outTxts"), ",")
	outGeoJSlice := splitOrNil(r.FormValue("outGeoJson"), ",")

	runID, err := psuUUID()
	handleError(output.Errors, err)

	err = os.Mkdir("./"+runID, 0777)
	handleError(output.Errors, err)
	defer os.RemoveAll("./" + runID)

	err = os.Chmod("./"+runID, 0777)
	handleError(output.Errors, err)

	// this is done to enable use of handleFList, which lets us
	// reduce a fair bit of code duplication in plowing through
	// our upload/download lists.  handleFList gets used a fair
	// bit more after the execute call.
	downlFunc := func(dataID string) (string, error) {
		return pzsvc.Download(dataID, runID, configObj.PzAddr, authKey)
	}
	handleFList(inFileSlice, downlFunc, &output)

	if len(cmdSlice) == 0 {
		output.Errors = append(output.Errors, `No cmd or CliCmd.  Please provide "cmd" param.`)
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
	handleError(output.Errors, err)
	
	output.ProgReturn = b.String()
				
	fmt.Printf("Program output: %s\n", output.ProgReturn)

	attMap := make(map[string]string)
	attMap["algoName"] = configObj.SvcName
	attMap["algoVersion"] = version
	attMap["algoCmd"] = configObj.CliCmd + cmdParam
	attMap["algoProcTime"] = time.Now().UTC().Format("20060102.150405.99999")
	
	// this is the other spot that handleFlist gets used.  It's a bit mroe compicated,
	// because we want to use it three times with slight changes, but ti works on the
	// same principles.
	type uploadFunc func(string, string, string, string, string, string, map[string]string) (string, error)
	curryFunc := func(uplFunc uploadFunc) curriedUploadFunc {
		cFunc := func(fname string) (string, error) {
			return uplFunc(fname, runID, configObj.PzAddr, cmdSlice[0], version, authKey, attMap)
		}
		return cFunc	
	}
	handleFList(outTiffSlice, curryFunc(pzsvc.IngestLocalTiff), &output)
	handleFList(outTxtSlice, curryFunc(pzsvc.IngestLocalTxt), &output)
	handleFList(outGeoJSlice, curryFunc(pzsvc.IngestLocalGeoJSON), &output)
	
	return output
}

type curriedUploadFunc func(string) (string, error)

func handleFList(upFList []string, upFunc curriedUploadFunc, output *outStruct) {
	for _, upF := range upFList {
		dataID, err := upFunc(upF)
		if err != nil {
			output.Errors = append(output.Errors, err.Error())
		} else {
			output.OutFiles[upF] = dataID
		}
	}
}

func handleError(errorSlice []string, err error) []string{
	if (err == nil) {
		return errorSlice
	}
	return append(errorSlice, err.Error())
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
