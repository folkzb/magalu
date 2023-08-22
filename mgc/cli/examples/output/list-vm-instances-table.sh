#!/bin/bash
set -xe

# Customize trace output for commands to stand out
PS4='\[\e[36m\]RAN COMMAND: \[\e[m\]'

MGC_CLI=${MGC_CLI:-./mgc}
D=`dirname $0`

# 1. Login
$MGC_CLI auth login

# 2. List instances with smart auto-definition
$MGC_CLI virtual-machine instances list -o table

# 3. List instances with smart auto-definition and max length
MAX_ROW_LENGTH=20 $MGC_CLI virtual-machine instances list -o table

# 3. Create table definition with the following format:
# <COLUMN_NAME>:<JSON-PATH>
# Concatenate multiple of the above using commas as separators
# JSON Path is a standard defined here: https://goessner.net/articles/JsonPath/
TABLE_DEFINITION="NAME:$.instances[*].name,ID:$.instances[*].id"

# 4. List instances with inline table definition
$MGC_CLI virtual-machine instances list -o "table=$TABLE_DEFINITION"

# 5. List instances with pre-defined table definition via file
TABLE_FILE="$D/list-vm-instances-table.yml"
$MGC_CLI virtual-machine instances list -o table-file=$TABLE_FILE
