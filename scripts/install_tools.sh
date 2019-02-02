#!/usr/bin/env bash
set -euo pipefail

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

    if [[ $OS == "macos" ]]; then
        ARTIFACT_URL=$(curl -s https://api.github.com/repos/buildpack/pack/releases/latest |   jq --raw-output '.assets[1] | .browser_download_url')
    else
        ARTIFACT_URL=$(curl -s https://api.github.com/repos/buildpack/pack/releases/latest |   jq --raw-output '.assets[0] | .browser_download_url')
    fi
 
    PACK_ARTIFACT=$(echo $ARTIFACT_URL | sed "s/.*\///")
    PACK_VERSION=v$(echo $PACK_ARTIFACT | sed 's/pack-//' | sed 's/-.*//')

    if [[ ! -f .bin/pack ]]; then
        echo "Installing Pack"
    elif [[ "$(.bin/pack version | cut -d ' ' -f 1)" != *$PACK_VERSION* ]]; then
        rm .bin/pack
        echo "Updating Pack"
    else
        echo "The latest version of pack is already installed"
        return 0
    fi

    wget $ARTIFACT_URL
    tar xzvf $PACK_ARTIFACT -C .bin
    rm $PACK_ARTIFACT
}

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

mkdir -p .bin
export PATH=$(pwd)/.bin:$PATH

install_pack
