#!/bin/bash
set -xe

# Customize trace output for commands to stand out
PS4='\[\e[36m\]RAN COMMAND: \[\e[m\]'

ALL_LEVELS_LOGIN_NAMESPACE="*:mgc.magalu.cloud/cli/auth.login"

# 1. Login
go run main.go auth login -l $ALL_LEVELS_LOGIN_NAMESPACE
