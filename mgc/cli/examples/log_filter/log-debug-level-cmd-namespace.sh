#!/bin/bash
set -xe

# Color trace output for commands to stand out
PS4='\[\e[36m\]\+ \[\e[m\]'

DEBUG_LEVEL_ROOT_NAMESPACE="debug:mgc.magalu.cloud/cli/cmd"

# 1. Login
go run main.go auth login -l $DEBUG_LEVEL_ROOT_NAMESPACE
