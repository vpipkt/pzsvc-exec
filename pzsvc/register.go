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

// The "ResourceMetadata" section of the service data.  In general has little
// programmatic effect at the Pz level - promarily exists as a way for users
// and consuming services to learn about the registered service
type resMeta struct {
	Name string `json:"name"`					// The service name
	Description string `json:"description"`		// Text block describing the service 
	Method string `json:"method"` 				// Method used to call this service (like "POST").
	Metadata map[string]string `json:"metadata"`// arbitrary key/value pairs.
}

// The overall service data.  
type pzService struct {
	ServiceId string `json:"serviceId"`			// The unique ID used by Pz to track this service
	Url string `json:"url"`						// The URL that Pz uses to call this service
	ResourceMetadata resMeta `json:"resourceMetadata"`	// See above
}

// Searches Pz for a service matching the input information.  if it finds one,
// returns the service ID.  If it does not, returns an empty string.  Currently
// only semi-functional.  Read through and vet carefully before declaring functional.
func FindMySvc(svcName, pzAddr string) (string, error) {

	// Pz generic list wrapper, around a list of service objects.
	type svcWrapper struct {
		Type       string			`json:"type"`
		Data       []pzService		`json:"data"`
		Pagination map[string]int	`json:"pagination"`
	}

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

	var respObj svcWrapper
	err = json.Unmarshal(respBuf, &respObj)
	if err != nil {
		return "", err
	}

	for _, checkServ := range respObj.Data {
		if checkServ.ResourceMetadata.Name == svcName {
			return checkServ.ServiceId, nil
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
func ManageRegistration(svcName, svcDesc, svcUrl, pzAddr string, imgReq map[string]string) error {
	if len(imgReq) > 0 {
		newImgReq:= map[string]string{}
		for key, val := range imgReq {
			newImgReq["imgReq - " + key] = val
		}
		imgReq = newImgReq
	}
	
	fmt.Println("Finding")
	svcId, err := FindMySvc(svcName, pzAddr)
	if err != nil {
		return err
	}

	metaObj := resMeta{ svcName, svcDesc, "POST", imgReq }
	svcObj := pzService{ svcId, svcUrl, metaObj }
	svcJson, err := json.Marshal(svcObj)

fmt.Println("attempting to register/update: " + string(svcJson))

	if svcId == "" {
		fmt.Println("Registering")
		_, err = SubmitSinglePart("POST", string(svcJson), pzAddr+"/service")
	} else {
		fmt.Println("Updating")
		_, err = SubmitSinglePart("PUT", string(svcJson), pzAddr+"/service/"+svcId)
	}
	if err != nil {
		return err
	}

	return nil
}
