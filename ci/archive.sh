#! /bin/bash -ex

pushd `dirname $0`/.. > /dev/null
root=$(pwd -P)
popd > /dev/null

export GOPATH=$root/gopath
mkdir -p $GOPATH

source $root/ci/vars.sh

go get -v github.com/venicegeo/$APP/...
#go install -v github.com/venicegeo/$APP/...
cp $GOPATH/src/thub.com/venicegeo/$APP/examplecfg.txt $GOPATH/bin/

tar -czf $APP.$EXT -C $root $GOPATH/bin
