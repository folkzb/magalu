#!/bin/bash
set -xe

# Customize trace output for commands to stand out
PS4='\[\e[36m\]RAN COMMAND: \[\e[m\]'

SPEC_STATS=${SPEC_STATS:-../../scripts/spec_stats.py}
OAPI_DIR=${OAPI_DIR:-./openapis}
BLOCK_STORAGE=$OAPI_DIR/block-storage.openapi.yaml

# 1. Install dependencies
python3 -m pip install typing argparse pyyaml jsonschema

# 2. Run script on dir without flags
python3 $SPEC_STATS $OAPI_DIR

# 3. Run script on file without flags
python3 $SPEC_STATS $BLOCK_STORAGE

# 4. Run script on dir with filter
python3 $SPEC_STATS $OAPI_DIR --filter missing-crud

# 5. Run script on file with filter
python3 $SPEC_STATS $BLOCK_STORAGE --filter missing-crud

# 6. Run script on dir with filter-out
python3 $SPEC_STATS $OAPI_DIR --filter-out computed-variables

# 7. Run script on file with filter-out
python3 $SPEC_STATS $BLOCK_STORAGE --filter-out computed-variables

TMP_FILE_OUTPUT=./tmp_spec_dump.yaml

# 8. Run script on file with filter-out and output to file
python3 $SPEC_STATS $BLOCK_STORAGE --filter-out computed-variables -o $TMP_FILE_OUTPUT

# 9. Print file contents
cat $TMP_FILE_OUTPUT

# 10. Delete tmp file
rm -rf $TMP_FILE_OUTPUT
