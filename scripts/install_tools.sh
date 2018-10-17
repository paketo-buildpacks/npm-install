#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

if [ ! -d .bin ]; then
  mkdir .bin
fi

export GOBIN=$PWD/.bin
export PATH=$GOBIN:$PATH

if [[ ! -f $GOBIN/ginkgo ]]; then
    go install github.com/onsi/ginkgo/ginkgo
fi

if [[ ! -f $GOBIN/pack ]]; then
    go get github.com/buildpack/pack@master
    go install github.com/buildpack/pack/cmd/pack
fi