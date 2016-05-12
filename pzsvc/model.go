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

// Intent: the following types are designed to correspond with the types
// defined in piazza, in order to simplify the management of JSON - both
// in interpreting the JSON that comes out of Piazza and in producing
// JSON for calls to Piazza.

// Commented links are to the object in the piazza code that corresponds with each
// type.  Links starting with "model" are at
// https://github.com/venicegeo/pz-jobcommon/blob/master/src/main/java/model/...

// The following objects are not necessarily complete.  Some fields that exist
// in the original java are ignored here because either we have no use for them
// or they are outright deprecated.  Similarly, there are at times differences
// between the outward piazza interfaces and the class files.  The class files
// are useful as a reference, but not fully dependable.


/***************************/
/***  Subordinate types  ***/
/***************************/

// multiple possibilities, all of which are of interface ResultType.  Currently implementing
// model/job/result/type/DataResult.java and model/job/result/type/FileResult.java.  Others
// exist, and would require additional fields - they may need ot be implemented int he future.
type DataResult struct { //name
	Type   string	`json:"type"`
	DataID string	`json:"dataId"`
}

// model/job/JobProgress.java
type JobProg struct { //name
	PercentComplete int `json:"percentComplete"`
}

// model/job/metadata/SpatialMetadata.java
type SpatMeta struct { //name
	CoordRefSystem	string	`json:"coordinateReferenceSystem"`
	EpsgCode		string	`json:"epsgCode"`
	MinX			float64	`json:"minX"`
	MinY			float64	`json:"minY"`
	MinZ			float64	`json:"minZ"`
	MaxX			float64	`json:"maxX"`
	MaxY			float64	`json:"maxY"`
	MaxZ			float64	`json:"maxZ"`
}

// model.security.SecurityClassification
type ClassType struct {
	Classification	string	`json:"classification"`
}

// model/job/metadata/ResourceMetadata.java
type ResMeta struct {
	Name			string			`json:"name"`
	Description		string			`json:"description"`
	ClassType		ClassType		`json:"classType"`
	Method			string			`json:"method"`
	Version			string			`json:"version"`
	Metadata		map[string]string `json:"metadata"` // Pz
}

// model/data/DataType.java
type DataType struct { //name
	Content			string			`json:"content"`
	Type			string			`json:"type"`
	MimeType		string			`json:"mimeType"`
}

// model/data/DataResource.java
type DataResource struct {
	DataType		DataType	`json:"dataType"`
	Metadata		ResMeta		`json:"metadata"`
	DataID			string		`json:"dataId"`
	SpatMeta		SpatMeta	`json:"spatialMetadata"`
}

// model/job/type/IngestJob.java
type IngJobType struct {
	Type		string			`json:"type"`
	Host		bool			`json:"host"`
	Data		DataResource	`json:"data"`
}

/***********************/
/***  Request types  ***/
/***********************/

// model/service/metadata/ExecuteServiceData.java
// used to call services through Pz
type ExecService struct {
	ServiceID		string				`json:"serviceId"`
	DataInputs		map[string]DataType	`json:"dataInputs"`
	DataOutput		DataType			`json:"dataOutput"`
}

// model/service/metadata/Service.java
// The overall service data.
// Also used as the payload in register service and update service jobs.
type Service struct {
	ServiceID string `json:"serviceId"`			// The unique ID used by Pz to track this service
	URL string `json:"url"`						// The URL that Pz uses to call this service
	ResMeta ResMeta `json:"resourceMetadata"`	// See above
}

// model/request/PiazzaJobRequest.java
type IngestCall struct { //name
	UserName		string		`json:"userName"`
	JobType			IngJobType	`json:"jobType"`
}

/***********************/
/***  Response types  **/
/***********************/

// model/job/result/type/JobResult.java
// the immediate result gotten back from asynch jobs (service calls and ingests)
type JobResult struct { //name
	Type	string	`json:"type"`
	JobID	string	`json:"jobId"`
}

// model/job/Job.java
// the response object for a Check Status call against a JobID
type JobResp struct { //name
	Type		string		`json:"type"`
	JobID		string		`json:"jobId"`
	Result		DataResult	`json:"result"`
	Status		string		`json:"status"`
	JobType		string		`json:"jobType"`
	SubmittedBy	string		`json:"submittedBy"`
	Progress	JobProg		`json:"progress"`
	Message		string		`json:"message"`	//important for error responses
}

// Pz generic list wrapper, around a list of service objects.
// the response object for a list/search services call
type SvcWrapper struct {
	Type       string			`json:"type"`
	Data       []Service		`json:"data"`
	Pagination map[string]int	`json:"pagination"`
}
