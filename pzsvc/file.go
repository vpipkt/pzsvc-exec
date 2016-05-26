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
	}
	return fmt.Sprintf(`./%s/%s`, subFold, fname)	
}

func submitGet(payload, authKey string) (*http.Response, error) {
	fileReq, err := http.NewRequest("GET", payload, nil)
	if err != nil {
		return nil, err
	}

	fileReq.Header.Add("Authorization", authKey)

	client := &http.Client{}
	return client.Do(fileReq)
}

// submitMultipart sends a multi-part POST call, including an optional uploaded file,
// and returns the response.  Primarily intended to support Ingest calls.
func submitMultipart(bodyStr, address, filename, authKey string, fileData []byte) (*http.Response, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err := writer.WriteField("body", bodyStr)
	if err != nil {
		return nil, err
	}

	if fileData != nil {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return nil, err
		}
		if (part == nil) {
			return nil, fmt.Errorf("Failure in Form File Creation.")
		}

		_, err = io.Copy(part, bytes.NewReader(fileData))
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
	fileReq.Header.Add("Authorization", authKey)

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}
	return resp, err
}

// Download retrieves a file from Pz using the file access API
func Download(dataID, subFold, pzAddr, authKey string) (string, error) {

	resp, err := submitGet(pzAddr + "/file/" + dataID, authKey)
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
		return "", fmt.Errorf(`File for DataID %s unnamed.  Probable ingest error.`, dataID)
	}
	
	out, err := os.Create(locString(subFold, filename))
	if err != nil {
		return "", err
	}

	defer out.Close()
	io.Copy(out, resp.Body)

	return filename, nil
}

// getDataID will repeatedly poll the job status on the given job Id
// until job completion, then acquires and returns the resulting DataId.
func getDataID(jobID, pzAddr, authKey string) (string, error) {

	time.Sleep(200 * time.Millisecond)
	for i := 0; i < 100; i++ {
		resp, err := submitGet(pzAddr + "/job/" + jobID, authKey)
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

		//fmt.Println(respBuf.String())

		var respObj JobResp
		err = json.Unmarshal(respBuf.Bytes(), &respObj)
		if err != nil {
			return "", err
		}

		if respObj.Status == "Submitted" || respObj.Status == "Running" || respObj.Status == "Pending" || respObj.Message == "Job Not Found" {
			time.Sleep(200 * time.Millisecond)
		} else {

			if respObj.Status == "Success" {
				return respObj.Result.DataID, nil
			}
			if respObj.Status == "Error" || respObj.Status == "Fail" {
				return "", errors.New("Error when acquiring DataId.  Response json: " + respBuf.String())
			}
			return "", errors.New("Unknown status when acquiring DataId.  Response json: " + respBuf.String())
		}
	}

	return "", errors.New("Never completed.")
}

// ingestMultipart handles the Pz Ingest process.  It uploads the file to Pz and
// returns the resulting DataId.

func ingestMultipart(bodyStr, pzAddr, authKey, filename string, fileData []byte) (string, error) {

	resp, err := submitMultipart(bodyStr, (pzAddr + "/job"), filename, authKey, fileData)
	if err != nil {
		return "", err
	}

	respBuf := &bytes.Buffer{}
	_, err = respBuf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}
	//fmt.Println(respBuf.String())

	var respObj JobResp
	err = json.Unmarshal(respBuf.Bytes(), &respObj)
	if err != nil {
		fmt.Println("error:", err)
	}

	return getDataID(respObj.JobID, pzAddr, authKey)
}

// genIngestJson constructs and returns the JSON for a Pz ingest call.
func genIngestJSON(	fName, fType, mimeType, cmdName, content, version string,
					props map[string]string) (string, error) {
	
	desc := fmt.Sprintf("%s uploaded by %s.", fType, cmdName)
	rMeta := ResMeta{fName, desc, ClassType{"UNCLASSIFIED"}, "POST", version, make(map[string]string)} //TODO: implement classification
	
	for key, val := range props {
		rMeta.Metadata[key] = val
	}
	
	dType := DataType{content, fType, mimeType, nil}
	dRes := DataResource{dType, rMeta, "", nil}
	jType := IngJobType{"ingest", true, dRes}
	iCall := IngestCall{"defaultUser", jType}	
	
	bbuff, err := json.Marshal(iCall)
	
	return string(bbuff), err
}

