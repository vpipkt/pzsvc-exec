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

## Using

Intended use is through the Piazza service