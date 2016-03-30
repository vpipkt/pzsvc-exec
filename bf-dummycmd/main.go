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
	"encoding/json"
	"github.com/spf13/cobra"
)

var algoCmd = &cobra.Command{
	Use: "bf-cmdline",
	Long: "bf-cmdline is a dummy algorithm for Beachfront.",
}

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Initiate beachfront algorithm.  Example Format: dummyBF process \"{\\\"BoundBox\\\":[0,0,5,5],\\\"ImageLink\\\":\\\"dummy\\\"}\"",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		type aoiStruct struct {
			BoundBox [4]float64 // {minX, minY, maxX, maxY}
			ImageLink string // URL of image to be examined
		}

		var dataAOI aoiStruct  

		err:= json.Unmarshal([]byte(args[0]), &dataAOI)
		if err != nil {
			fmt.Println(err.Error())
		}

		dataLoad := fmt.Sprintf(
			"{ \"type\": \"Feature\", \"geometry\": { \"type\": \"LineString\", \"coordinates\": [ [%f, %f], [%f, %f] ] }, \"properties\": { \"algorithm\": \"dummy\" } }", 
		dataAOI.BoundBox[0],
		dataAOI.BoundBox[1],
		dataAOI.BoundBox[2],
		dataAOI.BoundBox[3] )

		fmt.Println(dataLoad)
	},
}

func main() {
	algoCmd.AddCommand(processCmd)
	algoCmd.Execute()
}
