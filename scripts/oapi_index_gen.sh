#!/bin/bash

set -xe

BASEDIR=$(dirname $0)
ROOTDIR=$(builtin cd $BASEDIR/..; pwd)
MGCDIR=${MGCDIR:-"mgc/cli"}
OAPIDIR=${OAPIDIR:-"$MGCDIR/openapis"}
OAPIEMBED=${OAPIEMBED:-"mgc/sdk/openapi/embed_loader.go"}

python3 $BASEDIR/oapi_index_gen.py $OAPIDIR --embed $OAPIEMBED
