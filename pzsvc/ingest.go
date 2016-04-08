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

package pzsvc

// TODO: Will probably want to rename/rearrange/refactor the pzsvc-exec package so as to better conform
// to go coding standards/naming conventions at some point.

import (
	"bytes"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type IngJsonResp struct {
	Type string
	JobId string
}

type StatusJsonResult struct {
	ApiKey string
	DataId string
}

type StatusJsonResp struct {
	Type string
	JobId string
	Result StatusJsonResult
	Status string
}

func Download(dataId, address string) error {

	jsonStr := fmt.Sprintf(`body: { "userName": "my-api-key-38n987", "dataId": "%s"}`, dataId)
	jsonBuf := bytes.NewBufferString(jsonStr)

	resp, err := http.Post(address, "application/json", jsonBuf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	contDisp := resp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contDisp)
	filename := params["filename"]
	if filename == "" { filename = "dummy.txt" }
	filepath := fmt.Sprintf(`./%s`, filename)

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	io.Copy(out, resp.Body)
	return nil
}



func getStatus (jobId, jobAddress string) (string, error) {

	var respObj StatusJsonResp
	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "get", "jobId": "%s" } }`, jobId)
	jsonBuf := bytes.NewBufferString(jsonStr)

	lastErr := errors.New("Never completed.")

	for i:=0; i<600; i++{
		resp, err := http.Post(jobAddress, "application/json", jsonBuf)
		if err != nil {
			return "", err
		}

		var respJson []byte
		_, err = io.ReadFull (resp.Body, respJson)
		if err != nil {
			return "", err
		}

		err = json.Unmarshal(respJson, &respObj)
		if err != nil {
			return "", err
		}

		if respObj.Status == "Complete" {
			lastErr = nil
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return respObj.Result.DataId, lastErr	
}



func ingestMultipart (bodyStr, jobAddress, filename string) (string, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err := writer.WriteField("body", "{}")
	if err != nil {
		return "", err
	}

	if filename != "" {
		file, err := os.Open(fmt.Sprintf(`./%s`, filename))
		if err != nil {
			return "", err
		}

		defer file.Close()

		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return "", err
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return "", err
		}
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	fileReq, err := http.NewRequest("POST", jobAddress, body)
	if err != nil {
		return "", err
	}
	
	fileReq.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return "", err
	}
	respBuf := &bytes.Buffer{}

	_, err = respBuf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}

fmt.Println(string(respBuf.Bytes()))

	var respObj IngJsonResp
	err = json.Unmarshal(respBuf.Bytes(), &respObj)
	if err != nil {
		fmt.Println("error:", err)
	}	

	dataId, err := getStatus(respObj.JobId, jobAddress)

	return dataId, err
}



func IngestTiff (filename, jobAddress string) (string, error) {

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" : { "dataType": { "type": "raster" }, "metadata": { "name": "%s", "description": "raster uploaded by pzsvc-exec.", "classType": { "classification": "unclassified" } } } } }`, filename)

	return ingestMultipart(jsonStr, jobAddress, filename)
}



func IngestTxt (filename, jobAddress string) (string, error) {
	textblock, err := ioutil.ReadFile(fmt.Sprintf(`./%s`, filename))
	if err != nil {
		return "", err
	}

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" :{ "dataType": { "type": "text", "mimeType": "application/text", "content": "%s" }, "metadata": { "name": "%s", "description": "text output from pzsvc-exec.", "classType": { "classification": "unclassified" } } } } }`, string(textblock), filename)

//TODO: consider upgrading the description field, both here and for the TIFF upload.  Possibly include name of CLI program?

	return ingestMultipart(jsonStr, jobAddress, "")
}





/*

**** Ingest call (in Post - plus a tiff file.  ???)
https://pz-gateway.stage.geointservices.io/job
{ 	"userName": "my-api-key-38n987",
 	"jobType": {
 		"type": "ingest",
 		"host": "true",
 		"data" : {
 			"dataType": {
 				"type": "raster"
 			},
 			"metadata": {
 				"name": "My Test Raster",
 				"description": "This is a test.",
 				"classType": {
 					"classification": "unclassified"
 				}
			}
 		}
 	}
 }

**** response from same
{
  "type": "job",
  "jobId": "b91673c0-4140-4a8a-a27a-39ec251cfac3"
}
**** call to ask for data on status
https://pz-gateway.stage.geointservices.io/job
{
"userName": "my-api-key-38n987",
"jobType": {
	"type": "get",
	"jobId": "b91673c0-4140-4a8a-a27a-39ec251cfac3"
	}
}


**** format of the return from the get call, after a successful ingest
{
  "type": "status",
  "jobId": "b91673c0-4140-4a8a-a27a-39ec251cfac3",
  "result": {
    "type": "data",
    "dataId": "92e3ba56-a0be-4039-88d3-e988e2052d49"
  },
  "status": "Success",
  "progress": {
    "percentComplete": 100
  }
}

**** format of call to get the actual file
https://pz-gateway.stage.geointservices.io/file
{
	"userName": "my-api-key-38n987",
	"dataId": "8e510728-0fb5-419b-9838-1308a202be33"
}

other valid dataId: b4ac11f7-eedd-42ff-87f8-fc48e1079d2a


*/


