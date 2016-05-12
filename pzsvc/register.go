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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Searches Pz for a service matching the input information.  if it finds one,
// returns the service ID.  If it does not, returns an empty string.  Currently
// only semi-functional.  Read through and vet carefully before declaring functional.
func FindMySvc(svcName, pzAddr string) (string, error) {

	fmt.Println(pzAddr + "/service?per_page=1000&keyword=" + url.QueryEscape(svcName))

	resp, err := http.Get(pzAddr + "/service?per_page=1000&keyword=" + url.QueryEscape(svcName))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	respBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var respObj SvcWrapper
	err = json.Unmarshal(respBuf, &respObj)
	if err != nil {
		return "", err
	}

	for _, checkServ := range respObj.Data {
		if checkServ.ResMeta.Name == svcName {
			return checkServ.ServiceID, nil
		}
	}

	return "", nil
}

// Sends a single-part POST or a PUT call to Pz and returns the response.
// May work on some other methods, but not yet tested for them.  Includes
// the necessary headers.
func SubmitSinglePart(method, bodyStr, address string) (*http.Response, error) {

	fileReq, err := http.NewRequest(method, address, bytes.NewBuffer([]byte(bodyStr)))
	if err != nil {
		return nil, err
	}

	// The following header block is necessary for proper Pz function (as of 4 May 2016).
	fileReq.Header.Add("Content-Type", "application/json")
	fileReq.Header.Add("size", "30")
	fileReq.Header.Add("from", "0")
	fileReq.Header.Add("key", "stamp")
	fileReq.Header.Add("order", "true")

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}

	return resp, err
}

// Handles Pz registration for a service.  It checks the current service list to see if
// it has been registered already.  If it has not, it performs initial registration.
// if it has not, it re-registers.  Best practice is to do this every time your service
// starts up.  For those of you code-reading, the filter is still somewhat rudimentary.
// It will improve as better tools become available.
func ManageRegistration(svcName, svcDesc, svcURL, pzAddr string, imgReq map[string]string) error {
	if len(imgReq) > 0 {
		newImgReq:= map[string]string{}
		for key, val := range imgReq {
			newImgReq["imgReq - " + key] = val
		}
		imgReq = newImgReq
	}
//TODO: imgReq may not be generic enough.  Look into/reconsider.
	
	fmt.Println("Finding")
	svcID, err := FindMySvc(svcName, pzAddr)
	if err != nil {
		return err
	}
	svcVers := "" //TODO: Update this to current version numebr

	svcClass := ClassType{"UNCLASSIFIED"} // TODO: this will have to be updated at some point.
	metaObj := ResMeta{ svcName, svcDesc, svcClass, "POST", svcVers, imgReq }
	svcObj := Service{ svcID, svcURL, metaObj }
	svcJSON, err := json.Marshal(svcObj)

fmt.Println("attempting to register/update: " + string(svcJSON))

	if svcID == "" {
		fmt.Println("Registering")
		_, err = SubmitSinglePart("POST", string(svcJSON), pzAddr+"/service")
	} else {
		fmt.Println("Updating")
		_, err = SubmitSinglePart("PUT", string(svcJSON), pzAddr+"/service/"+svcID)
	}
	if err != nil {
		return err
	}

	return nil
}
