#!/bin/bash
set -xe

# Customize trace output for commands to stand out
PS4='\[\e[36m\]RAN COMMAND: \[\e[m\]'

MGC_CLI=${MGC_CLI:-./mgc}
D=`dirname $0`

# 1. Login
$MGC_CLI auth login

# 2. List instances with inline Go string template output
STR_TEMPLATE="{{range .instances}}Name: {{.name}}, ID: {{.id}}{{printf \"%s\" \"\n\"}}{{end}}"
$MGC_CLI virtual-machine instances list -o "template=$STR_TEMPLATE"

# 3. List instances with pre-defined Go string template via file
STR_TEMPLATE_FILE="$D/list-vm-instances-template.txt"
$MGC_CLI virtual-machine instances list -o "template-file=$STR_TEMPLATE_FILE"
