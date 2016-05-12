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

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"
)

func locString(subFold, fname string ) string {
	if subFold == "" {
		return fmt.Sprintf(`./%s`, fname)
	} else {
		return fmt.Sprintf(`./%s/%s`, subFold, fname)
	}	
}

// Sends a multi-part POST call, including optional uploaded file,
// and returns the response.  Primarily intended to support Ingest calls.
func submitMultipart(bodyStr, subFold, address, upload string) (*http.Response, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err := writer.WriteField("body", bodyStr)
	if err != nil {
		return nil, err
	}

	if upload != "" {
		
		file, err := os.Open(locString(subFold, upload))
		if err != nil {
			return nil, err
		}

		defer file.Close()

		part, err := writer.CreateFormFile("file", upload)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return nil, err
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	fileReq, err := http.NewRequest("POST", address, body)
	if err != nil {
		return nil, err
	}

	fileReq.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}

	return resp, err
}

// Downloads a file from Pz using the file access API
func Download(dataId, subFold, pzAddr string) (string, error) {

	resp, err := http.Get(pzAddr + "/file/" + dataId)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	contDisp := resp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contDisp)
	fmt.Println("%+v", params)
	filename := params["filename"]
	if filename == "" {
		filename = "dummy.txt"
	}
	
	out, err := os.Create(locString(subFold, filename))
	if err != nil {
		return "", err
	}

	defer out.Close()
	io.Copy(out, resp.Body)

	return filename, nil
}

// Given the JobId of an ingest call, polls job status
// until the job completes, then acquires and returns
// the resulting DataId.
func getDataId(jobId, pzAddr string) (string, error) {

	time.Sleep(1000 * time.Millisecond)

	for i := 0; i < 100; i++ {

		resp, err := http.Get(pzAddr + "/job/" + jobId)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return "", err
		}

		respBuf := &bytes.Buffer{}

		_, err = respBuf.ReadFrom(resp.Body)
		if err != nil {
			return "", err
		}

		fmt.Println(respBuf.String())

		var respObj JobResp
		err = json.Unmarshal(respBuf.Bytes(), &respObj)
		if err != nil {
			return "", err
		}

		if respObj.Status == "Submitted" || respObj.Status == "Running" || respObj.Status == "Pending" || respObj.Message == "Job Not Found" {
			time.Sleep(200 * time.Millisecond)
		} else if respObj.Status == "Success" {
			return respObj.Result.DataID, nil
		} else if respObj.Status == "Error" || respObj.Status == "Fail" {
			return "", errors.New(respObj.Status + ": " + respObj.Message)
		} else {
			return "", errors.New("Unknown status: " + respObj.Status)
		}
	}

	return "", errors.New("Never completed.")
}

// Handles the Pz Ingest process.  Will upload file to Pz and return the
// resulting DataId.
func ingestMultipart(bodyStr, subFold, pzAddr, filename string) (string, error) {

	resp, err := submitMultipart(bodyStr, subFold, (pzAddr + "/job"), filename)
	if err != nil {
		return "", err
	}

	respBuf := &bytes.Buffer{}

	_, err = respBuf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println(respBuf.String())

	var respObj JobResp
	err = json.Unmarshal(respBuf.Bytes(), &respObj)
	if err != nil {
		fmt.Println("error:", err)
	}

	return getDataId(respObj.JobID, pzAddr)
}

// Constructs the ingest call for a GeoTIFF
func IngestTiff(filename, subFold, pzAddr, cmdName string) (string, error) {

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" : { "dataType": { "type": "raster" }, "metadata": { "name": "%s", "description": "raster uploaded by pzsvc-exec for %s.", "classType": { "classification": "unclassified" } } } } }`, filename, cmdName)

	return ingestMultipart(jsonStr, subFold, pzAddr, filename)
}

// Constructs the ingest call for a GeoJson
func IngestGeoJson(filename, subFold, pzAddr, cmdName string) (string, error) {

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" : { "dataType": { "type": "geojson" }, "metadata": { "name": "%s", "description": "GeoJson uploaded by pzsvc-exec for %s.", "classType": { "classification": "unclassified" } } } } }`, filename, cmdName)

	return ingestMultipart(jsonStr, subFold, pzAddr, filename)
}

// Constructs the ingest call for standard text.
func IngestTxt(filename, subFold, pzAddr, cmdName string) (string, error) {
	
	textblock, err := ioutil.ReadFile(locString(subFold, filename))
	if err != nil {
		return "", err
	}

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" :{ "dataType": { "type": "text", "mimeType": "application/text", "content": "%s" }, "metadata": { "name": "%s", "description": "text output from pzsvc-exec for %s.", "classType": { "classification": "unclassified" } } } } }`, strconv.QuoteToASCII(string(textblock)), filename, cmdName)

	return ingestMultipart(jsonStr, "", pzAddr, "")
}