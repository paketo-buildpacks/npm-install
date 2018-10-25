#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source  ./scripts/install_tools.sh

export CNB_BUILD_IMAGE=${CNB_BUILD_IMAGE:-cfbuildpacks/cflinuxfs3-cnb-experimental:build}

# TODO: change default to `cfbuildpacks/cflinuxfs3-cnb-experimental:run` when pack cli can use it
export CNB_RUN_IMAGE=${CNB_RUN_IMAGE:-packs/run}

# Always pull latest images
# Most helpful for local testing consistency with CI (which would already pull the latest)
docker pull $CNB_BUILD_IMAGE
docker pull $CNB_RUN_IMAGE

echo "Run Buildpack Acceptance Tests"
go test ./acceptance/... -v -run Acceptance
