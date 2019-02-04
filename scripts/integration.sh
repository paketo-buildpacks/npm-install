#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source ./scripts/install_tools.sh

export CNB_BUILD_IMAGE=${CNB_BUILD_IMAGE:-cfbuildpacks/cflinuxfs3-cnb-experimental:build}
export CNB_RUN_IMAGE=${CNB_RUN_IMAGE:-cfbuildpacks/cflinuxfs3-cnb-experimental:run}

# Always pull latest images
# Most helpful for local testing consistency with CI (which would already pull the latest)
docker pull $CNB_BUILD_IMAGE
docker pull $CNB_RUN_IMAGE

echo "Run Buildpack Runtime Integration Tests"
go test ./integration/... -v -run Integration
