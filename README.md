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

cmd: the second part of the exec call, potentially allowing some degree of control.

inFiles: a comma separated list (no spaces) of Piazza dataIds.  the files corresponding to those dataIds will be downloaded into the same directory as the program being served prior to execution, allowing for remote file inputs to the process.

outTiffs: a comma separated list (no spaces) of filenames.  Those filenames should correspond to .tif files that will be in the same directory as the program being served after the program has finished execution.  They will be uploaded to the chosen Piazza instance, and the resulting dataIds will be returned with the service results, allowing for file-based returns of images.

outTxts: as with outTiffs, but text files.  Actual extension doesn't matter as long as the result can be meaningfully interpreted as raw text.

pz: if this parameter is defined as anything other than the empty string, the service will return its result in a format designed for Piazza consumption.  This is intended to support being called through Piazza as a job.  If the service is beign called directly, this should be left blank.