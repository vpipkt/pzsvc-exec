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
	PzAddr      string
	SvcName     string // meaningless without PzAddr, URL
	URL         string // meaningless without PzAddr, SvcName
	Port        int
	Description	string
	AuthEnVar	string // meaningless without PzAddr, URL, SvcName
	ImageReqs	map[string]string // meaningless without PzAddr, URL, SvcName
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

	cmdConfigSlice := splitOrNil(configObj.CliCmd, " ")

	if configObj.Port <= 0 {
		configObj.Port = 8080
	}

	portStr := ":" + strconv.Itoa(configObj.Port)
	authKey := os.Getenv(configObj.AuthEnVar)

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
	
	for key, val := range configObj.ImageReqs {
		configObj.Attributes["imgReq - " + key] = val
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
				//var usePz string

				var output outStruct 

				// might be time to start looking into that "help" thing.

				if r.Method == "GET" {
					cmdParam = r.URL.Query().Get("cmd")
					inFileStr = r.URL.Query().Get("inFiles")
					outTiffStr = r.URL.Query().Get("outTiffs")
					outTxtStr = r.URL.Query().Get("outTxts")
					outGeoJStr = r.URL.Query().Get("outGeoJson")
					//usePz = r.URL.Query().Get("pz")
				} else {
					cmdParam = r.FormValue("cmd")
					inFileStr = r.FormValue("inFiles")
					outTiffStr = r.FormValue("outTiffs")
					outTxtStr = r.FormValue("outTxts")
					outGeoJStr = r.FormValue("outGeoJson")
					//usePz = r.FormValue("pz")
				}

				cmdParamSlice := splitOrNil(cmdParam, " ")
				cmdSlice := append(cmdConfigSlice, cmdParamSlice...)

				inFileSlice := splitOrNil(inFileStr, ",")
				outTiffSlice := splitOrNil(outTiffStr, ",")
				outTxtSlice := splitOrNil(outTxtStr, ",")
				outGeoJSlice := splitOrNil(outGeoJStr, ",")

				runID, err := psuUUID()
				if err != nil {
					output.Errors = append(output.Errors, err.Error())
				}

				err = os.Mkdir("./"+runID, 0777)
				if err != nil {
					output.Errors = append(output.Errors, err.Error()+"(1)")
				}
				defer os.RemoveAll("./" + runID)

				err = os.Chmod("./"+runID, 0777)
				if err != nil {
					output.Errors = append(output.Errors, err.Error()+"(2)")
				}

				if len(inFileSlice) > 0 {
					output.InFiles = make(map[string]string)
				}
				if len(outTiffSlice)+len(outTxtSlice)+len(outGeoJSlice) > 0 {
					output.OutFiles = make(map[string]string)
				}

				for i, inFile := range inFileSlice {

					fmt.Printf("Downloading file %s - %d of %d.\n", inFile, i, len(inFileSlice))
					fname, err := pzsvc.Download(inFile, runID, configObj.PzAddr, authKey)
					if err != nil {
						output.Errors = append(output.Errors,err.Error()+"(3)")
						fmt.Printf("Download failed.  %s\n", err.Error())
					} else {
						output.InFiles[inFile] = fname
						fmt.Printf("Successfully downloaded %s.\n", fname)
					}
				}

				if len(cmdSlice) == 0 {
					output.Errors = append(output.Errors, `No cmd or CliCmd.  Please provide "cmd" param.(4)`)
					break
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
				if err != nil {
					output.Errors = append(output.Errors, err.Error()+"(5)")
				}
				output.ProgReturn = b.String()
				
				fmt.Printf("Program output: %s\n", output.ProgReturn)

				var attMap map[string]string
				attMap = make(map[string]string)
				attMap["algoName"] = configObj.SvcName
				attMap["algoVersion"] = version
				attMap["algoCmd"] = configObj.CliCmd + cmdParam
				attMap["algoProcTime"] = time.Now().UTC().Format("20060102.150405.99999")

				for i, outTiff := range outTiffSlice {
					fmt.Printf("Uploading Tiff %s - %d of %d.\n", outTiff, i, len(outTiffSlice))
					dataID, err := pzsvc.IngestLocalTiff(outTiff, runID, configObj.PzAddr, cmdSlice[0], version, authKey, attMap)
					if err != nil {
						output.Errors = append(output.Errors, err.Error()+"(6)")
						fmt.Printf("Upload failed.  %s", err.Error()+"(7)")
					} else {
						output.OutFiles[outTiff] = dataID
						fmt.Printf("Upload complete: %s", dataID)
					}
				}

				for i, outTxt := range outTxtSlice {
					fmt.Printf("Uploading Txt %s - %d of %d.\n", outTxt, i, len(outTxtSlice))
					dataID, err := pzsvc.IngestLocalTxt(outTxt, runID, configObj.PzAddr, cmdSlice[0], version, authKey, attMap)
					if err != nil {
						output.Errors = append(output.Errors, err.Error()+"(8)")
						fmt.Printf("Upload failed.  %s", err.Error()+"(9)")
					} else {
						output.OutFiles[outTxt] = dataID
						fmt.Printf("Upload complete: %s", dataID)
					}
				}

				for i, outGeoJ := range outGeoJSlice {
					fmt.Printf("Uploading GeoJson %s - %d of %d.\n", outGeoJ, i, len(outGeoJSlice))
					dataID, err := pzsvc.IngestLocalGeoJSON(outGeoJ, runID, configObj.PzAddr, cmdSlice[0], configObj.VersionStr, authKey, attMap)
					if err != nil {
						output.Errors = append(output.Errors, err.Error()+"(10)")
						fmt.Printf("Upload failed.  %s", err.Error())
					} else {
						output.OutFiles[outGeoJ] = dataID
						fmt.Printf("Upload complete: %s", dataID)
					}
				}
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
			fmt.Fprintf(w, "We're sorry, help is not yet implemented.\n")
		case "/version":

			b, err := exec.Command(vCmdSlice[0], vCmdSlice[1:]...).Output()
			if err == nil {
				fmt.Fprintf(w, string(b))
			}
			
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

func psuUUID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func printJSON (w http.ResponseWriter, output interface{}) {
	outBuf, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(w, `{"Errors":"Json marshalling failure.  Data not reportable."}`)
	}

	fmt.Fprintf(w, "%s", string(outBuf))
}