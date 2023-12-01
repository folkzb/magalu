#!/bin/bash

BASEDIR=$(dirname $0)
ROOTDIR=$(builtin cd $BASEDIR/..; pwd)
MGCDIR=${MGCDIR:-"mgc/cli/"}
OUT_DIR="script-qa/cli-help"

set -xe
cd $MGCDIR
go build

echo "generating $OUT_DIR..."
python3 ../../scripts/gen_expected_cli_help_output.py ./cli "../../$OUT_DIR"
echo "generating $OUT_DIR: done"
