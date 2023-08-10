#!/bin/bash
set -xe

# Color trace output for commands to stand out
PS4='\[\e[36m\]\+ \[\e[m\]'

MGC_CLI=${MGC_CLI:-./mgc}
D=`dirname $0`

# 1. Login
$MGC_CLI auth login

# 2. List instances with JSON output (which is the default for most commands)
$MGC_CLI virtual-machine instances list -o json

# 3. List instances with specific inline value for JSON Path
JSON_PATH="$.instances[*].name"
$MGC_CLI virtual-machine instances list -o "jsonpath=$JSON_PATH"

# 4. List instances with pre-defined JSON Path via file
JSON_PATH_FILE=$D/list-vm-instances-json-path.txt
$MGC_CLI virtual-machine instances list -o "jsonpath-file=$JSON_PATH_FILE"
