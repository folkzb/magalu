#!/bin/bash

BASEDIR=$(dirname $0)
ROOTDIR=$(builtin cd $BASEDIR/..; pwd)
MGCDIR=${MGCDIR:-"mgc/cli/"}
OUT_FILE="script-qa/cli-dump-tree.json"

set -xe
cd $MGCDIR
go build

echo "generating $OUT_FILE..."
python3 ../../scripts/gen_expected_cli_dump_tree.py ./cli -o "../../$OUT_FILE"
echo "generating $OUT_FILE: done"
