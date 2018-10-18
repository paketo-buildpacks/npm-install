#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
echo $PWD
source ./scripts/install_tools.sh

echo "Run Buildpack Runtime Unit Tests"
go test ./... -v -run Unit


