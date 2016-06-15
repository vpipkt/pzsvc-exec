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
// require additional fields.  They may need to be implemented in the future.
type DataResult struct {
	Type			string		`json:"type,omitempty"`
	DataID			string		`json:"dataId,omitempty"`
	Details			string		`json:"details,omitempty"`
}

// JobProg corresponds to model/job/JobProgress.java.  It does not currently include all
// of the fields of its counterpart.
type JobProg struct { //name
	PercentComplete int			`json:"percentComplete,omitempty"`
}

// SpatMeta corresponds to model/job/metadata/SpatialMetadata.java.
type SpatMeta struct { //name
	CoordRefSystem	string		`json:"coordinateReferenceSystem,omitempty"`
	EpsgCode		int			`json:"epsgCode,omitempty"`
	MinX			float64		`json:"minX,omitempty"`
	MinY			float64		`json:"minY,omitempty"`
	MinZ			float64		`json:"minZ,omitempty"`
	MaxX			float64		`json:"maxX,omitempty"`
	MaxY			float64		`json:"maxY,omitempty"`
	MaxZ			float64		`json:"maxZ,omitempty"`
	NumFeatures		int			`json:"numFeatures,omitempty"`
}

// ClassType corresponds to model.security.SecurityClassification.
type ClassType struct {
	Classification	string		`json:"classification,omitempty"`
}

// ResMeta corresponds to model/job/metadata/ResourceMetadata.java.
// Worth noting that Pz pays no attention to the contents of the
// Metadata map field except to act as a passthrough.
type ResMeta struct {
	Name			string		`json:"name,omitempty"`
	Description		string		`json:"description,omitempty"`
	ClassType		ClassType	`json:"classType,omitempty"`
	Version			string		`json:"version,omitempty"`
	Metadata		map[string]string `json:"metadata,omitempty"`
}

// S3Loc corresponds with model/data/location/S3FileStore.java
// It's the form of FileLocation that refers to things currently
// in the S3 Bucket.
type S3Loc struct {
	FileName		string		`json:"fileName,omitempty"`
	DomainName		string		`json:"domainName,omitempty"`
	BucketName		string		`json:"bucketName,omitempty"`
	FileSize		float64		`json:"fileSize,omitempty"`
}

// DataType corresponds to model/data/DataType.java.  It's an occasional
// misnomer.  In cases where the Content field is not empty, that contains
// the data that the type is referring to.  In cases where it is empty, the
// data is attached elsewhere.  Note that S3Loc is a struct pointer rather
// than a struct.  If you want to unmarshal JSON into it, you'll need to put
// an empty S3Loc object there first.
type DataType struct { //name
	Content			string		`json:"content,omitempty"`
	Type			string		`json:"type,omitempty"`
	MimeType		string		`json:"mimeType,omitempty"`
	Location		*S3Loc		`json:"location,omitempty"`
}

// DataResource corresponds to model/data/DataResource.java  It is also
// used as the response type for Get data/{dataID} calls.  Note that SpatMeta
// is a pointer rather than a struct, and has the same caveat as DataType.S3Loc
type DataResource struct {
	DataType		DataType	`json:"dataType,omitempty"`
	Metadata		ResMeta		`json:"metadata,omitempty"`
	DataID			string		`json:"dataId,omitempty"`
	SpatMeta		*SpatMeta	`json:"spatialMetadata,omitempty"`
}

// IngJobType corresponds to model/job/type/IngestJob.java
type IngJobType 	struct {
	Type			string		`json:"type,omitempty"`
	Host			bool		`json:"host,omitempty"`
	Data			DataResource	`json:"data,omitempty"`
}

/***********************/
/***  Request types  ***/
/***********************/

// ExecService corresponds to model/service/metadata/ExecuteServiceData.java
// It is used to call services through Pz
type ExecService struct {
	ServiceID		string				`json:"serviceId,omitempty"`
	DataInputs		map[string]DataType	`json:"dataInputs,omitempty"`
	DataOutput		DataType			`json:"dataOutput,omitempty"`
}

// Service corresponds to model/service/metadata/Service.java
// Used as the payload in register service and update service jobs.
// Also used in the response to the List Service job.
type Service struct {
	ServiceID		string		`json:"serviceId,omitempty"`
	URL			string		`json:"url,omitempty"`
	ContractUrl		string 		`json:"contractUrl,omitempty"`
	RestMethod		string 		`json:"method,omitempty"`
	ResMeta			ResMeta		`json:"resourceMetadata,omitempty"`
}

// IngestCall corresponds to model/request/PiazzaJobRequest.java
type IngestCall struct { //name
	UserName		string		`json:"userName,omitempty"`
	JobType			IngJobType	`json:"jobType,omitempty"`
}

/***********************/
/***  Response types  **/
/***********************/

// JobResult corresponds to model/job/result/type/JobResult.java
// It's the immediate result gotten back from asynch jobs (service calls and ingests, mostly)
type JobResult struct { //name
	Type			string		`json:"type,omitempty"`
	JobID			string		`json:"jobId,omitempty"`
}

// JobResp corresponds to model/job/Job.java
// It's the response object for a Check Status call against a JobID
type JobResp struct { //name
	Type			string		`json:"type,omitempty"`
	JobID			string		`json:"jobId,omitempty"`
	Result			DataResult	`json:"result,omitempty"`
	Status			string		`json:"status,omitempty"`
	JobType			string		`json:"jobType,omitempty"`
	SubmittedBy		string		`json:"submittedBy,omitempty"`
	Progress		JobProg		`json:"progress,omitempty"`
	Message			string		`json:"message,omitempty"`	//used for error responses
}

// SvcWrapper is the Pz generic list wrapper, around a list of service objects.
// It's the response object for a list/search services call
type SvcWrapper struct {
	Type       string			`json:"type,omitempty"`
	Data       []Service		`json:"data,omitempty"`
	Pagination map[string]int	`json:"pagination,omitempty"`
}