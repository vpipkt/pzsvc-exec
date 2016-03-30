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
	"fmt"
	"log"
	"io/ioutil"
	"encoding/json"
	"os"
	"os/exec"
	//"strconv"
	"net/http"
	"bytes"
)

type configType struct {
	cliProg string
	cliCmd string
}

func main() {

	// first argument after the program name should be the path to the config file.
	// ReadFile returns the contents of the file as a byte buffer.
	configBuf, _ := ioutil.ReadFile(os.Args[1])

	var configObj configType
	json.Unmarshal(configBuf, &configObj)

	//- check that config file data is complete.  Checks other dependency requirements (if any)
	//- register on Pz

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.URL.Path{
			case "/":
				fmt.Fprintf(w, "hello.")
			case "/execute": {

				var paramString string
				if r.Method == "GET" {
					paramString = r.URL.Query().Get("param")
				} else {
					paramString = r.FormValue("param")
				}


				//TODO: this should be removed once it is no longer necessary for debugging.  Reflection attack vuln
				fmt.Fprintf(w, "param string: %s\n", paramString)


 
				var b bytes.Buffer
				var clc exec.Cmd
				clc.Path = "./bf-dummycmd"
				clc.Args = []string{"./bf-dummycmd", "process", paramString}
				clc.Stdout = &b
				clc.Stderr = os.Stderr

				err := clc.Run()
				if err != nil {
					fmt.Fprintf(w, err.Error())
				} else {
					fmt.Fprintf(w, b.String())
				}
			}
			case "/help":
				help(w)
			default:
				other(w)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func other(w http.ResponseWriter) {
	fmt.Fprintf(w, "Command undefined.  Try help?\n")
}

func help(w http.ResponseWriter) {
	fmt.Fprintf(w, "We're sorry, help is not yet implemented\n")
}

