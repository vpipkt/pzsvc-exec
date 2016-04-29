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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

type ConfigType struct {
	CliCmd      string
	PzAddr      string
	SvcName     string
	Url         string
	Port        int
	Description string
	Attributes  map[string]string
}

type OutStruct struct {
	InFiles    map[string]string
	OutFiles   map[string]string
	ProgReturn string
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("error: Insufficient parameters.  You must specify a config file.")
		return
	}
	
	// first argument after the base call should be the path to the config file.
	// ReadFile returns the contents of the file as a byte buffer.
	configBuf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("error:", err)
	}

	var configObj ConfigType
	err = json.Unmarshal(configBuf, &configObj)
	if err != nil {
		fmt.Println("error:", err)
	}

	if configObj.Port <= 0 {
		configObj.Port = 8080
	}

	portStr := ":" + strconv.Itoa(configObj.Port)

	if configObj.SvcName != "" && configObj.PzAddr != "" {
fmt.Println("About to manage registration.")
		err = pzsvc.ManageRegistration(configObj.SvcName, configObj.Description, configObj.Url, configObj.PzAddr)
		if err != nil {
			fmt.Println("error:", err)
		}
fmt.Println("Registration managed.")

	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, "hello.")
		case "/execute":
			{

				var cmdParam string
				var inFileStr string
				var outTiffStr string
				var outTxtStr string
				var outGeoJStr string
				var usePz string

				// might be time to start looking into that "help" thing.

				if r.Method == "GET" {
					cmdParam = r.URL.Query().Get("cmd")
					inFileStr = r.URL.Query().Get("inFiles")
					outTiffStr = r.URL.Query().Get("outTiffs")
					outTxtStr = r.URL.Query().Get("outTxts")
					outGeoJStr = r.URL.Query().Get("outGeoJson")
					usePz = r.URL.Query().Get("pz")
				} else {
					cmdParam = r.FormValue("cmd")
					inFileStr = r.FormValue("inFiles")
					outTiffStr = r.FormValue("outTiffs")
					outTxtStr = r.FormValue("outTxts")
					outGeoJStr = r.FormValue("outGeoJson")
					usePz = r.FormValue("pz")
				}

				cmdConfigSlice := splitOrNil(configObj.CliCmd, " ")
				cmdParamSlice := splitOrNil(cmdParam, " ")
				cmdSlice := append(cmdConfigSlice, cmdParamSlice...)

				inFileSlice := splitOrNil(inFileStr, ",")
				outTiffSlice := splitOrNil(outTiffStr, ",")
				outTxtSlice := splitOrNil(outTxtStr, ",")
				outGeoJSlice := splitOrNil(outGeoJStr, ",")

				var output OutStruct

				uuid, err := exec.Command("uuidgen").Output()
				if err != nil {
					fmt.Fprintf(w, err.Error())
				}

				runId := string(uuid)

				err = os.Mkdir("./"+runId, 0777)
				if err != nil {
					fmt.Fprintf(w, err.Error())
				}
				defer os.RemoveAll("./" + runId)

				err = os.Chmod("./"+runId, 0777)
				if err != nil {
					fmt.Fprintf(w, err.Error())
				}
				
				if len(inFileSlice) > 0 {
					output.InFiles = make(map[string]string)
				}
				if len(outTiffSlice)+len(outTxtSlice)+len(outGeoJSlice) > 0 {
					output.OutFiles = make(map[string]string)
				}

				for i, inFile := range inFileSlice {

					fmt.Printf("Downloading file %s - %d of %d.\n", inFile, i, len(inFileSlice))
					fname, err := pzsvc.Download(inFile, runId, configObj.PzAddr)
					if err != nil {
						fmt.Fprintf(w, err.Error())
						fmt.Printf("Download failed.  %s", err.Error())
					} else {
						output.InFiles[inFile] = fname
						fmt.Printf("Successfully downloaded %s.", fname)
					}
				}

				if len(cmdSlice) == 0 {
					fmt.Fprintf(w, `No cmd specified in config file.  Please provide "cmd" param.`)
					break
				}

				fmt.Printf("Executing \"%s\".\n", configObj.CliCmd+" "+cmdParam)

				clc := exec.Command(cmdSlice[0], cmdSlice[1:]...)
				clc.Dir = runId
				var b bytes.Buffer
				clc.Stdout = &b
				clc.Stderr = os.Stderr

				err = clc.Run()
				if err != nil {
					fmt.Fprintf(w, err.Error())
				}
				fmt.Printf("Program output: %s\n", b.String())
				output.ProgReturn = b.String()

				for i, outTiff := range outTiffSlice {
					fmt.Printf("Uploading Tiff %s - %d of %d.\n", outTiff, i, len(outTiffSlice))
					dataId, err := pzsvc.IngestTiff(outTiff, runId, configObj.PzAddr, cmdSlice[0])
					if err != nil {
						fmt.Fprintf(w, err.Error())
						fmt.Printf("Upload failed.  %s", err.Error())
					} else {
						output.OutFiles[outTiff] = dataId
					}
				}

				for i, outTxt := range outTxtSlice {
					fmt.Printf("Uploading Txt %s - %d of %d.\n", outTxt, i, len(outTxtSlice))
					dataId, err := pzsvc.IngestTxt(outTxt, runId, configObj.PzAddr, cmdSlice[0])
					if err != nil {
						fmt.Fprintf(w, err.Error())
						fmt.Printf("Upload failed.  %s", err.Error())
					} else {
						output.OutFiles[outTxt] = dataId
					}
				}

				for i, outGeoJ := range outGeoJSlice {
					fmt.Printf("Uploading GeoJson %s - %d of %d.\n", outGeoJ, i, len(outGeoJSlice))
					dataId, err := pzsvc.IngestGeoJson(outGeoJ, runId, configObj.PzAddr, cmdSlice[0])
					if err != nil {
						fmt.Fprintf(w, err.Error())
						fmt.Printf("Upload failed.  %s", err.Error())
					} else {
						output.OutFiles[outGeoJ] = dataId
					}
				}

				outBuf, err := json.Marshal(output)
				if err != nil {
					fmt.Fprintf(w, err.Error())
				}

				outStr := string(outBuf)

				if usePz != "" {
					outStr = strconv.QuoteToASCII(outStr)
					// TODO: clean this up a bit, and possibly move it back into
					// the support function.
					// - possibly include metadata to help on results searches?  Talk with Marge on where/how to put it in.
					outStr = fmt.Sprintf(`{ "dataType": { "type": "text", "content": "%s" "mimeType": "text/plain" }, "metadata": {} }`, outStr)
				}

				fmt.Fprintf(w, outStr)
				
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
				// convert attributes back into Json
				// this might require specifying the interface a bit better.
				//					fmt.Fprintf(w, configObj.Attributes)
			}
		case "/help":
			fmt.Fprintf(w, "We're sorry, help is not yet implemented.\n")
		default:
			fmt.Fprintf(w, "Command undefined.  Try help?\n")
		}
	})

	log.Fatal(http.ListenAndServe(portStr, nil))
}

func splitOrNil(inString, knife string) []string {
	if inString == "" {
		return nil
	}
	return strings.Split(inString, knife)
}
