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

# change package, package-lock versions to a random number to prevent test polution
# randomVersion=$(printf "0.%s.%s" $(( (RANDOM % 100) + 1)) $(( (RANDOM % 100) + 1)))

echo "Run Buildpack Acceptance Tests"
echo "Package.json and package-lock.json will use random version:"
echo $randomVersion
# sed "s/\"version\": \"[0-9].[0-9].[0-9]\"/\"version\": \"$randomVersion\"/" fixtures/simple_app_vendored/package.json
go test ./acceptance/... -v -run Acceptance

# git co package.json changes