#!/bin/bash

set -xe

BASEDIR=$(dirname $0)
ROOTDIR=$(builtin cd $BASEDIR/..; pwd)
MGCDIR=${MGCDIR:-"mgc/cli"}
OAPIDIR=${OAPIDIR:-"$MGCDIR/openapis"}
OAPICUSTOMDIR=${OAPICUSTOMDIR:-"openapi-customizations"}

python3 $ROOTDIR/scripts/oapi_apply_customizations.py $OAPIDIR $OAPICUSTOMDIR
