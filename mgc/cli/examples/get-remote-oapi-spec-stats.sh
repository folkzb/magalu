#!/bin/bash
set -xe

# This script is relevant to magalu's OAPI's specs, directly from the services. That's why we filter-out
# 'missing-crud', as that's only relevant to this internal project

# Customize trace output for commands to stand out
PS4='\[\e[36m\]RAN COMMAND: \[\e[m\]'

SPEC_STATS=${SPEC_STATS:-../../scripts/spec_stats.py}
SPEC_URL=$1
SPEC=./remote-oapi.yaml

# 1. Install dependencies
python3 -m pip install typing argparse pyyaml jsonschema

# 2. Download remote spec
curl $SPEC_URL -o $SPEC

# 3. Run script with relevant filters
python3 $SPEC_STATS $SPEC --filter-out missing-crud --ignore-disabled=false

# 4. Delete remote spec
rm -rf $SPEC
