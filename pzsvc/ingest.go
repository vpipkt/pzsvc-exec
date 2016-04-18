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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type JobResp struct {
	Type  string
	JobId string
}

type StatusJsonResult struct {
	Type    string
	DataId  string
	Message string
	Details string
}

type StatusJsonResp struct {
	Type   string
	JobId  string
	Result StatusJsonResult
	Status string
}

func submitMultipart(bodyStr, address, upload string) (*http.Response, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
fmt.Println("1")
	err := writer.WriteField("body", bodyStr)
	if err != nil {
		return nil, err
	}
fmt.Println("2")
	if upload != "" {
		file, err := os.Open(fmt.Sprintf(`./%s`, upload))
		if err != nil {
			return nil, err
		}
fmt.Println("3")
		defer file.Close()

		part, err := writer.CreateFormFile("file", upload)
		if err != nil {
			return nil, err
		}
fmt.Println("4")
		_, err = io.Copy(part, file)
		if err != nil {
			return nil, err
		}
	}
fmt.Println("5")
	err = writer.Close()
	if err != nil {
		return nil, err
	}
fmt.Println("6")
	fileReq, err := http.NewRequest("POST", address, body)
	if err != nil {
		return nil, err
	}
fmt.Println("7")
	fileReq.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}
fmt.Println("8")
	return resp, err
}

func Download(dataId, pzAddr string) (string, error) {

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "dataId": "%s"}`, dataId)

	resp, err := submitMultipart(jsonStr, (pzAddr + "/file"), "")
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	contDisp := resp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contDisp)
	filename := params["filename"]
	if filename == "" {
		filename = "dummy.txt"
	}
	filepath := fmt.Sprintf(`./%s`, filename)

	out, err := os.Create(filepath)
	if err != nil {
		return "", err
	}

	defer out.Close()
	io.Copy(out, resp.Body)

	return filename, nil
}

func getStatus(jobId, pzAddr string) (string, error) {

	time.Sleep(1000 * time.Millisecond)

	var respObj StatusJsonResp
	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "get", "jobId": "%s" } }`, jobId)

	lastErr := errors.New("Never completed.")

	for i := 0; i < 100; i++ {

		resp, err := submitMultipart(jsonStr, (pzAddr + "/job"), "")
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

		err = json.Unmarshal(respBuf.Bytes(), &respObj)
		if err != nil {
			return "", err
		}

		if respObj.Status == "Submitted" || respObj.Status == "Running" || respObj.Status == "Pending" {
			time.Sleep(200 * time.Millisecond)
		} else if respObj.Status == "Success" {
			lastErr = nil
			break
		} else if respObj.Status == "Error" || respObj.Status == "Fail" {
			return "", errors.New(respObj.Status + ": " + respObj.Result.Message + respObj.Result.Details)
		} else {
			return "", errors.New("Unknown status: " + respObj.Status)
		}
	}

	return respObj.Result.DataId, lastErr
}

func ingestMultipart(bodyStr, pzAddr, filename string) (string, error) {

	resp, err := submitMultipart(bodyStr, (pzAddr + "/job"), filename)
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

	dataId, err := getStatus(respObj.JobId, pzAddr)

	return dataId, err
}

func IngestTiff(filename, pzAddr, cmdName string) (string, error) {

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" : { "dataType": { "type": "raster" }, "metadata": { "name": "%s", "description": "raster uploaded by pzsvc-exec for %s.", "classType": { "classification": "unclassified" } } } } }`, filename, cmdName)

	return ingestMultipart(jsonStr, pzAddr, filename)
}

func IngestGeoJson(filename, pzAddr, cmdName string) (string, error) {

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" : { "dataType": { "type": "geojson" }, "metadata": { "name": "%s", "description": "GeoJson uploaded by pzsvc-exec for %s.", "classType": { "classification": "unclassified" } } } } }`, filename, cmdName)

	return ingestMultipart(jsonStr, pzAddr, filename)
}

func IngestTxt(filename, pzAddr, cmdName string) (string, error) {
	textblock, err := ioutil.ReadFile(fmt.Sprintf(`./%s`, filename))
	if err != nil {
		return "", err
	}

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" :{ "dataType": { "type": "text", "mimeType": "application/text", "content": "%s" }, "metadata": { "name": "%s", "description": "text output from pzsvc-exec for %s.", "classType": { "classification": "unclassified" } } } } }`, string(textblock), filename, cmdName)

	return ingestMultipart(jsonStr, pzAddr, "")
}

func RegisterSvc(svcName, svcType, pzAddr string) error {
	// two paths: either we ask for the entire registration structure in the config file json,
	// or we ask for specific details, and fill in the details ourselves.  My inclination is
	// to go with the "specific details" route.  It'll mean that the metaservice will have to
	// be updated whenever the Pz input formats change, but that's going to have to happen anyway.
	// - service name
	// - service description
	// -
	//
	// - remember to include the BF-indicating metadata
	return nil
}

func UpdateSvc(svcName, svcType, svcId, pzAddr string) error {
	// as RegisterSvc, but updating service data rather than registering fresh.
	return nil
}

// Searches Pz for a service matching the input information.  if it finds one,
// returns the service ID.  If it does not, returns an empty string.
// currently only semi-functional.  Read through and vet carefully before declaring functional.

func FindSvc(svcName, pzAddr string) (string, error) {

	req, err := http.NewRequest("GET", (pzAddr + "/service/me"), nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("keyword", svcName)
	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	resp, err := client.Do(req)
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

	return "", nil
}

func ManageRegistration(svcName, svcType, pzAddr string) error {
	svcId, err := FindSvc(svcName, pzAddr)
	if err != nil {
		return err
	}

	if svcId == "" {
		err = RegisterSvc(svcName, svcType, pzAddr)
		if err != nil {
			return err
		}
	} else {
		err = UpdateSvc(svcName, svcType, svcId, pzAddr)
		if err != nil {
			return err
		}
	}
	return nil
}
