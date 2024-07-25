#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
#
midpath="src/tree/"
appname="tornadoAdmin"
prjdir="src/github.com/$appname"
# /User/user/go/src/test/goTest
root="$PWD"

# /User/user/go/src/test/goTest/build/_workspace/src/github.com/ethereum
# ethdir="$workspace/src/tree/ethereum"
appdir="$workspace/$prjdir"
if [ ! -L "$appdir/$appname" ]; then
    mkdir -p "$appdir"
    cd "$appdir"
    ln -s ../../../../../. $appname
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$appdir/$appname"
PWD="$appdir/$appname"

# Launch the arguments with the configured environment.
exec "$@"
