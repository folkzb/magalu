#!/bin/bash
set -xe

# Customize trace output for commands to stand out
PS4='\[\e[36m\]RAN COMMAND: \[\e[m\]'

DEBUG_LEVEL_ALL_NAMESPACES="debug:*"

# 1. Login
go run main.go auth login -l $DEBUG_LEVEL_ALL_NAMESPACES
