#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
wondir="$workspace/src/github.com/worldopennet"
if [ ! -L "$wondir/go-won" ]; then
    mkdir -p "$wondir"
    cd "$wondir"
    ln -s ../../../../../. go-won
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$wondir/go-won"
PWD="$wondir/go-won"

# Launch the arguments with the configured environment.
exec go build  -gcflags='-N -l' ./cmd/geth
