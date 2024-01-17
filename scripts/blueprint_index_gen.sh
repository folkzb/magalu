#!/bin/bash

set -xe

BASEDIR=$(dirname $0)
ROOTDIR=$(builtin cd $BASEDIR/..; pwd)
MGCDIR=${MGCDIR:-"mgc/cli"}
BLUEPRINTSIDIR=${BLUEPRINTSIDIR:-"$MGCDIR/blueprints"}
BLUEPRINTSEMBED=${BLUEPRINTSEMBED:-"mgc/sdk/blueprint/embed_loader.go"}

python3 $BASEDIR/blueprint_index_gen.py $BLUEPRINTSIDIR --embed $BLUEPRINTSEMBED