// IngestTiffReader generates and sends an ingest request to Pz, uploading the contents of the
// given reader as a TIFF file.
func IngestTiffReader (	filename, pzAddr, sourceName, version, authKey string,
						inTiff io.Reader, props map[string]string) (string, error) {					
	jStr, err := genIngestJSON(filename, "raster", "image/tiff", sourceName, "", version, props)
	if err != nil {
		return "", err
	}
				
	bSlice, err := ioutil.ReadAll(inTiff)
	if err != nil {
		return "", err
	}
					
	return ingestMultipart(jStr, pzAddr, authKey, filename, bSlice)
}

// IngestLocalFile finds and loads the local file to be read (if any) then passes the result
// on to ingestMultipart.
func ingestLocalFile(bodyStr, subFold, pzAddr, filename, authKey string) (string, error) {
	var fileData []byte
	var err error

	if 	filename != "" {
		fileData, err = ioutil.ReadFile(locString(subFold, filename))
		if err != nil {
			return "", err
		}
	}
	return ingestMultipart(bodyStr, pzAddr, authKey, filename, fileData)
}

// IngestLocalTiff constructs and executes the ingest call for a local GeoTIFF, returning the DataId
func IngestLocalTiff(filename, subFold, pzAddr, cmdName, version, authKey string,
					 props map[string]string) (string, error) {
						 
	jStr, err := genIngestJSON(filename, "raster", "image/tiff", cmdName, "", version, props)
	if err != nil {
		return "", err
	}
	return ingestLocalFile(jStr, subFold, pzAddr, filename, authKey)
}

// IngestLocalGeoJSON constructs and executes the ingest call for a local GeoJson, returning the DataId
func IngestLocalGeoJSON(filename, subFold, pzAddr, cmdName, version, authKey string,
						props map[string]string) (string, error) {

	jStr, err := genIngestJSON(filename, "geojson", "application/vnd.geo+json", cmdName, "", version, props)
	if err != nil {
		return "", err	
	}
	return ingestLocalFile(jStr, subFold, pzAddr, filename, authKey)
}

// IngestLocalTxt constructs and executes the ingest call for a local text file, returning the DataId
func IngestLocalTxt(filename, subFold, pzAddr, cmdName, version, authKey string,
					props map[string]string) (string, error) {
	
	textblock, err := ioutil.ReadFile(locString(subFold, filename))
	if err != nil {
		return "", err
	}
	
	jStr, err := genIngestJSON(filename, "text", "application/text", cmdName, strconv.QuoteToASCII(string(textblock)), version, props)
	if err != nil {
		return "", nil
	}
	return ingestLocalFile(jStr, "", pzAddr, "", authKey)
}

// GetFileMeta retrieves the metadata for a given dataID in the S3 bucket
func GetFileMeta(dataID, pzAddr, authKey string) (*DataResource, error) {

	call := fmt.Sprintf(`%s/data/%s`, pzAddr, dataID)
	
	resp, err := submitGet(call, authKey)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	
	respBuf := &bytes.Buffer{}
		
	_, err = respBuf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var respObj DataResource
	err = json.Unmarshal(respBuf.Bytes(), &respObj)
	if err != nil {
		return nil, err
	}	

	return &respObj, nil
}

// UpdateFileMeta updates the metadata for a given dataID in the S3 bucket
func UpdateFileMeta(dataID, pzAddr, authKey string, newMeta map[string]string ) error {
	
	var meta struct { Metadata map[string]string `json:"metadata"` }
	meta.Metadata = newMeta
	jbuff, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	
	_, err = SubmitSinglePart("POST", string(jbuff), fmt.Sprintf(`%s/data/%s`, pzAddr, dataID), authKey)
	return err
}




