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
	"strings"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

type ConfigType struct {
	CliCmd string
	PzJobAddr string
	PzFileAddr string
}

func main() {

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

	//- check that config file data is complete.  Checks other dependency requirements (if any)
	//- register on Pz

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.URL.Path{
			case "/":
				fmt.Fprintf(w, "hello.")
			case "/execute": {

				var cmdParam string
				var inFileStr string
				var outTiffStr string
				var outTxtStr string

// might be time to start looking into that "help" thing.

				if r.Method == "GET" {
					cmdParam = r.URL.Query().Get("cmd")
					inFileStr = r.URL.Query().Get("inFiles")
					outTiffStr = r.URL.Query().Get("outTiffs")
					outTxtStr = r.URL.Query().Get("outTxts")
				} else {
					cmdParam = r.FormValue("cmd")
					inFileStr = r.FormValue("inFiles")
					outTiffStr = r.FormValue("outTiffs")
					outTxtStr = r.FormValue("outTxts")
				}
fmt.Printf("params acquired\n")
				cmdConfigSlice := splitOrNil(configObj.CliCmd, " ")
				cmdParamSlice := splitOrNil(cmdParam, " ")
				cmdSlice := append(cmdConfigSlice, cmdParamSlice...)

				inFileSlice := splitOrNil(inFileStr, ",")
				outTiffSlice := splitOrNil(outTiffStr, ",")
				outTxtSlice := splitOrNil(outTxtStr, ",")
fmt.Printf("params sliced\n")
				for _, inFile := range inFileSlice {
fmt.Printf("\ndownloading file: %s\n", inFile)
					err := pzsvc.Download(inFile, configObj.PzFileAddr)
					if err != nil {
fmt.Printf(err.Error())
						fmt.Fprintf(w, err.Error())
					}
				}
fmt.Printf("downloaded\n")

				if len(cmdSlice) == 0 {
					fmt.Fprintf(w, `No cmd specified in config file.  Please provide "cmd" param.`)
					break
				}
				clc := exec.Command(cmdSlice[0], cmdSlice[1:]...)

				var b bytes.Buffer
				clc.Stdout = &b
				clc.Stderr = os.Stderr

				err = clc.Run()
				if err != nil {
					fmt.Fprintf(w, err.Error())
				} else {
					fmt.Fprintf(w, b.String())
				}
fmt.Printf("executed\n")
				for _, outTiff := range outTiffSlice {
					dataId, err := pzsvc.IngestTiff(outTiff, configObj.PzJobAddr)
					if err != nil {
						fmt.Fprintf(w, err.Error())
					} else {
						fmt.Fprintf(w, dataId)
					}
				}
fmt.Printf("Tiff sent\n")
				for _, outTxt := range outTxtSlice {
fmt.Fprintf(w, "sending Txt\n")
					dataId, err := pzsvc.IngestTxt(outTxt, configObj.PzJobAddr)
					if err != nil {
						fmt.Fprintf(w, err.Error())
						fmt.Fprintf(w, dataId)
					} else {
						fmt.Fprintf(w, dataId)
					}
fmt.Fprintf(w, "\ndone sending Txt\n")
				}
fmt.Printf("Txt sent\n")				
			}
			case "/help":
				help(w)
			default:
				other(w)
		}
	})

// might want to update Port number at some point - possibly to os.Getenve(“PORT”),
// possibly to some other defined port - talk with the Pz folks over what their
// system is
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func splitOrNil(inString, knife string) []string {
fmt.Printf("SplitOrNull: \"%s\", split by \"%s\".\n", inString, knife)
	if inString == "" {
		return nil
	}
	return strings.Split(inString, knife)
}


func other(w http.ResponseWriter) {
	fmt.Fprintf(w, "Command undefined.  Try help?\n")
}

func help(w http.ResponseWriter) {
	fmt.Fprintf(w, "We're sorry, help is not yet implemented\n")
}

