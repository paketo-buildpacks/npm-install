#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

if [ ! -d .bin ]; then
  mkdir .bin
fi

export GOBIN=$PWD/.bin
export PATH=$GOBIN:$PATH

if [[ ! -f $GOBIN/pack ]]; then
    go get github.com/buildpack/pack@a0f5edb5d97d9ac20c15386e64d7c75168758736
    go install github.com/buildpack/pack/cmd/pack
fi
