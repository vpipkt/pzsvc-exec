# pzsvc-exec

"pzsvc-exec" is designed to serve command-line programs to Piazza, based on the contents of a config file.

## Installing and running

Make sure you have Go installed on you machine, and an appropriate GOPATH (environment variable) set.

Use `go get` to install the latest version of both the CLI and the library.
	`$ go get -v github.com/venicegeo/pzsvc-exec/...`

To install:
	`$ go install github.com/venicegeo/pzsvc-exec/...`

Alternate install:
	navigate to `GOPATH/src/github.com/venicegeo/pzsvc-exec`
	then call `$ go install .`

To Run:
	`GOPATH/bin/pzsvc-exec <configfile.txt>`, where <configfile.txt> represents the path to an appropriately formatted config file, indicating what command line function to use, and where to find Piazza for registration

## Config File Format

The example config file in this directory includes all pertinent potential entries, and should be used as an example.  Additional entries are meaningless but nonharmful, as long as standard json format is maintained.  No entries are strictly speaking mandatory, though leaving them out will often eliminate various kinds of functionality.

CliCmd: the initial parameters of the exec call.  For security reasons, you are strongly encouraged to put something here, thus limiting your service to a single application.  If you do not, you are essentially offering open command-line access to anyone capable of calling your service.  

PzJobAddr: For use with a Piazza instance.  This is the general job scheduler endpoint of the chosen Piazza instance, and is necessary for the file upload and download functionalities.

PzFileAddr: For use with a Piazza instance.  This is the file download endpoint of the chosen Piazza instance, and is necessary for the file download functionality.

## Using

Intended use is through the Piazza service, though it can also be used as a standalone service.  Currently accepts both GET and POST calls, with identical parameters.  Actually using the service requires that you call the "execute" endpoint of whatever base the service is called on (example: "http://localhost:8080/execute").  Beyond that, valid and accepted parameters (query parameters for Get, form parameters for POST) are as follows:

cmd: The second part of the exec call (following CliCmd).  Additional commands after the first are not supported.  Allows the user some control over the process by influencing input params.

inFiles: a comma separated list (no spaces) of Piazza dataIds.  the files corresponding to those dataIds will be downloaded into the same directory as the program being served prior to execution, allowing for remote file inputs to the process.

outTiffs: a comma separated list (no spaces) of filenames.  Those filenames should correspond to .tif files that will be in the same directory as the program being served after the program has finished execution.  They will be uploaded to the chosen Piazza instance, and the resulting dataIds will be returned with the service results, allowing for file-based returns of images.  Must be in proper TIFF format

outTxts: as with outTiffs, but text files.  Actual extension doesn't matter as long as the result can be meaningfully interpreted as raw text.  Not suitable for large files.

outGeoJson: as with outTiffs and outTxts, but with GeoJson files.  Must be in proper GeoJson fromat.

pz: if this parameter is defined as anything other than the empty string, the service will return its result in a format designed for Piazza consumption.  This is intended to support being called through Piazza as a job.  If the service is beign called directly, this should be left blank.

## Example http calls

`http://<address:port>/execute`
- No uploads, no downloads, direct access rather than through piazza, just running whatever command CliCmd has to offer

`http://localhost:8080/execute?cmd=ls;inFiles=a10e6611-b996-4491-8988-ad0624ae8b6a,f71159c8-836d-4fcc-b8d9-4e9fb032e7a6,10fa1980-f0b5-4138-9f64-64b6fe7f73b2;outTiffs=garden_rgb.tif,garden_b6.tif,garden_b3.tif;outTxts=testSend.txt;outGeoJson=tester.json;pz=true`
- Assumes that CliCmd is blank.  Attempts to download 3 files, followed by checking the contents of the local directory, followed by uploading 5 files (3 Tiffs, a GeoJson, and a text file), and expects the results to be consumed by Piazza before being made available to the user.  Results should include the DataIds of all uploaded files in addition to the standard output for ls and the Piazza wrapper.



outTiffs=garden_rgb.tif,garden_b6.tif,garden_b3.tif