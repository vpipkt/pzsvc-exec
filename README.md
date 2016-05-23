# pzsvc-exec

"pzsvc-exec" is designed to serve command-line programs to Piazza, based on the contents of a config file.

## Contents

- How and Why: Useful background information and overview.
- Installing and Running: You need this if you want to run/administrate a pzsvc-exec instance
- Config File Format: You need this if you want to have any control over the instance you're running.  May be useful for understanding how pzsvc-exec works.
- Using: You need this if you want to access an existing instance of pzsvc-exec with any control at all.
-- Example http calls: examples that put the "using" section into practice.

## What and Why

Pzsvc-exec is at its most basic level something that publishes the exec() call as a service to Piazza (with lots of useful bells and whistles).

When it is launched, it is given a config file, from which it derives all persistent information.  If the config file allows, it will start by automatically registering itself as a service to a specified Piazza instance.  Regardless, it will then begin to serve.

When a request comes in, it has up to three parts - a set of files to download, a command string to execute, and a set of files to upload.  It will generate a temporary folder, download the files into the folder, execute its command in the folder, upload the files from the folder, reply to the service request, and then delete the folder.  The command it attempts to execute is the `CliCmd` parameter from the config file, with the `cmd` from the service request appended on.  The reply to the service request will take the form of a Json string, and contains a list of the files downloaded, a list of the files uploaded, and the stdout return of command executed.

The idea of this meta-service is to simplify the task of launch and maintenance on Pz services.  If you have execute access to an algorithm or similar program, its meaningful inputs consist of files and a command-line call, and its meaningful outputs consist of files, stderr, and stdout, you can provide it as a Piazza service.  All you should have to do is fill out the config file properly (and have a Piazza instance to connect to) and pzsvc-exec will take care of the rest.

As a secondary benefit, pzsvc-exec will be kept current with the existing Piazza interface, meaning that it can serve as living example code for those of you who find its limitations overly constraining.  For those of you writing in Go, it can even be used as a library.

## Installing and Running

Make sure you have Go installed on you machine, and an appropriate GOPATH (environment variable) set.

Use `go get` to install the latest version of both the CLI and the library.
	`$ go get -v github.com/venicegeo/pzsvc-exec/...`

To install:
	`$ go install github.com/venicegeo/pzsvc-exec/...`

Alternate install:
	navigate to `GOPATH/src/github.com/venicegeo/pzsvc-exec`
	then call `$ go install .`

To Run:
	`GOPATH/bin/pzsvc-exec <configfile.txt>`, where <configfile.txt> represents the path to an appropriately formatted config file, indicating what command line function to use, and where to find Piazza for registration.  Additionally, make sure that whatever application you wish to access is in your PATH.  Pzsvc-exec uses the creation of temporary folders as a way to keep from overwhelming the hard drive.  Just placing your application in the `GOPATH/bin` directory will not work.

## Config File Format

The example config file in this directory includes all pertinent potential entries, and should be used as an example.  Additional entries are meaningless but nonharmful, as long as standard json format is maintained.  No entries are strictly speaking mandatory, though leaving them out will often eliminate various kinds of functionality.

CliCmd: The initial parameters of the exec call.  For security reasons, you are strongly encouraged to define this entry as something other than the empty string, thus limiting your service to a single application.  If you do not, you are essentially offering open command-line access to anyone capable of calling your service.  

PzAddr: For use with a Piazza instance.  This is the base http or https address of the chosen Piazza instance, and is necessary for the file upload, file download, and autoregistration functionalities.

SvcName: Primarily for purposes of Piazza registration.  This is the name by which the service will identify itself.  Maintaining SvcName uniqueness among your services is important, as it will be used to determine on execution whether a service is being launched for the first time, or whether it is a continuation of a previous service.  Maintaining SvcName uniqueness in general is not as critical, as identity of launching user will also be used as a component.  It is necessary for the autoregistration functionality.

SvcType: Primarily for purposes of Piazza registration.  This is an indicator of type of service, and is intended to simplify searches by service consumers.  For example, the Beachfront GUI might search for all services with a SvcType of "Beachfront".  It is not necessary for any functionality.

Port: The port this service serves on.  If not defined, will default to 8080.

Description: A simple text description of your service.  Used in registration, and also available through the "/description" endpoint.

Attributes: A block of freeform Json for you to set additional descriptive attributes.  Used in registration, and also available through the "/attributes" endpoint.  Primarily intended to aid communication between services and service consumers with respect to the details of a service.

## Using

Intended use is through the Piazza service, though it can also be used as a standalone service.  Currently accepts both GET and POST calls, with identical parameters.  Actually using the service requires that you call the "execute" endpoint of whatever base the service is called on (example: "http://localhost:8080/execute", if running locally on port 8080).  Beyond that, valid and accepted parameters (query parameters for Get, form parameters for POST) are as follows:

cmd: The second part of the exec call (following CliCmd).  Additional commands after the first are not supported.  Allows the user some control over the process by influencing input params.

inFiles: a comma separated list (no spaces) of Piazza dataIds.  the files corresponding to those dataIds will be downloaded into the same directory as the program being served prior to execution, allowing for remote file inputs to the process.

outTiffs: a comma separated list (no spaces) of filenames.  Those filenames should correspond to .tif files that will be in the same directory as the program being served after the program has finished execution.  They will be uploaded to the chosen Piazza instance, and the resulting dataIds will be returned with the service results, allowing for file-based returns of images.  Must be in proper TIFF format

outTxts: as with outTiffs, but text files.  Actual extension doesn't matter as long as the result can be meaningfully interpreted as raw text.  Not suitable for large files.

outGeoJson: as with outTiffs and outTxts, but with GeoJson files.  Must be in proper GeoJson format.

### Example http calls

`http://<address:port>/execute`
- No uploads, no downloads, direct access rather than through piazza, just running whatever command CliCmd has to offer

`http://<address:port>/execute?cmd=ls -l`
- Assumes that CliCmd is blank.  Makes no uploads and no downloads.  Returns the contents of the local directory.

`http://<address:port>/execute?cmd=ls -l;inFiles=a10e6611-b996-4491-8988-ad0624ae8b6a,f71159c8-836d-4fcc-b8d9-4e9fb032e7a6,10fa1980-f0b5-4138-9f64-64b6fe7f73b2`
- As above, but downloads 3 files prior to running `ls -l`.  Results should include the identity of the downloaded files in addition to the standard output for ls (a list of the files downloaded) and the Piazza wrapper.

`http://<address:port>/execute?inFiles=a10e6611-b996-4491-8988-ad0624ae8b6a,f71159c8-836d-4fcc-b8d9-4e9fb032e7a6,10fa1980-f0b5-4138-9f64-64b6fe7f73b2;outTiffs=garden_rgb.tif,garden_b6.tif,garden_b3.tif;outTxts=testSend.txt;outGeoJson=tester.json`

Downloads 3 files, then runs whatever command CmdCli has to offer without addition, then attempts to upload 3 tiffs, a txt file, and a GeoJson.  Results should include a list of files downloaded, a list of files uploaded, and whatever the results of the CliCmd call is.