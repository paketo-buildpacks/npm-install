#!/usr/bin/env bash
set -euo pipefail

PACK_VERSION="0.0.9"

install_pack() {
    OS=$(uname -s)

    if [[ $OS == "Darwin" ]]; then
        OS="macos"
    elif [[ $OS == "Linux" ]]; then
        OS="linux"
    else
        echo "Unsupported operating system"
        exit 1
    fi

    PACK_ARTIFACT=pack-$PACK_VERSION-$OS.tar.gz

    curl -O -L -s https://github.com/buildpack/pack/releases/download/v$PACK_VERSION/$PACK_ARTIFACT
    tar xzvf $PACK_ARTIFACT -C .bin
    rm $PACK_ARTIFACT
}

configure_pack() {
    pack add-stack org.cloudfoundry.stacks.cflinuxfs3 \
        --build-image cfbuildpacks/cflinuxfs3-cnb-experimental:build \
        --run-image cfbuildpacks/cflinuxfs3-cnb-experimental:run || echo "Ignoring add stack error"
}

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

mkdir -p .bin
export PATH=$(pwd)/.bin:$PATH

if [[ ! -f .bin/pack ]]; then
    install_pack
elif [[ $(.bin/pack version | cut -d ' ' -f 2) != "v$PACK_VERSION" ]]; then
    rm .bin/pack
    install_pack
fi
configure_pack
