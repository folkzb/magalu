#!/bin/bash

BASEDIR=$(dirname $0)
ROOTDIR=$(builtin cd $BASEDIR/..; pwd)
MGCDIR=${MGCDIR:-"mgc/cli/"}

set -xe
cd $MGCDIR
go build

python3 ../../scripts/gen_expected_cli_help_output.py ./cli > ../../script-qa/cli-full-help-output.txt
