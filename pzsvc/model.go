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

// DataResult corresponds to an number of classes, all of which implement interface
// ResultType.  Of those classes, it currently handles model/job/result/type/DataResult.java
// and model/job/result/type/FileResult.java.  Other such classes do exist, and would
// require additional fields.  They may need ot be implemented in the future.
type DataResult struct {
	Type			string		`json:"type"`
	DataID			string		`json:"dataId"`
}

// JobProg corresponds to model/job/JobProgress.java.  It does not currently include all
// of the fields of its counterpart.
type JobProg struct { //name
	PercentComplete int			`json:"percentComplete"`
}

// SpatMeta corresponds to model/job/metadata/SpatialMetadata.java.
type SpatMeta struct { //name
	CoordRefSystem	string		`json:"coordinateReferenceSystem"`
	EpsgCode		string		`json:"epsgCode"`
	MinX			float64		`json:"minX"`
	MinY			float64		`json:"minY"`
	MinZ			float64		`json:"minZ"`
	MaxX			float64		`json:"maxX"`
	MaxY			float64		`json:"maxY"`
	MaxZ			float64		`json:"maxZ"`
}

// ClassType corresponds to model.security.SecurityClassification.
type ClassType struct {
	Classification	string		`json:"classification"`
}

// ResMeta corresponds to model/job/metadata/ResourceMetadata.java.
// Worth noting that Pz pays no attention to the contents of the
// Metadata map field except to act as a passthrough.
type ResMeta struct {
	Name			string		`json:"name"`
	Description		string		`json:"description"`
	ClassType		*ClassType	`json:"classType"`
	Method			string		`json:"method"`
	Version			string		`json:"version"`
	Metadata		map[string]string `json:"metadata"`
}

// S3Loc corresponds with model/data/location/S3FileStore.java
// It's the form of FileLocation that refers to things currently
// in the S3 Bucket.
type S3Loc struct {
	FileName		string		`json:"fileName"`
	DomainName		string		`json:"domainName"`
	BucketName		string		`json:"bucketName"`
	FileSize		float64		`json:"fileSize"`
}

// DataType corresponds to model/data/DataType.java.  It's an occasional
// misnomer.  In cases where the Content field is not empty, that contains
// the data that the type is referring to.  In cases where it is empty, the
// data is attached elsewhere.
type DataType struct { //name
	Content			string		`json:"content"`
	Type			string		`json:"type"`
	MimeType		string		`json:"mimeType"`
	Location		*S3Loc		`json:"location"`
}

// DataResource corresponds to model/data/DataResource.java  It is also
// used as the response type for Get data/{dataID} calls.
type DataResource struct {
	DataType		*DataType	`json:"dataType"`
	Metadata		*ResMeta	`json:"metadata"`
	DataID			string		`json:"dataId"`
	SpatMeta		*SpatMeta	`json:"spatialMetadata"`
}

// IngJobType corresponds to model/job/type/IngestJob.java
type IngJobType 	struct {
	Type			string		`json:"type"`
	Host			bool		`json:"host"`
	Data			*DataResource	`json:"data"`
}

/***********************/
/***  Request types  ***/
/***********************/

// ExecService corresponds to model/service/metadata/ExecuteServiceData.java
// It is used to call services through Pz
type ExecService struct {
	ServiceID		string				`json:"serviceId"`
	DataInputs		map[string]DataType	`json:"dataInputs"`
	DataOutput		DataType			`json:"dataOutput"`
}

// Service corresponds to model/service/metadata/Service.java
// Used as the payload in register service and update service jobs.
// Also used in the response to the List Service job.
type Service struct {
	ServiceID		string		`json:"serviceId"`
	URL				string		`json:"url"`
	ResMeta			ResMeta		`json:"resourceMetadata"`
}

// IngestCall corresponds to model/request/PiazzaJobRequest.java
type IngestCall struct { //name
	UserName		string		`json:"userName"`
	JobType			*IngJobType	`json:"jobType"`
}

/***********************/
/***  Response types  **/
/***********************/

// JobResult corresponds to model/job/result/type/JobResult.java
// It's the immediate result gotten back from asynch jobs (service calls and ingests, mostly)
type JobResult struct { //name
	Type			string		`json:"type"`
	JobID			string		`json:"jobId"`
}

// JobResp corresponds to model/job/Job.java
// It's the response object for a Check Status call against a JobID
type JobResp struct { //name
	Type			string		`json:"type"`
	JobID			string		`json:"jobId"`
	Result			*DataResult	`json:"result"`
	Status			string		`json:"status"`
	JobType			string		`json:"jobType"`
	SubmittedBy		string		`json:"submittedBy"`
	Progress		*JobProg	`json:"progress"`
	Message			string		`json:"message"`	//used for error responses
}

// SvcWrapper is the Pz generic list wrapper, around a list of service objects.
// It's the response object for a list/search services call
type SvcWrapper struct {
	Type       string			`json:"type"`
	Data       []Service		`json:"data"`
	Pagination map[string]int	`json:"pagination"`
}