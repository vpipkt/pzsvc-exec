#! /bin/bash -ex

pushd `dirname $0`/.. > /dev/null
root=$(pwd -P)
popd > /dev/null

export GOPATH=$root/gopath
mkdir -p $GOPATH

source $root/ci/vars.sh

go get -v github.com/venicegeo/$APP/...
src=$GOPATH/bin/pzsvc-exec
mv $src $root/$APP.$EXT
