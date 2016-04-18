#! /bin/bash -ex

pushd `dirname $0`/.. > /dev/null
root=$(pwd -P)
popd > /dev/null

export GOPATH=$root/gopath
mkdir -p $GOPATH
lsb_release -a || cat /proc/version || cat /etc/*-release
source $root/ci/vars.sh

go get -v github.com/venicegeo/pzsvc-exec/...
go install -v github.com/venicegeo/pzsvc-exec/bf-dummycmd
cp $GOPATH/src/github.com/venicegeo/pzsvc-exec/examplecfg.txt $GOPATH/bin/

tar -czfn $APP.$EXT -C $GOPATH/bin .
