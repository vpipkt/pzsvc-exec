# pzsvc-exec

"pzsvc-exec" is designed to serve command-line programs to Piazza, based on teh contents of a config file.

## Installing and running

Make sure you have Go installed on you machine, and an appropriate GOPATH (environment variable) set.

Use `go get` to install the latest version of both the CLI and the library.
	$ go get -v github.com/venicegeo/pzsvc-exec/...

To install
	$ go install github.com/venicegeo/pzsvc-exec/...
Alternate install:
	navigate to GOPATH/src/github.com/venicegeo/pzsvc-exec
	$ go install .

To Run
	GOPATH/bin/bf-service <configfile.txt>

in this case, <configfile.txt> represents the path to an appropriately formatted config file, indicating what 

## Using

Intended use is through the Piazza service