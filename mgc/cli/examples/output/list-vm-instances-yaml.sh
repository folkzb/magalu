#!/bin/bash
set -xe

# Color trace output for commands to stand out
PS4='\[\e[36m\]\+ \[\e[m\]'

MGC_CLI=${MGC_CLI:-./mgc}

# 1. Login
$MGC_CLI auth login

# 2. List instances with YAML output
$MGC_CLI virtual-machine instances list -o yaml
