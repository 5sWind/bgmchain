#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
bgmdir="$workspace/src/github.com/5sWind"
if [ ! -L "$bgmdir/bgmchain" ]; then
    mkdir -p "$bgmdir"
    cd "$bgmdir"
    ln -s ../../../../../. bgmchain
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$bgmdir/bgmchain"
PWD="$bgmdir/bgmchain"

# Launch the arguments with the configured environment.
exec "$@"
