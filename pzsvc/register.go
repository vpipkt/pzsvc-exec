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

// FindMySvc Searches Pz for a service matching the input information.  If it finds
// one, it returns the service ID.  If it does not, returns an empty string.  Currently
// only able to search on service name.  Will be much more viable as a long-term answer
// if/when it's able to search on both service name and submitting user.
func FindMySvc(svcName, pzAddr, authKey string) (string, error) {

	query := pzAddr + "/service?per_page=1000&keyword=" + url.QueryEscape(svcName)
	
	resp, err := submitGet(query, authKey)
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

// SubmitSinglePart sends a single-part POST or a PUT call to Pz and returns the
// response.  May work on some other methods, but not yet tested for them.  Includes
// the necessary headers.
func SubmitSinglePart(method, bodyStr, address, authKey string) (*http.Response, error) {

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
	fileReq.Header.Add("Authorization", authKey)

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}

	return resp, err
}

// ManageRegistration Handles Pz registration for a service.  It checks the current
// service list to see if it has been registered already.  If it has not, it performs
// initial registration.  If it has not, it re-registers.  Best practice is to do this
// every time your service starts up.  For those of you code-reading, the filter is
// still somewhat rudimentary.  It will improve as better tools become available.
func ManageRegistration(svcName, svcDesc, svcURL, pzAddr, svcVers, authKey string, attributes map[string]string) error {
	
	fmt.Println("Finding")
	svcID, err := FindMySvc(svcName, pzAddr, authKey)
	if err != nil {
		return err
	}
	
	svcClass := ClassType{"UNCLASSIFIED"} // TODO: this will have to be updated at some point.
	metaObj := ResMeta{ svcName, svcDesc, svcClass, "POST", svcVers, attributes }
	svcObj := Service{ svcID, svcURL, metaObj }
	svcJSON, err := json.Marshal(svcObj)

fmt.Println("attempting to register/update: " + string(svcJSON))

	if svcID == "" {
		fmt.Println("Registering")
		_, err = SubmitSinglePart("POST", string(svcJSON), pzAddr+"/service", authKey)
	} else {
		fmt.Println("Updating")
		_, err = SubmitSinglePart("PUT", string(svcJSON), pzAddr+"/service/"+svcID, authKey)
	}
	if err != nil {
		return err
	}

	return nil
}
