#! /bin/bash -ex

pushd `dirname $0`/.. > /dev/null
root=$(pwd -P)
popd > /dev/null

export GOPATH=$root/gopath
mkdir -p $GOPATH

source $root/ci/vars.sh

go get -v github.com/venicegeo/$APP/...
#go install -v github.com/venicegeo/$APP/...
cp $GOPATH/src/github.com/venicegeo/$APP/examplecfg.txt $GOPATH/bin/
pushd $GOPATH/bin

tar -czf $APP.$EXT pzsvc-exec bf-dummycmd examplecfg.txt
